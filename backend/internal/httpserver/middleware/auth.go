package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"fpgwiki/backend/internal/config"
	"fpgwiki/backend/internal/models"
	"fpgwiki/backend/internal/repository"
)

const (
	contextUserIDKey   = "user_id"
	contextUserRoleKey = "user_role"
	contextUserKey     = "user"
)

func RequireAuth(cfg config.Config, userRepo repository.UserRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := strings.TrimSpace(c.GetHeader("Authorization"))
		if authorization == "" {
			writeErr(c, http.StatusUnauthorized, "missing_token", "authorization token is required")
			c.Abort()
			return
		}

		tokenString, ok := extractBearerToken(authorization)
		if !ok {
			writeErr(c, http.StatusUnauthorized, "invalid_token", "authorization token is invalid")
			c.Abort()
			return
		}

		claims := &models.JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			writeErr(c, http.StatusUnauthorized, "invalid_token", "authorization token is invalid")
			c.Abort()
			return
		}

		userID, err := uuid.Parse(claims.Sub)
		if err != nil {
			writeErr(c, http.StatusUnauthorized, "invalid_token", "authorization token is invalid")
			c.Abort()
			return
		}

		user, err := userRepo.FindByID(c.Request.Context(), userID)
		if err != nil {
			writeErr(c, http.StatusInternalServerError, "internal_error", "internal server error")
			c.Abort()
			return
		}
		if user == nil {
			writeErr(c, http.StatusUnauthorized, "user_not_found", "user not found")
			c.Abort()
			return
		}

		if claims.TV != user.TokenVersion {
			writeErr(c, http.StatusUnauthorized, "token_revoked", "authorization token has been revoked")
			c.Abort()
			return
		}

		c.Set(contextUserIDKey, user.ID)
		c.Set(contextUserRoleKey, user.Role)
		c.Set(contextUserKey, user)
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := GetUserRole(c)
		for _, allowedRole := range roles {
			if role == allowedRole {
				c.Next()
				return
			}
		}

		writeErr(c, http.StatusForbidden, "forbidden", "forbidden")
		c.Abort()
	}
}

func OptionalAuth(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		_ = cfg
		c.Next()
	}
}

func GetUserID(c *gin.Context) uuid.UUID {
	value, ok := c.Get(contextUserIDKey)
	if !ok {
		return uuid.Nil
	}

	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}

	return userID
}

func GetUserRole(c *gin.Context) string {
	value, ok := c.Get(contextUserRoleKey)
	if !ok {
		return ""
	}

	role, ok := value.(string)
	if !ok {
		return ""
	}

	return role
}

func GetUser(c *gin.Context) *models.User {
	value, ok := c.Get(contextUserKey)
	if !ok {
		return nil
	}

	user, ok := value.(*models.User)
	if ok {
		return user
	}

	userValue, ok := value.(models.User)
	if ok {
		return &userValue
	}

	return nil
}

func extractBearerToken(authorization string) (string, bool) {
	authorization = strings.TrimSpace(authorization)
	if authorization == "" {
		return "", false
	}

	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 {
		return "", false
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}

	return token, true
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	Success bool           `json:"success"`
	Data    any            `json:"data"`
	Error   *errorResponse `json:"error"`
}

func writeErr(c *gin.Context, status int, code string, message string) {
	c.JSON(status, envelope{
		Success: false,
		Data:    nil,
		Error: &errorResponse{
			Code:    code,
			Message: message,
		},
	})
}
