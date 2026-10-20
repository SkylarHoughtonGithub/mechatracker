package handler

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/mechatracker/api/internal/apierr"
	"github.com/mechatracker/api/internal/config"
	"github.com/mechatracker/api/internal/middleware"
	"github.com/mechatracker/api/internal/session"
	"github.com/mechatracker/api/internal/store"
)

// OnboardingHandler owns the first-time setup flow: creating an org, inviting
// members, and accepting invites. All three endpoints end with a fresh session
// cookie carrying the post-onboarding claims so the client can land in-app
// immediately.
type OnboardingHandler struct {
	q   store.Querier
	cfg *config.Config
	log *zap.Logger
}

func NewOnboardingHandler(q store.Querier, cfg *config.Config, log *zap.Logger) *OnboardingHandler {
	return &OnboardingHandler{q: q, cfg: cfg, log: log}
}

type createOrgRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

// CreateOrg creates a new personal org for the signed-in Google account and
// a users row with role=owner, in that order. Rejects if the Google account
// already has a users row (single-org-per-user).
//
// POST /orgs
func (h *OnboardingHandler) CreateOrg(c *gin.Context) {
	if c.GetString(middleware.KeyUserID) != "" {
		apierr.Conflict(c, "This Google account is already in a workspace.")
		return
	}

	sub := c.GetString(middleware.KeyGoogleSub)
	g := googleFromCtx(c)
	if sub == "" || g.Email == "" {
		apierr.Unauthorized(c, "No Google credential on request.")
		return
	}
	if !g.EmailVerified {
		apierr.Forbidden(c, "Google email must be verified before creating a workspace.")
		return
	}

	var req createOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	slug := uniqueSlug(req.Name, sub)
	org, err := h.q.CreateOrg(c.Request.Context(), store.CreateOrgParams{Name: req.Name, Slug: slug})
	if err != nil {
		h.log.Error("create org failed", zap.Error(err))
		apierr.Internal(c)
		return
	}

	user, err := h.q.CreateUser(c.Request.Context(), store.CreateUserParams{
		GoogleSub:     sub,
		OrgID:         org.ID,
		Role:          store.UserRoleOwner,
		Email:         g.Email,
		EmailVerified: g.EmailVerified,
		Name:          g.Name,
		PictureUrl:    strPtrOrNil(g.PictureURL),
	})
	if err != nil {
		h.log.Error("create user failed", zap.Error(err))
		apierr.Internal(c)
		return
	}

	if err := h.rotateSession(c, sub, g, user); err != nil {
		return
	}

	c.JSON(http.StatusCreated, meResponse{
		User: userToDTO(user),
		Org:  orgToDTO(org),
	})
}

type createInviteRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role"  binding:"required,oneof=admin member"`
}

type createInviteResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// CreateInvite issues a single-use invite token with 24h expiry. Owner/admin
// only. The token is returned in the response for now; later this becomes an
// email send.
//
// POST /orgs/invites
func (h *OnboardingHandler) CreateInvite(c *gin.Context) {
	orgIDStr := c.GetString(middleware.KeyOrgID)
	callerIDStr := c.GetString(middleware.KeyUserID)
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		apierr.Unauthorized(c, "Malformed org ID.")
		return
	}
	callerID, err := uuid.Parse(callerIDStr)
	if err != nil {
		apierr.Unauthorized(c, "Malformed user ID.")
		return
	}

	var req createInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	token, err := randomToken()
	if err != nil {
		h.log.Error("invite token generation failed", zap.Error(err))
		apierr.Internal(c)
		return
	}

	expires := time.Now().Add(24 * time.Hour)
	inv, err := h.q.CreateInvite(c.Request.Context(), store.CreateInviteParams{
		OrgID:      orgID,
		Email:      strings.ToLower(req.Email),
		Role:       store.UserRole(req.Role),
		Token:      token,
		InvitedBy:  callerID,
		ExpiresAt:  pgtype.Timestamptz{Time: expires, Valid: true},
	})
	if err != nil {
		h.log.Error("create invite failed", zap.Error(err))
		apierr.Internal(c)
		return
	}

	c.JSON(http.StatusCreated, createInviteResponse{
		ID:        inv.ID.String(),
		Email:     inv.Email,
		Role:      string(inv.Role),
		Token:     inv.Token,
		ExpiresAt: inv.ExpiresAt.Time,
	})
}

// AcceptInvite consumes a pending invite and creates a users row for the
// signed-in Google account scoped to the invite's org with the invite's role.
// Rejects if the account already has a users row.
//
// POST /invites/:token/accept
func (h *OnboardingHandler) AcceptInvite(c *gin.Context) {
	if c.GetString(middleware.KeyUserID) != "" {
		apierr.Conflict(c, "This Google account is already in a workspace. Sign out and use a different Google account to accept this invite.")
		return
	}

	sub := c.GetString(middleware.KeyGoogleSub)
	g := googleFromCtx(c)
	if sub == "" || g.Email == "" {
		apierr.Unauthorized(c, "No Google credential on request.")
		return
	}
	if !g.EmailVerified {
		apierr.Forbidden(c, "Google email must be verified before accepting an invite.")
		return
	}

	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		apierr.BadRequest(c, "Missing invite token.")
		return
	}

	inv, err := h.q.GetInviteByToken(c.Request.Context(), token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierr.NotFound(c, "Invite not found.")
			return
		}
		h.log.Error("get invite failed", zap.Error(err))
		apierr.Internal(c)
		return
	}

	if inv.ConsumedAt.Valid {
		apierr.Conflict(c, "This invite has already been used.")
		return
	}
	if time.Now().After(inv.ExpiresAt.Time) {
		c.JSON(http.StatusGone, gin.H{"error": "This invite has expired."})
		return
	}
	if !strings.EqualFold(inv.Email, g.Email) {
		apierr.Forbidden(c, "This invite was issued to a different email address.")
		return
	}

	user, err := h.q.CreateUser(c.Request.Context(), store.CreateUserParams{
		GoogleSub:     sub,
		OrgID:         inv.OrgID,
		Role:          inv.Role,
		Email:         g.Email,
		EmailVerified: g.EmailVerified,
		Name:          g.Name,
		PictureUrl:    strPtrOrNil(g.PictureURL),
	})
	if err != nil {
		h.log.Error("create user from invite failed", zap.Error(err))
		apierr.Internal(c)
		return
	}

	if _, err := h.q.ConsumeInvite(c.Request.Context(), inv.ID); err != nil {
		// User row already exists at this point. Log the invite leak but
		// don't fail the request — the user is in. A background sweeper
		// could clean up (future).
		h.log.Warn("consume invite failed; user created but invite not marked consumed",
			zap.String("invite_id", inv.ID.String()),
			zap.Error(err),
		)
	}

	org, oerr := h.q.GetOrgByID(c.Request.Context(), inv.OrgID)
	if oerr != nil {
		h.log.Error("get org failed", zap.Error(oerr))
		apierr.Internal(c)
		return
	}

	if err := h.rotateSession(c, sub, g, user); err != nil {
		return
	}

	c.JSON(http.StatusCreated, meResponse{
		User: userToDTO(user),
		Org:  orgToDTO(org),
	})
}

// rotateSession mints a fresh post-onboarding session (with user_id/org_id/role)
// and sets the cookie.
func (h *OnboardingHandler) rotateSession(c *gin.Context, sub string, g googleDTO, u store.User) error {
	tok, err := session.Mint([]byte(h.cfg.SessionSigningKey), session.Claims{
		RegisteredClaims: jwtSubject(sub),
		Email:            g.Email,
		EmailVerified:    g.EmailVerified,
		Name:             g.Name,
		PictureURL:       g.PictureURL,
		UserID:           u.ID.String(),
		OrgID:            u.OrgID.String(),
		Role:             string(u.Role),
	}, h.cfg.SessionTTL)
	if err != nil {
		h.log.Error("session mint failed", zap.Error(err))
		apierr.Internal(c)
		return err
	}
	http.SetCookie(c.Writer, session.Cookie(tok, h.cfg.SessionTTL, h.cfg.SessionCookieDomain, h.cfg.SessionCookieSecure))
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

var slugCleaner = regexp.MustCompile(`[^a-z0-9-]+`)

// uniqueSlug derives a URL-safe slug from the org name and appends 6 chars of
// the google_sub for uniqueness.
func uniqueSlug(name, sub string) string {
	base := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))
	base = slugCleaner.ReplaceAllString(base, "")
	if base == "" {
		base = "workspace"
	}
	suffix := sub
	if len(suffix) > 6 {
		suffix = suffix[len(suffix)-6:]
	}
	return base + "-" + strings.ToLower(suffix)
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
