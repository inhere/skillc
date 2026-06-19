package webapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/profile"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

func TestManager_SummaryCountsSourcesProfilesSkillsAndStatus(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)

	result, err := NewManager(configFile, baseDir).Summary(ManagerReq{
		Agent:   "universal",
		Scope:   "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Eq(t, baseDir, result.ProjectPath)
	assert.Eq(t, 1, result.SourceCount)
	assert.Eq(t, 1, result.ProfileCount)
	assert.Eq(t, 2, result.SkillCount)
	assert.Eq(t, 1, result.Status.Outdated)
}

func TestManager_ListSourcesReturnsConfiguredSources(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)

	sources, err := NewManager(configFile, baseDir).Sources()

	assert.NoErr(t, err)
	assert.Len(t, sources, 1)
	assert.Eq(t, "gstack", sources[0].ID)
	assert.Eq(t, sourcepkg.TypeLocal, sources[0].Type)
}

func TestManager_ListProfilesReturnsSavedProfiles(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)

	profiles, err := NewManager(configFile, baseDir).Profiles()

	assert.NoErr(t, err)
	assert.Len(t, profiles, 1)
	assert.Eq(t, "go-dev", profiles[0].Name)
	assert.Eq(t, "Go dev", profiles[0].Description)
}

func TestManager_HistoryFileUsesUserCacheDir(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "project", "skillc.yaml")
	config := cfg.DefaultConfig()
	config.RegistryCacheDir = filepath.Join(baseDir, "cache", "registry")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, filepath.Dir(configFile)))

	got := NewManager(configFile, filepath.Dir(configFile)).historyFile()

	assert.Eq(t, filepath.Join(baseDir, "cache", "skillc-web-history.jsonl"), got)
}

func TestManager_InstallMapReadsLockRecords(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)

	items, err := NewManager(configFile, baseDir).InstallMap()

	assert.NoErr(t, err)
	assert.Len(t, items, 1)
	assert.Eq(t, filepath.Clean(baseDir), items[0].ProjectPath)
	assert.Eq(t, "project", items[0].Scope)
	assert.Eq(t, "universal", items[0].Agent)
	assert.Eq(t, "go-dev", items[0].Profile)
	assert.Eq(t, "gstack/tools/go-pro", items[0].SourceQualifiedName)
}

func TestManager_VersionDriftUsesIndexLatestVersion(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)

	groups, err := NewManager(configFile, baseDir).VersionDrift()

	assert.NoErr(t, err)
	assert.Len(t, groups, 1)
	assert.Eq(t, "go-pro", groups[0].SkillID)
	assert.Eq(t, "gstack/tools/go-pro", groups[0].SourceQualifiedName)
	assert.Eq(t, "2.0.0", groups[0].LatestVersion)
	assert.Len(t, groups[0].Versions, 1)
	assert.Eq(t, "1.0.0", groups[0].Versions[0].Version)
}

func TestManager_PlanProfileApplyReturnsPlan(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)

	plan, err := NewManager(configFile, baseDir).PlanProfileApply("go-dev", ManagerReq{
		Agent:   "universal",
		Scope:   "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Eq(t, "go-dev", plan.Profile)
	assert.Eq(t, "universal", plan.Agent)
	assert.Eq(t, "project", plan.Scope)
	assert.Len(t, plan.Items, 1)
	assert.Eq(t, "skip", plan.Items[0].Action)
}

func TestManager_MissingLockAndIndexAreEmptyForDerivedViews(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	config := cfg.DefaultConfig()
	config.LockFile = filepath.Join(baseDir, "missing.lock.json")
	config.IndexFile = filepath.Join(baseDir, "missing-index.json")
	config.Sources = []sourcepkg.Source{}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))

	manager := NewManager(configFile, baseDir)
	installs, err := manager.InstallMap()
	assert.NoErr(t, err)
	assert.Len(t, installs, 0)

	drift, err := manager.VersionDrift()
	assert.NoErr(t, err)
	assert.Len(t, drift, 0)

	skills, err := manager.Skills("")
	assert.NoErr(t, err)
	assert.Len(t, skills, 0)
}

func writeWebManagerFixture(t *testing.T, baseDir string) (string, cfg.Config) {
	t.Helper()

	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	projectKey := filepath.Clean(baseDir)

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.RegistryCacheDir = filepath.Join(baseDir, "cache", "registry")
	config.AgentTools["universal"] = cfg.AgentToolConfig{
		Dirname:    ".agents",
		ProjectDir: filepath.Join(baseDir, ".agents"),
	}
	config.Sources = []sourcepkg.Source{{
		ID:     "gstack",
		Name:   "gstack",
		Type:   sourcepkg.TypeLocal,
		Path:   filepath.Join(baseDir, "source"),
		Status: "ready",
	}}
	config.Profiles = map[string]profile.Profile{
		"go-dev": {
			Description:  "Go dev",
			DefaultAgent: "universal",
			DefaultScope: "project",
			Targets:      []profile.Target{{Source: "gstack", Skill: "go-pro"}},
		},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))

	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectKey: {{
			SkillID:             "go-pro",
			SourceID:            "gstack",
			QualifiedName:       "tools/go-pro",
			SourceQualifiedName: "gstack/tools/go-pro",
			Version:             "1.0.0",
			Profile:             "go-dev",
			Agents:              []string{"universal"},
		}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{
			ID:                  "go-pro",
			SourceID:            "gstack",
			SourceName:          "gstack",
			Collection:          "tools",
			QualifiedName:       "tools/go-pro",
			SourceQualifiedName: "gstack/tools/go-pro",
			Version:             "2.0.0",
			SourceType:          sourcepkg.TypeLocal,
		},
		{
			ID:                  "review",
			SourceID:            "gstack",
			SourceName:          "gstack",
			Collection:          "tools",
			QualifiedName:       "tools/review",
			SourceQualifiedName: "gstack/tools/review",
			Version:             "1.0.0",
			SourceType:          sourcepkg.TypeLocal,
		},
	}))
	return configFile, config
}
