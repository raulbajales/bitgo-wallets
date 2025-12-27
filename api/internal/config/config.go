package config

import (
	"os"
)

type Config struct {
	DatabaseURL   string
	Port          string
	GinMode       string
	AdminEmail    string
	AdminPassword string
}

func Load() *Config {
	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/bitgo_wallets?sslmode=disable"),
		Port:          getEnv("PORT", "8080"),
		GinMode:       getEnv("GIN_MODE", "debug"),
		AdminEmail:    getEnv("ADMIN_EMAIL", "admin@bitgo.com"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
