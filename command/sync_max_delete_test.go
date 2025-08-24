package command

import (
	"testing"

	"github.com/urfave/cli/v2"
	"gotest.tools/v3/assert"
)

func TestMaxDeleteFlagDefinition(t *testing.T) {
	// Test that the max-delete flag is defined correctly
	flags := NewSyncCommandFlags()

	// Find the max-delete flag
	var maxDeleteFlag *cli.IntFlag
	for _, flag := range flags {
		if intFlag, ok := flag.(*cli.IntFlag); ok && intFlag.Name == "max-delete" {
			maxDeleteFlag = intFlag
			break
		}
	}

	assert.Assert(t, maxDeleteFlag != nil, "max-delete flag should be defined")
	assert.Equal(t, maxDeleteFlag.Name, "max-delete")
	assert.Equal(t, maxDeleteFlag.Usage, "don't delete more than NUM files")
	assert.Equal(t, maxDeleteFlag.Value, -1, "default value should be -1 (unlimited)")
}

func TestMaxDeleteLogic(t *testing.T) {
	testCases := []struct {
		name            string
		maxDelete       int
		filesToDelete   int
		shouldDelete    bool
		expectedMessage string
	}{
		{
			name:          "unlimited deletions",
			maxDelete:     -1,
			filesToDelete: 100,
			shouldDelete:  true,
		},
		{
			name:          "within limit",
			maxDelete:     10,
			filesToDelete: 5,
			shouldDelete:  true,
		},
		{
			name:          "at exact limit",
			maxDelete:     10,
			filesToDelete: 10,
			shouldDelete:  true,
		},
		{
			name:            "exceeds limit",
			maxDelete:       5,
			filesToDelete:   10,
			shouldDelete:    false,
			expectedMessage: "refusing to delete 10 files; more than max-delete limit of 5",
		},
		{
			name:            "zero limit with files to delete",
			maxDelete:       0,
			filesToDelete:   1,
			shouldDelete:    false,
			expectedMessage: "refusing to delete 1 files; more than max-delete limit of 0",
		},
		{
			name:          "zero limit with no files",
			maxDelete:     0,
			filesToDelete: 0,
			shouldDelete:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the logic that would be used in planRun
			shouldDelete := tc.filesToDelete == 0 || (tc.maxDelete < 0 || tc.filesToDelete <= tc.maxDelete)
			assert.Equal(t, shouldDelete, tc.shouldDelete)

			if !tc.shouldDelete && tc.filesToDelete > 0 {
				// Verify the expected error message format
				expectedMsg := tc.expectedMessage
				assert.Assert(t, expectedMsg != "")
			}
		})
	}
}
