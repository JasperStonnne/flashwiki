package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	AppEnv           string        `envconfig:"APP_ENV" default:"dev"`
	HTTPAddr         string        `envconfig:"HTTP_ADDR" default:":8080"`
	PostgresDSN      string        `envconfig:"POSTGRES_DSN" required:"true"`
	PostgresMaxConns int32         `envconfig:"POSTGRES_MAX_CONNS" default:"20"`
	JWTSecret        string        `envconfig:"JWT_SECRET" required:"true"`
	JWTAccessTTL     time.Duration `envconfig:"JWT_ACCESS_TTL" default:"15m"`
	JWTRefreshTTL    time.Duration `envconfig:"JWT_REFRESH_TTL" default:"168h"`
	UploadDir        string        `envconfig:"UPLOAD_DIR" default:"/data/uploads"`
	MaxImageBytes    int64         `envconfig:"MAX_IMAGE_BYTES" default:"52428800"`
	MaxVideoBytes    int64         `envconfig:"MAX_VIDEO_BYTES" default:"524288000"`
	LogLevel         string        `envconfig:"LOG_LEVEL" default:"info"`
}

func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
