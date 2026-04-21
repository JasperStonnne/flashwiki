package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"fpgwiki/backend/internal/config"
)

func TestCORSPassThrough(t *testing.T) {
	cfg := config.Config{}
	nextCalled := false

	router := gin.New()
	router.Use(CORS(cfg))
	router.GET("/ok", func(c *gin.Context) {
		nextCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to execute")
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("expected CORS header to be set")
	}
}

func TestCORSOptionsRequest(t *testing.T) {
	cfg := config.Config{}
	router := gin.New()
	router.Use(CORS(cfg))
	router.OPTIONS("/ok", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/ok", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
}

func TestRequestIDGeneratesValue(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.GET("/ok", func(c *gin.Context) {
		if requestIDFromContext(c) == "" {
			t.Fatal("expected request id in context")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get(requestIDHeader) == "" {
		t.Fatal("expected response request id header")
	}
}

func TestRequestIDUsesProvidedHeader(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.GET("/ok", func(c *gin.Context) {
		if requestIDFromContext(c) != "req-123" {
			t.Fatalf("expected request id req-123, got %s", requestIDFromContext(c))
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set(requestIDHeader, "req-123")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get(requestIDHeader) != "req-123" {
		t.Fatalf("expected response header req-123, got %s", rec.Header().Get(requestIDHeader))
	}
}

func TestRecoverHandlesPanic(t *testing.T) {
	log := zerolog.New(io.Discard)
	router := gin.New()
	router.Use(RequestID())
	router.Use(Recover(log))
	router.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestLoggerPassThrough(t *testing.T) {
	log := zerolog.New(io.Discard)
	router := gin.New()
	router.Use(RequestID())
	router.Use(Logger(log))
	router.GET("/ok", func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
}
