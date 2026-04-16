package main

import (
	"errors"
	"net/url"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog"

	"fpgwiki/backend/internal/config"
	"fpgwiki/backend/internal/logger"
)

const migrationsSourceURL = "file://migrations"

func main() {
	cfg, err := config.Load()
	if err != nil {
		fallbackLog := logger.New("info")
		fallbackLog.Fatal().Err(err).Msg("load config")
	}

	log := logger.New(cfg.LogLevel)

	command, steps, err := parseArgs(os.Args[1:])
	if err != nil {
		log.Fatal().Err(err).Msg("parse command")
	}

	databaseURL, err := migrateDatabaseURL(cfg.PostgresDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("build migrate database URL")
	}

	m, err := migrate.New(migrationsSourceURL, databaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("create migrate instance")
	}
	defer closeMigrator(log, m)

	log.Info().Str("command", command).Msg("running migrations")

	switch command {
	case "up":
		err = m.Up()
	case "down":
		err = m.Steps(-steps)
	default:
		log.Fatal().Str("command", command).Msg("unsupported command")
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal().Err(err).Str("command", command).Msg("migration failed")
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Info().Str("command", command).Msg("no migration change")
		return
	}

	log.Info().Str("command", command).Msg("migration completed")
}

func parseArgs(args []string) (string, int, error) {
	if len(args) == 1 && args[0] == "up" {
		return "up", 0, nil
	}

	if len(args) == 2 && args[0] == "down" {
		steps, err := strconv.Atoi(args[1])
		if err != nil || steps <= 0 {
			return "", 0, errors.New("down requires a positive integer step count")
		}
		return "down", steps, nil
	}

	return "", 0, errors.New("usage: go run ./cmd/migrate up | go run ./cmd/migrate down <steps>")
}

func migrateDatabaseURL(dsn string) (string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}

	switch parsed.Scheme {
	case "postgres", "postgresql":
		parsed.Scheme = "pgx5"
	case "pgx5":
	default:
		return "", errors.New("POSTGRES_DSN must use postgres://, postgresql://, or pgx5:// scheme")
	}

	return parsed.String(), nil
}

func closeMigrator(log zerolog.Logger, m *migrate.Migrate) {
	sourceErr, databaseErr := m.Close()
	if sourceErr != nil {
		log.Warn().Err(sourceErr).Msg("close migrate source")
	}
	if databaseErr != nil {
		log.Warn().Err(databaseErr).Msg("close migrate database")
	}
}
