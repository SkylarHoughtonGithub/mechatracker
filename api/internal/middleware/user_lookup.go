package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/mechatracker/api/internal/apierr"
	"github.com/mechatracker/api/internal/store"
)

// RequireUser looks up the users row by the Google subject set by Auth and
// stores the internal user_id + org_id + role on the context. Aborts with 401
// if no users row exists (the caller is signed in with Google but hasn't
// onboarded yet; they should go through POST /orgs or POST /invites/{token}/accept).
//
// Must chain after Auth.
//
// If the Auth stage already populated KeyUserID (post-onboarding session),
// this is a no-op — the session already carries the lookup result.
func RequireUser(q store.Querier, log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString(KeyUserID) != "" {
			c.Next()
			return
		}
		sub := c.GetString(KeyGoogleSub)
		if sub == "" {
			apierr.Unauthorized(c, "No user context.")
			return
		}
		user, err := q.GetUserByGoogleSub(c.Request.Context(), sub)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.Unauthorized(c, "Account has no workspace yet. Create or join one at /orgs or /invites/{token}/accept.")
				return
			}
			log.Error("user lookup failed", zap.Error(err))
			apierr.Internal(c)
			return
		}
		c.Set(KeyUserID, user.ID.String())
		c.Set(KeyOrgID, user.OrgID.String())
		c.Set(KeyRole, string(user.Role))
		c.Next()
	}
}
