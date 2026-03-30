package fsx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestWriteFileAtomically_ReplacesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	assert.NoErr(t, os.WriteFile(path, []byte("old-value"), 0o644))

	err := WriteFileAtomically(path, []byte("new-value"), 0o600)
	assert.NoErr(t, err)

	data, err := os.ReadFile(path)
	assert.NoErr(t, err)
	assert.Eq(t, "new-value", string(data))

	info, err := os.Stat(path)
	assert.NoErr(t, err)
	assert.False(t, info.IsDir())
}

func TestWriteFileAtomically_CreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "skillc.lock")

	err := WriteFileAtomically(path, []byte("content"), 0o644)
	assert.NoErr(t, err)

	data, err := os.ReadFile(path)
	assert.NoErr(t, err)
	assert.Eq(t, "content", string(data))
}
