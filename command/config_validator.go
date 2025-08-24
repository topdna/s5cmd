package command

import (
	"fmt"
	"regexp"
	"strings"
)

// BandwidthConfigValidator validates bandwidth configuration parameters
type BandwidthConfigValidator struct {
	validFormats []string
	formatRegex  *regexp.Regexp
}

// NewBandwidthConfigValidator creates a new bandwidth configuration validator
func NewBandwidthConfigValidator() *BandwidthConfigValidator {
	validFormats := []string{
		"KB/s", "MB/s", "GB/s",
		"Kbps", "Mbps", "Gbps",
	}

	// Create regex pattern for valid bandwidth formats
	// Pattern: number (with optional decimal) followed by valid unit
	pattern := `^(\d+(?:\.\d+)?)\s*(KB/S|MB/S|GB/S|KBPS|MBPS|GBPS)$`
	formatRegex := regexp.MustCompile(pattern)

	return &BandwidthConfigValidator{
		validFormats: validFormats,
		formatRegex:  formatRegex,
	}
}

// ValidateBandwidthFormat validates the format of a bandwidth limit string
func (v *BandwidthConfigValidator) ValidateBandwidthFormat(limitStr string) error {
	if limitStr == "" {
		return nil // Empty string is valid (means no limit)
	}

	// Normalize to uppercase for consistent validation
	normalized := strings.TrimSpace(strings.ToUpper(limitStr))

	// Check against regex pattern
	if !v.formatRegex.MatchString(normalized) {
		return fmt.Errorf("invalid bandwidth format '%s'. Valid formats: %s",
			limitStr, strings.Join(v.validFormats, ", "))
	}

	// Additional validation: check for reasonable values
	bytesPerSecond, err := parseBandwidthLimit(limitStr)
	if err != nil {
		return fmt.Errorf("failed to parse bandwidth limit: %w", err)
	}

	// Check for reasonable bounds
	minBandwidth := float64(1024)                     // 1 KB/s minimum
	maxBandwidth := float64(100 * 1024 * 1024 * 1024) // 100 GB/s maximum

	if bytesPerSecond < minBandwidth {
		return fmt.Errorf("bandwidth limit too low: minimum 1KB/s")
	}

	if bytesPerSecond > maxBandwidth {
		return fmt.Errorf("bandwidth limit too high: maximum 100GB/s")
	}

	return nil
}

// GetSupportedFormats returns a list of supported bandwidth formats
func (v *BandwidthConfigValidator) GetSupportedFormats() []string {
	return append([]string{}, v.validFormats...) // Return a copy
}

// GetExampleFormats returns example bandwidth format strings
func (v *BandwidthConfigValidator) GetExampleFormats() []string {
	return []string{
		"100KB/s",
		"50MB/s",
		"1GB/s",
		"10Mbps",
		"100Mbps",
		"1Gbps",
	}
}

// ValidateAndNormalize validates and normalizes a bandwidth format string
func (v *BandwidthConfigValidator) ValidateAndNormalize(limitStr string) (string, error) {
	if err := v.ValidateBandwidthFormat(limitStr); err != nil {
		return "", err
	}

	if limitStr == "" {
		return "", nil
	}

	// Normalize the format for consistent usage
	normalized := strings.TrimSpace(strings.ToUpper(limitStr))

	// Ensure consistent spacing (remove any spaces between number and unit)
	spaceRegex := regexp.MustCompile(`(\d)\s+([A-Z])`)
	normalized = spaceRegex.ReplaceAllString(normalized, "${1}${2}")

	return normalized, nil
}

// SuggestCorrection suggests a corrected format for common mistakes
func (v *BandwidthConfigValidator) SuggestCorrection(invalidFormat string) string {
	lower := strings.ToLower(strings.TrimSpace(invalidFormat))

	// Common mistake corrections
	corrections := map[string]string{
		"mb":   "MB/s",
		"mbps": "Mbps",
		"gb":   "GB/s",
		"gbps": "Gbps",
		"kb":   "KB/s",
		"kbps": "Kbps",
		"mbs":  "MB/s",
		"gbs":  "GB/s",
		"kbs":  "KB/s",
	}

	// Extract numeric part and unit part
	numRegex := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(.*)$`)
	matches := numRegex.FindStringSubmatch(lower)

	if len(matches) == 3 {
		numPart := matches[1]
		unitPart := strings.TrimSpace(matches[2])

		if correction, exists := corrections[unitPart]; exists {
			return numPart + correction
		}
	}

	return "50MB/s" // Default suggestion
}

// ClientCopyConfigValidator validates complete client copy configuration
type ClientCopyConfigValidator struct {
	bandwidthValidator *BandwidthConfigValidator
}

// NewClientCopyConfigValidator creates a comprehensive client copy config validator
func NewClientCopyConfigValidator() *ClientCopyConfigValidator {
	return &ClientCopyConfigValidator{
		bandwidthValidator: NewBandwidthConfigValidator(),
	}
}

// ValidateClientCopyConfig validates all client copy configuration parameters
func (v *ClientCopyConfigValidator) ValidateClientCopyConfig(config ClientCopyConfig) []error {
	var errors []error

	// Validate bandwidth limit
	if err := v.bandwidthValidator.ValidateBandwidthFormat(config.BandwidthLimit); err != nil {
		errors = append(errors, fmt.Errorf("bandwidth validation: %w", err))
	}

	// Validate source and destination URLs
	if config.SourceURL == "" {
		errors = append(errors, fmt.Errorf("source URL cannot be empty"))
	}

	if config.DestinationURL == "" {
		errors = append(errors, fmt.Errorf("destination URL cannot be empty"))
	}

	if config.SourceURL == config.DestinationURL {
		errors = append(errors, fmt.Errorf("source and destination URLs cannot be the same"))
	}

	// Validate profiles and endpoints consistency
	if config.SourceProfile != "" && config.SourceEndpoint == "" {
		// This is usually fine - profile with default AWS endpoints
	} else if config.SourceProfile == "" && config.SourceEndpoint != "" {
		// Warning: custom endpoint without specific profile
		errors = append(errors, fmt.Errorf("warning: source endpoint specified without profile"))
	}

	if config.DestinationProfile != "" && config.DestinationEndpoint == "" {
		// This is usually fine - profile with default AWS endpoints
	} else if config.DestinationProfile == "" && config.DestinationEndpoint != "" {
		// Warning: custom endpoint without specific profile
		errors = append(errors, fmt.Errorf("warning: destination endpoint specified without profile"))
	}

	return errors
}

// ClientCopyConfig represents client copy configuration parameters
type ClientCopyConfig struct {
	SourceURL           string
	DestinationURL      string
	BandwidthLimit      string
	SourceProfile       string
	DestinationProfile  string
	SourceEndpoint      string
	DestinationEndpoint string
	SkipDiskCheck       bool
}

// GetConfigSummary returns a summary of the configuration for validation
func (c ClientCopyConfig) GetConfigSummary() string {
	return fmt.Sprintf("Source: %s, Destination: %s, Bandwidth: %s, SkipDiskCheck: %t",
		c.SourceURL, c.DestinationURL,
		getBandwidthStatusForMetrics(c.BandwidthLimit), c.SkipDiskCheck)
}
