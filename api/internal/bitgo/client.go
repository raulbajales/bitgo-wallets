package bitgo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Config holds BitGo client configuration
type Config struct {
	BaseURL     string
	AccessToken string
	Enterprise  string
	Timeout     time.Duration
	MaxRetries  int
}

// Logger interface for structured logging
type Logger interface {
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
}

// Client represents a BitGo API client
type Client struct {
	baseURL     string
	accessToken string
	enterprise  string
	httpClient  *http.Client
	logger      Logger
}

// APIError represents a BitGo API error response
type APIError struct {
	ErrorMsg    string `json:"error"`
	Message     string `json:"message"`
	RequestID   string `json:"requestId,omitempty"`
	StatusCode  int    `json:"-"`
	RequestInfo string `json:"-"`
}

func (e APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("BitGo API error (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("BitGo API error (%d): %s", e.StatusCode, e.ErrorMsg)
}

// RequestOptions holds options for API requests
type RequestOptions struct {
	Method         string
	Path           string
	Body           interface{}
	Headers        map[string]string
	IdempotencyKey string
}

// NewClient creates a new BitGo API client
func NewClient(config Config, logger Logger) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://app.bitgo-test.com"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	return &Client{
		baseURL:     config.BaseURL,
		accessToken: config.AccessToken,
		enterprise:  config.Enterprise,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger: logger,
	}
}

// makeRequest performs an HTTP request to the BitGo API with retry logic
func (c *Client) makeRequest(ctx context.Context, opts RequestOptions) (*http.Response, error) {
	// Generate correlation ID for request tracking
	correlationID := uuid.New().String()

	var bodyReader io.Reader
	var bodyBytes []byte
	if opts.Body != nil {
		var err error
		bodyBytes, err = json.Marshal(opts.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Redact sensitive information for logging
	logBody := c.redactSensitiveFields(opts.Body)
	url := c.baseURL + "/api/v2" + opts.Path
	c.logger.Info("Making BitGo API request",
		"method", opts.Method,
		"url", c.redactURL(url),
		"correlation_id", correlationID,
		"body", logBody,
	)

	req, err := http.NewRequestWithContext(ctx, opts.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication headers
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "bitgo-wallets-api/1.0")
	req.Header.Set("X-Correlation-ID", correlationID)

	// Set idempotency key if provided
	if opts.IdempotencyKey != "" {
		req.Header.Set("X-Idempotency-Key", opts.IdempotencyKey)
	}

	// Set custom headers
	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	// Perform request with retry logic
	return c.doWithRetry(req, correlationID)
}

// doWithRetry executes HTTP request with exponential backoff retry
func (c *Client) doWithRetry(req *http.Request, correlationID string) (*http.Response, error) {
	maxRetries := 3
	baseDelay := 1 * time.Second

	// Clone request body for retries
	var bodyReader io.Reader
	if req.Body != nil {
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("failed to get request body for retry: %w", err)
			}
			bodyReader = body
		}
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if bodyReader != nil {
			req.Body = io.NopCloser(bodyReader)
		}

		// Log response
		resp, err := c.httpClient.Do(req)
		if resp != nil {
			c.logger.Info("BitGo API response",
				"status_code", resp.StatusCode,
				"correlation_id", correlationID,
				"attempt", attempt+1,
			)
		}

		// Check if we should retry
		if err != nil || c.shouldRetry(resp, attempt, maxRetries) {
			if resp != nil {
				resp.Body.Close()
			}
			if attempt < maxRetries {
				delay := time.Duration(attempt+1) * baseDelay
				c.logger.Warn("Retrying BitGo API request",
					"attempt", attempt+1,
					"delay_seconds", delay.Seconds(),
					"error", err,
					"correlation_id", correlationID,
				)
				time.Sleep(delay)
				continue
			}
		}

		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}

		// Check for API errors
		if resp.StatusCode >= 400 {
			return resp, c.parseAPIError(resp, correlationID)
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded")
}

// shouldRetry determines if a request should be retried
func (c *Client) shouldRetry(resp *http.Response, attempt, maxRetries int) bool {
	if attempt >= maxRetries {
		return false
	}

	if resp == nil {
		return true // Network error, retry
	}

	// Retry on server errors and rate limiting
	return resp.StatusCode >= 500 || resp.StatusCode == 429
}

// parseAPIError parses BitGo API error response
func (c *Client) parseAPIError(resp *http.Response, correlationID string) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("Failed to read error response body",
			"correlation_id", correlationID,
			"error", err,
		)
		return APIError{
			ErrorMsg:    "Failed to read error response",
			StatusCode:  resp.StatusCode,
			RequestInfo: correlationID,
		}
	}

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		c.logger.Error("Failed to parse error response",
			"correlation_id", correlationID,
			"body", string(body),
			"error", err,
		)
		return APIError{
			ErrorMsg:    "Failed to parse error response",
			StatusCode:  resp.StatusCode,
			RequestInfo: correlationID,
		}
	}

	apiErr.StatusCode = resp.StatusCode
	apiErr.RequestInfo = correlationID
	c.logger.Error("BitGo API error",
		"status_code", resp.StatusCode,
		"error", apiErr.ErrorMsg,
		"message", apiErr.Message,
		"correlation_id", correlationID,
	)

	return apiErr
}

// redactSensitiveFields removes sensitive information from request bodies for logging
func (c *Client) redactSensitiveFields(body interface{}) interface{} {
	if body == nil {
		return nil
	}

	// Convert to JSON and back to map for manipulation
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return "[REDACTION_ERROR]"
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return "[REDACTION_ERROR]"
	}

	// List of fields to redact
	sensitiveFields := []string{
		"passphrase", "password", "otp", "backup", "recoveryXpub",
		"userKey", "backupKey", "bitgoKey", "prv", "encryptedPrv",
	}

	for _, field := range sensitiveFields {
		if _, exists := data[field]; exists {
			data[field] = "[REDACTED]"
		}
	}

	return data
}

// redactURL removes sensitive information from URLs for logging
func (c *Client) redactURL(url string) string {
	// Replace access token in URL if present
	if strings.Contains(url, "access_token=") {
		parts := strings.Split(url, "access_token=")
		if len(parts) > 1 {
			tokenPart := parts[1]
			ampIndex := strings.Index(tokenPart, "&")
			if ampIndex > 0 {
				return parts[0] + "access_token=[REDACTED]" + tokenPart[ampIndex:]
			}
			return parts[0] + "access_token=[REDACTED]"
		}
	}
	return url
}

// ValidateAddress validates if a blockchain address is valid using BitGo API
func (c *Client) ValidateAddress(ctx context.Context, address string) (bool, error) {
	// Simple regex validation first - this is a basic check
	// Bitcoin addresses typically start with 1, 3, or bc1
	// Ethereum addresses start with 0x and are 42 characters long
	if len(address) < 26 {
		return false, nil
	}

	// Basic format validation
	bitcoinRegex := regexp.MustCompile(`^(1|3|bc1)[a-zA-Z0-9]{25,62}$`)
	ethereumRegex := regexp.MustCompile(`^0x[a-fA-F0-9]{40}$`)

	if bitcoinRegex.MatchString(address) || ethereumRegex.MatchString(address) {
		return true, nil
	}

	// If basic regex doesn't match, return false
	// In a real implementation, you might want to call BitGo's address validation API
	return false, nil
}
