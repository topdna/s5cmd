package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/icmd"
)

// TestSyncErrorHandlingRequestError tests that RequestError properly stops sync operations
func TestSyncErrorHandlingRequestError(t *testing.T) {
	t.Parallel()

	_, s5cmd := setup(t)

	const (
		filename = "test_file.txt"
		content  = "test content"
	)

	// Create local file
	workdir := fs.NewDir(t, "workdir", fs.WithFile(filename, content))
	defer workdir.Remove()

	src := fmt.Sprintf("%v/", workdir.Path())
	src = filepath.ToSlash(src)

	// Use an endpoint that will generate RequestError
	// Set invalid endpoint that should cause RequestError
	fakeEndpoint := "http://invalid.endpoint.test:1234"
	os.Setenv("AWS_ENDPOINT_URL", fakeEndpoint)
	defer os.Unsetenv("AWS_ENDPOINT_URL")

	dst := "s3://fake-bucket/"

	// Should fail due to RequestError and stop sync
	cmd := s5cmd("sync", src, dst)
	result := icmd.RunCmd(cmd)

	// Should exit with error
	result.Assert(t, icmd.Expected{ExitCode: 1})

	// Check that error handling worked correctly
	errorOutput := result.Stderr()

	// Should contain network/request error indicators
	hasRequestError := strings.Contains(errorOutput, "RequestError") ||
		strings.Contains(errorOutput, "connection") ||
		strings.Contains(errorOutput, "dial") ||
		strings.Contains(errorOutput, "timeout") ||
		strings.Contains(errorOutput, "no such host") ||
		strings.Contains(errorOutput, "connect") ||
		strings.Contains(errorOutput, "NotFound") || // Fake endpoint causing NotFound
		strings.Contains(errorOutput, "404")

	if !hasRequestError {
		t.Logf("Expected request error, got: %s", errorOutput)
	}
	assert.Assert(t, hasRequestError, "Should detect request/connection errors")
}

// TestSyncErrorHandlingWithExitOnError tests that --exit-on-error flag works correctly with enhanced error detection
func TestSyncErrorHandlingWithExitOnError(t *testing.T) {
	t.Parallel()

	s3client, s5cmd := setup(t)

	bucket := s3BucketFromTestName(t)
	createBucket(t, s3client, bucket)

	const (
		filename = "test_file.txt"
		content  = "test content"
	)

	// Put object in S3
	putFile(t, s3client, bucket, filename, content)

	// Create local file
	workdir := fs.NewDir(t, "workdir", fs.WithFile(filename, content))
	defer workdir.Remove()

	src := fmt.Sprintf("%v/", workdir.Path())
	src = filepath.ToSlash(src)

	// Use non-existent destination bucket to trigger NoSuchBucket error
	dst := "s3://non-existent-bucket-for-testing/"

	// Should fail and exit immediately due to --exit-on-error
	cmd := s5cmd("sync", "--exit-on-error", src, dst)
	result := icmd.RunCmd(cmd)

	// Should exit with error
	result.Assert(t, icmd.Expected{ExitCode: 1})

	// Should contain NoSuchBucket or similar error
	errorOutput := result.Stderr()
	hasExpectedError := strings.Contains(errorOutput, "NoSuchBucket") ||
		strings.Contains(errorOutput, "404") ||
		strings.Contains(errorOutput, "not found") ||
		strings.Contains(errorOutput, "does not exist")

	if !hasExpectedError {
		t.Logf("Expected NoSuchBucket error, got: %s", errorOutput)
	}
	assert.Assert(t, hasExpectedError, "Should detect NoSuchBucket errors")
}

// TestSyncErrorHandlingGracefulShutdown tests that sync operations shut down gracefully on errors
func TestSyncErrorHandlingGracefulShutdown(t *testing.T) {
	t.Parallel()

	_, s5cmd := setup(t)

	// Create multiple files to ensure we can test cancellation behavior
	folderLayout := []fs.PathOp{
		fs.WithFile("file1.txt", "content 1"),
		fs.WithFile("file2.txt", "content 2"),
		fs.WithFile("file3.txt", "content 3"),
		fs.WithFile("file4.txt", "content 4"),
		fs.WithFile("file5.txt", "content 5"),
	}

	workdir := fs.NewDir(t, "workdir", folderLayout...)
	defer workdir.Remove()

	src := fmt.Sprintf("%v/", workdir.Path())
	src = filepath.ToSlash(src)

	// Use invalid endpoint to trigger errors
	fakeEndpoint := "http://127.0.0.1:1" // Likely to be refused
	os.Setenv("AWS_ENDPOINT_URL", fakeEndpoint)
	defer os.Unsetenv("AWS_ENDPOINT_URL")

	dst := "s3://test-bucket/"

	// Track start time
	startTime := time.Now()

	// Should fail but not hang
	cmd := s5cmd("sync", src, dst)
	result := icmd.RunCmd(cmd)

	// Check that it completed reasonably quickly (within 30 seconds)
	duration := time.Since(startTime)
	assert.Assert(t, duration < 30*time.Second,
		"Sync should complete quickly on errors, took %v", duration)

	// Should exit with error
	result.Assert(t, icmd.Expected{ExitCode: 1})
}

// TestSyncErrorHandlingHashOnlyWithErrors tests error handling specifically with hash-only mode
func TestSyncErrorHandlingHashOnlyWithErrors(t *testing.T) {
	t.Parallel()

	// Skip on Windows as file permission handling is different
	if runtime.GOOS == "windows" {
		t.Skip("Skipping file permission test on Windows")
	}

	s3client, s5cmd := setup(t)

	bucket := s3BucketFromTestName(t)
	createBucket(t, s3client, bucket)

	const (
		filename = "protected_file.txt"
		content  = "protected content"
	)

	// Put object in S3
	putFile(t, s3client, bucket, filename, content)

	// Create local file and make it unreadable to trigger errors
	workdir := fs.NewDir(t, "workdir", fs.WithFile(filename, content))
	defer workdir.Remove()

	filePath := filepath.Join(workdir.Path(), filename)

	// Store original permissions for cleanup
	originalInfo, err := os.Stat(filePath)
	assert.NilError(t, err)
	originalMode := originalInfo.Mode()

	// Remove read permissions
	err = os.Chmod(filePath, 0000)
	assert.NilError(t, err)

	// Ensure permissions are restored
	defer func() {
		if restoreErr := os.Chmod(filePath, originalMode); restoreErr != nil {
			t.Logf("Warning: failed to restore file permissions: %v", restoreErr)
		}
	}()

	src := fmt.Sprintf("%v/", workdir.Path())
	src = filepath.ToSlash(src)
	dst := fmt.Sprintf("s3://%s/", bucket)

	// Track start time to ensure no hanging
	startTime := time.Now()

	// Should fail due to file access error but handle gracefully
	cmd := s5cmd("sync", "--hash-only", src, dst)
	result := icmd.RunCmd(cmd)

	// Should complete within reasonable time
	duration := time.Since(startTime)
	assert.Assert(t, duration < 30*time.Second,
		"Hash-only sync should handle errors gracefully, took %v", duration)

	// Should exit with error
	result.Assert(t, icmd.Expected{ExitCode: 1})

	// Should contain permission error
	errorOutput := result.Stderr()
	hasPermissionError := strings.Contains(errorOutput, "permission denied") ||
		strings.Contains(errorOutput, "access is denied") ||
		strings.Contains(errorOutput, "operation not permitted")

	assert.Assert(t, hasPermissionError, "Should detect permission errors")
}

// TestSyncErrorHandlingMultipleWorkers tests error handling with multiple workers (hash-only mode)
func TestSyncErrorHandlingMultipleWorkers(t *testing.T) {
	t.Parallel()

	s3client, s5cmd := setup(t)

	bucket := s3BucketFromTestName(t)
	createBucket(t, s3client, bucket)

	// Create multiple files to test multiple workers
	folderLayout := []fs.PathOp{
		fs.WithFile("file1.txt", "content 1"),
		fs.WithFile("file2.txt", "content 2"),
		fs.WithFile("file3.txt", "content 3"),
		fs.WithFile("file4.txt", "content 4"),
	}

	workdir := fs.NewDir(t, "workdir", folderLayout...)
	defer workdir.Remove()

	// Put some files in S3 with different content to trigger hash checks
	putFile(t, s3client, bucket, "file1.txt", "different content 1")
	putFile(t, s3client, bucket, "file2.txt", "different content 2")

	src := fmt.Sprintf("%v/", workdir.Path())
	src = filepath.ToSlash(src)
	dst := fmt.Sprintf("s3://%s/", bucket)

	// Track start time
	startTime := time.Now()

	// Test that multiple workers can handle operations gracefully
	cmd := s5cmd("--numworkers", "4", "sync", "--hash-only", src, dst)
	result := icmd.RunCmd(cmd)

	// Should complete within reasonable time even with multiple workers
	duration := time.Since(startTime)
	assert.Assert(t, duration < 30*time.Second,
		"Multi-worker hash-only sync should complete gracefully, took %v", duration)

	// Should exit successfully (this test is about ensuring no panics/hangs)
	result.Assert(t, icmd.Success)

	// Should have some copy operations in output
	output := result.Stdout()
	assert.Assert(t, strings.Contains(output, "cp"), "Should have copy operations in output")
}

// TestSyncErrorHandlingWithDeleteAndMaxDelete tests error handling with delete operations
func TestSyncErrorHandlingWithDeleteAndMaxDelete(t *testing.T) {
	t.Parallel()

	_, s5cmd := setup(t)

	// Create empty local directory (source)
	workdir := fs.NewDir(t, "workdir")
	defer workdir.Remove()

	src := fmt.Sprintf("%v/", workdir.Path())
	src = filepath.ToSlash(src)

	// Use invalid endpoint
	fakeEndpoint := "http://invalid.endpoint.for.testing:8080"
	os.Setenv("AWS_ENDPOINT_URL", fakeEndpoint)
	defer os.Unsetenv("AWS_ENDPOINT_URL")

	dst := "s3://test-bucket/"

	// Track start time
	startTime := time.Now()

	// Should fail when trying to list destination objects for delete
	cmd := s5cmd("sync", "--delete", "--max-delete", "10", src, dst)
	result := icmd.RunCmd(cmd)

	// Should complete quickly on connection errors
	duration := time.Since(startTime)
	assert.Assert(t, duration < 30*time.Second,
		"Sync with delete should handle errors gracefully, took %v", duration)

	// Should exit with error
	result.Assert(t, icmd.Expected{ExitCode: 1})
}
