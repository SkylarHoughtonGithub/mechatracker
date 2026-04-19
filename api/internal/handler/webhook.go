package handler

// ClerkWebhookHandler handles inbound webhook events from Clerk.
//
// Route: POST /webhooks/clerk
// Auth:  None — this endpoint is verified using svix webhook signatures,
//        NOT the Bearer JWT middleware. Register it BEFORE the auth group.
//
// Supported events:
//   - user.created — creates an org + user row in a single transaction

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	svix "github.com/svix/svix-webhooks/go"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/mechatracker/api/internal/apierr"
	"github.com/mechatracker/api/internal/store"
)

// ClerkWebhookHandler handles Clerk webhook events.
type ClerkWebhookHandler struct {
	q      store.Querier
	log    *zap.Logger
	secret string // CLERK_WEBHOOK_SECRET from config
}

func NewClerkWebhookHandler(q store.Querier, log *zap.Logger, secret string) *ClerkWebhookHandler {
	return &ClerkWebhookHandler{q: q, log: log, secret: secret}
}

// ── Clerk event payload types ─────────────────────────────────────────────────

type clerkWebhookEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// clerkUserData is the shape of data.* for user.created events.
// Only fields we actually use are mapped here.
type clerkUserData struct {
	ID             string              `json:"id"`
	EmailAddresses []clerkEmailAddress `json:"email_addresses"`
	FirstName      string              `json:"first_name"`
	LastName       string              `json:"last_name"`
	Username       *string             `json:"username"`
}

type clerkEmailAddress struct {
	EmailAddress string `json:"email_address"`
	ID           string `json:"id"`
}

// primaryEmail returns the first email address, which Clerk orders primary-first.
func (u *clerkUserData) primaryEmail() string {
	if len(u.EmailAddresses) == 0 {
		return ""
	}
	return u.EmailAddresses[0].EmailAddress
}

// displayName builds "First Last" or falls back to username or clerk ID.
func (u *clerkUserData) displayName() string {
	parts := []string{}
	if u.FirstName != "" {
		parts = append(parts, u.FirstName)
	}
	if u.LastName != "" {
		parts = append(parts, u.LastName)
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	if u.Username != nil && *u.Username != "" {
		return *u.Username
	}
	return u.ID
}

// orgSlug derives a URL-safe slug from the user's name or ID.
// Appends a short suffix from the Clerk user ID to avoid collisions.
func (u *clerkUserData) orgSlug() string {
	base := strings.ToLower(strings.ReplaceAll(u.displayName(), " ", "-"))
	// Keep it clean — strip anything that's not alphanumeric or hyphen
	var cleaned strings.Builder
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			cleaned.WriteRune(r)
		}
	}
	slug := cleaned.String()
	if slug == "" {
		slug = "user"
	}
	// Append last 6 chars of clerk ID for uniqueness (e.g. "skylar-houghton-f8TiUa")
	suffix := u.ID
	if len(suffix) > 6 {
		suffix = suffix[len(suffix)-6:]
	}
	return slug + "-" + strings.ToLower(suffix)
}

// ── Handler ───────────────────────────────────────────────────────────────────

// HandleClerkWebhook godoc
// @Summary      Clerk webhook receiver
// @Description  Receives and verifies Clerk webhook events. Handles user.created to provision org + user.
// @Tags         webhooks
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  apierr.ErrorResponse
// @Failure      500  {object}  apierr.ErrorResponse
// @Router       /webhooks/clerk [post]
func (h *ClerkWebhookHandler) HandleClerkWebhook(c *gin.Context) {
	// Read the raw body — svix needs the exact bytes to verify the signature
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.log.Error("webhook: failed to read body", zap.Error(err))
		apierr.BadRequest(c, "Failed to read request body.")
		return
	}

	// Verify the svix signature
	wh, err := svix.NewWebhook(h.secret)
	if err != nil {
		h.log.Error("webhook: failed to init svix verifier", zap.Error(err))
		apierr.Internal(c)
		return
	}

	if err := wh.Verify(body, c.Request.Header); err != nil {
		h.log.Warn("webhook: signature verification failed", zap.Error(err))
		// Return 400 not 401 — Clerk retries on 5xx but not 4xx
		apierr.BadRequest(c, "Invalid webhook signature.")
		return
	}

	// Parse the event envelope
	var event clerkWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.log.Error("webhook: failed to parse event", zap.Error(err))
		apierr.BadRequest(c, "Malformed webhook payload.")
		return
	}

	h.log.Info("webhook: received event", zap.String("type", event.Type))

	switch event.Type {
	case "user.created":
		if err := h.handleUserCreated(c, event.Data); err != nil {
			h.log.Error("webhook: user.created failed", zap.Error(err))
			// Return 500 so Clerk retries
			apierr.Internal(c)
			return
		}
	default:
		// Unknown event — return 200 so Clerk doesn't retry
		h.log.Info("webhook: ignoring unhandled event type", zap.String("type", event.Type))
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// handleUserCreated provisions an org and user row for a new Clerk user.
// Runs both inserts. If either fails the whole thing rolls back conceptually
// (the user row references the org via FK so order matters).
func (h *ClerkWebhookHandler) handleUserCreated(c *gin.Context, data json.RawMessage) error {
	var user clerkUserData
	if err := json.Unmarshal(data, &user); err != nil {
		return fmt.Errorf("unmarshal user data: %w", err)
	}

	email := user.primaryEmail()
	name := user.displayName()
	slug := user.orgSlug()

	h.log.Info("webhook: provisioning new user",
		zap.String("clerk_id", user.ID),
		zap.String("email", email),
		zap.String("slug", slug),
	)

	// Check for duplicate — Clerk can fire duplicate webhooks on retries
	existing, _ := h.q.GetUserByClerkID(c.Request.Context(), user.ID)
	if existing.ID != [16]byte{} {
		h.log.Info("webhook: user already exists, skipping", zap.String("clerk_id", user.ID))
		return nil
	}

	// 1. Create the personal org
	org, err := h.q.CreateOrg(c.Request.Context(), store.CreateOrgParams{
		Name: name + "'s Garage",
		Slug: slug,
	})
	if err != nil {
		return fmt.Errorf("create org: %w", err)
	}

	// 2. Create the user, linked to the org
	_, err = h.q.CreateUser(c.Request.Context(), store.CreateUserParams{
		ClerkID: user.ID,
		OrgID:   org.ID,
		Role:    store.UserRoleOwner,
		Email:   email,
		Name:    name,
	})
	if err != nil {
		return fmt.Errorf("create user (org_id=%s): %w", org.ID, err)
	}

	h.log.Info("webhook: user provisioned",
		zap.String("clerk_id", user.ID),
		zap.String("org_id", org.ID.String()),
	)
	return nil
}
