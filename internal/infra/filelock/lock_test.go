package filelock

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestLocker_LockBlocksSecondAcquire(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skillc.lock")

	first := New(path)
	assert.NoErr(t, first.Lock())
	defer first.Unlock()

	second := New(path)
	err := second.Lock()
	assert.True(t, errors.Is(err, ErrLocked))
}

func TestLocker_UnlockAllowsRelock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skillc.lock")

	locker := New(path)
	assert.NoErr(t, locker.Lock())
	assert.NoErr(t, locker.Unlock())

	reopened := New(path)
	assert.NoErr(t, reopened.Lock())
	assert.NoErr(t, reopened.Unlock())
}
