package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/mechatracker/api/internal/apierr"
	"github.com/mechatracker/api/internal/config"
	"github.com/mechatracker/api/internal/middleware"
	"github.com/mechatracker/api/internal/session"
	"github.com/mechatracker/api/internal/store"
)

// AuthHandler owns the session lifecycle: Google sign-in, logout, and /me.
type AuthHandler struct {
	q   store.Querier
	cfg *config.Config
	log *zap.Logger
}

func NewAuthHandler(q store.Querier, cfg *config.Config, log *zap.Logger) *AuthHandler {
	return &AuthHandler{q: q, cfg: cfg, log: log}
}

// userDTO is the stable public shape returned by every auth-ish endpoint.
type userDTO struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	PictureURL    string `json:"picture_url,omitempty"`
	Role          string `json:"role"`
	OrgID         string `json:"org_id"`
}

type orgDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Plan string `json:"plan"`
}

type meResponse struct {
	User            *userDTO    `json:"user"`
	Org             *orgDTO     `json:"org,omitempty"`
	NeedsOnboarding bool        `json:"needs_onboarding"`
	Google          *googleDTO  `json:"google,omitempty"`
}

type googleDTO struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	PictureURL    string `json:"picture_url,omitempty"`
}

// Google exchanges a verified Google ID token (carried as Bearer by Auth
// middleware) for a session cookie. Returns the user + org if the Google
// account already has a users row; otherwise returns needs_onboarding so the
// client can show the create-or-join screen.
//
// POST /auth/google
func (h *AuthHandler) Google(c *gin.Context) {
	sub := c.GetString(middleware.KeyGoogleSub)
	if sub == "" {
		apierr.Unauthorized(c, "No Google credential on request.")
		return
	}

	g := googleFromCtx(c)

	user, err := h.q.GetUserByGoogleSub(c.Request.Context(), sub)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		h.log.Error("get user by google sub failed", zap.Error(err))
		apierr.Internal(c)
		return
	}

	if errors.Is(err, pgx.ErrNoRows) {
		// Pre-onboarding session: carries only Google claims.
		if err := h.setSession(c, session.Claims{
			RegisteredClaims: jwtSubject(sub),
			Email:            g.Email,
			EmailVerified:    g.EmailVerified,
			Name:             g.Name,
			PictureURL:       g.PictureURL,
		}); err != nil {
			return
		}
		c.JSON(http.StatusOK, meResponse{
			NeedsOnboarding: true,
			Google:          &g,
		})
		return
	}

	org, oerr := h.q.GetOrgByID(c.Request.Context(), user.OrgID)
	if oerr != nil {
		h.log.Error("get org by id failed", zap.Error(oerr))
		apierr.Internal(c)
		return
	}

	if err := h.setSession(c, session.Claims{
		RegisteredClaims: jwtSubject(sub),
		Email:            g.Email,
		EmailVerified:    g.EmailVerified,
		Name:             g.Name,
		PictureURL:       g.PictureURL,
		UserID:           user.ID.String(),
		OrgID:            user.OrgID.String(),
		Role:             string(user.Role),
	}); err != nil {
		return
	}
	c.JSON(http.StatusOK, meResponse{
		User: userToDTO(user),
		Org:  orgToDTO(org),
	})
}

// Logout clears the session cookie.
//
// POST /auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	http.SetCookie(c.Writer, session.Cookie("", 0, h.cfg.SessionCookieDomain, h.cfg.SessionCookieSecure))
	c.Status(http.StatusNoContent)
}

// Me returns the current user + org, or needs_onboarding + Google claims if
// the signed-in Google account has no users row yet.
//
// GET /auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	g := googleFromCtx(c)
	if uidStr := c.GetString(middleware.KeyUserID); uidStr != "" {
		user, err := h.q.GetUserByGoogleSub(c.Request.Context(), c.GetString(middleware.KeyGoogleSub))
		if err != nil {
			h.log.Error("get user failed", zap.Error(err))
			apierr.Internal(c)
			return
		}
		org, oerr := h.q.GetOrgByID(c.Request.Context(), user.OrgID)
		if oerr != nil {
			h.log.Error("get org failed", zap.Error(oerr))
			apierr.Internal(c)
			return
		}
		c.JSON(http.StatusOK, meResponse{
			User: userToDTO(user),
			Org:  orgToDTO(org),
		})
		return
	}
	c.JSON(http.StatusOK, meResponse{
		NeedsOnboarding: true,
		Google:          &g,
	})
}

// setSession mints a new session JWT and attaches it as an httpOnly cookie.
// Returns false (and writes an error response) on failure.
func (h *AuthHandler) setSession(c *gin.Context, claims session.Claims) error {
	tok, err := session.Mint([]byte(h.cfg.SessionSigningKey), claims, h.cfg.SessionTTL)
	if err != nil {
		h.log.Error("session mint failed", zap.Error(err))
		apierr.Internal(c)
		return err
	}
	http.SetCookie(c.Writer, session.Cookie(tok, h.cfg.SessionTTL, h.cfg.SessionCookieDomain, h.cfg.SessionCookieSecure))
	return nil
}
