package agentfs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestInstaller_InstallCopiesFileTree(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "source")
	targetDir := filepath.Join(baseDir, "target")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "skill.txt"), []byte("hello"), 0o644))

	installer := NewInstallerWithMode(ModeCopy)
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

// TestInstaller_InstallSymlinkOrFallback 验证默认 symlink 模式：
// 在不支持 symlink 的环境（如 Windows 无 Dev Mode）会自动回退到 copy。
func TestInstaller_InstallSymlinkOrFallback(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "source")
	targetDir := filepath.Join(baseDir, "target")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "skill.txt"), []byte("hello"), 0o644))

	var fellBack bool
	installer := NewInstaller()
	installer.OnSymlinkFallback = func(_, _ string, _ error) { fellBack = true }
	assert.NoErr(t, installer.Install(sourceDir, targetDir))

	// 无论是 symlink 还是回退后的 copy，都应能从 targetDir 读取到内容
	data, err := os.ReadFile(filepath.Join(targetDir, "skill.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))

	info, err := os.Lstat(targetDir)
	assert.NoErr(t, err)
	if info.Mode()&os.ModeSymlink != 0 {
		// symlink 安装：修改源文件，从 target 应读到新内容
		assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "skill.txt"), []byte("world"), 0o644))
		data, err := os.ReadFile(filepath.Join(targetDir, "skill.txt"))
		assert.NoErr(t, err)
		assert.Eq(t, "world", string(data))
		assert.False(t, fellBack)

		// 移除 symlink 不应影响源
		assert.NoErr(t, installer.Remove(targetDir))
		_, err = os.Stat(filepath.Join(sourceDir, "skill.txt"))
		assert.NoErr(t, err)
	} else {
		// 在 Windows 无权限场景下应触发回退
		if runtime.GOOS == "windows" {
			assert.True(t, fellBack)
		}
	}
}

// TestInstaller_InstallSymlinkOverridesExistingTarget 当 target 已是普通目录时，
// symlink 模式应先清理再创建链接（或回退后用 copy 覆盖）。
func TestInstaller_InstallSymlinkOverridesExistingTarget(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "source")
	targetDir := filepath.Join(baseDir, "target")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "new.txt"), []byte("new"), 0o644))
	assert.NoErr(t, os.MkdirAll(targetDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "old.txt"), []byte("old"), 0o644))

	installer := NewInstaller()
	assert.NoErr(t, installer.Install(sourceDir, targetDir))

	_, err := os.Stat(filepath.Join(targetDir, "new.txt"))
	assert.NoErr(t, err)
}

func TestNormalizeMode(t *testing.T) {
	assert.Eq(t, ModeSymlink, NormalizeMode(""))
	assert.Eq(t, ModeSymlink, NormalizeMode("symlink"))
	assert.Eq(t, ModeCopy, NormalizeMode("copy"))
	assert.Eq(t, ModeSymlink, NormalizeMode("invalid"))
}
