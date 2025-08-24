package command

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/peak/s5cmd/v2/log"
)

// RetryConfig configures retry behavior for client copy operations
type RetryConfig struct {
	MaxRetries      int
	BaseDelay       time.Duration
	MaxDelay        time.Duration
	BackoffExponent float64
	Jitter          bool
}

// DefaultClientCopyRetryConfig returns default retry configuration for client copy
func DefaultClientCopyRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      3,
		BaseDelay:       1 * time.Second,
		MaxDelay:        30 * time.Second,
		BackoffExponent: 2.0,
		Jitter:          true,
	}
}

// IsRetryableError determines if an error is retryable for client copy operations
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Network and connectivity errors
	retryablePatterns := []string{
		"connection",
		"timeout",
		"temporary failure",
		"service unavailable",
		"internal error",
		"slow down",
		"throttling",
		"rate limit",
		"too many requests",
		"request timeout",
		"dial tcp",
		"connection reset",
		"connection refused",
		"no such host",
		"i/o timeout",
		"context deadline exceeded",
		"eof",
		"unexpected eof",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	// AWS-specific retryable errors
	awsRetryablePatterns := []string{
		"provisionedthroughputexceeded",
		"throttlingexception",
		"requestlimitexceeded",
		"serviceunavailable",
		"internalerror",
		"slowdown",
		"requesttimeout",
	}

	for _, pattern := range awsRetryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// CalculateDelay calculates the delay for a retry attempt with exponential backoff
func (rc RetryConfig) CalculateDelay(attempt int) time.Duration {
	if attempt < 0 {
		return 0
	}

	// Calculate exponential backoff delay
	delay := float64(rc.BaseDelay) * math.Pow(rc.BackoffExponent, float64(attempt))

	// Apply jitter if enabled (±25% randomization)
	if rc.Jitter && delay > 0 {
		jitterRange := delay * 0.25
		jitter := (rand.Float64() - 0.5) * 2 * jitterRange
		delay += jitter
	}

	// Ensure delay is within bounds
	if delay < 0 {
		delay = float64(rc.BaseDelay)
	}
	if delay > float64(rc.MaxDelay) {
		delay = float64(rc.MaxDelay)
	}

	return time.Duration(delay)
}

// WithRetry executes a function with retry logic and exponential backoff
func WithRetry(ctx context.Context, config RetryConfig, operation func() error, operationName string) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the operation
		err := operation()
		if err == nil {
			// Success - log retry success if this wasn't the first attempt
			if attempt > 0 {
				log.Debug(log.DebugMessage{
					Err: fmt.Sprintf("Client copy: %s succeeded after %d retries", operationName, attempt),
				})
			}
			return nil
		}

		lastErr = err

		// Check if this is the last attempt
		if attempt == config.MaxRetries {
			break
		}

		// Check if the error is retryable
		if !IsRetryableError(err) {
			log.Debug(log.DebugMessage{
				Err: fmt.Sprintf("Client copy: %s failed with non-retryable error: %v", operationName, err),
			})
			return err
		}

		// Calculate delay for next attempt
		delay := config.CalculateDelay(attempt)

		log.Debug(log.DebugMessage{
			Err: fmt.Sprintf("Client copy: %s failed (attempt %d/%d), retrying in %v: %v",
				operationName, attempt+1, config.MaxRetries+1, delay, err),
		})

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All retries exhausted
	log.Debug(log.DebugMessage{
		Err: fmt.Sprintf("Client copy: %s failed after %d retries: %v", operationName, config.MaxRetries+1, lastErr),
	})

	return fmt.Errorf("operation failed after %d retries: %w", config.MaxRetries+1, lastErr)
}

// RetryableClientCopyOperation wraps client copy operations with retry logic
type RetryableClientCopyOperation struct {
	config RetryConfig
}

// NewRetryableClientCopyOperation creates a new retryable operation wrapper
func NewRetryableClientCopyOperation() *RetryableClientCopyOperation {
	return &RetryableClientCopyOperation{
		config: DefaultClientCopyRetryConfig(),
	}
}

// WithCustomConfig sets a custom retry configuration
func (r *RetryableClientCopyOperation) WithCustomConfig(config RetryConfig) *RetryableClientCopyOperation {
	r.config = config
	return r
}

// ExecuteDownload executes a download operation with retry logic
func (r *RetryableClientCopyOperation) ExecuteDownload(ctx context.Context, downloadFunc func() error) error {
	return WithRetry(ctx, r.config, downloadFunc, "download")
}

// ExecuteUpload executes an upload operation with retry logic
func (r *RetryableClientCopyOperation) ExecuteUpload(ctx context.Context, uploadFunc func() error) error {
	return WithRetry(ctx, r.config, uploadFunc, "upload")
}
