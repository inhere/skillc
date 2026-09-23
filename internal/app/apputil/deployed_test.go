package apputil

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/infra/hashx"
)

func TestOverwriteGuardRefusesModifiedCopyInstall(t *testing.T) {
	baseDir := t.TempDir()
	target := filepath.Join(baseDir, "skills", "demo")
	assert.NoErr(t, os.MkdirAll(target, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(target, "SKILL.md"), []byte("skill"), 0o644))
	record := copyRecord(t, target)
	assert.NoErr(t, os.WriteFile(filepath.Join(target, "SKILL.md"), []byte("local edit"), 0o644))

	guard := OverwriteGuard{BackupRoot: filepath.Join(baseDir, "backups")}
	_, err := guard.GuardOverwrite(record, filepath.Join(baseDir, "project"), target)

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "local changes")
	assert.Contains(t, err.Error(), "--force")
	assertFileContent(t, filepath.Join(target, "SKILL.md"), "local edit")
}

func TestOverwriteGuardBacksUpBeforeForcedOverwrite(t *testing.T) {
	baseDir := t.TempDir()
	target := filepath.Join(baseDir, "skills", "demo")
	assert.NoErr(t, os.MkdirAll(target, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(target, "SKILL.md"), []byte("skill"), 0o644))
	record := copyRecord(t, target)
	assert.NoErr(t, os.WriteFile(filepath.Join(target, "SKILL.md"), []byte("local edit"), 0o644))

	guard := OverwriteGuard{BackupRoot: filepath.Join(baseDir, "backups"), Force: true}
	backupPath, err := guard.GuardOverwrite(record, filepath.Join(baseDir, "project"), target)

	assert.NoErr(t, err)
	assert.NotEq(t, "", backupPath)
	assert.Contains(t, backupPath, "demo")
	assertFileContent(t, filepath.Join(backupPath, "SKILL.md"), "local edit")
}

func TestOverwriteGuardBacksUpTargetWithoutFingerprint(t *testing.T) {
	baseDir := t.TempDir()
	target := filepath.Join(baseDir, "skills", "demo")
	assert.NoErr(t, os.MkdirAll(target, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(target, "SKILL.md"), []byte("legacy"), 0o644))

	guard := OverwriteGuard{BackupRoot: filepath.Join(baseDir, "backups")}
	backupPath, err := guard.GuardOverwrite(lockpkg.Record{SkillID: "demo", InstallMode: "copy"}, filepath.Join(baseDir, "project"), target)

	assert.NoErr(t, err)
	assert.NotEq(t, "", backupPath)
	assertFileContent(t, filepath.Join(backupPath, "SKILL.md"), "legacy")
}

func TestOverwriteGuardSkipsUnchangedAndMissingTargets(t *testing.T) {
	baseDir := t.TempDir()
	unchanged := filepath.Join(baseDir, "skills", "demo")
	assert.NoErr(t, os.MkdirAll(unchanged, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(unchanged, "SKILL.md"), []byte("skill"), 0o644))

	guard := OverwriteGuard{BackupRoot: filepath.Join(baseDir, "backups")}
	backupPath, err := guard.GuardOverwrite(copyRecord(t, unchanged), filepath.Join(baseDir, "project"), unchanged)
	assert.NoErr(t, err)
	assert.Eq(t, "", backupPath)

	backupPath, err = guard.GuardOverwrite(copyRecord(t, unchanged), filepath.Join(baseDir, "project"), filepath.Join(baseDir, "skills", "missing"))
	assert.NoErr(t, err)
	assert.Eq(t, "", backupPath)
	assertNoDir(t, filepath.Join(baseDir, "backups"))
}

// link 安装（symlink/junction）的目标目录就是源目录，绝不能备份或改动它。
func TestOverwriteGuardSkipsLinkedTargets(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "source")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("skill"), 0o644))

	target := filepath.Join(baseDir, "skills", "demo")
	assert.NoErr(t, os.MkdirAll(filepath.Dir(target), 0o755))
	if err := linkDir(sourceDir, target); err != nil {
		t.Skipf("link install is not supported here: %v", err)
	}

	guard := OverwriteGuard{BackupRoot: filepath.Join(baseDir, "backups"), Force: true}
	backupPath, err := guard.GuardOverwrite(lockpkg.Record{SkillID: "demo", InstallMode: "copy", InstalledChecksum: "stale"}, filepath.Join(baseDir, "project"), target)

	assert.NoErr(t, err)
	assert.Eq(t, "", backupPath)
	assertNoDir(t, filepath.Join(baseDir, "backups"))
}

func TestScopeLabelSeparatesProjectsWithSameName(t *testing.T) {
	first := ScopeLabel(filepath.Join(t.TempDir(), "alpha", "project"))
	second := ScopeLabel(filepath.Join(t.TempDir(), "beta", "project"))

	assert.NotEq(t, first, second)
	assert.Contains(t, first, "project-")
	assert.Eq(t, "user", ScopeLabel(lockpkg.GlobalKey))
}

// copyRecord 生成一条与 target 当前内容匹配的 copy 模式记录。
func copyRecord(t *testing.T, target string) lockpkg.Record {
	t.Helper()
	sum, err := hashx.SumDeployed(target)
	assert.NoErr(t, err)
	return lockpkg.Record{SkillID: "demo", InstallMode: "copy", InstalledChecksum: sum}
}

func linkDir(source string, target string) error {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", "mklink", "/J", target, source).Run()
	}
	return os.Symlink(source, target)
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	assert.NoErr(t, err)
	assert.Eq(t, want, string(data))
}

func assertNoDir(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be absent, got err=%v", path, err)
	}
}
