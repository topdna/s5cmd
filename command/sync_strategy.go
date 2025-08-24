package command

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"strings"

	errorpkg "github.com/peak/s5cmd/v2/error"
	"github.com/peak/s5cmd/v2/storage"
)

// SyncStrategy is the interface to make decision whether given source object should be synced
// to destination object
type SyncStrategy interface {
	ShouldSync(srcObject, dstObject *storage.Object) error
}

func NewStrategy(sizeOnly bool, hashOnly bool) SyncStrategy {
	if sizeOnly {
		return &SizeOnlyStrategy{}
	} else if hashOnly {
		return &HashStrategy{}
	} else {
		return &SizeAndModificationStrategy{}
	}
}

// SizeOnlyStrategy determines to sync based on objects' file sizes.
type SizeOnlyStrategy struct{}

func (s *SizeOnlyStrategy) ShouldSync(srcObj, dstObj *storage.Object) error {
	if srcObj.Size == dstObj.Size {
		return errorpkg.ErrObjectSizesMatch
	}
	return nil
}

// SizeAndModificationStrategy determines to sync based on objects' both sizes and modification times.
// It treats source object as the source-of-truth;
//
//	time: src > dst        size: src != dst    should sync: yes
//	time: src > dst        size: src == dst    should sync: yes
//	time: src <= dst       size: src != dst    should sync: yes
//	time: src <= dst       size: src == dst    should sync: no
type SizeAndModificationStrategy struct{}

func (sm *SizeAndModificationStrategy) ShouldSync(srcObj, dstObj *storage.Object) error {
	srcMod, dstMod := srcObj.ModTime, dstObj.ModTime
	if srcMod.After(*dstMod) {
		return nil
	}

	if srcObj.Size != dstObj.Size {
		return nil
	}

	return errorpkg.ErrObjectIsNewerAndSizesMatch
}

// HashStrategy determines to sync based on objects' hashes and sizes.
// It treats source object as the source-of-truth; Source object can be local file or remote (s3).
//
//	md5 hash: src 		!= dst			should sync: yes
//	md5 hash: src 		== dst			should sync: no
//	md5 hash: src multipart upload		should sync: yes (always)
//	md5 hash: can't open src			should sync: yes (but cp won't be able to open the file)
type HashStrategy struct{}

// isMultipartETag detects if an ETag is from a multipart upload
// Multipart upload ETags contain a dash followed by part count (e.g., "abc123-5")
func isMultipartETag(etag string) bool {
	return strings.Contains(etag, "-")
}

func (s *HashStrategy) ShouldSync(srcObj, dstObj *storage.Object) error {
	// Firstly check size. Maybe the sizes will be different.
	if srcObj.Size != dstObj.Size {
		return nil
	}

	srcHash := getHash(srcObj)
	dstHash := getHash(dstObj)

	// Always sync multipart uploads as ETags are not reliable for comparison
	if isMultipartETag(srcHash) || isMultipartETag(dstHash) {
		return nil
	}

	if srcHash == dstHash {
		return errorpkg.ErrObjectEtagsMatch
	}

	return nil
}

func getHash(obj *storage.Object) string {
	// if remote (s3) then should has Etag
	// if not remote (s3) but has Etag then return it
	if obj.URL.IsRemote() || obj.Etag != "" {
		return obj.Etag
	} else {
		// cp.go opens the file again. It MAY be possible not to open the file again to calculate the hash.
		// fs.go Stat loads file metadata. It is possible to calculate md5 hash in that place, but not necessary.
		file, err := os.OpenFile(obj.URL.String(), os.O_RDONLY, 0644)
		// Can't open source file? Push it to the storage.
		// Not sure about this place. Maybe should throw exception and stop execution.
		// But if can't open file here, then can't open file in cp and upload it.
		if err != nil {
			// Return empty string to force sync, allowing cp to handle the actual error
			return ""
		}
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				// Intentionally ignore close errors as this is best-effort cleanup
				// The file operation has already completed successfully
				_ = closeErr
			}
		}()

		md5Obj := md5.New()
		// Use fixed buffer size instead of file size to prevent OOM for large files
		const bufferSize = 32 * 1024 // 32KB chunks
		buf := make([]byte, bufferSize)
		if _, err := io.CopyBuffer(md5Obj, file, buf); err != nil {
			// Return empty string to force sync if hash calculation fails
			// This ensures the file will be copied and the actual error will surface during cp
			return ""
		}

		return hex.EncodeToString(md5Obj.Sum(nil))
	}
}
