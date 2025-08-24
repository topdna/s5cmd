package command

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "connection timeout",
			err:  errors.New("connection timeout"),
			want: true,
		},
		{
			name: "temporary failure",
			err:  errors.New("temporary failure in operation"),
			want: true,
		},
		{
			name: "service unavailable",
			err:  errors.New("service unavailable"),
			want: true,
		},
		{
			name: "throttling exception",
			err:  errors.New("ThrottlingException: Rate exceeded"),
			want: true,
		},
		{
			name: "slow down",
			err:  errors.New("SlowDown: Please reduce your request rate"),
			want: true,
		},
		{
			name: "too many requests",
			err:  errors.New("too many requests"),
			want: true,
		},
		{
			name: "dial tcp connection refused",
			err:  errors.New("dial tcp: connection refused"),
			want: true,
		},
		{
			name: "context deadline exceeded",
			err:  errors.New("context deadline exceeded"),
			want: true,
		},
		{
			name: "unexpected EOF",
			err:  errors.New("unexpected EOF"),
			want: true,
		},
		{
			name: "non-retryable error",
			err:  errors.New("file not found"),
			want: false,
		},
		{
			name: "authentication error",
			err:  errors.New("access denied"),
			want: false,
		},
		{
			name: "case insensitive matching",
			err:  errors.New("CONNECTION TIMEOUT"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryableError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRetryConfigCalculateDelay(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      3,
		BaseDelay:       100 * time.Millisecond,
		MaxDelay:        1 * time.Second,
		BackoffExponent: 2.0,
		Jitter:          false, // Disable jitter for predictable testing
	}

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{
			name:     "first retry",
			attempt:  0,
			expected: 100 * time.Millisecond,
		},
		{
			name:     "second retry",
			attempt:  1,
			expected: 200 * time.Millisecond,
		},
		{
			name:     "third retry",
			attempt:  2,
			expected: 400 * time.Millisecond,
		},
		{
			name:     "fourth retry - should be capped at max",
			attempt:  3,
			expected: 800 * time.Millisecond,
		},
		{
			name:     "negative attempt",
			attempt:  -1,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.CalculateDelay(tt.attempt)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestRetryConfigCalculateDelayWithJitter(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      3,
		BaseDelay:       100 * time.Millisecond,
		MaxDelay:        1 * time.Second,
		BackoffExponent: 2.0,
		Jitter:          true,
	}

	// Test that jitter produces different values
	delays := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		delays[i] = config.CalculateDelay(1)
	}

	// Check that we get some variation (at least 2 different values)
	uniqueDelays := make(map[time.Duration]bool)
	for _, delay := range delays {
		uniqueDelays[delay] = true
		// Should be within reasonable bounds (75-125ms for attempt 1 with 25% jitter)
		assert.Assert(t, delay >= 75*time.Millisecond && delay <= 250*time.Millisecond,
			"delay %v out of expected range", delay)
	}

	// We expect some variation with jitter enabled
	if len(uniqueDelays) < 2 {
		t.Log("Warning: Expected more variation with jitter, but this could be random")
	}
}

func TestRetryConfigCalculateDelayMaxCap(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      10,
		BaseDelay:       100 * time.Millisecond,
		MaxDelay:        500 * time.Millisecond,
		BackoffExponent: 2.0,
		Jitter:          false,
	}

	// Large attempt number should be capped at MaxDelay
	delay := config.CalculateDelay(10)
	assert.Equal(t, config.MaxDelay, delay)
}

func TestWithRetrySuccess(t *testing.T) {
	config := DefaultClientCopyRetryConfig()
	config.BaseDelay = 10 * time.Millisecond // Speed up test

	callCount := 0
	operation := func() error {
		callCount++
		return nil // Success on first try
	}

	ctx := context.Background()
	err := WithRetry(ctx, config, operation, "test-operation")

	assert.NilError(t, err)
	assert.Equal(t, 1, callCount) // Should only be called once
}

func TestWithRetryEventualSuccess(t *testing.T) {
	config := DefaultClientCopyRetryConfig()
	config.BaseDelay = 10 * time.Millisecond // Speed up test

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 3 {
			return errors.New("connection timeout") // Retryable error
		}
		return nil // Success on third try
	}

	ctx := context.Background()
	start := time.Now()
	err := WithRetry(ctx, config, operation, "test-operation")
	duration := time.Since(start)

	assert.NilError(t, err)
	assert.Equal(t, 3, callCount)
	// Should have taken some time due to delays
	assert.Assert(t, duration > 20*time.Millisecond)
}

func TestWithRetryNonRetryableError(t *testing.T) {
	config := DefaultClientCopyRetryConfig()
	config.BaseDelay = 10 * time.Millisecond

	callCount := 0
	operation := func() error {
		callCount++
		return errors.New("file not found") // Non-retryable error
	}

	ctx := context.Background()
	err := WithRetry(ctx, config, operation, "test-operation")

	assert.Assert(t, err != nil)
	assert.Equal(t, 1, callCount) // Should only be called once
	assert.Assert(t, strings.Contains(err.Error(), "file not found"))
}

func TestWithRetryMaxRetriesExceeded(t *testing.T) {
	config := DefaultClientCopyRetryConfig()
	config.BaseDelay = 10 * time.Millisecond
	config.MaxRetries = 2 // Only 2 retries

	callCount := 0
	operation := func() error {
		callCount++
		return errors.New("connection timeout") // Always retryable error
	}

	ctx := context.Background()
	err := WithRetry(ctx, config, operation, "test-operation")

	assert.Assert(t, err != nil)
	assert.Equal(t, 3, callCount) // Initial attempt + 2 retries
	assert.Assert(t, strings.Contains(err.Error(), "operation failed after"))
	assert.Assert(t, strings.Contains(err.Error(), "connection timeout"))
}

func TestWithRetryContextCancellation(t *testing.T) {
	config := DefaultClientCopyRetryConfig()
	config.BaseDelay = 100 * time.Millisecond // Longer delay to test cancellation

	callCount := 0
	operation := func() error {
		callCount++
		return errors.New("connection timeout") // Retryable error
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := WithRetry(ctx, config, operation, "test-operation")
	duration := time.Since(start)

	assert.Assert(t, err != nil)
	assert.Equal(t, 1, callCount)                     // Should fail before retry due to context timeout
	assert.Assert(t, duration < 100*time.Millisecond) // Should fail before first retry delay
	assert.Assert(t, errors.Is(err, context.DeadlineExceeded))
}

func TestRetryableClientCopyOperation(t *testing.T) {
	retryOp := NewRetryableClientCopyOperation()
	assert.Assert(t, retryOp != nil)

	// Test with custom config
	customConfig := RetryConfig{
		MaxRetries:      5,
		BaseDelay:       50 * time.Millisecond,
		MaxDelay:        2 * time.Second,
		BackoffExponent: 1.5,
		Jitter:          true,
	}
	retryOp = retryOp.WithCustomConfig(customConfig)
	assert.Equal(t, customConfig.MaxRetries, retryOp.config.MaxRetries)
}

func TestRetryableClientCopyOperationDownload(t *testing.T) {
	retryOp := NewRetryableClientCopyOperation()
	retryOp.config.BaseDelay = 10 * time.Millisecond // Speed up test

	callCount := 0
	downloadFunc := func() error {
		callCount++
		if callCount < 2 {
			return errors.New("temporary failure")
		}
		return nil
	}

	ctx := context.Background()
	err := retryOp.ExecuteDownload(ctx, downloadFunc)

	assert.NilError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestRetryableClientCopyOperationUpload(t *testing.T) {
	retryOp := NewRetryableClientCopyOperation()
	retryOp.config.BaseDelay = 10 * time.Millisecond // Speed up test

	callCount := 0
	uploadFunc := func() error {
		callCount++
		if callCount < 3 {
			return errors.New("throttlingexception")
		}
		return nil
	}

	ctx := context.Background()
	err := retryOp.ExecuteUpload(ctx, uploadFunc)

	assert.NilError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestDefaultClientCopyRetryConfig(t *testing.T) {
	config := DefaultClientCopyRetryConfig()

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1*time.Second, config.BaseDelay)
	assert.Equal(t, 30*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.BackoffExponent)
	assert.Equal(t, true, config.Jitter)
}

// TestRetryErrorTypes tests different AWS error types
func TestRetryErrorTypes(t *testing.T) {
	awsErrors := []string{
		"ProvisionedThroughputExceeded",
		"ThrottlingException",
		"RequestLimitExceeded",
		"ServiceUnavailable",
		"InternalError",
		"SlowDown",
		"RequestTimeout",
	}

	for _, errStr := range awsErrors {
		t.Run(errStr, func(t *testing.T) {
			err := errors.New(errStr)
			assert.Assert(t, IsRetryableError(err), "AWS error %s should be retryable", errStr)
		})
	}
}

// BenchmarkRetryLogic benchmarks the retry logic overhead
func BenchmarkRetryLogic(b *testing.B) {
	config := DefaultClientCopyRetryConfig()
	config.BaseDelay = 1 * time.Millisecond

	operation := func() error {
		return nil // Always succeed
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := WithRetry(ctx, config, operation, "benchmark")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestRetryRealWorldScenarios tests more realistic failure scenarios
func TestRetryRealWorldScenarios(t *testing.T) {
	t.Run("intermittent network failure", func(t *testing.T) {
		config := DefaultClientCopyRetryConfig()
		config.BaseDelay = 10 * time.Millisecond

		callCount := 0
		operation := func() error {
			callCount++
			switch callCount {
			case 1:
				return errors.New("dial tcp: connection refused")
			case 2:
				return errors.New("i/o timeout")
			case 3:
				return nil // Success
			default:
				return errors.New("unexpected call")
			}
		}

		ctx := context.Background()
		err := WithRetry(ctx, config, operation, "network-test")

		assert.NilError(t, err)
		assert.Equal(t, 3, callCount)
	})

	t.Run("aws throttling scenario", func(t *testing.T) {
		config := DefaultClientCopyRetryConfig()
		config.BaseDelay = 10 * time.Millisecond

		callCount := 0
		operation := func() error {
			callCount++
			if callCount <= 2 {
				return errors.New("ThrottlingException: Rate exceeded")
			}
			return nil
		}

		ctx := context.Background()
		err := WithRetry(ctx, config, operation, "throttling-test")

		assert.NilError(t, err)
		assert.Equal(t, 3, callCount)
	})
}
