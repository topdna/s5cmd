package command

import (
	"fmt"
	"time"

	"github.com/peak/s5cmd/v2/log"
)

// ClientCopyMetrics collects metrics for client copy operations
type ClientCopyMetrics struct {
	StartTime         time.Time
	DownloadStartTime time.Time
	DownloadEndTime   time.Time
	UploadStartTime   time.Time
	UploadEndTime     time.Time
	TotalBytes        int64
	SourceURL         string
	DestinationURL    string
	BandwidthLimit    string
	DiskCheckSkipped  bool
	TempDir           string

	// Enhanced metrics
	RetryAttempts      int
	DiskSpaceUsed      int64
	DiskSpaceAvailable int64
	NetworkLatency     time.Duration
	ThroughputSamples  []ThroughputSample
	ErrorCount         int
	LastError          string
}

// ThroughputSample represents a throughput measurement at a point in time
type ThroughputSample struct {
	Timestamp  time.Time
	BytesTotal int64
	Phase      string // "download" or "upload"
}

// NewClientCopyMetrics creates a new metrics collection instance
func NewClientCopyMetrics(sourceURL, destinationURL, bandwidthLimit string, diskCheckSkipped bool, tempDir string) *ClientCopyMetrics {
	return &ClientCopyMetrics{
		StartTime:         time.Now(),
		SourceURL:         sourceURL,
		DestinationURL:    destinationURL,
		BandwidthLimit:    bandwidthLimit,
		DiskCheckSkipped:  diskCheckSkipped,
		TempDir:           tempDir,
		ThroughputSamples: make([]ThroughputSample, 0),
		RetryAttempts:     0,
		ErrorCount:        0,
	}
}

// StartDownload records the start of the download phase
func (m *ClientCopyMetrics) StartDownload() {
	m.DownloadStartTime = time.Now()
}

// EndDownload records the end of the download phase
func (m *ClientCopyMetrics) EndDownload() {
	m.DownloadEndTime = time.Now()
}

// StartUpload records the start of the upload phase
func (m *ClientCopyMetrics) StartUpload() {
	m.UploadStartTime = time.Now()
}

// EndUpload records the end of the upload phase
func (m *ClientCopyMetrics) EndUpload() {
	m.UploadEndTime = time.Now()
}

// SetTotalBytes sets the total bytes transferred
func (m *ClientCopyMetrics) SetTotalBytes(bytes int64) {
	m.TotalBytes = bytes
}

// AddRetryAttempt increments the retry counter
func (m *ClientCopyMetrics) AddRetryAttempt() {
	m.RetryAttempts++
}

// SetDiskSpaceInfo sets disk space usage information
func (m *ClientCopyMetrics) SetDiskSpaceInfo(used, available int64) {
	m.DiskSpaceUsed = used
	m.DiskSpaceAvailable = available
}

// SetNetworkLatency sets the measured network latency
func (m *ClientCopyMetrics) SetNetworkLatency(latency time.Duration) {
	m.NetworkLatency = latency
}

// AddThroughputSample adds a throughput measurement sample
func (m *ClientCopyMetrics) AddThroughputSample(bytesTotal int64, phase string) {
	m.ThroughputSamples = append(m.ThroughputSamples, ThroughputSample{
		Timestamp:  time.Now(),
		BytesTotal: bytesTotal,
		Phase:      phase,
	})
}

// RecordError records an error occurrence
func (m *ClientCopyMetrics) RecordError(err error) {
	m.ErrorCount++
	if err != nil {
		m.LastError = err.Error()
	}
}

// GetDownloadDuration returns the duration of the download phase
func (m *ClientCopyMetrics) GetDownloadDuration() time.Duration {
	if m.DownloadStartTime.IsZero() || m.DownloadEndTime.IsZero() {
		return 0
	}
	return m.DownloadEndTime.Sub(m.DownloadStartTime)
}

// GetUploadDuration returns the duration of the upload phase
func (m *ClientCopyMetrics) GetUploadDuration() time.Duration {
	if m.UploadStartTime.IsZero() || m.UploadEndTime.IsZero() {
		return 0
	}
	return m.UploadEndTime.Sub(m.UploadStartTime)
}

// GetTotalDuration returns the total duration of the client copy operation
func (m *ClientCopyMetrics) GetTotalDuration() time.Duration {
	if m.UploadEndTime.IsZero() {
		return time.Since(m.StartTime)
	}
	return m.UploadEndTime.Sub(m.StartTime)
}

// GetAverageSpeed returns the average transfer speed in bytes per second
func (m *ClientCopyMetrics) GetAverageSpeed() float64 {
	totalDuration := m.GetTotalDuration()
	if totalDuration == 0 || m.TotalBytes == 0 {
		return 0
	}
	return float64(m.TotalBytes) / totalDuration.Seconds()
}

// GetDownloadSpeed returns the download speed in bytes per second
func (m *ClientCopyMetrics) GetDownloadSpeed() float64 {
	downloadDuration := m.GetDownloadDuration()
	if downloadDuration == 0 || m.TotalBytes == 0 {
		return 0
	}
	return float64(m.TotalBytes) / downloadDuration.Seconds()
}

// GetUploadSpeed returns the upload speed in bytes per second
func (m *ClientCopyMetrics) GetUploadSpeed() float64 {
	uploadDuration := m.GetUploadDuration()
	if uploadDuration == 0 || m.TotalBytes == 0 {
		return 0
	}
	return float64(m.TotalBytes) / uploadDuration.Seconds()
}

// LogSummary logs a comprehensive summary of the client copy operation
func (m *ClientCopyMetrics) LogSummary() {
	summary := fmt.Sprintf(`Client Copy Operation Summary:
  Source: %s
  Destination: %s
  Total Bytes: %s
  Total Duration: %v
  Download Duration: %v
  Upload Duration: %v
  Average Speed: %.2f MB/s
  Download Speed: %.2f MB/s
  Upload Speed: %.2f MB/s
  Peak Throughput: %.2f MB/s
  Bandwidth Limit: %s
  Disk Check Skipped: %t
  Disk Space Used: %s
  Disk Space Available: %s
  Network Latency: %v
  Retry Attempts: %d
  Error Count: %d
  Last Error: %s
  Temp Directory: %s`,
		m.SourceURL,
		m.DestinationURL,
		formatBytes(m.TotalBytes),
		m.GetTotalDuration(),
		m.GetDownloadDuration(),
		m.GetUploadDuration(),
		m.GetAverageSpeed()/(1024*1024),
		m.GetDownloadSpeed()/(1024*1024),
		m.GetUploadSpeed()/(1024*1024),
		m.GetPeakThroughput()/(1024*1024),
		getBandwidthStatusForMetrics(m.BandwidthLimit),
		m.DiskCheckSkipped,
		formatBytesOrNA(m.DiskSpaceUsed),
		formatBytesOrNA(m.DiskSpaceAvailable),
		m.NetworkLatency,
		m.RetryAttempts,
		m.ErrorCount,
		m.getLastErrorSummary(),
		m.TempDir,
	)

	log.Debug(log.DebugMessage{
		Err: summary,
	})
}

// getBandwidthStatusForMetrics returns bandwidth status for metrics
func getBandwidthStatusForMetrics(limitStr string) string {
	if limitStr == "" {
		return "unlimited"
	}
	return limitStr
}

// formatBytesOrNA formats bytes into human-readable format or "N/A" if zero
func formatBytesOrNA(bytes int64) string {
	if bytes == 0 {
		return "N/A"
	}
	return formatBytes(bytes)
}

// getLastErrorSummary returns a truncated version of the last error
func (m *ClientCopyMetrics) getLastErrorSummary() string {
	if m.LastError == "" {
		return "none"
	}
	if len(m.LastError) > 100 {
		return m.LastError[:97] + "..."
	}
	return m.LastError
}

// GetPeakThroughput calculates the peak throughput from samples
func (m *ClientCopyMetrics) GetPeakThroughput() float64 {
	if len(m.ThroughputSamples) < 2 {
		return 0
	}

	var maxThroughput float64
	for i := 1; i < len(m.ThroughputSamples); i++ {
		prev := m.ThroughputSamples[i-1]
		curr := m.ThroughputSamples[i]

		timeDiff := curr.Timestamp.Sub(prev.Timestamp).Seconds()
		bytesDiff := curr.BytesTotal - prev.BytesTotal

		if timeDiff > 0 && bytesDiff > 0 {
			throughput := float64(bytesDiff) / timeDiff
			if throughput > maxThroughput {
				maxThroughput = throughput
			}
		}
	}

	return maxThroughput
}

// GetEfficiency calculates the efficiency ratio (actual vs theoretical max speed)
func (m *ClientCopyMetrics) GetEfficiency() float64 {
	if m.BandwidthLimit == "" {
		return 0 // No limit to compare against
	}

	limitBytes, err := parseBandwidthLimit(m.BandwidthLimit)
	if err != nil {
		return 0
	}

	actualSpeed := m.GetAverageSpeed()
	if limitBytes > 0 {
		return (actualSpeed / limitBytes) * 100 // Return as percentage
	}

	return 0
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatSpeed formats speed into human-readable format
func formatSpeed(bytesPerSecond float64) string {
	return formatBytes(int64(bytesPerSecond)) + "/s"
}
