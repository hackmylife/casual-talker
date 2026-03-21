package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port         string `env:"PORT"          envDefault:"8080"`
	DatabaseURL  string `env:"DATABASE_URL"  required:"true"`
	JWTSecret    string `env:"JWT_SECRET"    required:"true"`
	OpenAIAPIKey string `env:"OPENAI_API_KEY" required:"true"`
	StaticDir    string `env:"STATIC_DIR"    envDefault:"./static"`
}

// Load parses environment variables into a Config struct.
// Returns an error if any required fields are missing.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters (got %d)", len(cfg.JWTSecret))
	}
	return cfg, nil
}
