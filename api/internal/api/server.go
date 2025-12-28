package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"bitgo-wallets-api/internal/bitgo"
	"bitgo-wallets-api/internal/config"
	"bitgo-wallets-api/internal/repository"
	"bitgo-wallets-api/internal/services"

	"github.com/gin-gonic/gin"
)

type Server struct {
	db     *sql.DB
	config *config.Config
	router *gin.Engine

	// External services
	bitgoClient     *bitgo.Client
	pollingWorker   *services.TransferPollingWorker
	notificationSvc services.NotificationService
	coldWalletSvc   *services.ColdWalletService

	// Repositories
	walletRepo          repository.WalletRepository
	transferRequestRepo repository.TransferRequestRepository
}

func NewServer(db *sql.DB, cfg *config.Config) *Server {
	server := &Server{
		db:     db,
		config: cfg,
	}

	// Initialize BitGo client
	server.initBitGoClient()

	// Initialize notification service
	server.initNotificationService()

	// Initialize repositories
	server.walletRepo = repository.NewWalletRepository(db)
	server.transferRequestRepo = repository.NewTransferRequestRepository(db)

	// Initialize background services
	server.initBackgroundServices()

	// Initialize cold wallet service
	server.initColdWalletService()

	// Setup router
	server.setupRouter()

	return server
}

func (s *Server) initBitGoClient() {
	// Create a simple logger implementation
	logger := &SimpleLogger{}

	bitgoConfig := bitgo.Config{
		BaseURL:     s.config.BitGoBaseURL,
		AccessToken: s.config.BitGoAccessToken,
		Logger:      logger,
		Timeout:     30 * time.Second,
		MaxRetries:  3,
		APIVersion:  "v2",
	}

	s.bitgoClient = bitgo.NewClient(bitgoConfig)
}

func (s *Server) initNotificationService() {
	// Create notification service configuration
	notificationConfig := services.DefaultNotificationConfig()

	// Configure webhook URL if provided
	if s.config.WebhookURL != "" {
		notificationConfig.WebhookURL = s.config.WebhookURL
	}

	// Create notification service
	logger := &SimpleLogger{}
	s.notificationSvc = services.NewNotificationService(notificationConfig, logger)
}

func (s *Server) initBackgroundServices() {
	// Create polling worker configuration
	workerConfig := services.DefaultPollingWorkerConfig()

	// Override defaults based on environment
	if s.config.GinMode == "release" {
		workerConfig.PollInterval = 30 * time.Second
		workerConfig.ConcurrentWorkers = 5
	} else {
		// Development settings
		workerConfig.PollInterval = 10 * time.Second
		workerConfig.ConcurrentWorkers = 2
	}

	// Create polling worker
	logger := &SimpleLogger{}
	s.pollingWorker = services.NewTransferPollingWorker(
		workerConfig,
		logger,
		s.bitgoClient,
		s.transferRequestRepo,
		s.walletRepo,
		s.notificationSvc,
	)
}

func (s *Server) initColdWalletService() {
	// Create cold wallet service configuration
	coldConfig := services.DefaultColdWalletConfig()

	// Override with environment-specific settings
	if s.config.GinMode == "release" {
		coldConfig.RequiredApprovals = 3
		coldConfig.ApprovalTimeoutHours = 72
	} else {
		// Development settings
		coldConfig.RequiredApprovals = 2
		coldConfig.ApprovalTimeoutHours = 24
	}

	// Create cold wallet service
	logger := &SimpleLogger{}
	s.coldWalletSvc = services.NewColdWalletService(
		s.bitgoClient,
		s.walletRepo,
		s.transferRequestRepo,
		s.notificationSvc,
		logger,
		coldConfig,
	)
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
	s.router.GET("/health/detailed", s.detailedHealthCheck)
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
				wallets.GET("/discover", s.discoverWallets)
				wallets.GET("/:id", s.getWallet)
				wallets.PUT("/:id", s.updateWallet)
				wallets.DELETE("/:id", s.deleteWallet)
				wallets.POST("/:id/sync-balance", s.syncWalletBalance)

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
				transfers.POST("/:id/submit", s.submitTransfer)
				transfers.GET("/:id/status", s.getTransferStatus)
				transfers.PUT("/:id/offline-workflow-state", s.updateOfflineWorkflowState)
				transfers.POST("/verify-address", s.verifyAddress)

				// Cold transfer routes
				cold := transfers.Group("/cold")
				{
					cold.POST("", s.createColdTransfer)
					cold.GET("/sla", s.getColdTransfersSLA)
					cold.GET("/admin-queue", s.getColdTransfersAdminQueue)
				}
			}

			// Admin routes
			admin := protected.Group("/admin")
			{
				admin.GET("/approvers", s.getApprovers)
			}
		}
	}
}

func (s *Server) Start() error {
	// Start background services
	if err := s.pollingWorker.Start(); err != nil {
		return fmt.Errorf("failed to start polling worker: %w", err)
	}

	return s.router.Run(":" + s.config.Port)
}

func (s *Server) Stop() error {
	// Stop background services gracefully
	if err := s.pollingWorker.Stop(); err != nil {
		return fmt.Errorf("failed to stop polling worker: %w", err)
	}

	return nil
}

// SimpleLogger implements the bitgo.Logger interface
type SimpleLogger struct{}

func (l *SimpleLogger) Info(msg string, fields ...interface{}) {
	// In a real implementation, use a proper logger like logrus or zap
	println("[INFO]", msg, fmt.Sprint(fields...))
}

func (l *SimpleLogger) Warn(msg string, fields ...interface{}) {
	println("[WARN]", msg, fmt.Sprint(fields...))
}

func (l *SimpleLogger) Error(msg string, fields ...interface{}) {
	println("[ERROR]", msg, fmt.Sprint(fields...))
}

func (l *SimpleLogger) Debug(msg string, fields ...interface{}) {
	println("[DEBUG]", msg, fmt.Sprint(fields...))
}
