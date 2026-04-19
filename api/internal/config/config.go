package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all environment-driven configuration for the API server.
type Config struct {
	Port int
	Env  string // "development" | "staging" | "production"

	DatabaseURL string

	// ClerkDevBypass skips JWT verification and injects a synthetic user.
	// Only allowed when Env == "development". Never set in staging/prod.
	ClerkDevBypass     bool
	ClerkJWKSURL       string
	ClerkAudience      string
	ClerkWebhookSecret string

	SpacesEndpoint  string
	SpacesBucket    string
	SpacesAccessKey string
	SpacesSecretKey string

	SentryDSN string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:               envInt("PORT", 8080),
		Env:                envStr("ENV", "development"),
		DatabaseURL:        envStr("DATABASE_URL", ""),
		ClerkDevBypass:     envBool("CLERK_DEV_BYPASS", false),
		ClerkJWKSURL:       envStr("CLERK_JWKS_URL", ""),
		ClerkAudience:      envStr("CLERK_AUDIENCE", ""),
		ClerkWebhookSecret: envStr("CLERK_WEBHOOK_SECRET", ""),
		SpacesEndpoint:     envStr("SPACES_ENDPOINT", ""),
		SpacesBucket:       envStr("SPACES_BUCKET", ""),
		SpacesAccessKey:    envStr("SPACES_ACCESS_KEY", ""),
		SpacesSecretKey:    envStr("SPACES_SECRET_KEY", ""),
		SentryDSN:          envStr("SENTRY_DSN", ""),
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("config: DATABASE_URL is required")
	}
	if !c.ClerkDevBypass && c.ClerkJWKSURL == "" {
		return fmt.Errorf("config: CLERK_JWKS_URL must be set (or set CLERK_DEV_BYPASS=true for local dev)")
	}
	if c.ClerkDevBypass && c.Env != "development" {
		return fmt.Errorf("config: CLERK_DEV_BYPASS=true is not allowed in ENV=%s", c.Env)
	}
	return nil
}

func (c *Config) IsDevelopment() bool { return c.Env == "development" }
func (c *Config) IsProduction() bool  { return c.Env == "production" }

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
