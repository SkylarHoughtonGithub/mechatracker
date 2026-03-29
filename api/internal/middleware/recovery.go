package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/mechatracker/api/internal/apierr"
)

// Recovery catches any panic in a downstream handler, logs it at Error level,
// and returns a structured 500 via apierr rather than Gin's default HTML page.
func Recovery(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("panic recovered",
					zap.Any("error", err),
					zap.String("request_id", c.GetString(KeyRequestID)),
					zap.String("path", c.Request.URL.Path),
				)
				apierr.Internal(c)
			}
		}()
		c.Next()
	}
}
