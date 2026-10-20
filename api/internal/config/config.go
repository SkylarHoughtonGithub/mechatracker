package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all environment-driven configuration for the API server.
type Config struct {
	Port int
	Env  string // "development" | "staging" | "production"

	DatabaseURL string

	// DevAuthBypass skips Google ID-token verification and injects a synthetic
	// user. Only allowed when Env == "development". Never set in staging/prod.
	DevAuthBypass bool

	// Google OAuth — the API verifies ID tokens issued to this client ID and
	// treats it as the ID-token audience.
	GoogleClientID string
	// GoogleAllowedHD, if set, restricts sign-in to a Google Workspace hosted
	// domain (e.g. "example.com"). Leave empty to accept any Google account.
	GoogleAllowedHD string

	// Session cookie — signed JWT the browser carries after /auth/google.
	SessionSigningKey   string
	SessionCookieDomain string
	SessionCookieSecure bool
	SessionTTL          time.Duration

	SpacesEndpoint  string
	SpacesBucket    string
	SpacesAccessKey string
	SpacesSecretKey string

	SentryDSN string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:                envInt("PORT", 8080),
		Env:                 envStr("ENV", "development"),
		DatabaseURL:         envStr("DATABASE_URL", ""),
		DevAuthBypass:       envBool("DEV_AUTH_BYPASS", false),
		GoogleClientID:      envStr("GOOGLE_CLIENT_ID", ""),
		GoogleAllowedHD:     envStr("GOOGLE_ALLOWED_HD", ""),
		SessionSigningKey:   envStr("SESSION_SIGNING_KEY", ""),
		SessionCookieDomain: envStr("SESSION_COOKIE_DOMAIN", ""),
		SessionCookieSecure: envBool("SESSION_COOKIE_SECURE", true),
		SessionTTL:          envDuration("SESSION_TTL", 12*time.Hour),
		SpacesEndpoint:      envStr("SPACES_ENDPOINT", ""),
		SpacesBucket:        envStr("SPACES_BUCKET", ""),
		SpacesAccessKey:     envStr("SPACES_ACCESS_KEY", ""),
		SpacesSecretKey:     envStr("SPACES_SECRET_KEY", ""),
		SentryDSN:           envStr("SENTRY_DSN", ""),
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("config: DATABASE_URL is required")
	}
	if c.DevAuthBypass && c.Env != "development" {
		return fmt.Errorf("config: DEV_AUTH_BYPASS=true is not allowed in ENV=%s", c.Env)
	}
	if !c.DevAuthBypass {
		if c.GoogleClientID == "" {
			return fmt.Errorf("config: GOOGLE_CLIENT_ID must be set (or set DEV_AUTH_BYPASS=true for local dev)")
		}
		if c.SessionSigningKey == "" {
			return fmt.Errorf("config: SESSION_SIGNING_KEY must be set (or set DEV_AUTH_BYPASS=true for local dev)")
		}
		if len(c.SessionSigningKey) < 32 {
			return fmt.Errorf("config: SESSION_SIGNING_KEY must be at least 32 bytes")
		}
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

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
