package bitgo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Logger interface for structured logging
type Logger interface {
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
}

// Client represents a BitGo API client
type Client struct {
	httpClient  *http.Client
	baseURL     string
	accessToken string
	logger      Logger
	apiVersion  string
	idempotency *IdempotencyService
}

// Config represents the BitGo client configuration
type Config struct {
	BaseURL     string
	AccessToken string
	Logger      Logger
	Timeout     time.Duration
	MaxRetries  int
	APIVersion  string
}

// APIError represents a BitGo API error response
type APIError struct {
	StatusCode int    `json:"statusCode"`
	Error      string `json:"error"`
	Message    string `json:"message"`
	RequestID  string `json:"requestId"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("BitGo API error (status %d): %s - %s", e.StatusCode, e.Error, e.Message)
}

// NewClient creates a new BitGo API client
func NewClient(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://app.bitgo.com"
	}
	if config.APIVersion == "" {
		config.APIVersion = "v2"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	
	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}
	
	// Create idempotency service
	idempotency := NewIdempotencyService(config.Logger, 24*time.Hour)
	
	return &Client{
		httpClient:  httpClient,
		baseURL:     config.BaseURL,
		accessToken: config.AccessToken,
		logger:      config.Logger,
		apiVersion:  config.APIVersion,
		idempotency: idempotency,
	}
}

// makeRequest performs an HTTP request with retry logic and error handling
func (c *Client) makeRequest(ctx context.Context, method, path string, body interface{}, response interface{}) error {
	maxRetries := 3
	backoffDuration := time.Second
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := c.doRequest(ctx, method, path, body, response)
		
		// Check if we should retry
		if err != nil && attempt < maxRetries {
			if apiErr, ok := err.(*APIError); ok {
				// Retry on 5xx errors and rate limiting
				if apiErr.StatusCode >= 500 || apiErr.StatusCode == 429 {
					c.logger.Warn("Request failed, retrying",
						"attempt", attempt+1,
						"max_retries", maxRetries,
						"status_code", apiErr.StatusCode,
						"error", apiErr.Message,
					)
					
					// Exponential backoff
					select {
					case <-time.After(backoffDuration):
						backoffDuration *= 2
					case <-ctx.Done():
						return ctx.Err()
					}
					continue
				}
			}
		}
		
		return err
	}
	
	return fmt.Errorf("request failed after %d attempts", maxRetries+1)
}

// doRequest performs a single HTTP request
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, response interface{}) error {
	// Build URL
	u, err := url.Parse(c.baseURL + "/api/" + c.apiVersion + path)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	
	// Generate correlation ID for request tracking
	correlationID := uuid.New().String()
	
	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("X-Correlation-ID", correlationID)
	req.Header.Set("User-Agent", "BitGo-Wallets-Client/1.0")
	
	// Log request (with sensitive data redacted)
	c.logRequest(req, body, correlationID)
	
	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("HTTP request failed",
			"method", method,
			"url", c.redactURL(u.String()),
			"correlation_id", correlationID,
			"error", err,
		)
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Log response (with sensitive data redacted)
	c.logResponse(resp, respBody, correlationID)
	
	// Check for API errors
	if resp.StatusCode >= 400 {
		apiErr := c.parseAPIError(resp.StatusCode, respBody, correlationID)
		return apiErr
	}
	
	// Parse response
	if response != nil {
		if err := json.Unmarshal(respBody, response); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}
	
	return nil
}

// logRequest logs HTTP requests with sensitive data redacted
func (c *Client) logRequest(req *http.Request, body interface{}, correlationID string) {
	fields := []interface{}{
		"method", req.Method,
		"url", c.redactURL(req.URL.String()),
		"correlation_id", correlationID,
	}
	
	if body != nil {
		redactedBody := c.redactSensitiveFields(body)
		fields = append(fields, "body", redactedBody)
	}
	
	c.logger.Info("Making BitGo API request", fields...)
}

// logResponse logs HTTP responses with sensitive data redacted
func (c *Client) logResponse(resp *http.Response, body []byte, correlationID string) {
	var responseData interface{}
	if len(body) > 0 {
		json.Unmarshal(body, &responseData)
		responseData = c.redactSensitiveFields(responseData)
	}
	
	c.logger.Info("Received BitGo API response",
		"status_code", resp.StatusCode,
		"correlation_id", correlationID,
		"response", responseData,
	)
}

// parseAPIError parses BitGo API error responses
func (c *Client) parseAPIError(statusCode int, body []byte, correlationID string) *APIError {
	var apiErr APIError
	
	if len(body) > 0 {
		json.Unmarshal(body, &apiErr)
	}
	
	apiErr.StatusCode = statusCode
	apiErr.RequestID = correlationID
	
	// Set default error message if not provided
	if apiErr.Error == "" {
		apiErr.Error = fmt.Sprintf("HTTP %d", statusCode)
	}
	if apiErr.Message == "" {
		apiErr.Message = "Unknown error"
	}
	
	return &apiErr
}

// redactSensitiveFields removes sensitive information from logged data
func (c *Client) redactSensitiveFields(data interface{}) interface{} {
	if data == nil {
		return nil
	}
	
	// Convert to JSON and back to get a clean map/slice structure
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "[REDACTION_ERROR]"
	}
	
	var result interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return "[REDACTION_ERROR]"
	}
	
	return c.redactValue(result)
}

// redactValue recursively redacts sensitive fields
func (c *Client) redactValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			if c.isSensitiveField(key) {
				result[key] = "[REDACTED]"
			} else {
				result[key] = c.redactValue(val)
			}
		}
		return result
		
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = c.redactValue(val)
		}
		return result
		
	default:
		return value
	}
}

// isSensitiveField checks if a field name contains sensitive information
func (c *Client) isSensitiveField(fieldName string) bool {
	sensitiveFields := []string{
		"passphrase", "password", "token", "secret", "key", "private",
		"accessToken", "privateKey", "walletPassphrase", "otp",
		"xprv", "seed", "mnemonic",
	}
	
	fieldLower := strings.ToLower(fieldName)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(fieldLower, sensitive) {
			return true
		}
	}
	return false
}

// redactURL removes sensitive information from URLs
func (c *Client) redactURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "[INVALID_URL]"
	}
	
	// Remove query parameters that might contain sensitive data
	sensitiveParams := []string{"token", "key", "secret", "passphrase"}
	query := u.Query()
	
	for param := range query {
		for _, sensitive := range sensitiveParams {
			if strings.Contains(strings.ToLower(param), sensitive) {
				query.Set(param, "[REDACTED]")
			}
		}
	}
	
	u.RawQuery = query.Encode()
	return u.String()
}

// GetIdempotencyService returns the idempotency service
func (c *Client) GetIdempotencyService() *IdempotencyService {
	return c.idempotency
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




}	return url	}		}			return parts[0] + "access_token=[REDACTED]"			}				return parts[0] + "access_token=[REDACTED]" + tokenPart[ampIndex:]			if ampIndex > 0 {			ampIndex := strings.Index(tokenPart, "&")			tokenPart := parts[1]		if len(parts) > 1 {		parts := strings.Split(url, "access_token=")	if strings.Contains(url, "access_token=") {	// Replace access token in URL if presentfunc (c *Client) redactURL(url string) string {// redactURL removes sensitive information from URLs for logging}	return data	}		}			data[field] = "[REDACTED]"		if _, exists := data[field]; exists {	for _, field := range sensitiveFields {	}		"userKey", "backupKey", "bitgoKey", "prv", "encryptedPrv",		"passphrase", "password", "otp", "backup", "recoveryXpub",	sensitiveFields := []string{	// List of fields to redact	}		return "[REDACTION_ERROR]"	if err := json.Unmarshal(jsonBytes, &data); err != nil {	var data map[string]interface{}	}		return "[REDACTION_ERROR]"	if err != nil {	jsonBytes, err := json.Marshal(body)	// Convert to JSON and back to map for manipulation	}		return nil	if body == nil {func (c *Client) redactSensitiveFields(body interface{}) interface{} {// redactSensitiveFields removes sensitive information from request bodies for logging}	return apiErr	)		"correlation_id", correlationID,		"message", apiErr.Message,		"error", apiErr.Error,		"status_code", resp.StatusCode,	c.logger.Error("BitGo API error",		apiErr.RequestInfo = correlationID	apiErr.StatusCode = resp.StatusCode	}		}			RequestInfo: correlationID,			StatusCode:  resp.StatusCode,			Error:       "Failed to parse error response",		return APIError{		)			"error", err,			"body", string(body),			"correlation_id", correlationID,		c.logger.Error("Failed to parse error response",	if err := json.Unmarshal(body, &apiErr); err != nil {	var apiErr APIError	}		}			RequestInfo: correlationID,			StatusCode:  resp.StatusCode,			Error:       "Failed to read error response",		return APIError{		)			"error", err,			"correlation_id", correlationID,		c.logger.Error("Failed to read error response body",	if err != nil {	body, err := io.ReadAll(resp.Body)func (c *Client) parseAPIError(resp *http.Response, correlationID string) error {// parseAPIError parses BitGo API error response}	return resp.StatusCode >= 500 || resp.StatusCode == 429	// Retry on server errors and rate limiting		}		return true // Network error, retry	if resp == nil {		}		return false	if attempt >= maxRetries {func (c *Client) shouldRetry(resp *http.Response, attempt, maxRetries int) bool {// shouldRetry determines if a request should be retried}	return nil, fmt.Errorf("max retries exceeded")	}		return resp, nil		}			return resp, c.parseAPIError(resp, correlationID)		if resp.StatusCode >= 400 {		// Check for API errors		}			return nil, fmt.Errorf("HTTP request failed: %w", err)		if err != nil {		}			}				continue				time.Sleep(delay)				)					"correlation_id", correlationID,					"error", err,					"delay_seconds", delay.Seconds(),					"attempt", attempt+1,				c.logger.Warn("Retrying BitGo API request",				delay := time.Duration(attempt+1) * baseDelay			if attempt < maxRetries {						}				resp.Body.Close()			if resp != nil {		if err != nil || c.shouldRetry(resp, attempt, maxRetries) {		// Check if we should retry		}			)				"attempt", attempt+1,				"correlation_id", correlationID,				"status_code", resp.StatusCode,			c.logger.Info("BitGo API response",		if resp != nil {		// Log response				resp, err := c.httpClient.Do(req)		}			req.Body = io.NopCloser(bodyReader)			}				bodyReader = body				}					return nil, fmt.Errorf("failed to get request body for retry: %w", err)				if err != nil {				body, err := req.GetBody()			if req.GetBody != nil {		if req.Body != nil {		var bodyReader io.Reader		// Clone request body for retries	for attempt := 0; attempt <= maxRetries; attempt++ {		baseDelay := 1 * time.Second	maxRetries := 3func (c *Client) doWithRetry(req *http.Request, correlationID string) (*http.Response, error) {// doWithRetry executes HTTP request with exponential backoff retry}	return c.doWithRetry(req, correlationID)	// Perform request with retry logic	}		req.Header.Set(key, value)	for key, value := range opts.Headers {	// Set custom headers		}		req.Header.Set("X-Idempotency-Key", opts.IdempotencyKey)	if opts.IdempotencyKey != "" {	// Set idempotency key if provided		req.Header.Set("X-Correlation-ID", correlationID)	req.Header.Set("User-Agent", "bitgo-wallets-api/1.0")	req.Header.Set("Content-Type", "application/json")	req.Header.Set("Authorization", "Bearer "+c.accessToken)	// Set authentication headers	}		return nil, fmt.Errorf("failed to create request: %w", err)	if err != nil {	req, err := http.NewRequestWithContext(ctx, opts.Method, url, bodyReader)	)		"body", logBody,		"correlation_id", correlationID,		"url", c.redactURL(url),		"method", opts.Method,	c.logger.Info("Making BitGo API request", 	logBody := c.redactSensitiveFields(opts.Body)	// Redact sensitive information for logging		url := c.baseURL + "/api/v2" + opts.Path	}		bodyReader = bytes.NewReader(bodyBytes)		}			return nil, fmt.Errorf("failed to marshal request body: %w", err)		if err != nil {		bodyBytes, err = json.Marshal(opts.Body)		var err error	if opts.Body != nil {		var bodyBytes []byte	var bodyReader io.Reader		correlationID := uuid.New().String()	// Generate correlation ID for request trackingfunc (c *Client) makeRequest(ctx context.Context, opts RequestOptions) (*http.Response, error) {// makeRequest performs an HTTP request to the BitGo API with retry logic}	IdempotencyKey string	Headers     map[string]string	Body        interface{}	Path        string	Method      stringtype RequestOptions struct {// RequestOptions holds options for API requests}	return fmt.Sprintf("BitGo API error (%d): %s", e.StatusCode, e.Error)	}		return fmt.Sprintf("BitGo API error (%d): %s", e.StatusCode, e.Message)	if e.Message != "" {func (e APIError) Error() string {}	RequestInfo string `json:"-"`	StatusCode  int    `json:"-"`	RequestID   string `json:"requestId,omitempty"`	Message     string `json:"message"`	Error       string `json:"error"`type APIError struct {// APIError represents a BitGo API error response}	}		logger: logger,		},			Timeout: config.Timeout,		httpClient: &http.Client{		enterprise:  config.Enterprise,		accessToken: config.AccessToken,		baseURL:     config.BaseURL,	return &Client{	}		config.MaxRetries = 3	if config.MaxRetries == 0 {	}		config.Timeout = 30 * time.Second	if config.Timeout == 0 {	}		config.BaseURL = "https://app.bitgo-test.com"	if config.BaseURL == "" {func NewClient(config Config, logger Logger) *Client {// NewClient creates a new BitGo API client}	Debug(msg string, fields ...interface{})	Error(msg string, fields ...interface{})	Warn(msg string, fields ...interface{})	Info(msg string, fields ...interface{})type Logger interface {// Logger interface for structured logging}	MaxRetries  int	Timeout     time.Duration	Enterprise  string	AccessToken string	BaseURL     stringtype Config struct {// Config holds BitGo client configuration}	logger      Logger	httpClient  *http.Client	enterprise  string	accessToken string	baseURL     stringtype Client struct {// Client represents a BitGo API client)	"github.com/google/uuid"	"time"	"strings"	"net/http"	"io"	"fmt"	"encoding/json"	"context"	"bytes"import (package bitgo