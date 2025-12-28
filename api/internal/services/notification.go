package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"bitgo-wallets-api/internal/bitgo"
	"bitgo-wallets-api/internal/models"

	"github.com/google/uuid"
)

// NotificationService handles sending notifications for various events
type NotificationService interface {
	SendTransferStatusNotification(transfer *models.TransferRequest, oldStatus, newStatus models.TransferStatus)
	SendPendingApprovalNotification(transfer *models.TransferRequest, approval *bitgo.ApprovalStatus)
	SendTransferCreatedNotification(transfer *models.TransferRequest)
	SendTransferCompletedNotification(transfer *models.TransferRequest)
	SendTransferFailedNotification(transfer *models.TransferRequest, reason string)
}

// NotificationChannel represents different notification delivery methods
type NotificationChannel string

const (
	NotificationChannelWebhook NotificationChannel = "webhook"
	NotificationChannelEmail   NotificationChannel = "email"
	NotificationChannelInApp   NotificationChannel = "in_app"
	NotificationChannelSMS     NotificationChannel = "sms"
	NotificationChannelSlack   NotificationChannel = "slack"
)

// NotificationType represents different types of notifications
type NotificationType string

const (
	NotificationTypeTransferStatusChange NotificationType = "transfer_status_change"
	NotificationTypePendingApproval      NotificationType = "pending_approval"
	NotificationTypeTransferCreated      NotificationType = "transfer_created"
	NotificationTypeTransferCompleted    NotificationType = "transfer_completed"
	NotificationTypeTransferFailed       NotificationType = "transfer_failed"
	NotificationTypeApprovalExpiring     NotificationType = "approval_expiring"
)

// NotificationPriority represents the urgency of a notification
type NotificationPriority string

const (
	NotificationPriorityLow      NotificationPriority = "low"
	NotificationPriorityNormal   NotificationPriority = "normal"
	NotificationPriorityHigh     NotificationPriority = "high"
	NotificationPriorityCritical NotificationPriority = "critical"
)

// Notification represents a notification message
type Notification struct {
	ID          string                 `json:"id"`
	Type        NotificationType       `json:"type"`
	Priority    NotificationPriority   `json:"priority"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Recipients  []string               `json:"recipients"`
	Channels    []NotificationChannel  `json:"channels"`
	Data        map[string]interface{} `json:"data"`
	CreatedAt   time.Time              `json:"createdAt"`
	ScheduledAt *time.Time             `json:"scheduledAt,omitempty"`
	DeliveredAt *time.Time             `json:"deliveredAt,omitempty"`
	FailedAt    *time.Time             `json:"failedAt,omitempty"`
	RetryCount  int                    `json:"retryCount"`
	MaxRetries  int                    `json:"maxRetries"`
}

// NotificationConfig configures the notification service
type NotificationConfig struct {
	DefaultChannels []NotificationChannel `json:"defaultChannels"`
	WebhookURL      string                `json:"webhookUrl,omitempty"`
	EmailConfig     *EmailConfig          `json:"emailConfig,omitempty"`
	SlackConfig     *SlackConfig          `json:"slackConfig,omitempty"`
	RetryAttempts   int                   `json:"retryAttempts"`
	RetryDelay      time.Duration         `json:"retryDelay"`
	BatchSize       int                   `json:"batchSize"`
	QueueSize       int                   `json:"queueSize"`
	Workers         int                   `json:"workers"`
}

// EmailConfig contains email notification configuration
type EmailConfig struct {
	SMTPHost    string `json:"smtpHost"`
	SMTPPort    int    `json:"smtpPort"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	FromAddress string `json:"fromAddress"`
	FromName    string `json:"fromName"`
	UseSTARTTLS bool   `json:"useStartTLS"`
}

// SlackConfig contains Slack notification configuration
type SlackConfig struct {
	WebhookURL string `json:"webhookUrl"`
	Channel    string `json:"channel"`
	Username   string `json:"username"`
	IconEmoji  string `json:"iconEmoji"`
}

// DefaultNotificationConfig returns sensible defaults
func DefaultNotificationConfig() NotificationConfig {
	return NotificationConfig{
		DefaultChannels: []NotificationChannel{NotificationChannelInApp},
		RetryAttempts:   3,
		RetryDelay:      5 * time.Second,
		BatchSize:       10,
		QueueSize:       1000,
		Workers:         2,
	}
}

// notificationService implements NotificationService
type notificationService struct {
	config    NotificationConfig
	logger    Logger
	queue     chan *Notification
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.RWMutex

	// In-memory storage for demo (in production, use database)
	notifications   map[string]*Notification
	notificationsMu sync.RWMutex
}

// NewNotificationService creates a new notification service
func NewNotificationService(config NotificationConfig, logger Logger) NotificationService {
	ctx, cancel := context.WithCancel(context.Background())

	service := &notificationService{
		config:        config,
		logger:        logger,
		queue:         make(chan *Notification, config.QueueSize),
		ctx:           ctx,
		cancel:        cancel,
		notifications: make(map[string]*Notification),
	}

	// Start worker goroutines
	service.start()

	return service
}

// start begins the notification workers
func (ns *notificationService) start() {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if ns.isRunning {
		return
	}

	ns.isRunning = true
	ns.logger.Info("Starting notification service",
		"workers", ns.config.Workers,
		"queue_size", ns.config.QueueSize,
	)

	// Start worker goroutines
	for i := 0; i < ns.config.Workers; i++ {
		ns.wg.Add(1)
		go ns.worker(i)
	}
}

// stop gracefully stops the notification service
func (ns *notificationService) stop() {
	ns.mu.Lock()
	if !ns.isRunning {
		ns.mu.Unlock()
		return
	}
	ns.isRunning = false
	ns.mu.Unlock()

	ns.logger.Info("Stopping notification service")

	close(ns.queue)
	ns.cancel()
	ns.wg.Wait()

	ns.logger.Info("Notification service stopped")
}

// worker processes notifications from the queue
func (ns *notificationService) worker(workerID int) {
	defer ns.wg.Done()

	ns.logger.Debug("Starting notification worker", "worker_id", workerID)

	for {
		select {
		case notification, ok := <-ns.queue:
			if !ok {
				ns.logger.Debug("Notification queue closed, worker stopping", "worker_id", workerID)
				return
			}
			ns.processNotification(notification)

		case <-ns.ctx.Done():
			ns.logger.Debug("Worker context cancelled", "worker_id", workerID)
			return
		}
	}
}

// processNotification handles delivery of a single notification
func (ns *notificationService) processNotification(notification *Notification) {
	ns.logger.Info("Processing notification",
		"id", notification.ID,
		"type", notification.Type,
		"priority", notification.Priority,
		"channels", notification.Channels,
	)

	success := false
	var lastError error

	// Try each configured channel
	for _, channel := range notification.Channels {
		switch channel {
		case NotificationChannelWebhook:
			if err := ns.sendWebhook(notification); err != nil {
				ns.logger.Error("Failed to send webhook notification",
					"notification_id", notification.ID,
					"error", err,
				)
				lastError = err
			} else {
				success = true
			}

		case NotificationChannelInApp:
			if err := ns.sendInApp(notification); err != nil {
				ns.logger.Error("Failed to send in-app notification",
					"notification_id", notification.ID,
					"error", err,
				)
				lastError = err
			} else {
				success = true
			}

		case NotificationChannelSlack:
			if err := ns.sendSlack(notification); err != nil {
				ns.logger.Error("Failed to send Slack notification",
					"notification_id", notification.ID,
					"error", err,
				)
				lastError = err
			} else {
				success = true
			}

		default:
			ns.logger.Warn("Unsupported notification channel",
				"channel", channel,
				"notification_id", notification.ID,
			)
		}
	}

	// Update notification status
	now := time.Now()
	if success {
		notification.DeliveredAt = &now
		ns.logger.Info("Notification delivered successfully",
			"id", notification.ID,
			"type", notification.Type,
		)
	} else {
		notification.RetryCount++
		if notification.RetryCount >= notification.MaxRetries {
			notification.FailedAt = &now
			ns.logger.Error("Notification failed after max retries",
				"id", notification.ID,
				"type", notification.Type,
				"retry_count", notification.RetryCount,
				"last_error", lastError,
			)
		} else {
			// Retry after delay
			go ns.scheduleRetry(notification)
		}
	}

	// Store notification (in production, save to database)
	ns.storeNotification(notification)
}

// scheduleRetry schedules a notification for retry
func (ns *notificationService) scheduleRetry(notification *Notification) {
	delay := ns.config.RetryDelay * time.Duration(notification.RetryCount)

	ns.logger.Info("Scheduling notification retry",
		"id", notification.ID,
		"retry_count", notification.RetryCount,
		"delay", delay,
	)

	time.Sleep(delay)

	select {
	case ns.queue <- notification:
		// Queued for retry
	case <-ns.ctx.Done():
		// Service is shutting down
	default:
		// Queue is full
		ns.logger.Error("Failed to queue notification retry, queue full",
			"id", notification.ID,
		)
	}
}

// storeNotification stores the notification (in-memory for demo)
func (ns *notificationService) storeNotification(notification *Notification) {
	ns.notificationsMu.Lock()
	defer ns.notificationsMu.Unlock()
	ns.notifications[notification.ID] = notification
}

// sendWebhook sends notification via webhook
func (ns *notificationService) sendWebhook(notification *Notification) error {
	if ns.config.WebhookURL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	// In a real implementation, make HTTP POST request to webhook URL
	ns.logger.Info("Sending webhook notification",
		"url", ns.config.WebhookURL,
		"notification_id", notification.ID,
	)

	return nil // Simulated success
}

// sendInApp stores notification for in-app display
func (ns *notificationService) sendInApp(notification *Notification) error {
	ns.logger.Info("Storing in-app notification",
		"notification_id", notification.ID,
		"recipients", notification.Recipients,
	)

	// In a real implementation, store in database for in-app display
	return nil // Simulated success
}

// sendSlack sends notification to Slack
func (ns *notificationService) sendSlack(notification *Notification) error {
	if ns.config.SlackConfig == nil || ns.config.SlackConfig.WebhookURL == "" {
		return fmt.Errorf("Slack webhook URL not configured")
	}

	ns.logger.Info("Sending Slack notification",
		"webhook_url", ns.config.SlackConfig.WebhookURL,
		"notification_id", notification.ID,
	)

	// In a real implementation, send to Slack webhook
	return nil // Simulated success
}

// enqueueNotification adds a notification to the processing queue
func (ns *notificationService) enqueueNotification(notification *Notification) {
	// Set defaults
	if notification.ID == "" {
		notification.ID = uuid.New().String()
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}
	if notification.MaxRetries == 0 {
		notification.MaxRetries = ns.config.RetryAttempts
	}
	if len(notification.Channels) == 0 {
		notification.Channels = ns.config.DefaultChannels
	}

	select {
	case ns.queue <- notification:
		ns.logger.Debug("Notification queued",
			"id", notification.ID,
			"type", notification.Type,
		)
	default:
		ns.logger.Error("Notification queue full, dropping notification",
			"id", notification.ID,
			"type", notification.Type,
		)
	}
}

// SendTransferStatusNotification sends notification when transfer status changes
func (ns *notificationService) SendTransferStatusNotification(transfer *models.TransferRequest, oldStatus, newStatus models.TransferStatus) {
	notification := &Notification{
		Type:       NotificationTypeTransferStatusChange,
		Priority:   ns.getStatusChangePriority(oldStatus, newStatus),
		Title:      fmt.Sprintf("Transfer Status Updated"),
		Message:    fmt.Sprintf("Transfer %s status changed from %s to %s", transfer.ID, oldStatus, newStatus),
		Recipients: []string{transfer.RequestedByUserID.String()},
		Data: map[string]interface{}{
			"transfer_id": transfer.ID.String(),
			"old_status":  string(oldStatus),
			"new_status":  string(newStatus),
			"amount":      transfer.AmountString,
			"coin":        transfer.Coin,
			"recipient":   transfer.RecipientAddress,
		},
	}

	ns.enqueueNotification(notification)
}

// SendPendingApprovalNotification sends notification about pending approvals
func (ns *notificationService) SendPendingApprovalNotification(transfer *models.TransferRequest, approval *bitgo.ApprovalStatus) {
	notification := &Notification{
		Type:       NotificationTypePendingApproval,
		Priority:   NotificationPriorityHigh,
		Title:      fmt.Sprintf("Transfer Requires Approval"),
		Message:    fmt.Sprintf("Transfer %s requires %d approval(s). %d received, %d pending.", transfer.ID, approval.RequiredApprovals, approval.ReceivedApprovals, approval.PendingApprovals),
		Recipients: []string{transfer.RequestedByUserID.String()}, // In real app, send to approvers
		Data: map[string]interface{}{
			"transfer_id":        transfer.ID.String(),
			"approval_id":        approval.ID,
			"required_approvals": approval.RequiredApprovals,
			"received_approvals": approval.ReceivedApprovals,
			"pending_approvals":  approval.PendingApprovals,
			"expires_at":         approval.Expires,
			"time_remaining":     approval.TimeRemaining.String(),
		},
	}

	ns.enqueueNotification(notification)
}

// SendTransferCreatedNotification sends notification when transfer is created
func (ns *notificationService) SendTransferCreatedNotification(transfer *models.TransferRequest) {
	notification := &Notification{
		Type:       NotificationTypeTransferCreated,
		Priority:   NotificationPriorityNormal,
		Title:      fmt.Sprintf("Transfer Created"),
		Message:    fmt.Sprintf("Transfer of %s %s to %s has been created", transfer.AmountString, transfer.Coin, transfer.RecipientAddress),
		Recipients: []string{transfer.RequestedByUserID.String()},
		Data: map[string]interface{}{
			"transfer_id": transfer.ID.String(),
			"amount":      transfer.AmountString,
			"coin":        transfer.Coin,
			"recipient":   transfer.RecipientAddress,
		},
	}

	ns.enqueueNotification(notification)
}

// SendTransferCompletedNotification sends notification when transfer completes
func (ns *notificationService) SendTransferCompletedNotification(transfer *models.TransferRequest) {
	notification := &Notification{
		Type:       NotificationTypeTransferCompleted,
		Priority:   NotificationPriorityNormal,
		Title:      fmt.Sprintf("Transfer Completed"),
		Message:    fmt.Sprintf("Transfer of %s %s has been completed successfully", transfer.AmountString, transfer.Coin),
		Recipients: []string{transfer.RequestedByUserID.String()},
		Data: map[string]interface{}{
			"transfer_id":      transfer.ID.String(),
			"amount":           transfer.AmountString,
			"coin":             transfer.Coin,
			"recipient":        transfer.RecipientAddress,
			"transaction_hash": transfer.TransactionHash,
		},
	}

	ns.enqueueNotification(notification)
}

// SendTransferFailedNotification sends notification when transfer fails
func (ns *notificationService) SendTransferFailedNotification(transfer *models.TransferRequest, reason string) {
	notification := &Notification{
		Type:       NotificationTypeTransferFailed,
		Priority:   NotificationPriorityHigh,
		Title:      fmt.Sprintf("Transfer Failed"),
		Message:    fmt.Sprintf("Transfer of %s %s has failed: %s", transfer.AmountString, transfer.Coin, reason),
		Recipients: []string{transfer.RequestedByUserID.String()},
		Data: map[string]interface{}{
			"transfer_id": transfer.ID.String(),
			"amount":      transfer.AmountString,
			"coin":        transfer.Coin,
			"recipient":   transfer.RecipientAddress,
			"reason":      reason,
		},
	}

	ns.enqueueNotification(notification)
}

// getStatusChangePriority determines notification priority based on status change
func (ns *notificationService) getStatusChangePriority(oldStatus, newStatus models.TransferStatus) NotificationPriority {
	switch newStatus {
	case models.TransferStatusCompleted:
		return NotificationPriorityNormal
	case models.TransferStatusFailed, models.TransferStatusRejected:
		return NotificationPriorityHigh
	case models.TransferStatusPendingApproval:
		return NotificationPriorityHigh
	case models.TransferStatusBroadcast:
		return NotificationPriorityNormal
	default:
		return NotificationPriorityLow
	}
}
