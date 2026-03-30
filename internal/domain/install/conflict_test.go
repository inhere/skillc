package install

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestDetectConflict_ReportsExistingTarget(t *testing.T) {
	plan := Plan{TargetPath: "existing-path"}
	conflict := DetectConflict(plan, true)
	assert.True(t, conflict.Exists)
	assert.Eq(t, ConflictModeOverwrite, conflict.Mode)
}

func TestDetectConflict_SkipsMissingTarget(t *testing.T) {
	conflict := DetectConflict(Plan{TargetPath: "new-path"}, false)
	assert.False(t, conflict.Exists)
	assert.Eq(t, ConflictModeOverwrite, conflict.Mode)
}
