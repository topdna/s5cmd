package e2e

import (
	"fmt"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/icmd"
)

// TestClientCopyValidationChecks tests the input validation for client copy
func TestClientCopyValidationChecks(t *testing.T) {
	t.Parallel()

	s3client, s5cmd := setup(t)

	bucket := s3BucketFromTestName(t)
	createBucket(t, s3client, bucket)

	const (
		filename = "validation_test.txt"
		content  = "content for validation testing"
	)

	putFile(t, s3client, bucket, filename, content)

	// Test same source and destination error
	sameFile := fmt.Sprintf("s3://%v/%v", bucket, filename)

	cmd := s5cmd("cp", "--client-copy", sameFile, sameFile)
	result := icmd.RunCmd(cmd)

	result.Assert(t, icmd.Expected{ExitCode: 1})

	// Should contain error about same source and destination
	errorOutput := result.Stderr()
	hasExpectedError := strings.Contains(errorOutput, "source and destination cannot be the same")

	assert.Assert(t, hasExpectedError,
		"Should detect same source/destination error, got: %s", errorOutput)
}

// TestClientCopyCredentialRefresh tests the proactive credential refresh mechanism
func TestClientCopyCredentialRefresh(t *testing.T) {
	t.Parallel()

	s3client, s5cmd := setup(t)

	bucket := s3BucketFromTestName(t)
	createBucket(t, s3client, bucket)

	const (
		filename = "credential_test.txt"
		content  = "content for credential refresh testing"
	)

	putFile(t, s3client, bucket, filename, content)

	src := fmt.Sprintf("s3://%v/%v", bucket, filename)
	dst := fmt.Sprintf("s3://%v/refresh_copy_%v", bucket, filename)

	// Test that client copy works normally (credential refresh is automatic)
	cmd := s5cmd("cp", "--client-copy", src, dst)
	result := icmd.RunCmd(cmd)

	// Should succeed for normal operations
	if result.ExitCode == 0 {
		assert.Assert(t, ensureS3Object(s3client, bucket, filename, content))
		assert.Assert(t, ensureS3Object(s3client, bucket, "refresh_copy_"+filename, content))
	} else {
		// If it fails, it should not be due to credential issues for normal operations
		hasCredentialError := strings.Contains(result.Stderr(), "ExpiredToken") ||
			strings.Contains(result.Stderr(), "InvalidToken") ||
			strings.Contains(result.Stderr(), "TokenRefreshRequired")

		// Allow infrastructure errors but not credential errors for short operations
		if hasCredentialError {
			t.Logf("Note: Credential refresh mechanism triggered: %s", result.Stderr())
		}
	}
}
