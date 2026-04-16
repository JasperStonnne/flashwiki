package logger

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
)

const (
	RequestIDField = "request_id"
	UserIDField    = "user_id"
	NodeIDField    = "node_id"
	LatencyMSField = "latency_ms"
)

func New(logLevel string) zerolog.Logger {
	level, err := zerolog.ParseLevel(strings.ToLower(logLevel))
	if err != nil {
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)
	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}
