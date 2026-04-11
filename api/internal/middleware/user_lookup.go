package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/mechatracker/api/internal/apierr"
	"github.com/mechatracker/api/internal/store"
	"go.uber.org/zap"
)

// UserLookup resolves the Clerk subject string (KeyUserID) to an internal
// users.id UUID and stores it under KeyInternalUserID.
// Must run after auth middleware
func UserLookup(q store.Querier, log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		clerkID := c.GetString(KeyUserID)
		if clerkID == "" {
			apierr.Unauthorized(c, "No user context.")
			return
		}

		user, err := q.GetUserByClerkID(c.Request.Context(), clerkID)
		if err != nil {
			apierr.Unauthorized(c, "User not found. Please complete signup")
			return
		}

		c.Set(KeyInternalUserID, user.ID.String())
		c.Set(KeyOrgID, user.OrgID.String())
		c.Next()
	}
}
