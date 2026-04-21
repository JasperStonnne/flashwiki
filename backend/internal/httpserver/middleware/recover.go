package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func Recover(log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error().
					Str("request_id", requestIDFromContext(c)).
					Str("method", c.Request.Method).
					Str("path", c.Request.URL.Path).
					Str("panic", fmt.Sprint(rec)).
					Msg("panic recovered")

				writeErr(c, http.StatusInternalServerError, "internal_error", "internal server error")
				c.Abort()
			}
		}()

		c.Next()
	}
}
