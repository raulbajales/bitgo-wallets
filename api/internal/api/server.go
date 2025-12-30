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
	bitgoClient        *bitgo.Client
	bitgoRequestLogger *BitGoRequestLogger
	pollingWorker      *services.TransferPollingWorker
	notificationSvc    services.NotificationService
	coldWalletSvc      *services.ColdWalletService
	warmWalletSvc      *services.WarmWalletService

	// Repositories
	walletRepo          repository.WalletRepository
	transferRequestRepo repository.TransferRequestRepository
}

func NewServer(db *sql.DB, cfg *config.Config) *Server {
	server := &Server{
		db:     db,
		config: cfg,
	}

	// Initialize BitGo request logger first (needed by BitGo client)
	server.bitgoRequestLogger = NewBitGoRequestLogger()

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

	// Initialize warm wallet service
	server.initWarmWalletService()

	// Setup router
	server.setupRouter()

	return server
}

func (s *Server) initBitGoClient() {
	// Create BitGo logger that captures requests for debug console
	logger := NewBitGoLogger(s.bitgoRequestLogger)

	bitgoConfig := bitgo.Config{
		BaseURL:     s.config.BitGoBaseURL,
		AccessToken: s.config.BitGoAccessToken,
		Timeout:     30 * time.Second,
		MaxRetries:  3,
	}

	s.bitgoClient = bitgo.NewClient(bitgoConfig, logger)
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

func (s *Server) initWarmWalletService() {
	// Create warm wallet service configuration
	warmConfig := services.DefaultWarmWalletConfig()

	// Override with environment-specific settings
	if s.config.GinMode == "release" {
		warmConfig.RequiredApprovals = 1
		warmConfig.ApprovalTimeoutHours = 24
		warmConfig.AutoProcessThreshold = "10.0"
	} else {
		// Development settings
		warmConfig.RequiredApprovals = 0
		warmConfig.ApprovalTimeoutHours = 12
		warmConfig.AutoProcessThreshold = "5.0"
	}

	// Create warm wallet service
	logger := &SimpleLogger{}
	s.warmWalletSvc = services.NewWarmWalletService(
		s.bitgoClient,
		s.walletRepo,
		s.transferRequestRepo,
		s.notificationSvc,
		logger,
		warmConfig,
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

	// WebSocket endpoint for BitGo request logs
	s.router.GET("/ws/bitgo-requests", s.HandleBitGoRequestLogs)

	api := s.router.Group("/api/v1")
	// NO MIDDLEWARE APPLIED - ALL ROUTES ARE PUBLIC

	// Test endpoints
	api.GET("/test-bitgo", s.testBitGo)
	api.POST("/test-wallet", s.createWallet)
	api.GET("/test-bitgo-direct", s.testBitGoLogging)

	// Auth routes (for compatibility)
	api.POST("/auth/login", s.login)

	// Wallet routes - NO AUTH REQUIRED
	api.GET("/wallets", s.listWallets)
	api.POST("/wallets", s.createWallet)
	api.GET("/wallets/discover", s.discoverWallets)
	api.GET("/wallets/:id", s.getWallet)
	api.PUT("/wallets/:id", s.updateWallet)
	api.DELETE("/wallets/:id", s.deleteWallet)
	api.POST("/wallets/:id/sync-balance", s.syncWalletBalance)
	api.GET("/wallets/:id/transfers", s.listTransfers)
	api.POST("/wallets/:id/transfers", s.createTransfer)

	// Transfer routes - NO AUTH REQUIRED
	api.GET("/transfers/:id", s.getTransfer)
	api.PUT("/transfers/:id", s.updateTransfer)
	api.PUT("/transfers/:id/status", s.updateTransferStatus)
	api.POST("/transfers/:id/submit", s.submitTransfer)
	api.GET("/transfers/:id/status", s.getTransferStatus)
	api.PUT("/transfers/:id/offline-workflow-state", s.updateOfflineWorkflowState)
	api.POST("/transfers/verify-address", s.verifyAddress)

	// Cold transfer routes - NO AUTH REQUIRED
	api.POST("/transfers/cold", s.createColdTransfer)
	api.GET("/transfers/cold/sla", s.getColdTransfersSLA)
	api.GET("/transfers/cold/admin-queue", s.getColdTransfersAdminQueue)

	// Warm transfer routes - NO AUTH REQUIRED
	api.POST("/transfers/warm", s.createWarmTransfer)
	api.GET("/transfers/warm/sla", s.getWarmTransfersSLA)
	api.GET("/transfers/warm/analytics", s.getWarmTransfersAnalytics)
	api.POST("/transfers/warm/:id/process", s.processWarmTransfer)

	// Admin routes - NO AUTH REQUIRED
	api.GET("/admin/approvers", s.getApprovers)
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
