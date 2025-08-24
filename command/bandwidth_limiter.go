package command

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/time/rate"
)

// BandwidthLimiter provides rate limiting for I/O operations
type BandwidthLimiter struct {
	limiter *rate.Limiter
	enabled bool
}

// NewBandwidthLimiter creates a new bandwidth limiter from a limit string
// Supports formats like "100MB/s", "1GB/s", "500KB/s", "10Mbps", "1Gbps"
func NewBandwidthLimiter(limitStr string) (*BandwidthLimiter, error) {
	if limitStr == "" {
		return &BandwidthLimiter{enabled: false}, nil
	}

	bytesPerSecond, err := parseBandwidthLimit(limitStr)
	if err != nil {
		return nil, fmt.Errorf("invalid bandwidth limit format: %w", err)
	}

	// Use a burst size of 10% of the rate or minimum 64KB
	burstSize := int(bytesPerSecond / 10)
	if burstSize < 64*1024 {
		burstSize = 64 * 1024
	}

	return &BandwidthLimiter{
		limiter: rate.NewLimiter(rate.Limit(bytesPerSecond), burstSize),
		enabled: true,
	}, nil
}

// Wait blocks until the limiter allows n bytes to be processed
func (bl *BandwidthLimiter) Wait(ctx context.Context, n int) error {
	if !bl.enabled {
		return nil
	}
	return bl.limiter.WaitN(ctx, n)
}

// IsEnabled returns whether the bandwidth limiter is active
func (bl *BandwidthLimiter) IsEnabled() bool {
	return bl.enabled
}

// parseBandwidthLimit parses bandwidth limit strings into bytes per second
func parseBandwidthLimit(limitStr string) (float64, error) {
	limitStr = strings.TrimSpace(strings.ToUpper(limitStr))

	// Handle different formats
	var multiplier float64 = 1
	var numStr string

	if strings.HasSuffix(limitStr, "BPS") {
		// Handle "Mbps", "Gbps", etc. (bits per second)
		if strings.HasSuffix(limitStr, "GBPS") {
			multiplier = 1024 * 1024 * 1024 / 8 // Convert Gbps to bytes/sec
			numStr = strings.TrimSuffix(limitStr, "GBPS")
		} else if strings.HasSuffix(limitStr, "MBPS") {
			multiplier = 1024 * 1024 / 8 // Convert Mbps to bytes/sec
			numStr = strings.TrimSuffix(limitStr, "MBPS")
		} else if strings.HasSuffix(limitStr, "KBPS") {
			multiplier = 1024 / 8 // Convert Kbps to bytes/sec
			numStr = strings.TrimSuffix(limitStr, "KBPS")
		} else {
			return 0, fmt.Errorf("unsupported bandwidth format: %s", limitStr)
		}
	} else if strings.HasSuffix(limitStr, "B/S") {
		// Handle "MB/s", "GB/s", etc. (bytes per second)
		if strings.HasSuffix(limitStr, "GB/S") {
			multiplier = 1024 * 1024 * 1024
			numStr = strings.TrimSuffix(limitStr, "GB/S")
		} else if strings.HasSuffix(limitStr, "MB/S") {
			multiplier = 1024 * 1024
			numStr = strings.TrimSuffix(limitStr, "MB/S")
		} else if strings.HasSuffix(limitStr, "KB/S") {
			multiplier = 1024
			numStr = strings.TrimSuffix(limitStr, "KB/S")
		} else {
			return 0, fmt.Errorf("unsupported bandwidth format: %s", limitStr)
		}
	} else {
		return 0, fmt.Errorf("bandwidth limit must end with /s or bps (e.g., '100MB/s', '10Mbps')")
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in bandwidth limit: %w", err)
	}

	if num <= 0 {
		return 0, fmt.Errorf("bandwidth limit must be positive")
	}

	return num * multiplier, nil
}

// LimitedReader wraps an io.Reader with bandwidth limiting
type LimitedReader struct {
	reader  io.Reader
	limiter *BandwidthLimiter
	ctx     context.Context
}

// NewLimitedReader creates a new bandwidth-limited reader
func NewLimitedReader(reader io.Reader, limiter *BandwidthLimiter, ctx context.Context) *LimitedReader {
	return &LimitedReader{
		reader:  reader,
		limiter: limiter,
		ctx:     ctx,
	}
}

// Read implements io.Reader with bandwidth limiting
func (lr *LimitedReader) Read(p []byte) (int, error) {
	n, err := lr.reader.Read(p)
	if n > 0 && lr.limiter.enabled {
		// Wait for rate limiter before returning
		if waitErr := lr.limiter.Wait(lr.ctx, n); waitErr != nil {
			return n, waitErr
		}
	}
	return n, err
}

// LimitedWriter wraps an io.Writer with bandwidth limiting
type LimitedWriter struct {
	writer  io.Writer
	limiter *BandwidthLimiter
	ctx     context.Context
}

// NewLimitedWriter creates a new bandwidth-limited writer
func NewLimitedWriter(writer io.Writer, limiter *BandwidthLimiter, ctx context.Context) *LimitedWriter {
	return &LimitedWriter{
		writer:  writer,
		limiter: limiter,
		ctx:     ctx,
	}
}

// Write implements io.Writer with bandwidth limiting
func (lw *LimitedWriter) Write(p []byte) (int, error) {
	if lw.limiter.enabled {
		// Wait for rate limiter before writing
		if err := lw.limiter.Wait(lw.ctx, len(p)); err != nil {
			return 0, err
		}
	}
	return lw.writer.Write(p)
}
