package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port          string
	DBPath        string
	CORSOrigin    string
	EncryptionKey string
	JWTSecret     string
	AuthEnabled   bool
	JWTExpiry     time.Duration

	// OAuth2
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	OAuthRedirectBase  string

	// SMTP (for notifications and shared link delivery)
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	cfg := &Config{
		Port:          envOr("PORT", "8080"),
		DBPath:        envOr("DB_PATH", "data.db"),
		CORSOrigin:    envOr("CORS_ORIGIN", "*"),
		EncryptionKey: os.Getenv("ENCRYPTION_KEY"),
		JWTSecret:     envOr("JWT_SECRET", "change-me-in-production"),
		AuthEnabled:   envBool("AUTH_ENABLED", true),
		JWTExpiry:     envDuration("JWT_EXPIRY", 24*time.Hour),

		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		OAuthRedirectBase:  envOr("OAUTH_REDIRECT_BASE", "http://localhost:5173"),

		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     envOr("SMTP_PORT", "587"),
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:     os.Getenv("SMTP_FROM"),
	}
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
