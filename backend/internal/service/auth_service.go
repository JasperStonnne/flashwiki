package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"fpgwiki/backend/internal/config"
	"fpgwiki/backend/internal/models"
	"fpgwiki/backend/internal/repository"
)

var (
	ErrEmailTaken         = errors.New("email_taken")
	ErrInvalidCredentials = errors.New("invalid_credentials")
	ErrInvalidToken       = errors.New("invalid_token")
	ErrTokenExpired       = errors.New("token_expired")
	ErrTokenReused        = errors.New("token_reused")
	ErrInvalidPassword    = errors.New("invalid_password")
	ErrUserNotFound       = errors.New("user_not_found")
)

const (
	minPasswordLength = 8
	minDisplayNameLen = 1
	maxDisplayNameLen = 50
	refreshTokenBytes = 32
	bcryptCost        = 12
)

type AuthService interface {
	Register(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error)
	Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error)
	Refresh(ctx context.Context, req models.RefreshRequest) (*models.TokenResponse, error)
	Logout(ctx context.Context, refreshTokenHex string) error
	ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) (*models.TokenResponse, error)
}

type authService struct {
	userRepo         repository.UserRepo
	refreshTokenRepo repository.RefreshTokenRepo
	cfg              config.Config

	now        func() time.Time
	randReader io.Reader
	bcryptCost int
}

func NewAuthService(userRepo repository.UserRepo, rtRepo repository.RefreshTokenRepo, cfg config.Config) AuthService {
	return &authService{
		userRepo:         userRepo,
		refreshTokenRepo: rtRepo,
		cfg:              cfg,
		now:              time.Now,
		randReader:       rand.Reader,
		bcryptCost:       bcryptCost,
	}
}

func (s *authService) Register(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error) {
	email := normalizeEmail(req.Email)
	displayName := strings.TrimSpace(req.DisplayName)

	if err := validateRegisterInput(email, req.Password, displayName); err != nil {
		return nil, err
	}

	existing, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Email:        email,
		PasswordHash: string(passwordHash),
		DisplayName:  displayName,
		Role:         "member",
		Locale:       "zh",
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	accessToken, refreshToken, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         sanitizeUserForResponse(user),
	}, nil
}

func (s *authService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	email := normalizeEmail(req.Email)

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, refreshToken, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         sanitizeUserForResponse(user),
	}, nil
}

func (s *authService) Refresh(ctx context.Context, req models.RefreshRequest) (*models.TokenResponse, error) {
	rawToken, err := decodeRefreshTokenHex(req.RefreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	tokenHash := hashBytes(rawToken)
	oldToken, err := s.refreshTokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("failed to find refresh token by hash: %w", err)
	}
	if oldToken == nil {
		return nil, ErrInvalidToken
	}

	now := s.now().UTC()
	if oldToken.ExpiresAt.Before(now) {
		return nil, ErrTokenExpired
	}

	if oldToken.RevokedAt != nil || oldToken.ReplacedBy != nil {
		if err := s.refreshTokenRepo.RevokeAllByUserID(ctx, oldToken.UserID); err != nil {
			return nil, fmt.Errorf("failed to revoke refresh tokens by user id: %w", err)
		}
		if err := s.userRepo.IncrementTokenVersion(ctx, oldToken.UserID); err != nil {
			return nil, fmt.Errorf("failed to increment token version: %w", err)
		}
		return nil, ErrTokenReused
	}

	newRefreshRaw, err := s.generateRefreshTokenRaw()
	if err != nil {
		return nil, err
	}
	newRefreshHash := hashBytes(newRefreshRaw)

	newToken := &models.RefreshToken{
		UserID:    oldToken.UserID,
		TokenHash: newRefreshHash,
		ExpiresAt: now.Add(s.cfg.JWTRefreshTTL),
	}
	if err := s.refreshTokenRepo.Create(ctx, newToken); err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	if err := s.refreshTokenRepo.MarkReplaced(ctx, oldToken.ID, newToken.ID); err != nil {
		return nil, fmt.Errorf("failed to mark refresh token replaced: %w", err)
	}

	user, err := s.userRepo.FindByID(ctx, oldToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	accessToken, err := s.issueAccessToken(user)
	if err != nil {
		return nil, err
	}

	return &models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: hex.EncodeToString(newRefreshRaw),
	}, nil
}

func (s *authService) Logout(ctx context.Context, refreshTokenHex string) error {
	if strings.TrimSpace(refreshTokenHex) == "" {
		return nil
	}

	rawToken, err := decodeRefreshTokenHex(refreshTokenHex)
	if err != nil {
		return ErrInvalidToken
	}

	tokenHash := hashBytes(rawToken)
	token, err := s.refreshTokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to find refresh token by hash: %w", err)
	}
	if token == nil || token.RevokedAt != nil {
		return nil
	}

	if err := s.refreshTokenRepo.RevokeAllByUserID(ctx, token.UserID); err != nil {
		return fmt.Errorf("failed to revoke refresh tokens by user id: %w", err)
	}

	return nil
}

func (s *authService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) (*models.TokenResponse, error) {
	if len(newPassword) < minPasswordLength {
		return nil, fmt.Errorf("validation failed: new password must be at least %d characters", minPasswordLength)
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return nil, ErrInvalidPassword
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.userRepo.UpdatePasswordHash(ctx, userID, string(newHash)); err != nil {
		return nil, fmt.Errorf("failed to update password hash: %w", err)
	}

	if err := s.refreshTokenRepo.RevokeAllByUserID(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to revoke refresh tokens by user id: %w", err)
	}

	user, err = s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	accessToken, refreshToken, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	return &models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *authService) issueTokens(ctx context.Context, user *models.User) (string, string, error) {
	accessToken, err := s.issueAccessToken(user)
	if err != nil {
		return "", "", err
	}

	refreshRaw, err := s.generateRefreshTokenRaw()
	if err != nil {
		return "", "", err
	}
	refreshHash := hashBytes(refreshRaw)

	refreshToken := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: s.now().UTC().Add(s.cfg.JWTRefreshTTL),
	}

	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return "", "", fmt.Errorf("failed to create refresh token: %w", err)
	}

	return accessToken, hex.EncodeToString(refreshRaw), nil
}

func (s *authService) issueAccessToken(user *models.User) (string, error) {
	now := s.now().UTC()

	claims := models.JWTClaims{
		Sub:  user.ID.String(),
		Role: user.Role,
		TV:   user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.JWTAccessTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return accessToken, nil
}

func (s *authService) generateRefreshTokenRaw() ([]byte, error) {
	refreshRaw := make([]byte, refreshTokenBytes)
	if _, err := io.ReadFull(s.randReader, refreshRaw); err != nil {
		return nil, fmt.Errorf("failed to generate refresh token bytes: %w", err)
	}
	return refreshRaw, nil
}

func validateRegisterInput(email, password, displayName string) error {
	if !isValidEmail(email) {
		return errors.New("validation failed: invalid email format")
	}
	if len(password) < minPasswordLength {
		return fmt.Errorf("validation failed: password must be at least %d characters", minPasswordLength)
	}

	displayNameLen := utf8.RuneCountInString(displayName)
	if displayNameLen < minDisplayNameLen || displayNameLen > maxDisplayNameLen {
		return fmt.Errorf(
			"validation failed: display_name must be between %d and %d characters",
			minDisplayNameLen,
			maxDisplayNameLen,
		)
	}

	return nil
}

func sanitizeUserForResponse(user *models.User) models.User {
	if user == nil {
		return models.User{}
	}

	responseUser := *user
	responseUser.PasswordHash = ""
	return responseUser
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isValidEmail(email string) bool {
	parsed, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	return parsed.Address == email
}

func decodeRefreshTokenHex(refreshTokenHex string) ([]byte, error) {
	trimmed := strings.TrimSpace(refreshTokenHex)
	if trimmed == "" {
		return nil, errors.New("refresh token is empty")
	}

	rawToken, err := hex.DecodeString(trimmed)
	if err != nil {
		return nil, err
	}
	if len(rawToken) == 0 {
		return nil, errors.New("refresh token is empty")
	}

	return rawToken, nil
}

func hashBytes(raw []byte) []byte {
	sum := sha256.Sum256(raw)
	return sum[:]
}
