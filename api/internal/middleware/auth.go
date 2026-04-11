package middleware

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"go.uber.org/zap"

	"github.com/mechatracker/api/internal/apierr"
	"github.com/mechatracker/api/internal/config"
)

// Context keys — use these to read values downstream in handlers.
const (
	KeyInternalUserID = "internal_user_id" // Clerk user ID, e.g. "user_2abc..."
	KeyUserID         = "user_id"          // Clerk user ID, e.g. "user_2abc..."
	KeyOrgID          = "org_id"           // Clerk org ID, e.g. "org_2abc..." (may be empty)
	KeyOrgRole        = "org_role"         // "org:admin" | "org:member" etc.
	KeyRequestID      = "request_id"       // set by RequestID middleware
)

// clerkClaims maps the JWT claims issued by Clerk.
type clerkClaims struct {
	jwt.RegisteredClaims
	// Active org context — present when user has selected an org in the
	// Clerk session. Handlers should treat missing org as "personal" context.
	OrgID   string `json:"org_id"`
	OrgRole string `json:"org_role"`
	OrgSlug string `json:"org_slug"`
}

// Auth returns a Gin middleware that verifies the Bearer JWT issued by Clerk.
//
// On success it sets KeyUserID, KeyOrgID, KeyOrgRole in the Gin context.
// On failure it aborts with 401 using the apierr envelope.
//
// Dev bypass: when cfg.ClerkDevBypass is true (ENV=development +
// CLERK_DEV_BYPASS=true) JWT verification is skipped entirely and a synthetic
// user is injected. This lets the API run locally without a Clerk account.
//
// JWKS are fetched from cfg.ClerkJWKSURL on first use and cached for 1 hour
// so we don't hammer Clerk on every request.
func Auth(cfg *config.Config, log *zap.Logger) gin.HandlerFunc {
	const (
		DevUserID = "00000000-0000-0000-0000-000000000002"
		DevOrgID  = "00000000-0000-0000-0000-000000000003"
	)

	if cfg.ClerkDevBypass {
		log.Warn("⚠️  Clerk auth is BYPASSED — dev mode only. Never set CLERK_DEV_BYPASS in staging/prod.")
		return func(c *gin.Context) {
			c.Set(KeyUserID, DevUserID)
			c.Set(KeyOrgID, DevOrgID)
			c.Set(KeyOrgRole, "org:admin")
			c.Next()
		}
	}

	cache := &jwksCache{url: cfg.ClerkJWKSURL, log: log}

	return func(c *gin.Context) {
		rawToken := extractBearer(c.GetHeader("Authorization"))
		if rawToken == "" {
			apierr.Unauthorized(c, "Missing or malformed Authorization header.")
			return
		}

		keyFunc := func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			kid, _ := token.Header["kid"].(string)
			return cache.getKey(kid)
		}

		opts := []jwt.ParserOption{
			jwt.WithExpirationRequired(),
		}
		if cfg.ClerkAudience != "" {
			opts = append(opts, jwt.WithAudience(cfg.ClerkAudience))
		}

		token, err := jwt.ParseWithClaims(rawToken, &clerkClaims{}, keyFunc, opts...)

		if err != nil || !token.Valid {
			log.Debug("JWT validation failed", zap.Error(err))
			apierr.Unauthorized(c, "Invalid or expired token.")
			return
		}

		claims, ok := token.Claims.(*clerkClaims)
		if !ok {
			apierr.Unauthorized(c, "Malformed token claims.")
			return
		}

		c.Set(KeyUserID, claims.Subject)
		c.Set(KeyOrgID, claims.OrgID)
		c.Set(KeyOrgRole, claims.OrgRole)

		c.Next()
	}
}

// RequireOrg is an additional middleware (applied after Auth) that aborts
// with 403 if the request does not carry an active Clerk org context.
// Use on routes that require an org (shop/fleet operations).
func RequireOrg(c *gin.Context) {
	if c.GetString(KeyOrgID) == "" {
		apierr.Forbidden(c, "This endpoint requires an active organization context.")
		return
	}
	c.Next()
}

// extractBearer pulls the token from "Bearer <token>" or returns "".
func extractBearer(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// ── JWKS cache ────────────────────────────────────────────────────────────────

type jwksCache struct {
	url  string
	log  *zap.Logger
	mu   sync.RWMutex
	keys map[string]any // kid → rsa.PublicKey
	exp  time.Time
}

func (c *jwksCache) getKey(kid string) (any, error) {
	c.mu.RLock()
	if time.Now().Before(c.exp) {
		if key, ok := c.keys[kid]; ok {
			c.mu.RUnlock()
			return key, nil
		}
	}
	c.mu.RUnlock()

	// Refresh
	if err := c.refresh(); err != nil {
		return nil, fmt.Errorf("jwks: refresh failed: %w", err)
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	key, ok := c.keys[kid]
	if !ok {
		return nil, fmt.Errorf("jwks: unknown kid %q", kid)
	}
	return key, nil
}

func (c *jwksCache) refresh() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	set, err := jwk.Fetch(ctx, c.url)
	if err != nil {
		return fmt.Errorf("jwks: fetch failed: %w", err)
	}

	keys := make(map[string]any, set.Len())
	for i := 0; i < set.Len(); i++ {
		key, ok := set.Key(i)
		if !ok {
			continue
		}
		var raw any
		if err := key.Raw(&raw); err != nil {
			c.log.Warn("jwks: failed to extract raw key",
				zap.String("kid", key.KeyID()),
				zap.Error(err),
			)
			continue
		}
		keys[key.KeyID()] = raw
	}

	c.mu.Lock()
	c.keys = keys
	c.exp = time.Now().Add(time.Hour)
	c.mu.Unlock()

	c.log.Info("jwks: refreshed", zap.Int("keys", len(keys)))
	return nil
}
