package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger returns a Gin middleware that emits one structured log line per
// request using the provided zap logger. Fields mirror what you'd find in
// a typical access log plus Gin-specific extras.
//
// Log level:
//   - 5xx  → Error
//   - 4xx  → Warn
//   - 2xx/3xx → Info
func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next() // process request

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
			zap.String("request_id", c.GetString(KeyRequestID)),
		}

		// Include user context when available (set by Auth middleware).
		if userID := c.GetString(KeyUserID); userID != "" {
			fields = append(fields, zap.String("user_id", userID))
		}
		if orgID := c.GetString(KeyOrgID); orgID != "" {
			fields = append(fields, zap.String("org_id", orgID))
		}

		msg := path
		switch {
		case status >= 500:
			log.Error(msg, fields...)
		case status >= 400:
			log.Warn(msg, fields...)
		default:
			log.Info(msg, fields...)
		}
	}
}
