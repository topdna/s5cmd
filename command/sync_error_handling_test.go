package command

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/urfave/cli/v2"
	"gotest.tools/v3/assert"

	"github.com/peak/s5cmd/v2/log"
	"github.com/peak/s5cmd/v2/storage"
	"github.com/peak/s5cmd/v2/storage/url"
)

// Mock pipe writer that can simulate blocking operations
type mockPipeWriter struct {
	io.WriteCloser
	blockingEnabled bool
	writeCh         chan []byte
	closeCh         chan struct{}
}

func newMockPipeWriter() *mockPipeWriter {
	return &mockPipeWriter{
		writeCh: make(chan []byte, 100),
		closeCh: make(chan struct{}),
	}
}

func (m *mockPipeWriter) Write(p []byte) (int, error) {
	if m.blockingEnabled {
		select {
		case m.writeCh <- p:
			return len(p), nil
		case <-m.closeCh:
			return 0, fmt.Errorf("writer closed")
		}
	}
	return len(p), nil
}

func (m *mockPipeWriter) Close() error {
	close(m.closeCh)
	return nil
}

func (m *mockPipeWriter) enableBlocking() {
	m.blockingEnabled = true
}

func TestSyncShouldStopSyncEnhancedErrors(t *testing.T) {
	// Initialize logger to prevent nil pointer panics
	log.Init("error", false)
	defer log.Close()

	s := Sync{
		exitOnError: false, // Test without exit-on-error flag first
	}

	testCases := []struct {
		name           string
		err            error
		exitOnError    bool
		expectedResult bool
	}{
		{
			name:           "NoSuchObjectFound should not stop",
			err:            storage.ErrNoObjectFound,
			exitOnError:    false,
			expectedResult: false,
		},
		{
			name:           "AccessDenied should stop",
			err:            awserr.New("AccessDenied", "access denied", nil),
			exitOnError:    false,
			expectedResult: true,
		},
		{
			name:           "NoSuchBucket should stop",
			err:            awserr.New("NoSuchBucket", "bucket does not exist", nil),
			exitOnError:    false,
			expectedResult: true,
		},
		{
			name:           "RequestError should stop",
			err:            awserr.New("RequestError", "request error", nil),
			exitOnError:    false,
			expectedResult: true,
		},
		{
			name:           "SerializationError should stop",
			err:            awserr.New("SerializationError", "serialization error", nil),
			exitOnError:    false,
			expectedResult: true,
		},
		{
			name:           "Other AWS error with exitOnError false should not stop",
			err:            awserr.New("SomeOtherError", "other error", nil),
			exitOnError:    false,
			expectedResult: false,
		},
		{
			name:           "Other AWS error with exitOnError true should stop",
			err:            awserr.New("SomeOtherError", "other error", nil),
			exitOnError:    true,
			expectedResult: true,
		},
		{
			name:           "Non-AWS error with exitOnError false should not stop",
			err:            fmt.Errorf("generic error"),
			exitOnError:    false,
			expectedResult: false,
		},
		{
			name:           "Non-AWS error with exitOnError true should stop",
			err:            fmt.Errorf("generic error"),
			exitOnError:    true,
			expectedResult: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s.exitOnError = tc.exitOnError
			result := s.shouldStopSync(tc.err)
			assert.Equal(t, tc.expectedResult, result, "shouldStopSync result mismatch for %s", tc.name)
		})
	}
}

func TestSyncContextCancellationInPlanRun(t *testing.T) {
	// Initialize logger to prevent nil pointer panics
	log.Init("error", false)
	defer log.Close()

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Create mock channels
	onlySource := make(chan *url.URL, 10)
	onlyDest := make(chan *url.URL, 10)
	common := make(chan *ObjectPair, 10)

	// Create mock destination URL
	dsturl, err := url.New("s3://test-bucket/")
	assert.NilError(t, err)

	// Create mock CLI context
	app := cli.NewApp()
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cliCtx := cli.NewContext(app, flagSet, nil)

	// Create sync instance
	s := Sync{
		op:          "sync",
		fullCommand: "s5cmd sync",
		delete:      false,
		maxDelete:   -1,
	}

	// Create mock strategy
	strategy := &SizeOnlyStrategy{}

	// Create mock pipe writer
	mockWriter := newMockPipeWriter()

	// Add some test URLs to channels
	testURL1, _ := url.New("s3://source-bucket/file1.txt")
	testURL2, _ := url.New("s3://source-bucket/file2.txt")

	onlySource <- testURL1
	onlySource <- testURL2

	// Cancel the context immediately to test cancellation behavior
	cancel()

	// Close channels to signal completion
	close(onlySource)
	close(onlyDest)
	close(common)

	// Run planRun - should exit gracefully due to context cancellation
	s.planRun(ctx, cliCtx, onlySource, onlyDest, common, dsturl, strategy, mockWriter, true, 1)

	// Test should complete without hanging
	// If context cancellation is not working properly, this test would hang
}

func TestSyncContextCancellationWithBlocking(t *testing.T) {
	// Initialize logger to prevent nil pointer panics
	log.Init("error", false)
	defer log.Close()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Create channels that will block
	onlySource := make(chan *url.URL)
	onlyDest := make(chan *url.URL)
	common := make(chan *ObjectPair)

	// Create mock destination URL
	dsturl, err := url.New("s3://test-bucket/")
	assert.NilError(t, err)

	// Create mock CLI context
	app := cli.NewApp()
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cliCtx := cli.NewContext(app, flagSet, nil)

	// Create sync instance with delete enabled
	s := Sync{
		op:          "sync",
		fullCommand: "s5cmd sync",
		delete:      true,
		maxDelete:   -1,
	}

	// Create mock strategy
	strategy := &SizeOnlyStrategy{}

	// Create mock pipe writer
	mockWriter := newMockPipeWriter()
	mockWriter.enableBlocking()

	// Run planRun in goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.planRun(ctx, cliCtx, onlySource, onlyDest, common, dsturl, strategy, mockWriter, true, 1)
	}()

	// Wait for either completion or timeout
	select {
	case <-done:
		// Good! planRun completed due to context cancellation
	case <-time.After(200 * time.Millisecond):
		t.Fatal("planRun did not respect context cancellation and timed out")
	}
}

func TestSyncErrorHandlingIntegration(t *testing.T) {
	// Test that critical errors (RequestError, SerializationError) are properly handled
	testCases := []struct {
		name       string
		errorCode  string
		shouldStop bool
	}{
		{
			name:       "RequestError should trigger stop",
			errorCode:  "RequestError",
			shouldStop: true,
		},
		{
			name:       "SerializationError should trigger stop",
			errorCode:  "SerializationError",
			shouldStop: true,
		},
		{
			name:       "AccessDenied should trigger stop",
			errorCode:  "AccessDenied",
			shouldStop: true,
		},
		{
			name:       "NoSuchBucket should trigger stop",
			errorCode:  "NoSuchBucket",
			shouldStop: true,
		},
		{
			name:       "InternalError should not trigger stop",
			errorCode:  "InternalError",
			shouldStop: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := Sync{exitOnError: false}

			awsErr := awserr.New(tc.errorCode, fmt.Sprintf("Mock %s error", tc.errorCode), nil)
			result := s.shouldStopSync(awsErr)

			assert.Equal(t, tc.shouldStop, result,
				"shouldStopSync result mismatch for error code %s", tc.errorCode)
		})
	}
}

func TestSyncMaxDeleteWithContextCancellation(t *testing.T) {
	// Initialize logger to prevent nil pointer panics
	log.Init("error", false)
	defer log.Close()

	// Test that max-delete logic works properly with context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create channels
	onlySource := make(chan *url.URL)
	onlyDest := make(chan *url.URL, 10)
	common := make(chan *ObjectPair)

	// Add URLs that would exceed max-delete limit
	testURLs := make([]*url.URL, 5)
	for i := 0; i < 5; i++ {
		testURL, _ := url.New(fmt.Sprintf("s3://dest-bucket/file%d.txt", i))
		testURLs[i] = testURL
		onlyDest <- testURL
	}
	close(onlyDest)
	close(onlySource)
	close(common)

	// Create mock destination URL
	dsturl, err := url.New("s3://test-bucket/")
	assert.NilError(t, err)

	// Create mock CLI context
	app := cli.NewApp()
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	cliCtx := cli.NewContext(app, flagSet, nil)

	// Create sync instance with max-delete limit of 3
	s := Sync{
		op:          "sync",
		fullCommand: "s5cmd sync",
		delete:      true,
		maxDelete:   3, // Limit to 3 files, but we have 5
	}

	// Create mock strategy
	strategy := &SizeOnlyStrategy{}

	// Capture output
	var output strings.Builder
	mockWriter := &mockWriteCloser{writer: &output}

	// Run planRun
	s.planRun(ctx, cliCtx, onlySource, onlyDest, common, dsturl, strategy, mockWriter, true, 1)

	// Should not have any rm commands in output due to max-delete limit
	outputStr := output.String()
	assert.Assert(t, !strings.Contains(outputStr, "rm"),
		"Should not generate rm commands when exceeding max-delete limit")
}

// Mock WriteCloser for testing
type mockWriteCloser struct {
	writer io.Writer
}

func (m *mockWriteCloser) Write(p []byte) (int, error) {
	return m.writer.Write(p)
}

func (m *mockWriteCloser) Close() error {
	return nil
}
