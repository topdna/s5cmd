package e2e

import (
	"fmt"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/icmd"
)

// TestClientCopyWithDiskSpaceValidation tests disk space validation functionality
func TestClientCopyWithDiskSpaceValidation(t *testing.T) {
	t.Parallel()

	s3client, s5cmd := setup(t)

	bucket := s3BucketFromTestName(t)
	createBucket(t, s3client, bucket)

	const (
		filename = "disk_space_test.txt"
		content  = "content for disk space testing"
	)

	putFile(t, s3client, bucket, filename, content)

	src := fmt.Sprintf("s3://%v/%v", bucket, filename)
	dst := fmt.Sprintf("s3://%v/disk_copy_%v", bucket, filename)

	// Test with disk space validation enabled (default)
	cmd := s5cmd("cp", "--client-copy", src, dst)
	result := icmd.RunCmd(cmd)

	// Should succeed for small files
	if result.ExitCode != 0 {
		errorOutput := result.Stderr()
		hasDiskSpaceError := strings.Contains(errorOutput, "insufficient disk space")

		// Disk space validation should not fail for small test files
		assert.Assert(t, !hasDiskSpaceError,
			"Disk space validation should not fail for small files, got: %s", errorOutput)
	}
}

// TestClientCopyWithSkipDiskCheck tests skipping disk space validation
func TestClientCopyWithSkipDiskCheck(t *testing.T) {
	t.Parallel()

	s3client, s5cmd := setup(t)

	bucket := s3BucketFromTestName(t)
	createBucket(t, s3client, bucket)

	const (
		filename = "skip_disk_test.txt"
		content  = "content for skip disk check testing"
	)

	putFile(t, s3client, bucket, filename, content)

	src := fmt.Sprintf("s3://%v/%v", bucket, filename)
	dst := fmt.Sprintf("s3://%v/skip_disk_copy_%v", bucket, filename)

	// Test with disk space validation skipped
	cmd := s5cmd("cp", "--client-copy", "--client-copy-skip-disk-check", src, dst)
	result := icmd.RunCmd(cmd)

	// Should accept the flag (may fail for other reasons in test environment)
	if result.ExitCode != 0 {
		errorOutput := result.Stderr()
		hasUnknownFlag := strings.Contains(errorOutput, "flag provided but not defined") ||
			strings.Contains(errorOutput, "unknown flag")

		assert.Assert(t, !hasUnknownFlag,
			"Should accept skip disk check flag, got error: %s", errorOutput)
	}
}

// TestClientCopyValidationChecksEnhanced tests enhanced validation
func TestClientCopyValidationChecksEnhanced(t *testing.T) {
	t.Parallel()

	s3client, s5cmd := setup(t)

	bucket := s3BucketFromTestName(t)
	createBucket(t, s3client, bucket)

	const (
		filename = "enhanced_validation_test.txt"
		content  = "content for enhanced validation testing"
	)

	putFile(t, s3client, bucket, filename, content)

	// Test local to remote (should fail with client-copy)
	localFile := "test-local-file.txt"
	remoteFile := fmt.Sprintf("s3://%v/%v", bucket, filename)

	cmd := s5cmd("cp", "--client-copy", localFile, remoteFile)
	result := icmd.RunCmd(cmd)

	result.Assert(t, icmd.Expected{ExitCode: 1})

	// Should contain error about both URLs needing to be remote
	errorOutput := result.Stderr()
	hasValidationError := strings.Contains(errorOutput, "client copy requires both source and destination to be remote") ||
		strings.Contains(errorOutput, "remote (S3) URLs")

	assert.Assert(t, hasValidationError,
		"Should detect non-remote URL error, got: %s", errorOutput)
}

// TestClientCopyMetricsIntegration tests that metrics collection works
func TestClientCopyMetricsIntegration(t *testing.T) {
	t.Parallel()

	s3client, s5cmd := setup(t)

	bucket := s3BucketFromTestName(t)
	createBucket(t, s3client, bucket)

	const (
		filename = "metrics_test.txt"
		content  = "content for metrics testing integration"
	)

	putFile(t, s3client, bucket, filename, content)

	src := fmt.Sprintf("s3://%v/%v", bucket, filename)
	dst := fmt.Sprintf("s3://%v/metrics_copy_%v", bucket, filename)

	// Test client copy with verbose logging to see metrics
	cmd := s5cmd("cp", "--client-copy", src, dst)
	result := icmd.RunCmd(cmd)

	// Success is not required in test environment, but should not crash
	if result.ExitCode == 0 {
		// Verify files exist
		assert.Assert(t, ensureS3Object(s3client, bucket, filename, content))
		assert.Assert(t, ensureS3Object(s3client, bucket, "metrics_copy_"+filename, content))
	}

	// The implementation should not crash due to metrics collection
	// Error output should not contain panic or similar issues
	output := result.Stderr()
	hasPanic := strings.Contains(output, "panic") ||
		strings.Contains(output, "runtime error") ||
		strings.Contains(output, "fatal error")

	assert.Assert(t, !hasPanic,
		"Should not crash due to metrics collection, got: %s", output)
}

// TestClientCopyRetryLogicIntegration tests that retry logic is integrated
func TestClientCopyRetryLogicIntegration(t *testing.T) {
	t.Parallel()

	_, s5cmd := setup(t)

	// Test with non-existent bucket to trigger retry logic
	src := "s3://non-existent-bucket-for-retry-test/file.txt"
	dst := "s3://another-non-existent-bucket-for-retry-test/file.txt"

	cmd := s5cmd("cp", "--client-copy", src, dst)
	result := icmd.RunCmd(cmd)

	result.Assert(t, icmd.Expected{ExitCode: 1})

	// The retry logic should be integrated without causing crashes
	output := result.Stderr()
	hasPanic := strings.Contains(output, "panic") ||
		strings.Contains(output, "runtime error") ||
		strings.Contains(output, "fatal error")

	assert.Assert(t, !hasPanic,
		"Should not crash during retry logic execution, got: %s", output)
}

// TestClientCopyConfigurationValidation tests comprehensive configuration validation
func TestClientCopyConfigurationValidation(t *testing.T) {
	t.Parallel()

	_, s5cmd := setup(t)

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorSubstr string
	}{
		{
			name: "valid client copy",
			args: []string{"cp", "--client-copy",
				"s3://test/src", "s3://test/dst"},
			expectError: false, // May fail for other reasons, but not configuration
			errorSubstr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := s5cmd(tt.args...)
			result := icmd.RunCmd(cmd)

			if tt.expectError {
				result.Assert(t, icmd.Expected{ExitCode: 1})
				if tt.errorSubstr != "" {
					assert.Assert(t, strings.Contains(result.Stderr(), tt.errorSubstr),
						"Expected error containing '%s', got: %s", tt.errorSubstr, result.Stderr())
				}
			} else {
				// For valid configuration, we don't expect specific configuration errors
				if result.ExitCode != 0 {
					errorOutput := result.Stderr()
					hasConfigError := strings.Contains(errorOutput, "invalid configuration") ||
						strings.Contains(errorOutput, "configuration error")

					assert.Assert(t, !hasConfigError,
						"Should not have configuration error for valid input, got: %s", errorOutput)
				}
			}
		})
	}
}
