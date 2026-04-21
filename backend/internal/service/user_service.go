package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"fpgwiki/backend/internal/models"
	"fpgwiki/backend/internal/repository"
)

const (
	maxUserDisplayNameLen = 50
	minUserDisplayNameLen = 1
)

type UserService interface {
	GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
	Update(ctx context.Context, userID uuid.UUID, req models.UpdateUserRequest) (*models.User, error)
}

type userService struct {
	userRepo repository.UserRepo
}

func NewUserService(userRepo repository.UserRepo) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func (s *userService) Update(ctx context.Context, userID uuid.UUID, req models.UpdateUserRequest) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if req.DisplayName != nil {
		displayName := strings.TrimSpace(*req.DisplayName)
		displayNameLen := utf8.RuneCountInString(displayName)
		if displayNameLen < minUserDisplayNameLen || displayNameLen > maxUserDisplayNameLen {
			return nil, fmt.Errorf(
				"validation failed: display_name must be between %d and %d characters",
				minUserDisplayNameLen,
				maxUserDisplayNameLen,
			)
		}
		user.DisplayName = displayName
	}

	if req.Locale != nil {
		if *req.Locale != "en" && *req.Locale != "zh" {
			return nil, errors.New("validation failed: locale must be en or zh")
		}
		user.Locale = *req.Locale
	}

	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}
