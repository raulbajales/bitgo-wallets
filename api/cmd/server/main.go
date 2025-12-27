package main

import (
	"log"
	"os"

	"bitgo-wallets-api/internal/api"
	"bitgo-wallets-api/internal/config"
	"bitgo-wallets-api/internal/database"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables in development
	if os.Getenv("GIN_MODE") != "release" {
		if err := godotenv.Load(); err != nil {
			log.Printf("Warning: .env file not found")
		}
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize and start API server
	server := api.NewServer(db, cfg)
	log.Printf("Starting server on port %s", cfg.Port)

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
