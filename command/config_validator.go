package command

import (
	"fmt"
)

// ClientCopyConfigValidator validates complete client copy configuration
type ClientCopyConfigValidator struct {
}

// NewClientCopyConfigValidator creates a comprehensive client copy config validator
func NewClientCopyConfigValidator() *ClientCopyConfigValidator {
	return &ClientCopyConfigValidator{}
}

// ValidateClientCopyConfig validates all client copy configuration parameters
func (v *ClientCopyConfigValidator) ValidateClientCopyConfig(config ClientCopyConfig) []error {
	var errors []error

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
	SourceProfile       string
	DestinationProfile  string
	SourceEndpoint      string
	DestinationEndpoint string
	SkipDiskCheck       bool
}

// GetConfigSummary returns a summary of the configuration for validation
func (c ClientCopyConfig) GetConfigSummary() string {
	return fmt.Sprintf("Source: %s, Destination: %s, SkipDiskCheck: %t",
		c.SourceURL, c.DestinationURL, c.SkipDiskCheck)
}
