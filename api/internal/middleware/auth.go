package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/api/idtoken"

	"github.com/mechatracker/api/internal/apierr"
	"github.com/mechatracker/api/internal/config"
	"github.com/mechatracker/api/internal/session"
)

// Context keys — use these to read values downstream in handlers.
const (
	KeyGoogleSub           = "google_sub"
	KeyGoogleEmail         = "google_email"
	KeyGoogleEmailVerified = "google_email_verified"
	KeyGoogleName          = "google_name"
	KeyGooglePicture       = "google_picture"

	// Set after RequireUser (or after Auth when a signed-in session carries
	// post-onboarding claims). Empty for pre-onboarding sessions.
	KeyUserID = "user_id"
	KeyOrgID  = "org_id"
	KeyRole   = "role"

	KeyRequestID = "request_id"
)

// Auth validates one of two credentials, in order:
//
//  1. The session cookie (minted by POST /auth/google and carrying Google
//     claims + optionally the internal user_id + org_id + role).
//  2. An `Authorization: Bearer <id-token>` header, verified against Google's
//     JWKS. Used by POST /auth/google itself and by non-browser clients.
//
// On success it sets the Google-identity context keys. If the credential also
// carries user_id/org_id/role (post-onboarding session), those are set too.
// This middleware does NOT require a users row; chain RequireUser after it
// for routes that need one.
//
// Dev bypass: when cfg.DevAuthBypass is true (ENV=development only) credential
// verification is skipped and the dev seed user is injected directly.
func Auth(cfg *config.Config, log *zap.Logger) gin.HandlerFunc {
	const (
		DevUserID     = "00000000-0000-0000-0000-000000000002"
		DevOrgID      = "00000000-0000-0000-0000-000000000003"
		DevGoogleSub  = "dev_google_sub_local"
		DevEmail      = "dev@mechatracker.local"
		DevName       = "Dev User"
	)

	if cfg.DevAuthBypass {
		log.Warn("⚠️  Auth is BYPASSED — dev mode only. Never set DEV_AUTH_BYPASS in staging/prod.")
		return func(c *gin.Context) {
			c.Set(KeyGoogleSub, DevGoogleSub)
			c.Set(KeyGoogleEmail, DevEmail)
			c.Set(KeyGoogleEmailVerified, true)
			c.Set(KeyGoogleName, DevName)
			c.Set(KeyUserID, DevUserID)
			c.Set(KeyOrgID, DevOrgID)
			c.Set(KeyRole, "owner")
			c.Next()
		}
	}

	signingKey := []byte(cfg.SessionSigningKey)

	return func(c *gin.Context) {
		// 1. Session cookie
		if raw, err := c.Cookie(session.CookieName); err == nil && raw != "" {
			claims, perr := session.Parse(signingKey, raw)
			if perr != nil {
				log.Debug("session parse failed", zap.Error(perr))
				apierr.Unauthorized(c, "Invalid or expired session.")
				return
			}
			setFromSession(c, claims)
			c.Next()
			return
		}

		// 2. Bearer Google ID token
		if raw := extractBearer(c.GetHeader("Authorization")); raw != "" {
			payload, verr := idtoken.Validate(c.Request.Context(), raw, cfg.GoogleClientID)
			if verr != nil {
				log.Debug("google id-token validation failed", zap.Error(verr))
				apierr.Unauthorized(c, "Invalid or expired token.")
				return
			}
			if cfg.GoogleAllowedHD != "" {
				if hd, _ := payload.Claims["hd"].(string); hd != cfg.GoogleAllowedHD {
					apierr.Forbidden(c, "This Google account is not permitted.")
					return
				}
			}
			setFromGooglePayload(c, payload.Subject, payload.Claims)
			c.Next()
			return
		}

		apierr.Unauthorized(c, "Missing session cookie or Authorization header.")
	}
}

// setFromSession copies session claims onto the gin context.
func setFromSession(c *gin.Context, claims *session.Claims) {
	c.Set(KeyGoogleSub, claims.Subject)
	c.Set(KeyGoogleEmail, claims.Email)
	c.Set(KeyGoogleEmailVerified, claims.EmailVerified)
	c.Set(KeyGoogleName, claims.Name)
	c.Set(KeyGooglePicture, claims.PictureURL)
	if claims.UserID != "" {
		c.Set(KeyUserID, claims.UserID)
		c.Set(KeyOrgID, claims.OrgID)
		c.Set(KeyRole, claims.Role)
	}
}

// setFromGooglePayload copies a verified Google ID-token payload onto the
// gin context. Used when a non-browser client hits an endpoint directly with
// Bearer auth instead of going through POST /auth/google first.
func setFromGooglePayload(c *gin.Context, sub string, claims map[string]any) {
	c.Set(KeyGoogleSub, sub)
	if email, _ := claims["email"].(string); email != "" {
		c.Set(KeyGoogleEmail, email)
	}
	if verified, _ := claims["email_verified"].(bool); verified {
		c.Set(KeyGoogleEmailVerified, verified)
	}
	if name, _ := claims["name"].(string); name != "" {
		c.Set(KeyGoogleName, name)
	}
	if picture, _ := claims["picture"].(string); picture != "" {
		c.Set(KeyGooglePicture, picture)
	}
}

// RequireOrg aborts with 403 if the request does not carry an active org
// (i.e. the user has signed in but not onboarded). Use on routes that must
// run inside an org.
func RequireOrg(c *gin.Context) {
	if c.GetString(KeyOrgID) == "" {
		apierr.Forbidden(c, "This endpoint requires an active organization context.")
		return
	}
	c.Next()
}

// RequireRole aborts with 403 unless the caller holds one of the given roles.
// Must run after Auth + RequireUser.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		have := c.GetString(KeyRole)
		for _, r := range roles {
			if have == r {
				c.Next()
				return
			}
		}
		apierr.Forbidden(c, "This endpoint requires one of: "+strings.Join(roles, ", "))
	}
}

// extractBearer pulls the token from "Bearer <token>" or returns "".
func extractBearer(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
