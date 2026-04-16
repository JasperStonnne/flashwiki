package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"fpgwiki/backend/internal/db"
)

type pingResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

func Health(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := db.HealthCheck(c.Request.Context(), pool); err != nil {
			WriteErr(c, http.StatusServiceUnavailable, "db_unavailable", safeErrorMessage(err))
			return
		}

		WriteOK(c, http.StatusOK, pingResponse{
			Status: "ok",
			DB:     "ok",
		})
	}
}

func safeErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			break
		}
		err = unwrapped
	}

	message := strings.TrimSpace(err.Error())
	if message == "" || looksSensitive(message) {
		message = "database health check failed"
	}
	if len(message) > 100 {
		return message[:100]
	}
	return message
}

func looksSensitive(message string) bool {
	lower := strings.ToLower(message)
	sensitiveParts := []string{
		"postgres://",
		"postgresql://",
		"host=",
		"port=",
		"password",
		"dial",
		"connect",
	}

	for _, part := range sensitiveParts {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return false
}
