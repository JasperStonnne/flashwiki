package middleware

import (
	"github.com/gin-gonic/gin"

	"fpgwiki/backend/internal/config"
)

func RequireAuth(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		_ = cfg
		c.Next()
	}
}

func OptionalAuth(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		_ = cfg
		c.Next()
	}
}
