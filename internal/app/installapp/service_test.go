package installapp

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/agent"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
)

func TestService_InstallCopiesFilesAndWritesLock(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "source")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	targetRoot := filepath.Join(baseDir, ".claude", "skills")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "commands", "hello.txt"), []byte("hello"), 0o644))

	item := skill.Skill{
		ID:           "hello-skill",
		Name:         "Hello Skill",
		Version:      "1.0.0",
		SourceID:     "local-demo",
		SourceType:   sourcepkg.TypeLocal,
		InstallEntry: "commands",
		Path:         sourceDir,
	}

	service := NewService(lockFile)
	service.now = func() time.Time { return time.Unix(1710000000, 0).UTC() }

	record, err := service.Install(item, "claude-code", agent.ScopeProject, targetRoot)
	assert.NoErr(t, err)
	assert.Eq(t, "hello-skill", record.SkillID)
	assert.Eq(t, "commands", record.InstallEntry)
	assert.Eq(t, filepath.Join(targetRoot, "hello-skill"), record.InstalledPath)

	data, err := os.ReadFile(filepath.Join(targetRoot, "hello-skill", "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))

	locks, err := service.store.Load(lockFile)
	assert.NoErr(t, err)
	assert.Len(t, locks, 1)
	assert.Eq(t, record.SkillID, locks[0].SkillID)
	assert.Eq(t, "commands", locks[0].InstallEntry)
}

func TestService_InstallAppendsLockRecords(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	targetRoot := filepath.Join(baseDir, ".claude", "skills")
	sourceDir := filepath.Join(baseDir, "source")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "commands", "hello.txt"), []byte("hello"), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "commands", "world.txt"), []byte("world"), 0o644))

	service := NewService(lockFile)
	_, err := service.Install(skill.Skill{ID: "hello-skill", Version: "1.0.0", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: sourceDir}, "claude-code", agent.ScopeProject, targetRoot)
	assert.NoErr(t, err)
	_, err = service.Install(skill.Skill{ID: "world-skill", Version: "1.0.0", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: sourceDir}, "codex", agent.ScopeProject, targetRoot)
	assert.NoErr(t, err)

	locks, err := service.store.Load(lockFile)
	assert.NoErr(t, err)
	assert.Len(t, locks, 2)
}

func TestService_UninstallRemovesFilesAndLockRecord(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	installedPath := filepath.Join(baseDir, ".claude", "skills", "hello-skill")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(installedPath, "hello.txt"), []byte("hello"), 0o644))
	assert.NoErr(t, NewService(lockFile).store.Save(lockFile, []lockpkg.Record{{
		SkillID:       "hello-skill",
		Agent:         "claude-code",
		Scope:         "project",
		InstalledPath: installedPath,
	}}))

	service := NewService(lockFile)
	err := service.Uninstall("hello-skill", "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)

	_, err = os.Stat(installedPath)
	assert.True(t, os.IsNotExist(err))

	locks, err := service.store.Load(lockFile)
	assert.NoErr(t, err)
	assert.Len(t, locks, 0)
}

func TestService_RestoreUsesRecordedInstallEntry(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := filepath.Join(baseDir, "cache", "hello-skill")
	commandsDir := filepath.Join(sourceDir, "commands")
	installedPath := filepath.Join(baseDir, ".claude", "skills", "hello-skill")
	assert.NoErr(t, os.MkdirAll(commandsDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(commandsDir, "hello.txt"), []byte("restored"), 0o644))
	assert.NoErr(t, NewService(lockFile).store.Save(lockFile, []lockpkg.Record{{
		SkillID:       "hello-skill",
		Agent:         "claude-code",
		Scope:         "project",
		InstalledPath: installedPath,
		SourceID:      "local-demo",
		SourceType:    "local",
		InstallEntry:  "commands",
	}}))

	service := NewService(lockFile)
	restored, err := service.Restore(map[string]string{"local-demo": sourceDir})
	assert.NoErr(t, err)
	assert.Len(t, restored, 1)
	assert.Eq(t, "hello-skill", restored[0].SkillID)

	data, err := os.ReadFile(filepath.Join(installedPath, "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "restored", string(data))
}
