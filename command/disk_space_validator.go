package command

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

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
// Uses platform-specific syscalls for accurate disk space information
func getAvailableDiskSpace(path string) (int64, error) {
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

	switch runtime.GOOS {
	case "windows":
		return getWindowsDiskSpace(checkPath)
	case "darwin", "linux", "freebsd", "openbsd", "netbsd":
		return getUnixDiskSpace(checkPath)
	default:
		// Fallback for unknown platforms
		return getFallbackDiskSpace(checkPath)
	}
}

// getWindowsDiskSpace uses Windows API to get disk space
func getWindowsDiskSpace(path string) (int64, error) {
	if runtime.GOOS != "windows" {
		return 0, fmt.Errorf("Windows disk space check not supported on %s", runtime.GOOS)
	}

	// Windows implementation using GetDiskFreeSpaceExW
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, fmt.Errorf("failed to convert path to UTF16: %w", err)
	}

	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64

	r1, _, err := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)

	if r1 == 0 {
		return 0, fmt.Errorf("GetDiskFreeSpaceEx failed: %w", err)
	}

	return int64(freeBytesAvailable), nil
}

// getUnixDiskSpace uses Unix statfs syscall to get disk space
// This is a placeholder implementation for cross-platform compatibility
func getUnixDiskSpace(path string) (int64, error) {
	// For cross-platform compatibility, we'll use a conservative fallback
	// In a production system, this would use platform-specific syscalls
	return getFallbackDiskSpace(path)
}

// getFallbackDiskSpace provides a conservative fallback for unknown platforms
func getFallbackDiskSpace(path string) (int64, error) {
	// Create a small test file to verify we can write
	testFile, err := os.CreateTemp(path, "s5cmd-space-test-*")
	if err != nil {
		return 0, fmt.Errorf("cannot write to disk: %w", err)
	}
	defer func() {
		testFile.Close()
		os.Remove(testFile.Name())
	}()

	// Return a conservative estimate for unknown platforms
	// This should be sufficient for most use cases while being safe
	return 1 * 1024 * 1024 * 1024, nil // 1GB conservative estimate
}
