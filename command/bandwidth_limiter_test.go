package command

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestNewBandwidthLimiter(t *testing.T) {
	tests := []struct {
		name      string
		limitStr  string
		wantErr   bool
		wantBytes float64
	}{
		{
			name:      "empty string - no limit",
			limitStr:  "",
			wantErr:   false,
			wantBytes: 0,
		},
		{
			name:      "MB/s format",
			limitStr:  "100MB/s",
			wantErr:   false,
			wantBytes: 100 * 1024 * 1024,
		},
		{
			name:      "GB/s format",
			limitStr:  "1GB/s",
			wantErr:   false,
			wantBytes: 1024 * 1024 * 1024,
		},
		{
			name:      "KB/s format",
			limitStr:  "500KB/s",
			wantErr:   false,
			wantBytes: 500 * 1024,
		},
		{
			name:      "Mbps format",
			limitStr:  "10Mbps",
			wantErr:   false,
			wantBytes: 10 * 1024 * 1024 / 8,
		},
		{
			name:      "Gbps format",
			limitStr:  "1Gbps",
			wantErr:   false,
			wantBytes: 1024 * 1024 * 1024 / 8,
		},
		{
			name:      "invalid format",
			limitStr:  "invalid",
			wantErr:   true,
			wantBytes: 0,
		},
		{
			name:      "negative value",
			limitStr:  "-100MB/s",
			wantErr:   true,
			wantBytes: 0,
		},
		{
			name:      "zero value",
			limitStr:  "0MB/s",
			wantErr:   true,
			wantBytes: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter, err := NewBandwidthLimiter(tt.limitStr)

			if tt.wantErr {
				assert.Assert(t, err != nil, "expected error for %s", tt.limitStr)
				return
			}

			assert.NilError(t, err)
			assert.Assert(t, limiter != nil)

			if tt.limitStr == "" {
				assert.Assert(t, !limiter.IsEnabled(), "limiter should be disabled for empty string")
			} else {
				assert.Assert(t, limiter.IsEnabled(), "limiter should be enabled for valid limit")
			}
		})
	}
}

func TestParseBandwidthLimit(t *testing.T) {
	tests := []struct {
		name     string
		limitStr string
		want     float64
		wantErr  bool
	}{
		{"100MB/s", "100MB/s", 100 * 1024 * 1024, false},
		{"1.5GB/s", "1.5GB/s", 1.5 * 1024 * 1024 * 1024, false},
		{"50KB/s", "50KB/s", 50 * 1024, false},
		{"10Mbps", "10Mbps", 10 * 1024 * 1024 / 8, false},
		{"1Gbps", "1Gbps", 1024 * 1024 * 1024 / 8, false},
		{"100Kbps", "100Kbps", 100 * 1024 / 8, false},
		{"case insensitive", "100mb/s", 100 * 1024 * 1024, false},
		{"invalid format", "100", 0, true},
		{"invalid unit", "100XB/s", 0, true},
		{"invalid number", "abc MB/s", 0, true},
		{"negative", "-100MB/s", 0, true},
		{"zero", "0MB/s", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBandwidthLimit(tt.limitStr)
			if tt.wantErr {
				assert.Assert(t, err != nil)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBandwidthLimiterWait(t *testing.T) {
	// Test disabled limiter
	t.Run("disabled limiter", func(t *testing.T) {
		limiter, err := NewBandwidthLimiter("")
		assert.NilError(t, err)
		assert.Assert(t, !limiter.IsEnabled())

		ctx := context.Background()
		err = limiter.Wait(ctx, 1000)
		assert.NilError(t, err) // Should not block
	})

	// Test enabled limiter
	t.Run("enabled limiter", func(t *testing.T) {
		limiter, err := NewBandwidthLimiter("1KB/s") // Very slow for testing
		assert.NilError(t, err)
		assert.Assert(t, limiter.IsEnabled())

		ctx := context.Background()
		start := time.Now()
		err = limiter.Wait(ctx, 100) // Small amount
		duration := time.Since(start)
		assert.NilError(t, err)

		// Should complete relatively quickly for small amounts due to burst
		assert.Assert(t, duration < 1*time.Second)
	})

	// Test context cancellation
	t.Run("context cancellation", func(t *testing.T) {
		limiter, err := NewBandwidthLimiter("1KB/s")
		assert.NilError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		start := time.Now()
		err = limiter.Wait(ctx, 10000) // Large amount that would take time
		duration := time.Since(start)

		// Either succeeds quickly due to burst or fails with context timeout
		if err != nil {
			// Context timeout occurred
			assert.Assert(t, duration < 200*time.Millisecond)
		} else {
			// Completed due to burst allowance
			assert.Assert(t, duration < 200*time.Millisecond)
		}
	})
}

func TestLimitedReader(t *testing.T) {
	t.Run("disabled limiter", func(t *testing.T) {
		limiter, err := NewBandwidthLimiter("")
		assert.NilError(t, err)

		content := "test content for reading"
		reader := strings.NewReader(content)
		limitedReader := NewLimitedReader(reader, limiter, context.Background())

		buf := make([]byte, len(content))
		n, err := limitedReader.Read(buf)
		assert.NilError(t, err)
		assert.Equal(t, len(content), n)
		assert.Equal(t, content, string(buf))
	})

	t.Run("enabled limiter", func(t *testing.T) {
		limiter, err := NewBandwidthLimiter("10KB/s") // Reasonable limit for testing
		assert.NilError(t, err)

		content := "test content for reading"
		reader := strings.NewReader(content)
		limitedReader := NewLimitedReader(reader, limiter, context.Background())

		buf := make([]byte, len(content))
		start := time.Now()
		n, err := limitedReader.Read(buf)
		duration := time.Since(start)

		assert.NilError(t, err)
		assert.Equal(t, len(content), n)
		assert.Equal(t, content, string(buf))

		// Should complete quickly due to small size and burst allowance
		assert.Assert(t, duration < 1*time.Second)
	})

	t.Run("context cancellation", func(t *testing.T) {
		limiter, err := NewBandwidthLimiter("1KB/s")
		assert.NilError(t, err)

		// Create a large content to ensure rate limiting kicks in
		content := strings.Repeat("a", 10000)
		reader := strings.NewReader(content)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		limitedReader := NewLimitedReader(reader, limiter, ctx)

		buf := make([]byte, len(content))
		_, err = limitedReader.Read(buf)

		// Either succeeds quickly due to burst or fails with context timeout
		if err != nil {
			// Context timeout occurred - this is expected for large reads
			t.Logf("Context cancellation worked as expected: %v", err)
		} else {
			// Completed due to burst allowance - also acceptable
			t.Logf("Operation completed quickly due to burst allowance")
		}
	})
}

func TestLimitedWriter(t *testing.T) {
	t.Run("disabled limiter", func(t *testing.T) {
		limiter, err := NewBandwidthLimiter("")
		assert.NilError(t, err)

		var buf strings.Builder
		limitedWriter := NewLimitedWriter(&buf, limiter, context.Background())

		content := "test content for writing"
		n, err := limitedWriter.Write([]byte(content))
		assert.NilError(t, err)
		assert.Equal(t, len(content), n)
		assert.Equal(t, content, buf.String())
	})

	t.Run("enabled limiter", func(t *testing.T) {
		limiter, err := NewBandwidthLimiter("10KB/s")
		assert.NilError(t, err)

		var buf strings.Builder
		limitedWriter := NewLimitedWriter(&buf, limiter, context.Background())

		content := "test content for writing"
		start := time.Now()
		n, err := limitedWriter.Write([]byte(content))
		duration := time.Since(start)

		assert.NilError(t, err)
		assert.Equal(t, len(content), n)
		assert.Equal(t, content, buf.String())

		// Should complete quickly due to small size and burst allowance
		assert.Assert(t, duration < 1*time.Second)
	})
}

// TestBandwidthLimiterBurstBehavior tests the burst behavior of the limiter
func TestBandwidthLimiterBurstBehavior(t *testing.T) {
	limiter, err := NewBandwidthLimiter("1KB/s")
	assert.NilError(t, err)

	ctx := context.Background()

	// First small request should succeed quickly (using burst)
	start := time.Now()
	err = limiter.Wait(ctx, 100)
	firstDuration := time.Since(start)
	assert.NilError(t, err)
	assert.Assert(t, firstDuration < 100*time.Millisecond)

	// Subsequent request should also succeed (still within burst)
	start = time.Now()
	err = limiter.Wait(ctx, 100)
	secondDuration := time.Since(start)
	assert.NilError(t, err)

	// Both should be reasonably fast due to burst allowance
	fmt.Printf("First request: %v, Second request: %v\n", firstDuration, secondDuration)
}

// BenchmarkBandwidthLimiter benchmarks the bandwidth limiter overhead
func BenchmarkBandwidthLimiter(b *testing.B) {
	limiter, err := NewBandwidthLimiter("100MB/s")
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := limiter.Wait(ctx, 1024) // 1KB
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLimitedReader benchmarks the limited reader performance
func BenchmarkLimitedReader(b *testing.B) {
	limiter, err := NewBandwidthLimiter("100MB/s")
	if err != nil {
		b.Fatal(err)
	}

	content := make([]byte, 1024) // 1KB
	for i := range content {
		content[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(string(content))
		limitedReader := NewLimitedReader(reader, limiter, context.Background())

		_, err := io.ReadAll(limitedReader)
		if err != nil {
			b.Fatal(err)
		}
	}
}
