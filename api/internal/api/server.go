package api

import (
	"database/sql"
	"net/http"

	"bitgo-wallets-api/internal/config"
	"bitgo-wallets-api/internal/repository"

	"github.com/gin-gonic/gin"
)

type Server struct {
	db     *sql.DB
	config *config.Config
	router *gin.Engine

	// Repositories
	walletRepo          repository.WalletRepository
	transferRequestRepo repository.TransferRequestRepository
}

func NewServer(db *sql.DB, cfg *config.Config) *Server {
	server := &Server{
		db:     db,
		config: cfg,
	}

	// Initialize repositories
	server.walletRepo = repository.NewWalletRepository(db)
	server.transferRequestRepo = repository.NewTransferRequestRepository(db)

	// Setup router
	server.setupRouter()

	return server
}

func (s *Server) setupRouter() {
	gin.SetMode(s.config.GinMode)
	s.router = gin.Default()

	// Add CORS middleware
	s.router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Health check
	s.router.GET("/health", s.healthCheck)

	// API routes
	api := s.router.Group("/api/v1")
	{
		// Auth routes
		auth := api.Group("/auth")
		{
			auth.POST("/login", s.login)
		}

		// Protected routes
		protected := api.Group("/")
		protected.Use(s.authMiddleware())
		{
			// Wallet routes
			wallets := protected.Group("/wallets")
			{
				wallets.GET("", s.listWallets)
				wallets.POST("", s.createWallet)
				wallets.GET("/:id", s.getWallet)
				wallets.PUT("/:id", s.updateWallet)
				wallets.DELETE("/:id", s.deleteWallet)

				// Transfer routes under wallets
				wallets.GET("/:id/transfers", s.listTransfers)
				wallets.POST("/:id/transfers", s.createTransfer)
			}

			// Transfer routes
			transfers := protected.Group("/transfers")
			{
				transfers.GET("/:id", s.getTransfer)
				transfers.PUT("/:id", s.updateTransfer)
				transfers.PUT("/:id/status", s.updateTransferStatus)
			}
		}
	}
}

func (s *Server) Start() error {
	return s.router.Run(":" + s.config.Port)
}
