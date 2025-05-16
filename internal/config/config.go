package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Port     int
	LogLevel string

	// Database configuration
	DatabasePath string

	// Authentication configuration
	Auth AuthConfig
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	// Authentication type: "oidc" for Google
	AuthType string

	// OIDC Configuration for Google
	OIDCIssuerURL    string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string

	// Session Configuration
	SessionSecret    string
	SessionDuration  time.Duration

	// Initial setup configuration
	InitialSuperAdminEmail string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		// Default values
		Port:         80,
		LogLevel:     "development",
		DatabasePath: "./db/badges.db",
		Auth: AuthConfig{
			AuthType:               "oidc",
			OIDCIssuerURL:          "https://accounts.google.com",
			OIDCRedirectURL:        "http://localhost:80/auth/callback",
			SessionDuration:        24 * time.Hour,
			InitialSuperAdminEmail: "badge-admin@gmail.com",
		},
	}

	// Override with environment variables if they exist
	if port := os.Getenv("PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err == nil {
			cfg.Port = p
		}
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}

	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		cfg.DatabasePath = dbPath
	}

	// Load authentication configuration
	if authType := os.Getenv("AUTH_TYPE"); authType != "" {
		cfg.Auth.AuthType = authType
	}

	if issuerURL := os.Getenv("OIDC_ISSUER_URL"); issuerURL != "" {
		cfg.Auth.OIDCIssuerURL = issuerURL
	}

	if clientID := os.Getenv("OIDC_CLIENT_ID"); clientID != "" {
		cfg.Auth.OIDCClientID = clientID
	}

	if clientSecret := os.Getenv("OIDC_CLIENT_SECRET"); clientSecret != "" {
		cfg.Auth.OIDCClientSecret = clientSecret
	}

	if redirectURL := os.Getenv("OIDC_REDIRECT_URL"); redirectURL != "" {
		cfg.Auth.OIDCRedirectURL = redirectURL
	}

	if sessionSecret := os.Getenv("SESSION_SECRET"); sessionSecret != "" {
		cfg.Auth.SessionSecret = sessionSecret
	} else {
		// Generate a random session secret if not provided
		cfg.Auth.SessionSecret = "default-session-secret-change-in-production"
	}

	if sessionDuration := os.Getenv("SESSION_DURATION"); sessionDuration != "" {
		duration, err := time.ParseDuration(sessionDuration)
		if err == nil {
			cfg.Auth.SessionDuration = duration
		}
	}

	if superAdminEmail := os.Getenv("INITIAL_SUPER_ADMIN_EMAIL"); superAdminEmail != "" {
		cfg.Auth.InitialSuperAdminEmail = superAdminEmail
	}

	return cfg, nil
}
