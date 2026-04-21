//go:build integration

package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"fpgwiki/backend/internal/config"
	"fpgwiki/backend/internal/httpserver"
)

const defaultIntegrationDSN = "postgres://fpgwiki:fpgwiki_dev@localhost:5432/fpgwiki?sslmode=disable"

type httpEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type authResponseBody struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func TestAuthFlowIntegration(t *testing.T) {
	pool := newIntegrationPool(t)
	router := newIntegrationRouter(t, pool)

	email := fmt.Sprintf("auth_integration_%d@example.com", time.Now().UnixNano())
	password := "OldPass123!"
	newPassword := "NewPass123!"

	registerRec := doJSONRequest(t, router, http.MethodPost, "/api/auth/register", map[string]any{
		"email":        email,
		"password":     password,
		"display_name": "Integration User",
	}, "")
	if registerRec.Code != http.StatusCreated {
		t.Fatalf("register status: expected 201, got %d, body=%s", registerRec.Code, registerRec.Body.String())
	}

	registerBody := parseAuthEnvelope(t, registerRec)
	if registerBody.AccessToken == "" || registerBody.RefreshToken == "" {
		t.Fatal("register response should contain access and refresh tokens")
	}

	meRec := doJSONRequest(t, router, http.MethodGet, "/api/me", nil, registerBody.AccessToken)
	if meRec.Code != http.StatusOK {
		t.Fatalf("get /api/me status: expected 200, got %d, body=%s", meRec.Code, meRec.Body.String())
	}

	refreshRec := doJSONRequest(t, router, http.MethodPost, "/api/auth/refresh", map[string]any{
		"refresh_token": registerBody.RefreshToken,
	}, "")
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("refresh status: expected 200, got %d, body=%s", refreshRec.Code, refreshRec.Body.String())
	}
	refreshedBody := parseAuthEnvelope(t, refreshRec)
	if refreshedBody.AccessToken == "" || refreshedBody.RefreshToken == "" {
		t.Fatal("refresh response should contain new access and refresh tokens")
	}

	refreshOldRec := doJSONRequest(t, router, http.MethodPost, "/api/auth/refresh", map[string]any{
		"refresh_token": registerBody.RefreshToken,
	}, "")
	if refreshOldRec.Code != http.StatusUnauthorized {
		t.Fatalf("old refresh reuse status: expected 401, got %d, body=%s", refreshOldRec.Code, refreshOldRec.Body.String())
	}

	loginBeforeChangeRec := doJSONRequest(t, router, http.MethodPost, "/api/auth/login", map[string]any{
		"email":    email,
		"password": password,
	}, "")
	if loginBeforeChangeRec.Code != http.StatusOK {
		t.Fatalf("login before change password status: expected 200, got %d, body=%s", loginBeforeChangeRec.Code, loginBeforeChangeRec.Body.String())
	}
	loginBeforeChangeBody := parseAuthEnvelope(t, loginBeforeChangeRec)

	changePasswordRec := doJSONRequest(t, router, http.MethodPost, "/api/me/password", map[string]any{
		"old_password": password,
		"new_password": newPassword,
	}, loginBeforeChangeBody.AccessToken)
	if changePasswordRec.Code != http.StatusOK {
		t.Fatalf("change password status: expected 200, got %d, body=%s", changePasswordRec.Code, changePasswordRec.Body.String())
	}

	oldAccessRec := doJSONRequest(t, router, http.MethodGet, "/api/me", nil, loginBeforeChangeBody.AccessToken)
	if oldAccessRec.Code != http.StatusUnauthorized {
		t.Fatalf("old access status: expected 401, got %d, body=%s", oldAccessRec.Code, oldAccessRec.Body.String())
	}

	loginRec := doJSONRequest(t, router, http.MethodPost, "/api/auth/login", map[string]any{
		"email":    email,
		"password": newPassword,
	}, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status: expected 200, got %d, body=%s", loginRec.Code, loginRec.Body.String())
	}
	loginBody := parseAuthEnvelope(t, loginRec)

	logoutRec := doJSONRequest(t, router, http.MethodPost, "/api/auth/logout", map[string]any{
		"refresh_token": loginBody.RefreshToken,
	}, loginBody.AccessToken)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("logout status: expected 200, got %d, body=%s", logoutRec.Code, logoutRec.Body.String())
	}

	refreshAfterLogoutRec := doJSONRequest(t, router, http.MethodPost, "/api/auth/refresh", map[string]any{
		"refresh_token": loginBody.RefreshToken,
	}, "")
	if refreshAfterLogoutRec.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout status: expected 401, got %d, body=%s", refreshAfterLogoutRec.Code, refreshAfterLogoutRec.Body.String())
	}
}

func newIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = defaultIntegrationDSN
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skip integration tests: connect postgres failed: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skip integration tests: postgres unavailable: %v", err)
	}

	var usersTable *string
	if err := pool.QueryRow(ctx, "SELECT to_regclass('public.users')::text").Scan(&usersTable); err != nil {
		t.Skipf("skip integration tests: schema check failed: %v", err)
	}
	if usersTable == nil {
		t.Skip("skip integration tests: users table not found, run migrations first")
	}

	return pool
}

func newIntegrationRouter(t *testing.T, pool *pgxpool.Pool) http.Handler {
	t.Helper()

	cfg := config.Config{
		PostgresDSN:      defaultIntegrationDSN,
		PostgresMaxConns: 20,
		JWTSecret:        "integration-secret",
		JWTAccessTTL:     15 * time.Minute,
		JWTRefreshTTL:    7 * 24 * time.Hour,
	}
	log := zerolog.New(io.Discard)
	return httpserver.NewRouter(cfg, log, pool)
}

func doJSONRequest(t *testing.T, router http.Handler, method, path string, body any, accessToken string) *httptest.ResponseRecorder {
	t.Helper()

	var bodyReader *bytes.Reader
	if body == nil {
		bodyReader = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func parseAuthEnvelope(t *testing.T, rec *httptest.ResponseRecorder) authResponseBody {
	t.Helper()

	var env httpEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if !env.Success {
		t.Fatalf("expected success response, got error: %+v", env.Error)
	}

	var body authResponseBody
	if err := json.Unmarshal(env.Data, &body); err != nil {
		t.Fatalf("unmarshal auth response body: %v", err)
	}
	return body
}
