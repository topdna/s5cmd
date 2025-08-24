package command

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	errorpkg "github.com/peak/s5cmd/v2/error"
	"github.com/peak/s5cmd/v2/storage"
	"github.com/peak/s5cmd/v2/storage/url"
	"gotest.tools/v3/assert"
)

func TestSizeOnlyStrategy(t *testing.T) {
	strategy := &SizeOnlyStrategy{}

	// Create test objects with different sizes
	srcObj := &storage.Object{Size: 100}
	dstObj := &storage.Object{Size: 200}

	// Different sizes should sync
	err := strategy.ShouldSync(srcObj, dstObj)
	assert.NilError(t, err)

	// Same sizes should not sync
	dstObj.Size = 100
	err = strategy.ShouldSync(srcObj, dstObj)
	assert.Equal(t, err, errorpkg.ErrObjectSizesMatch)
}

func TestSizeAndModificationStrategy(t *testing.T) {
	strategy := &SizeAndModificationStrategy{}

	now := time.Now()
	older := now.Add(-time.Hour)
	newer := now.Add(time.Hour)

	testCases := []struct {
		name        string
		srcModTime  time.Time
		dstModTime  time.Time
		srcSize     int64
		dstSize     int64
		shouldSync  bool
		expectedErr error
	}{
		{
			name:       "newer source, different size",
			srcModTime: newer,
			dstModTime: older,
			srcSize:    100,
			dstSize:    200,
			shouldSync: true,
		},
		{
			name:       "newer source, same size",
			srcModTime: newer,
			dstModTime: older,
			srcSize:    100,
			dstSize:    100,
			shouldSync: true,
		},
		{
			name:       "older source, different size",
			srcModTime: older,
			dstModTime: newer,
			srcSize:    100,
			dstSize:    200,
			shouldSync: true,
		},
		{
			name:        "older source, same size",
			srcModTime:  older,
			dstModTime:  newer,
			srcSize:     100,
			dstSize:     100,
			shouldSync:  false,
			expectedErr: errorpkg.ErrObjectIsNewerAndSizesMatch,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srcObj := &storage.Object{
				ModTime: &tc.srcModTime,
				Size:    tc.srcSize,
			}
			dstObj := &storage.Object{
				ModTime: &tc.dstModTime,
				Size:    tc.dstSize,
			}

			err := strategy.ShouldSync(srcObj, dstObj)
			if tc.shouldSync {
				assert.NilError(t, err)
			} else {
				assert.Equal(t, err, tc.expectedErr)
			}
		})
	}
}

func TestIsMultipartETag(t *testing.T) {
	testCases := []struct {
		etag        string
		isMultipart bool
	}{
		{"d41d8cd98f00b204e9800998ecf8427e", false},  // Regular MD5
		{"d41d8cd98f00b204e9800998ecf8427e-5", true}, // Multipart with dash
		{"", false},           // Empty
		{"abc-def-ghi", true}, // Multiple dashes
		{"nodash", false},     // No dash
	}

	for _, tc := range testCases {
		t.Run(tc.etag, func(t *testing.T) {
			result := isMultipartETag(tc.etag)
			assert.Equal(t, result, tc.isMultipart)
		})
	}
}

func TestHashStrategy(t *testing.T) {
	strategy := &HashStrategy{}

	// Create URLs for test objects
	remoteURL, _ := url.New("s3://bucket/key")
	remoteURL2, _ := url.New("s3://bucket/key2")

	// Test different sizes - should always sync
	srcObj := &storage.Object{URL: remoteURL, Size: 100, Etag: "etag1"}
	dstObj := &storage.Object{URL: remoteURL2, Size: 200, Etag: "etag2"}
	err := strategy.ShouldSync(srcObj, dstObj)
	assert.NilError(t, err)

	// Test same ETags - should not sync
	srcObj.Size = 100
	dstObj.Size = 100
	srcObj.Etag = "sameetag"
	dstObj.Etag = "sameetag"
	err = strategy.ShouldSync(srcObj, dstObj)
	assert.Equal(t, err, errorpkg.ErrObjectEtagsMatch)

	// Test different ETags - should sync
	dstObj.Etag = "differentetag"
	err = strategy.ShouldSync(srcObj, dstObj)
	assert.NilError(t, err)

	// Test multipart ETags - should always sync
	srcObj.Etag = "etag1-5" // Multipart ETag
	dstObj.Etag = "etag2"
	err = strategy.ShouldSync(srcObj, dstObj)
	assert.NilError(t, err)

	dstObj.Etag = "etag2-3" // Both multipart
	err = strategy.ShouldSync(srcObj, dstObj)
	assert.NilError(t, err)
}

func TestGetHashWithRemoteObject(t *testing.T) {
	// Test remote object (should return existing Etag)
	remoteURL, _ := url.New("s3://bucket/key")
	obj := &storage.Object{
		URL:  remoteURL,
		Etag: "remote-etag",
	}

	hash := getHash(obj)
	assert.Equal(t, hash, "remote-etag")
}

func TestGetHashWithLocalFileEtag(t *testing.T) {
	// Test local object with existing Etag
	localURL, _ := url.New("/local/file")
	obj := &storage.Object{
		URL:  localURL,
		Etag: "existing-etag",
	}

	hash := getHash(obj)
	assert.Equal(t, hash, "existing-etag")
}

func TestGetHashWithLocalFile(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "testfile")

	content := "Hello, World!"
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	assert.NilError(t, err)

	// Calculate expected MD5
	md5Hash := md5.Sum([]byte(content))
	expectedHash := hex.EncodeToString(md5Hash[:])

	// Test local file hash calculation
	localURL, _ := url.New(tmpFile)
	obj := &storage.Object{
		URL:  localURL,
		Etag: "", // No existing Etag
		Size: int64(len(content)),
	}

	hash := getHash(obj)
	assert.Equal(t, hash, expectedHash)
}

func TestGetHashWithLargeFile(t *testing.T) {
	// Create a temporary large file to test memory usage
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "largefile")

	// Create 1MB file
	content := strings.Repeat("A", 1024*1024)
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	assert.NilError(t, err)

	// Calculate expected MD5
	md5Hash := md5.Sum([]byte(content))
	expectedHash := hex.EncodeToString(md5Hash[:])

	// Test large file hash calculation
	localURL, _ := url.New(tmpFile)
	obj := &storage.Object{
		URL:  localURL,
		Etag: "",
		Size: int64(len(content)),
	}

	hash := getHash(obj)
	assert.Equal(t, hash, expectedHash)
}

func TestGetHashWithNonExistentFile(t *testing.T) {
	// Test with non-existent file
	localURL, _ := url.New("/non/existent/file")
	obj := &storage.Object{
		URL:  localURL,
		Etag: "",
		Size: 100,
	}

	hash := getHash(obj)
	assert.Equal(t, hash, "") // Should return empty string on error
}

func TestNewStrategy(t *testing.T) {
	// Test creating different strategies
	sizeOnly := NewStrategy(true, false)
	_, ok := sizeOnly.(*SizeOnlyStrategy)
	assert.Assert(t, ok)

	hashOnly := NewStrategy(false, true)
	_, ok = hashOnly.(*HashStrategy)
	assert.Assert(t, ok)

	sizeAndMod := NewStrategy(false, false)
	_, ok = sizeAndMod.(*SizeAndModificationStrategy)
	assert.Assert(t, ok)

	// Test priority: sizeOnly takes precedence over hashOnly
	sizeOnlyPriority := NewStrategy(true, true)
	_, ok = sizeOnlyPriority.(*SizeOnlyStrategy)
	assert.Assert(t, ok)
}

func TestHashStrategyWithEmptyFiles(t *testing.T) {
	strategy := &HashStrategy{}

	// Create empty temporary files
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "src")
	dstFile := filepath.Join(tmpDir, "dst")

	err := os.WriteFile(srcFile, []byte{}, 0644)
	assert.NilError(t, err)
	err = os.WriteFile(dstFile, []byte{}, 0644)
	assert.NilError(t, err)

	srcURL, _ := url.New(srcFile)
	dstURL, _ := url.New(dstFile)

	srcObj := &storage.Object{URL: srcURL, Size: 0, Etag: ""}
	dstObj := &storage.Object{URL: dstURL, Size: 0, Etag: ""}

	// Empty files should have same hash and not sync
	err = strategy.ShouldSync(srcObj, dstObj)
	assert.Equal(t, err, errorpkg.ErrObjectEtagsMatch)
}

func TestGetHashWithFileReadError(t *testing.T) {
	// Create a temporary file and then remove it to simulate read error
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "testfile")

	content := "test content"
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	assert.NilError(t, err)

	// Remove the file to simulate file not found error
	err = os.Remove(tmpFile)
	assert.NilError(t, err)

	// Test hash calculation with missing file
	localURL, _ := url.New(tmpFile)
	obj := &storage.Object{
		URL:  localURL,
		Etag: "",
		Size: int64(len(content)),
	}

	hash := getHash(obj)
	assert.Equal(t, hash, "") // Should return empty string on file access error
}
