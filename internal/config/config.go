package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Port               string `envconfig:"PORT" default:"9000"`
	DatabaseURL        string `envconfig:"DATABASE_URL" required:"true"`
	JWTSecret          string `envconfig:"JWT_SECRET" required:"true" default:"your_very_strong_encypted_secret"`
	JWTExpiry          int    `envconfig:"JWT_EXPIRY" default:"15"` // in minutes
	JWTRefreshSecret   string `envconfig:"JWT_REFRESH_SECRET" default:"your_very_strong_encypted_secret"`
	JWTRefreshExpiry   int    `envconfig:"JWT_REFRESH_EXPIRY" default:"720"` // in hours
	ApiVersion         string `envconfig:"API_VERSION" default:"v1.0.0"`
	PlunkBaseUrl       string `envconfig:"PLUNK_BASE_URL"`
	PlunkSecretKey     string `envconfig:"PLUNK_SECRET_KEY"`
	RedisHost          string `envconfig:"REDIS_HOST" default:"localhost"`
	RedisPassword      string `envconfig:"REDIS_PASSWORD"`
	RedisPort          string `envconfig:"REDIS_PORT" default:"6379"`
	LoginRateLimit     int    `envconfig:"LOGIN_RATE_LIMIT" default:"5"`
	LoginRateWindow    int    `envconfig:"LOGIN_RATE_WINDOW" default:"15"`
	LoginBlockDuration int    `envconfig:"LOGIN_BLOCK_DURATION" default:"30"`
	IPRateLimit        int    `envconfig:"IP_RATE_LIMIT" default:"50"`
	GinMode            string `envconfig:"GIN_MODE" default:"release"`
	PapertrailAddr     string `envconfig:"PAPERTRAIL_ADDR"`
	PapertrailAppName string `envconfig:"PAPERTRAIL_APPNAME"`
}

func Load() (*Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
