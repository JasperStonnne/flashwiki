package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDKey = "request_id"
const requestIDHeader = "X-Request-ID"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(requestIDHeader))
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set(requestIDKey, requestID)
		c.Writer.Header().Set(requestIDHeader, requestID)
		c.Next()
	}
}

func requestIDFromContext(c *gin.Context) string {
	value, ok := c.Get(requestIDKey)
	if !ok {
		return ""
	}

	requestID, ok := value.(string)
	if !ok {
		return ""
	}

	return requestID
}
