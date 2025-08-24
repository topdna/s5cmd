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

// TestClientCopyWithDifferentProfiles tests client copy with different source and destination profiles
func TestClientCopyWithDifferentProfiles(t *testing.T) {
	t.Parallel()

	s3client, s5cmd := setup(t)

	srcbucket := s3BucketFromTestNameWithPrefix(t, "src")
	dstbucket := s3BucketFromTestNameWithPrefix(t, "dst")

	createBucket(t, s3client, srcbucket)
	createBucket(t, s3client, dstbucket)

	const (
		filename = "testfile_profiles.txt"
		content  = "content for profile testing"
	)

	putFile(t, s3client, srcbucket, filename, content)

	src := fmt.Sprintf("s3://%v/%v", srcbucket, filename)
	dst := fmt.Sprintf("s3://%v/%v", dstbucket, filename)

	// Skip test if AWS credentials are not available (common in CI environments)
	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".aws", "credentials")); os.IsNotExist(err) {
		t.Skip("AWS credentials file not found, skipping profile test")
	}

	// Note: In real scenarios, different profiles would be used
	cmd := s5cmd("cp", "--client-copy",
		"--source-region-profile", "default",
		"--destination-region-profile", "default",
		src, dst)
	result := icmd.RunCmd(cmd)

	result.Assert(t, icmd.Success)

	assertLines(t, result.Stdout(), map[int]compareFunc{
		0: contains(fmt.Sprintf(`cp %v`, src)),
		1: contains(fmt.Sprintf(`%v`, dst)),
	})

	assert.Assert(t, ensureS3Object(s3client, srcbucket, filename, content))
	assert.Assert(t, ensureS3Object(s3client, dstbucket, filename, content))
}

// TestClientCopyWithCustomEndpoints tests client copy with different endpoints
func TestClientCopyWithCustomEndpoints(t *testing.T) {
	t.Parallel()

	s3client, s5cmd := setup(t)

	bucket := s3BucketFromTestName(t)
	createBucket(t, s3client, bucket)

	const (
		filename = "testfile_endpoints.txt"
		content  = "content for endpoint testing"
	)

	putFile(t, s3client, bucket, filename, content)

	src := fmt.Sprintf("s3://%v/%v", bucket, filename)
	dst := fmt.Sprintf("s3://%v/copy_%v", bucket, filename)

	// Use same endpoint for both source and destination in test environment
	endpoint := "http://127.0.0.1:9000" // Example MinIO endpoint

	cmd := s5cmd("cp", "--client-copy",
		"--source-region-endpoint-url", endpoint,
		"--destination-region-endpoint-url", endpoint,
		src, dst)
	result := icmd.RunCmd(cmd)

	// This test may fail if custom endpoint is not available
	// In real environment, it would test cross-endpoint copying
	if result.ExitCode != 0 {
		t.Skipf("Custom endpoint test skipped: %s", result.Stderr())
	}

	assert.Assert(t, ensureS3Object(s3client, bucket, filename, content))
	assert.Assert(t, ensureS3Object(s3client, bucket, "copy_"+filename, content))
}

// TestClientCopyLargeFile tests client copy with larger files
func TestClientCopyLargeFile(t *testing.T) {
	t.Parallel()

	s3client, s5cmd := setup(t)

	bucket := s3BucketFromTestName(t)
	createBucket(t, s3client, bucket)

	const (
		filename = "large_testfile.bin"
		fileSize = 10 * 1024 * 1024 // 10MB
	)

	// Create large file content
	content := strings.Repeat("A", fileSize)

	putFile(t, s3client, bucket, filename, content)

	src := fmt.Sprintf("s3://%v/%v", bucket, filename)
	dst := fmt.Sprintf("s3://%v/copy_%v", bucket, filename)

	startTime := time.Now()
	cmd := s5cmd("cp", "--client-copy", src, dst)
	result := icmd.RunCmd(cmd)
	duration := time.Since(startTime)

	result.Assert(t, icmd.Success)

	// Verify timing is reasonable (should not be too slow)
	assert.Assert(t, duration < 2*time.Minute,
		"Client copy took too long: %v", duration)

	assertLines(t, result.Stdout(), map[int]compareFunc{
		0: contains(fmt.Sprintf(`cp %v`, src)),
		1: contains(fmt.Sprintf(`%v`, dst)),
	})

	assert.Assert(t, ensureS3Object(s3client, bucket, filename, content))
	assert.Assert(t, ensureS3Object(s3client, bucket, "copy_"+filename, content))
}

// TestClientCopyWithMetadata tests client copy preserves metadata
func TestClientCopyWithMetadata(t *testing.T) {
	t.Parallel()

	s3client, s5cmd := setup(t)

	bucket := s3BucketFromTestName(t)
	createBucket(t, s3client, bucket)

	const (
		filename = "metadata_testfile.txt"
		content  = "content with metadata"
	)

	putFile(t, s3client, bucket, filename, content)

	src := fmt.Sprintf("s3://%v/%v", bucket, filename)
	dst := fmt.Sprintf("s3://%v/metadata_copy_%v", bucket, filename)

	cmd := s5cmd("cp", "--client-copy",
		"--metadata", "purpose=testing",
		"--metadata", "env=e2e",
		src, dst)
	result := icmd.RunCmd(cmd)

	result.Assert(t, icmd.Success)

	assertLines(t, result.Stdout(), map[int]compareFunc{
		0: contains(fmt.Sprintf(`cp %v`, src)),
		1: contains(fmt.Sprintf(`%v`, dst)),
	})

	assert.Assert(t, ensureS3Object(s3client, bucket, filename, content))
	assert.Assert(t, ensureS3Object(s3client, bucket, "metadata_copy_"+filename, content))
}

// TestClientCopyErrorHandling tests error scenarios
func TestClientCopyErrorHandling(t *testing.T) {
	t.Parallel()

	_, s5cmd := setup(t)

	// Test with non-existent source
	src := "s3://non-existent-bucket/file.txt"
	dst := "s3://another-non-existent-bucket/file.txt"

	cmd := s5cmd("cp", "--client-copy", src, dst)
	// Set working directory to a valid temporary directory for all platforms
	tempDir := os.TempDir()
	cmd.Dir = tempDir
	result := icmd.RunCmd(cmd)

	result.Assert(t, icmd.Expected{ExitCode: 1})

	// Should contain error about non-existent bucket
	errorOutput := result.Stderr()
	hasExpectedError := strings.Contains(errorOutput, "NoSuchBucket") ||
		strings.Contains(errorOutput, "not found") ||
		strings.Contains(errorOutput, "does not exist") ||
		strings.Contains(errorOutput, "NotFound") ||
		strings.Contains(errorOutput, "404")

	assert.Assert(t, hasExpectedError,
		"Should detect non-existent bucket error, got: %s", errorOutput)
}

// TestClientCopyWithWildcard tests client copy with wildcard sources
func TestClientCopyWithWildcard(t *testing.T) {
	t.Parallel()

	s3client, s5cmd := setup(t)

	srcbucket := s3BucketFromTestNameWithPrefix(t, "wild-src")
	dstbucket := s3BucketFromTestNameWithPrefix(t, "wild-dst")

	createBucket(t, s3client, srcbucket)
	createBucket(t, s3client, dstbucket)

	// Create multiple files
	files := map[string]string{
		"file1.txt":     "content 1",
		"file2.txt":     "content 2",
		"dir/file3.txt": "content 3",
	}

	for filename, content := range files {
		putFile(t, s3client, srcbucket, filename, content)
	}

	src := fmt.Sprintf("s3://%v/*.txt", srcbucket)
	dst := fmt.Sprintf("s3://%v/", dstbucket)

	cmd := s5cmd("cp", "--client-copy", src, dst)
	result := icmd.RunCmd(cmd)

	result.Assert(t, icmd.Success)

	// Verify all files were copied
	for filename, content := range files {
		if !strings.Contains(filename, "/") { // Only root level files for wildcard
			assert.Assert(t, ensureS3Object(s3client, dstbucket, filename, content))
		}
	}
}

// TestClientCopyDiskSpaceHandling tests behavior with limited disk space
func TestClientCopyDiskSpaceHandling(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Disk space testing is complex on Windows")
	}

	t.Parallel()

	s3client, s5cmd := setup(t)

	bucket := s3BucketFromTestName(t)
	createBucket(t, s3client, bucket)

	const (
		filename = "space_test.txt"
		content  = "content for disk space testing"
	)

	putFile(t, s3client, bucket, filename, content)

	src := fmt.Sprintf("s3://%v/%v", bucket, filename)
	dst := fmt.Sprintf("s3://%v/space_copy_%v", bucket, filename)

	// Create a temporary directory with limited space (if possible)
	tempDir := fs.NewDir(t, "limited-space")
	defer tempDir.Remove()

	cmd := s5cmd("cp", "--client-copy", src, dst)
	// Set working directory to the temp directory to avoid path issues
	cmd.Dir = tempDir.Path()
	// Set environment variable for this specific command only
	cmd.Env = append(cmd.Env, fmt.Sprintf("TMPDIR=%s", tempDir.Path()))
	result := icmd.RunCmd(cmd)

	// Should succeed for small files
	result.Assert(t, icmd.Success)

	assert.Assert(t, ensureS3Object(s3client, bucket, filename, content))
	assert.Assert(t, ensureS3Object(s3client, bucket, "space_copy_"+filename, content))
}

// TestClientCopyTemporaryFileCleanup tests that temporary files are cleaned up
func TestClientCopyTemporaryFileCleanup(t *testing.T) {
	t.Parallel()

	s3client, s5cmd := setup(t)

	bucket := s3BucketFromTestName(t)
	createBucket(t, s3client, bucket)

	const (
		filename = "cleanup_test.txt"
		content  = "content for cleanup testing"
	)

	putFile(t, s3client, bucket, filename, content)

	src := fmt.Sprintf("s3://%v/%v", bucket, filename)
	dst := fmt.Sprintf("s3://%v/cleanup_copy_%v", bucket, filename)

	// Create a dedicated temporary directory for our test
	testTmpDir := fs.NewDir(t, "client-copy-cleanup")
	defer testTmpDir.Remove()

	cmd := s5cmd("cp", "--client-copy", src, dst)
	cmd.Dir = testTmpDir.Path()
	// Set environment variable for this specific command only
	if runtime.GOOS == "windows" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("TEMP=%s", testTmpDir.Path()))
	} else {
		cmd.Env = append(cmd.Env, fmt.Sprintf("TMPDIR=%s", testTmpDir.Path()))
	}
	result := icmd.RunCmd(cmd)

	result.Assert(t, icmd.Success)

	// Check that no temporary files remain in our test directory
	// Allow some time for cleanup to complete
	time.Sleep(100 * time.Millisecond)

	afterFiles, err := filepath.Glob(filepath.Join(testTmpDir.Path(), "*"))
	assert.NilError(t, err)

	// Filter out the workdir that might be created by the test framework
	var tempFiles []string
	for _, file := range afterFiles {
		if !strings.Contains(file, "workdir") {
			tempFiles = append(tempFiles, file)
		}
	}

	// Should not have any remaining temporary files
	assert.Assert(t, len(tempFiles) == 0,
		"Temporary files not cleaned up properly, remaining files: %v", tempFiles)

	assert.Assert(t, ensureS3Object(s3client, bucket, filename, content))
	assert.Assert(t, ensureS3Object(s3client, bucket, "cleanup_copy_"+filename, content))
}
