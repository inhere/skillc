package agentfs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestInstaller_InstallCopiesFileTree(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "source")
	targetDir := filepath.Join(baseDir, "target")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "skill.txt"), []byte("hello"), 0o644))

	installer := NewInstaller()
	assert.NoErr(t, installer.Install(sourceDir, targetDir))

	data, err := os.ReadFile(filepath.Join(targetDir, "skill.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))
}

func TestInstaller_RemoveDeletesInstalledTree(t *testing.T) {
	targetDir := filepath.Join(t.TempDir(), "target")
	assert.NoErr(t, os.MkdirAll(targetDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "skill.txt"), []byte("hello"), 0o644))

	installer := NewInstaller()
	assert.NoErr(t, installer.Remove(targetDir))

	_, err := os.Stat(targetDir)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}
