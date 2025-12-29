package config

import (
	"os"
)

type Config struct {
	DatabaseURL      string
	Port             string
	GinMode          string
	AdminEmail       string
	AdminPassword    string
	BitGoBaseURL     string
	BitGoAccessToken string
	BitGoEnvironment string
	WebhookURL       string
}

func Load() *Config {
	return &Config{
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/bitgo_wallets?sslmode=disable"),
		Port:             getEnv("PORT", "8080"),
		GinMode:          getEnv("GIN_MODE", "debug"),
		AdminEmail:       getEnv("ADMIN_EMAIL", "admin@bitgo.com"),
		AdminPassword:    getEnv("ADMIN_PASSWORD", "admin123"),
		BitGoBaseURL:     getEnv("BITGO_BASE_URL", "https://test.bitgo.com"),
		BitGoAccessToken: getEnv("BITGO_ACCESS_TOKEN", ""),
		BitGoEnvironment: getEnv("BITGO_ENVIRONMENT", "test"),
		WebhookURL:       getEnv("WEBHOOK_URL", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
