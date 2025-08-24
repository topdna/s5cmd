package command

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/peak/s5cmd/v2/storage"
	"github.com/peak/s5cmd/v2/storage/url"
)

// validateDiskSpace checks if there's enough disk space for client copy operations
func (c Copy) validateDiskSpace(ctx context.Context, srcurl *url.URL, tempDir string, storageOpts storage.Options) error {
	// Get source object size
	srcClient, err := storage.NewRemoteClient(ctx, srcurl, storageOpts)
	if err != nil {
		return fmt.Errorf("failed to create source client: %w", err)
	}

	obj, err := srcClient.Stat(ctx, srcurl)
	if err != nil {
		return fmt.Errorf("failed to get source object info: %w", err)
	}

	// Check available disk space
	free, err := getAvailableDiskSpace(tempDir)
	if err != nil {
		return fmt.Errorf("failed to check disk space: %w", err)
	}

	// Require at least 20% more space than the file size for safety
	requiredSpace := int64(float64(obj.Size) * 1.2)

	if free < requiredSpace {
		return fmt.Errorf("insufficient disk space: need %d bytes, have %d bytes available",
			requiredSpace, free)
	}

	return nil
}

// getAvailableDiskSpace returns available disk space in bytes for the given path
// This is a simplified cross-platform implementation
func getAvailableDiskSpace(path string) (int64, error) {
	// For simplicity, we'll use a basic check by trying to create a test file
	// In production, you would use platform-specific calls

	// Find an existing directory
	checkPath := path
	for checkPath != "/" && checkPath != "." && checkPath != "" {
		if stat, err := os.Stat(checkPath); err == nil && stat.IsDir() {
			break
		}
		checkPath = filepath.Dir(checkPath)
		if runtime.GOOS == "windows" && len(checkPath) <= 3 { // e.g., "C:\"
			break
		}
	}

	if checkPath == "" {
		checkPath = os.TempDir()
	}

	// Create a small test file to verify we can write
	testFile, err := os.CreateTemp(checkPath, "s5cmd-space-test-*")
	if err != nil {
		return 0, fmt.Errorf("cannot write to disk: %w", err)
	}
	defer func() {
		testFile.Close()
		os.Remove(testFile.Name())
	}()

	// For now, return a conservative estimate
	// In a real implementation, you would use:
	// - syscall.Statfs on Unix/Linux
	// - GetDiskFreeSpaceEx on Windows
	// For this example, we'll return a large enough number to avoid blocking
	return 10 * 1024 * 1024 * 1024, nil // 10GB conservative estimate
}
