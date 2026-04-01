package installapp

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
)

type skillLookupFunc func(id string) ([]skill.Skill, error)

func (f skillLookupFunc) Resolve(id string) ([]skill.Skill, error) {
	return f(id)
}

func TestService_RunInstallsIndexedSkill(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := filepath.Join(baseDir, "source")
	commandsDir := filepath.Join(sourceDir, "commands")
	assert.NoErr(t, os.MkdirAll(commandsDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(commandsDir, "hello.txt"), []byte("hello"), 0o644))

	service := NewService(lockFile)
	req := InstallReq{
		SkillID: "hello-skill",
		Agent:   "claude-code",
		Scope:   "project",
		WorkDir: baseDir,
	}
	result, err := service.Run(cfg.Config{
		AgentTools: map[string]cfg.AgentToolConfig{
			"claude-code": {ProjectDir: filepath.Join(baseDir, ".claude")},
		},
	}, req, skillLookupFunc(func(id string) ([]skill.Skill, error) {
		return []skill.Skill{{
			ID:                  "hello-skill",
			QualifiedName:       "marketplaces/hello-skill",
			SourceQualifiedName: "repo-a/marketplaces/hello-skill",
			Version:             "1.0.0",
			SourceID:            "local-demo",
			SourceType:          sourcepkg.TypeLocal,
			InstallEntry:        "commands",
			Path:                sourceDir,
		}}, nil
	}))
	assert.NoErr(t, err)
	assert.Len(t, result.Installed, 1)
	assert.Eq(t, "hello-skill", result.Installed[0].SkillID)
	assert.Eq(t, "marketplaces/hello-skill", result.Installed[0].QualifiedName)
	assert.Len(t, result.Restored, 0)

	data, err := os.ReadFile(filepath.Join(baseDir, ".claude", "skills", "hello-skill", "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))
}

func TestService_RunInstallsCollectionTarget(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	firstSourceDir := filepath.Join(baseDir, "source", "hello-skill")
	secondSourceDir := filepath.Join(baseDir, "source", "world-skill")
	assert.NoErr(t, os.MkdirAll(filepath.Join(firstSourceDir, "commands"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(secondSourceDir, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(firstSourceDir, "commands", "hello.txt"), []byte("hello"), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(secondSourceDir, "commands", "world.txt"), []byte("world"), 0o644))

	service := NewService(lockFile)
	req := InstallReq{
		SkillID: "repo-a/marketplaces",
		Agent:   "claude-code",
		Scope:   "project",
		WorkDir: baseDir,
	}
	result, err := service.Run(cfg.Config{
		AgentTools: map[string]cfg.AgentToolConfig{
			"claude-code": {ProjectDir: filepath.Join(baseDir, ".claude")},
		},
	}, req, skillLookupFunc(func(id string) ([]skill.Skill, error) {
		return []skill.Skill{
			{ID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", Collection: "marketplaces", Version: "1.0.0", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: firstSourceDir},
			{ID: "world-skill", QualifiedName: "marketplaces/world-skill", SourceQualifiedName: "repo-a/marketplaces/world-skill", Collection: "marketplaces", Version: "1.0.0", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: secondSourceDir},
		}, nil
	}))
	assert.NoErr(t, err)
	assert.Len(t, result.Installed, 2)
}

func TestService_RunRestoresWhenNoArgs(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := filepath.Join(baseDir, "cache", "hello-skill")
	commandsDir := filepath.Join(sourceDir, "commands")
	installedPath := filepath.Join(baseDir, ".claude", "skills", "hello-skill")
	assert.NoErr(t, os.MkdirAll(commandsDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(commandsDir, "hello.txt"), []byte("restored"), 0o644))
	assert.NoErr(t, NewService(lockFile).store.Save(lockFile, []lockpkg.Record{{
		SkillID:             "hello-skill",
		QualifiedName:       "marketplaces/hello-skill",
		SourceQualifiedName: "repo-a/marketplaces/hello-skill",
		Agent:               "claude-code",
		Scope:               "project",
		InstalledPath:       installedPath,
		SourceID:            "local-demo",
		SourceType:          "local",
		InstallEntry:        "commands",
	}}))

	service := NewService(lockFile)
	req := InstallReq{
		WorkDir: baseDir,
	}
	result, err := service.Run(cfg.Config{Sources: []sourcepkg.Source{{ID: "local-demo", Path: sourceDir}}}, req, nil)
	assert.NoErr(t, err)
	assert.Len(t, result.Installed, 0)
	assert.Len(t, result.Restored, 1)
	assert.Eq(t, "hello-skill", result.Restored[0].SkillID)

	data, err := os.ReadFile(filepath.Join(installedPath, "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "restored", string(data))
}

func TestService_RunRequiresSkillLookupForInstall(t *testing.T) {
	service := NewService(filepath.Join(t.TempDir(), "skillc-install.lock"))
	req := InstallReq{
		SkillID: "hello-skill",
		Agent:   "claude-code",
		Scope:   "project",
		WorkDir: t.TempDir(),
	}
	_, err := service.Run(cfg.Config{}, req, nil)
	assert.Err(t, err)
	assert.Contains(t, err.Error(), "skill lookup is required")
}

func TestService_RunReturnsLookupErrors(t *testing.T) {
	service := NewService(filepath.Join(t.TempDir(), "skillc-install.lock"))
	req := InstallReq{
		SkillID: "missing",
		Agent:   "claude-code",
		Scope:   "project",
	}

	_, err := service.Run(cfg.Config{}, req, skillLookupFunc(func(id string) ([]skill.Skill, error) {
		return nil, fmt.Errorf("skill not found: %s", id)
	}))
	assert.Err(t, err)
	assert.Contains(t, err.Error(), "skill not found")
}

func TestService_InstallCopiesFilesAndWritesLock(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "source")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	targetRoot := filepath.Join(baseDir, ".claude", "skills")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "commands", "hello.txt"), []byte("hello"), 0o644))

	item := skill.Skill{
		ID:                  "hello-skill",
		Name:                "Hello Skill",
		QualifiedName:       "marketplaces/hello-skill",
		SourceQualifiedName: "repo-a/marketplaces/hello-skill",
		Version:             "1.0.0",
		SourceID:            "local-demo",
		SourceType:          sourcepkg.TypeLocal,
		InstallEntry:        "commands",
		Path:                sourceDir,
	}

	service := NewService(lockFile)
	service.now = func() time.Time { return time.Unix(1710000000, 0).UTC() }

	record, err := service.Install(item, "claude-code", agent.ScopeProject, targetRoot)
	assert.NoErr(t, err)
	assert.Eq(t, "hello-skill", record.SkillID)
	assert.Eq(t, "marketplaces/hello-skill", record.QualifiedName)
	assert.Eq(t, "commands", record.InstallEntry)
	assert.Eq(t, filepath.Join(targetRoot, "hello-skill"), record.InstalledPath)

	data, err := os.ReadFile(filepath.Join(targetRoot, "hello-skill", "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))

	locks, err := service.store.Load(lockFile)
	assert.NoErr(t, err)
	assert.Len(t, locks, 1)
	assert.Eq(t, record.SkillID, locks[0].SkillID)
	assert.Eq(t, "marketplaces/hello-skill", locks[0].QualifiedName)
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
	_, err := service.Install(skill.Skill{ID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", Version: "1.0.0", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: sourceDir}, "claude-code", agent.ScopeProject, targetRoot)
	assert.NoErr(t, err)
	_, err = service.Install(skill.Skill{ID: "world-skill", QualifiedName: "marketplaces/world-skill", SourceQualifiedName: "repo-a/marketplaces/world-skill", Version: "1.0.0", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: sourceDir}, "codex", agent.ScopeProject, targetRoot)
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
		SkillID:             "hello-skill",
		QualifiedName:       "marketplaces/hello-skill",
		SourceQualifiedName: "repo-a/marketplaces/hello-skill",
		Agent:               "claude-code",
		Scope:               "project",
		InstalledPath:       installedPath,
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
		SkillID:             "hello-skill",
		QualifiedName:       "marketplaces/hello-skill",
		SourceQualifiedName: "repo-a/marketplaces/hello-skill",
		Agent:               "claude-code",
		Scope:               "project",
		InstalledPath:       installedPath,
		SourceID:            "local-demo",
		SourceType:          "local",
		InstallEntry:        "commands",
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
