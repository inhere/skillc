package statusapp

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

type syncerStub struct {
	syncFn func(id string) error
}

func (s syncerStub) Sync(id string) error {
	return s.syncFn(id)
}

func TestService_RunClassifiesInstalledMissingOutdatedAndOrphan(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	installedPath := filepath.Join(baseDir, ".agents", "skills", "installed")
	outdatedPath := filepath.Join(baseDir, ".agents", "skills", "outdated")
	orphanPath := filepath.Join(baseDir, ".agents", "skills", "orphan")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))
	assert.NoErr(t, os.MkdirAll(outdatedPath, 0o755))
	assert.NoErr(t, os.MkdirAll(orphanPath, 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	config.Sources = []sourcepkg.Source{{ID: "gstack", Type: sourcepkg.TypeLocal, Path: filepath.Join(baseDir, "source")}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{SkillID: "installed", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}},
			{SkillID: "missing", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}},
			{SkillID: "outdated", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}},
			{SkillID: "orphan", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "installed", SourceID: "gstack", Version: "1.0.0"},
		{ID: "missing", SourceID: "gstack", Version: "1.0.0"},
		{ID: "outdated", SourceID: "gstack", Version: "2.0.0"},
	}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	got := statusBySkill(result.Items)
	assert.Eq(t, "installed", got["installed"])
	assert.Eq(t, "missing", got["missing"])
	assert.Eq(t, "outdated", got["outdated"])
	assert.Eq(t, "orphan", got["orphan"])
	assert.Eq(t, 1, result.Summary.Installed)
	assert.Eq(t, 1, result.Summary.Missing)
	assert.Eq(t, 1, result.Summary.Outdated)
	assert.Eq(t, 1, result.Summary.Orphan)
}

func TestService_RunIncludesUnmanagedInstalledDirectories(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "manual"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Len(t, result.Items, 1)
	assert.Eq(t, "manual", result.Items[0].SkillID)
	assert.Eq(t, "unmanaged", result.Items[0].Status)
	assert.Eq(t, 1, result.Summary.Unmanaged)
}

func TestService_RunFiltersByProfile(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "review"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", Profile: "go-dev", Agents: []string{"universal"}},
			{SkillID: "review", SourceID: "gstack", Version: "1.0.0", Profile: "review", Agents: []string{"universal"}},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Version: "1.0.0"},
		{ID: "review", SourceID: "gstack", Version: "1.0.0"},
	}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", Profile: "go-dev", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Len(t, result.Items, 1)
	assert.Eq(t, "go-pro", result.Items[0].SkillID)
}

func TestService_RunReportsSourceSyncErrorsWithoutUpdatingLock(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	config.Sources = []sourcepkg.Source{{ID: "gstack", Type: sourcepkg.TypeLocal, Path: filepath.Join(baseDir, "source")}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{{ID: "go-pro", SourceID: "gstack", Version: "2.0.0", SourceType: sourcepkg.TypeLocal}}))
	svc := NewService(configFile, baseDir)
	svc.syncer = syncerStub{syncFn: func(id string) error {
		if id == "gstack" {
			return errors.New("sync failed")
		}
		return nil
	}}

	result, err := svc.Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir, Sync: true})

	assert.NoErr(t, err)
	assert.Len(t, result.SyncFailed, 1)
	assert.Eq(t, "source-error", result.Items[0].Status)
	assert.Eq(t, "sync failed", result.Items[0].Reason)
}

func statusBySkill(items []Item) map[string]string {
	out := map[string]string{}
	for _, item := range items {
		out[item.SkillID] = item.Status
	}
	return out
}
