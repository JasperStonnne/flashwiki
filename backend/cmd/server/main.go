package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"fpgwiki/backend/internal/config"
	"fpgwiki/backend/internal/db"
	"fpgwiki/backend/internal/httpserver"
	"fpgwiki/backend/internal/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	appLog := logger.New(cfg.LogLevel)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, cfg)
	if err != nil {
		appLog.Fatal().Err(err).Msg("database startup probe failed")
	}
	defer pool.Close()

	router := httpserver.NewRouter(cfg, appLog, pool)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	appLog.Info().Str("addr", cfg.HTTPAddr).Msg("starting HTTP server")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		appLog.Fatal().Err(err).Msg("HTTP server stopped")
	}
}
