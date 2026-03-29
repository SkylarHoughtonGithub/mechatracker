package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const headerRequestID = "X-Request-ID"

// RequestID sets a unique request ID on each request. It reads an existing
// X-Request-ID header from the client (useful when a load balancer sets one)
// and falls back to a randomly generated ID. The value is echoed back in the
// response header and stored in the Gin context under KeyRequestID.
func RequestID(c *gin.Context) {
	id := c.GetHeader(headerRequestID)
	if id == "" {
		id = newRequestID()
	}
	c.Set(KeyRequestID, id)
	c.Header(headerRequestID, id)
	c.Next()
}

func newRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
