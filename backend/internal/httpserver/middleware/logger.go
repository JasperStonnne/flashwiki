package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	applogger "fpgwiki/backend/internal/logger"
)

func Logger(log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		log.Info().
			Str(applogger.RequestIDField, requestIDFromContext(c)).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Int64(applogger.LatencyMSField, time.Since(start).Milliseconds()).
			Msg("request completed")
	}
}
