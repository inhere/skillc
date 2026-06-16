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

type registryResolverStub struct {
	resolveFn func(record lockpkg.Record) (skill.Skill, bool, error)
}

func (s registryResolverStub) Resolve(record lockpkg.Record) (skill.Skill, bool, error) {
	return s.resolveFn(record)
}

func (s registryResolverStub) Latest(record lockpkg.Record) (skill.Skill, bool, error) {
	return s.resolveFn(record)
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

func TestService_RunCarriesSourceQualifiedName(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID:             "go-pro",
			SourceID:            "gstack",
			SourceQualifiedName: "gstack/tools/go-pro",
			Version:             "1.0.0",
			Agents:              []string{"universal"},
		}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", SourceQualifiedName: "gstack/tools/go-pro", Version: "1.0.0"},
	}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Len(t, result.Items, 1)
	assert.Eq(t, "gstack/tools/go-pro", result.Items[0].SourceQualifiedName)
}

func TestService_RunFillsIndexIdentityForMissingSkill(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID:  "go-pro",
			SourceID: "gstack",
			Version:  "1.0.0",
			Agents:   []string{"universal"},
		}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{{
		ID:                  "go-pro",
		SourceID:            "gstack",
		QualifiedName:       "tools/go-pro",
		SourceQualifiedName: "gstack/tools/go-pro",
		Version:             "2.0.0",
	}}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Len(t, result.Items, 1)
	assert.Eq(t, StatusMissing, result.Items[0].Status)
	assert.Eq(t, "tools/go-pro", result.Items[0].QualifiedName)
	assert.Eq(t, "gstack/tools/go-pro", result.Items[0].SourceQualifiedName)
	assert.Eq(t, "2.0.0", result.Items[0].LatestVersion)
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

func TestService_RunProfileFilterExcludesUnmanagedDirectories(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "manual"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", Profile: "go-dev", Agents: []string{"universal"}},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Version: "1.0.0"},
	}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", Profile: "go-dev", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Len(t, result.Items, 1)
	assert.Eq(t, "go-pro", result.Items[0].SkillID)
	assert.Eq(t, 0, result.Summary.Unmanaged)
	for _, item := range result.Items {
		assert.NotEq(t, "manual", item.SkillID)
		assert.NotEq(t, StatusUnmanaged, item.Status)
	}
}

func TestService_RunFiltersByAgent(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "universal-skill"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".codex", "skills", "codex-skill"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	config.AgentTools["codex"] = cfg.AgentToolConfig{Dirname: ".codex", ProjectDir: filepath.Join(baseDir, ".codex")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{SkillID: "universal-skill", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}},
			{SkillID: "codex-skill", SourceID: "gstack", Version: "1.0.0", Agents: []string{"codex"}},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "universal-skill", SourceID: "gstack", Version: "1.0.0"},
		{ID: "codex-skill", SourceID: "gstack", Version: "1.0.0"},
	}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "codex", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Len(t, result.Items, 1)
	assert.Eq(t, "codex-skill", result.Items[0].SkillID)
	assert.Eq(t, "codex", result.Items[0].Agent)
	assert.Eq(t, StatusInstalled, result.Items[0].Status)
	assert.Eq(t, 1, result.Summary.Installed)
}

func TestService_RunFiltersByScope(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	userDir := filepath.Join(baseDir, "user-agents")
	projectDir := filepath.Join(baseDir, ".agents")
	assert.NoErr(t, os.MkdirAll(filepath.Join(userDir, "skills", "global-skill"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(userDir, "skills", "global-manual"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(projectDir, "skills", "project-skill"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(projectDir, "skills", "project-manual"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", UserDir: userDir, ProjectDir: projectDir}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		lockpkg.GlobalKey: {
			{SkillID: "global-skill", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}},
		},
		filepath.Clean(baseDir): {
			{SkillID: "project-skill", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "global-skill", SourceID: "gstack", Version: "1.0.0"},
		{ID: "project-skill", SourceID: "gstack", Version: "1.0.0"},
	}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "user", WorkDir: baseDir})

	assert.NoErr(t, err)
	got := statusBySkill(result.Items)
	assert.Eq(t, "installed", got["global-skill"])
	assert.Eq(t, "unmanaged", got["global-manual"])
	assert.Eq(t, "", got["project-skill"])
	assert.Eq(t, "", got["project-manual"])
	assert.Eq(t, 1, result.Summary.Installed)
	assert.Eq(t, 1, result.Summary.Unmanaged)
}

func TestService_RunMatchesIndexByQualifiedIdentityWhenSourceIDIsMissing(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{
				SkillID:             "go-pro",
				SourceQualifiedName: "gstack/go-pro",
				QualifiedName:       "gstack/go-pro",
				Version:             "1.0.0",
				Agents:              []string{"universal"},
			},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{
			ID:                  "go-pro",
			SourceID:            "gstack",
			SourceQualifiedName: "gstack/go-pro",
			QualifiedName:       "gstack/go-pro",
			Version:             "1.0.0",
		},
	}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Len(t, result.Items, 1)
	assert.Eq(t, "installed", result.Items[0].Status)
	assert.Eq(t, "gstack", result.Items[0].SourceID)
}

func TestService_RunTreatsQualifiedOnlyLockAsOrphanWhenIndexHasAmbiguousSources(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{
				SkillID:       "go-pro",
				QualifiedName: "tools/go-pro",
				Version:       "1.0.0",
				Agents:        []string{"universal"},
			},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{
			ID:                  "go-pro",
			SourceID:            "repo-a",
			SourceQualifiedName: "repo-a/tools/go-pro",
			QualifiedName:       "tools/go-pro",
			Version:             "2.0.0",
		},
		{
			ID:                  "go-pro",
			SourceID:            "repo-b",
			SourceQualifiedName: "repo-b/tools/go-pro",
			QualifiedName:       "tools/go-pro",
			Version:             "3.0.0",
		},
	}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Len(t, result.Items, 1)
	assert.Eq(t, StatusOrphan, result.Items[0].Status)
	assert.Eq(t, "skill not found in source index", result.Items[0].Reason)
	assert.Eq(t, "", result.Items[0].LatestVersion)
	assert.Eq(t, "", result.Items[0].SourceID)
}

func TestService_RunReportsSourceSyncErrorForQualifiedLockWithoutSourceID(t *testing.T) {
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
		filepath.Clean(baseDir): {
			{
				SkillID:             "go-pro",
				SourceQualifiedName: "gstack/go-pro",
				QualifiedName:       "gstack/go-pro",
				Version:             "1.0.0",
				Agents:              []string{"universal"},
			},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{
			ID:                  "go-pro",
			SourceID:            "gstack",
			SourceQualifiedName: "gstack/go-pro",
			QualifiedName:       "gstack/go-pro",
			Version:             "2.0.0",
		},
	}))
	svc := NewService(configFile, baseDir)
	svc.syncer = syncerStub{syncFn: func(id string) error {
		if id == "gstack" {
			return errors.New("sync failed")
		}
		return nil
	}}

	result, err := svc.Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir, Sync: true})

	assert.NoErr(t, err)
	assert.Len(t, result.Items, 1)
	assert.Eq(t, StatusSourceError, result.Items[0].Status)
	assert.Eq(t, "gstack", result.Items[0].SourceID)
	assert.Eq(t, "sync failed", result.Items[0].Reason)
	assert.Eq(t, 1, result.Summary.SourceError)
	assert.Eq(t, 0, result.Summary.Outdated)
}

func TestService_RunReportsSourceSyncErrorWhenSourceQualifiedNameUsesSourceName(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	config.Sources = []sourcepkg.Source{{ID: "git-workflow-repo", Name: "workflow-repo", Type: sourcepkg.TypeGit, Path: filepath.Join(baseDir, "source")}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{
				SkillID:             "go-pro",
				SourceQualifiedName: "workflow-repo/go-pro",
				QualifiedName:       "workflow-repo/go-pro",
				Version:             "1.0.0",
				Agents:              []string{"universal"},
			},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{
			ID:                  "go-pro",
			SourceID:            "git-workflow-repo",
			SourceQualifiedName: "workflow-repo/go-pro",
			QualifiedName:       "workflow-repo/go-pro",
			Version:             "2.0.0",
		},
	}))
	svc := NewService(configFile, baseDir)
	svc.syncer = syncerStub{syncFn: func(id string) error {
		if id == "git-workflow-repo" {
			return errors.New("sync failed")
		}
		return nil
	}}

	result, err := svc.Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir, Sync: true})

	assert.NoErr(t, err)
	assert.Len(t, result.Items, 1)
	assert.Eq(t, StatusSourceError, result.Items[0].Status)
	assert.Eq(t, "git-workflow-repo", result.Items[0].SourceID)
	assert.Eq(t, "sync failed", result.Items[0].Reason)
	assert.Eq(t, 1, result.Summary.SourceError)
	assert.Eq(t, 0, result.Summary.Outdated)
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
	originalLock := lockpkg.File{
		filepath.Clean(baseDir): {{SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}}},
	}
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, originalLock))
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
	assert.Eq(t, "gstack", result.SyncFailed[0].SourceID)
	assert.Eq(t, "source-error", result.Items[0].Status)
	assert.Eq(t, "sync failed", result.Items[0].Reason)
	assert.Eq(t, 1, result.Summary.SourceError)
	gotLock, err := lockstore.NewStore().Load(lockFile)
	assert.NoErr(t, err)
	assert.Eq(t, originalLock, gotLock)
}

func TestService_RunMarksOutdatedWhenGitResolvedRefChanges(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeStatusDriftFixture(t, baseDir)
	projectKey := baseDir
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))
	assert.NoErr(t, lockstore.NewStore().Save(config.LockFile, lockpkg.File{
		projectKey: {{
			SkillID: "go-pro", SourceID: "gstack", SourceType: "git", Version: "1.0.0",
			SourceResolvedRef: "oldcommit", Agents: []string{"universal"},
		}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(config.IndexFile, []skill.Skill{{
		ID: "go-pro", SourceID: "gstack", SourceType: sourcepkg.TypeGit, Version: "1.0.0",
		SourceResolvedRef: "newcommit", Path: filepath.Join(baseDir, "source", "go-pro"),
	}}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Eq(t, StatusOutdated, result.Items[0].Status)
	assert.Contains(t, result.Items[0].Reason, "git ref")
}

func TestService_RunMarksOutdatedWhenLocalChecksumChanges(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeStatusDriftFixture(t, baseDir)
	projectKey := baseDir
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "rules"), 0o755))
	assert.NoErr(t, lockstore.NewStore().Save(config.LockFile, lockpkg.File{
		projectKey: {{
			SkillID: "rules", SourceID: "local", SourceType: "local", Version: "1.0.0",
			Checksum: "oldsum", Agents: []string{"universal"},
		}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(config.IndexFile, []skill.Skill{{
		ID: "rules", SourceID: "local", SourceType: sourcepkg.TypeLocal, Version: "1.0.0",
		Checksum: "newsum", Path: filepath.Join(baseDir, "source", "rules"),
	}}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Eq(t, StatusOutdated, result.Items[0].Status)
	assert.Contains(t, result.Items[0].Reason, "checksum")
}

func TestService_RunMarksRegistrySkillInstalledFromRegistryCache(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeStatusDriftFixture(t, baseDir)
	cacheDir := filepath.Join(baseDir, "registry-cache")
	config.RegistryCacheDir = cacheDir
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))
	assert.NoErr(t, lockstore.NewStore().Save(config.LockFile, lockpkg.File{
		baseDir: {{
			SkillID: "go-pro", SourceID: "team", SourceType: "registry", RegistryEntryID: "go-pro",
			Version: "1.0.0", Checksum: "abc123", Agents: []string{"universal"},
		}},
	}))

	service := NewService(configFile, baseDir)
	service.registryResolver = registryResolverStub{resolveFn: func(record lockpkg.Record) (skill.Skill, bool, error) {
		return skill.Skill{
			ID: "go-pro", SourceID: "team", SourceType: sourcepkg.TypeRegistry,
			RegistryEntryID: "go-pro", Version: "1.0.0", Checksum: "abc123",
			SourceURL: "https://example.com/skills.git", InstallEntry: "skills/go-pro",
		}, true, nil
	}}
	result, err := service.Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Len(t, result.Items, 1)
	assert.Eq(t, StatusInstalled, result.Items[0].Status)
	assert.Eq(t, "team", result.Items[0].SourceID)
}

func writeStatusDriftFixture(t *testing.T, baseDir string) (string, cfg.Config) {
	t.Helper()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	return configFile, config
}

func statusBySkill(items []Item) map[string]string {
	out := map[string]string{}
	for _, item := range items {
		out[item.SkillID] = item.Status
	}
	return out
}
