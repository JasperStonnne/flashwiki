package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"fpgwiki/backend/internal/models"
)

type mockUserRepoForUserService struct {
	findByIDFn              func(ctx context.Context, id uuid.UUID) (*models.User, error)
	listAllFn               func(ctx context.Context) ([]*models.User, error)
	updateUserFn            func(ctx context.Context, user *models.User) error
	updateRoleFn            func(ctx context.Context, id uuid.UUID, role string) error
	incrementTokenVersionFn func(ctx context.Context, id uuid.UUID) error
}

func (m *mockUserRepoForUserService) CreateUser(ctx context.Context, user *models.User) error {
	return errors.New("unexpected CreateUser call")
}

func (m *mockUserRepoForUserService) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return nil, errors.New("unexpected FindByEmail call")
}

func (m *mockUserRepoForUserService) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if m.findByIDFn == nil {
		return nil, errors.New("unexpected FindByID call")
	}
	return m.findByIDFn(ctx, id)
}

func (m *mockUserRepoForUserService) ListAll(ctx context.Context) ([]*models.User, error) {
	if m.listAllFn == nil {
		return nil, errors.New("unexpected ListAll call")
	}
	return m.listAllFn(ctx)
}

func (m *mockUserRepoForUserService) UpdateUser(ctx context.Context, user *models.User) error {
	if m.updateUserFn == nil {
		return errors.New("unexpected UpdateUser call")
	}
	return m.updateUserFn(ctx, user)
}

func (m *mockUserRepoForUserService) UpdateRole(ctx context.Context, id uuid.UUID, role string) error {
	if m.updateRoleFn == nil {
		return errors.New("unexpected UpdateRole call")
	}
	return m.updateRoleFn(ctx, id, role)
}

func (m *mockUserRepoForUserService) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	return errors.New("unexpected UpdatePasswordHash call")
}

func (m *mockUserRepoForUserService) IncrementTokenVersion(ctx context.Context, id uuid.UUID) error {
	if m.incrementTokenVersionFn == nil {
		return errors.New("unexpected IncrementTokenVersion call")
	}
	return m.incrementTokenVersionFn(ctx, id)
}

func TestUserServiceGetByIDSuccess(t *testing.T) {
	userID := uuid.New()
	expectedUser := &models.User{
		ID:          userID,
		Email:       "user@example.com",
		DisplayName: "User",
		Role:        "member",
		Locale:      "zh",
	}

	repo := &mockUserRepoForUserService{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			if id != userID {
				t.Fatalf("unexpected user id %s", id)
			}
			return expectedUser, nil
		},
	}

	svc := NewUserService(repo)
	got, err := svc.GetByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil || got.ID != expectedUser.ID {
		t.Fatalf("expected user %s, got %+v", expectedUser.ID, got)
	}
}

func TestUserServiceGetByIDNotFound(t *testing.T) {
	repo := &mockUserRepoForUserService{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return nil, nil
		},
	}

	svc := NewUserService(repo)
	_, err := svc.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserServiceUpdateDisplayNameSuccess(t *testing.T) {
	userID := uuid.New()
	displayName := "Updated User"
	user := &models.User{
		ID:          userID,
		DisplayName: "Old Name",
		Locale:      "zh",
	}

	repo := &mockUserRepoForUserService{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return user, nil
		},
		updateUserFn: func(ctx context.Context, updated *models.User) error {
			if updated.DisplayName != displayName {
				t.Fatalf("expected display name %q, got %q", displayName, updated.DisplayName)
			}
			return nil
		},
	}

	svc := NewUserService(repo)
	got, err := svc.Update(context.Background(), userID, models.UpdateUserRequest{
		DisplayName: &displayName,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.DisplayName != displayName {
		t.Fatalf("expected updated display name %q, got %q", displayName, got.DisplayName)
	}
}

func TestUserServiceUpdateLocaleSuccess(t *testing.T) {
	userID := uuid.New()
	locale := "en"
	user := &models.User{
		ID:          userID,
		DisplayName: "User",
		Locale:      "zh",
	}

	repo := &mockUserRepoForUserService{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return user, nil
		},
		updateUserFn: func(ctx context.Context, updated *models.User) error {
			if updated.Locale != locale {
				t.Fatalf("expected locale %q, got %q", locale, updated.Locale)
			}
			return nil
		},
	}

	svc := NewUserService(repo)
	got, err := svc.Update(context.Background(), userID, models.UpdateUserRequest{
		Locale: &locale,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Locale != locale {
		t.Fatalf("expected updated locale %q, got %q", locale, got.Locale)
	}
}

func TestUserServiceUpdateDisplayNameTooLong(t *testing.T) {
	userID := uuid.New()
	tooLong := strings.Repeat("a", 51)
	user := &models.User{ID: userID, DisplayName: "old", Locale: "zh"}

	repo := &mockUserRepoForUserService{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return user, nil
		},
		updateUserFn: func(ctx context.Context, user *models.User) error {
			t.Fatal("UpdateUser should not be called on invalid display_name")
			return nil
		},
	}

	svc := NewUserService(repo)
	_, err := svc.Update(context.Background(), userID, models.UpdateUserRequest{
		DisplayName: &tooLong,
	})
	if err == nil || !strings.Contains(err.Error(), "validation failed") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestUserServiceUpdateLocaleInvalid(t *testing.T) {
	userID := uuid.New()
	locale := "jp"
	user := &models.User{ID: userID, DisplayName: "old", Locale: "zh"}

	repo := &mockUserRepoForUserService{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return user, nil
		},
		updateUserFn: func(ctx context.Context, user *models.User) error {
			t.Fatal("UpdateUser should not be called on invalid locale")
			return nil
		},
	}

	svc := NewUserService(repo)
	_, err := svc.Update(context.Background(), userID, models.UpdateUserRequest{
		Locale: &locale,
	})
	if err == nil || !strings.Contains(err.Error(), "validation failed") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestUserServiceUpdateUserNotFound(t *testing.T) {
	displayName := "Updated"
	repo := &mockUserRepoForUserService{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return nil, nil
		},
		updateUserFn: func(ctx context.Context, user *models.User) error {
			t.Fatal("UpdateUser should not be called when user not found")
			return nil
		},
	}

	svc := NewUserService(repo)
	_, err := svc.Update(context.Background(), uuid.New(), models.UpdateUserRequest{
		DisplayName: &displayName,
	})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
