package profileapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/profile"
	"github.com/inhere/skillc/internal/domain/skill"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

func TestService_ListAndShowProfiles(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	config := cfg.DefaultConfig()
	config.Profiles = map[string]profile.Profile{
		"review": {Description: "Review", Targets: []profile.Target{{Source: "gstack", Skill: "review"}}},
		"go-dev": {Description: "Go dev", Targets: []profile.Target{{Source: "gstack", Skill: "go-pro"}}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))

	svc := NewService(configFile, baseDir)
	list, err := svc.List()

	assert.NoErr(t, err)
	assert.Len(t, list, 2)
	assert.Eq(t, "go-dev", list[0].Name)
	assert.Eq(t, "review", list[1].Name)

	got, err := svc.Show("go-dev")
	assert.NoErr(t, err)
	assert.Eq(t, "Go dev", got.Description)
}

func TestService_ShowMissingProfileReturnsError(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, cfg.DefaultConfig(), baseDir))

	svc := NewService(configFile, baseDir)
	_, err := svc.Show("missing")

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "profile not found: missing")
}

func TestService_SaveProfileNormalizesTargets(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, cfg.DefaultConfig(), baseDir))

	svc := NewService(configFile, baseDir)
	err := svc.Save("go-dev", profile.Profile{
		Description: "Go dev",
		Targets: []profile.Target{
			{Source: "gstack", Skill: "review"},
			{Source: "gstack", Skill: "go-pro"},
			{Source: "gstack", Skill: "review"},
		},
	})
	assert.NoErr(t, err)

	got, err := svc.Show("go-dev")
	assert.NoErr(t, err)
	assert.Eq(t, "Go dev", got.Description)
	assert.Len(t, got.Targets, 2)
	assert.Eq(t, "go-pro", got.Targets[0].Skill)
	assert.Eq(t, "review", got.Targets[1].Skill)
}

func TestService_CreateProfileRejectsExistingName(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	config := cfg.DefaultConfig()
	config.Profiles = map[string]profile.Profile{
		"go-dev": {Description: "Existing", Targets: []profile.Target{{Source: "gstack", Skill: "go-pro"}}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))

	svc := NewService(configFile, baseDir)
	got, err := svc.Create("new-dev", profile.Profile{
		Description: "New dev",
		Targets:     []profile.Target{{Source: "gstack", Skill: "review"}},
	})
	assert.NoErr(t, err)
	assert.Eq(t, "New dev", got.Description)

	_, err = svc.Create("go-dev", profile.Profile{
		Description: "Overwrite",
		Targets:     []profile.Target{{Source: "gstack", Skill: "review"}},
	})
	assert.Err(t, err)
	assert.Contains(t, err.Error(), "profile already exists: go-dev")

	existing, err := svc.Show("go-dev")
	assert.NoErr(t, err)
	assert.Eq(t, "Existing", existing.Description)
}

func TestService_CreateFromInstalled(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	installedPath := filepath.Join(baseDir, ".agents", "skills", "go-pro")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID:  "go-pro",
			SourceID: "gstack",
			Agents:   []string{"universal"},
		}},
	}))

	svc := NewService(configFile, baseDir)
	got, err := svc.CreateFromInstalled("go-dev", CreateFromInstalledReq{
		Agent:   "universal",
		Scope:   "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Eq(t, "universal", got.DefaultAgent)
	assert.Eq(t, "project", got.DefaultScope)
	assert.Len(t, got.Targets, 1)
	assert.Eq(t, "gstack", got.Targets[0].Source)
	assert.Eq(t, "go-pro", got.Targets[0].Skill)
}

func TestService_CreateFromInstalledSkipsMissingLockRecords(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	installedPath := filepath.Join(baseDir, ".agents", "skills", "go-pro")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{
				SkillID:  "go-pro",
				SourceID: "gstack",
				Agents:   []string{"universal"},
			},
			{
				SkillID:  "missing",
				SourceID: "gstack",
				Agents:   []string{"universal"},
			},
		},
	}))

	got, err := NewService(configFile, baseDir).CreateFromInstalled("go-dev", CreateFromInstalledReq{
		Agent:   "universal",
		Scope:   "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Len(t, got.Targets, 1)
	assert.Eq(t, "go-pro", got.Targets[0].Skill)
}

func TestService_CreateFromInstalledRejectsInvalidRuntimeOptions(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))

	tests := []struct {
		name    string
		req     CreateFromInstalledReq
		wantErr string
	}{
		{name: "unknown agent", req: CreateFromInstalledReq{Agent: "missing-agent", Scope: "project", WorkDir: baseDir}, wantErr: "unsupported agent: missing-agent"},
		{name: "invalid scope", req: CreateFromInstalledReq{Agent: "universal", Scope: "projcet", WorkDir: baseDir}, wantErr: "unsupported scope: projcet"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewService(configFile, baseDir).CreateFromInstalled("go-dev", tt.req)

			if err == nil {
				t.Fatalf("CreateFromInstalled() error = nil, want %q", tt.wantErr)
			}
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestService_CreateFromCollection(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Collection: "go"},
		{ID: "review", SourceID: "gstack", Collection: "go"},
		{ID: "python-pro", SourceID: "gstack", Collection: "python"},
	}))

	svc := NewService(configFile, baseDir)
	got, err := svc.CreateFromCollection("go-dev", "gstack/go")

	assert.NoErr(t, err)
	assert.Len(t, got.Targets, 2)
	assert.Eq(t, "gstack", got.Targets[0].Source)
	assert.Eq(t, "go-pro", got.Targets[0].Skill)
	assert.Eq(t, "review", got.Targets[1].Skill)
}

func TestService_BuildFromCollectionReturnsUnsavedProfile(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Collection: "go"},
		{ID: "review", SourceID: "gstack", Collection: "ops"},
	}))

	got, err := NewService(configFile, baseDir).BuildFromCollection("gstack/go")

	assert.NoErr(t, err)
	assert.Len(t, got.Targets, 1)
	assert.Eq(t, "gstack", got.Targets[0].Source)
	assert.Eq(t, "go-pro", got.Targets[0].Skill)
	list, err := NewService(configFile, baseDir).List()
	assert.NoErr(t, err)
	assert.Len(t, list, 0)
}

func TestService_PlanSaveReportsAddedRemovedKeptTargets(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	config := cfg.DefaultConfig()
	config.Profiles = map[string]profile.Profile{
		"go-dev": {
			Targets: []profile.Target{
				{Source: "gstack", Skill: "go-pro"},
				{Source: "gstack", Skill: "old"},
			},
		},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))

	plan, err := NewService(configFile, baseDir).PlanSave("go-dev", profile.Profile{
		Targets: []profile.Target{
			{Source: "gstack", Skill: "go-pro"},
			{Source: "gstack", Skill: "review"},
		},
	})

	assert.NoErr(t, err)
	assert.Eq(t, "edit", plan.Mode)
	assert.Len(t, plan.Added, 1)
	assert.Eq(t, "review", plan.Added[0].Skill)
	assert.Len(t, plan.Removed, 1)
	assert.Eq(t, "old", plan.Removed[0].Skill)
	assert.Len(t, plan.Kept, 1)
}

func TestService_CreateFromCollectionRejectsInvalidSelector(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, cfg.DefaultConfig(), baseDir))

	_, err := NewService(configFile, baseDir).CreateFromCollection("go-dev", "go")

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "collection selector must be <source>/<collection>")
}

func TestService_PlanApplySkipsInstalledAndInstallsMissing(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	installedPath := filepath.Join(baseDir, ".agents", "skills", "go-pro")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))

	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.LockFile = lockFile
	config.Profiles = map[string]profile.Profile{
		"go-dev": {Targets: []profile.Target{
			{Source: "gstack", Skill: "go-pro"},
			{Source: "gstack", Skill: "review"},
		}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack"},
		{ID: "review", SourceID: "gstack"},
	}))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID:  "go-pro",
			SourceID: "gstack",
			Agents:   []string{"universal"},
		}},
	}))

	plan, err := NewService(configFile, baseDir).PlanApply("go-dev", ApplyReq{
		Agent:   "universal",
		Scope:   "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Len(t, plan.Items, 2)
	assert.Eq(t, "skip", plan.Items[0].Action)
	assert.Eq(t, "install", plan.Items[1].Action)
}

func TestService_PlanApplyTreatsMissingLockRecordAsInstall(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")

	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.LockFile = lockFile
	config.Profiles = map[string]profile.Profile{
		"go-dev": {Targets: []profile.Target{{Source: "gstack", Skill: "go-pro"}}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack"},
	}))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID:  "go-pro",
			SourceID: "gstack",
			Agents:   []string{"universal"},
		}},
	}))

	plan, err := NewService(configFile, baseDir).PlanApply("go-dev", ApplyReq{
		Agent:   "universal",
		Scope:   "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Len(t, plan.Items, 1)
	assert.Eq(t, "install", plan.Items[0].Action)
}

func TestService_PlanApplyReportsMissingTarget(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")

	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.Profiles = map[string]profile.Profile{
		"go-dev": {Targets: []profile.Target{{Source: "gstack", Skill: "missing"}}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack"},
	}))

	plan, err := NewService(configFile, baseDir).PlanApply("go-dev", ApplyReq{
		Agent:   "universal",
		Scope:   "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Len(t, plan.Items, 1)
	assert.Eq(t, "error", plan.Items[0].Action)
	assert.Contains(t, plan.Items[0].Reason, "skill not found in index")
}

func TestService_PlanApplyReportsAmbiguousUnqualifiedTarget(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")

	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.Profiles = map[string]profile.Profile{
		"go-dev": {Targets: []profile.Target{{Skill: "shared-skill"}}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "shared-skill", SourceID: "repo-a", QualifiedName: "alpha/shared-skill", SourceQualifiedName: "repo-a/alpha/shared-skill"},
		{ID: "shared-skill", SourceID: "repo-b", QualifiedName: "beta/shared-skill", SourceQualifiedName: "repo-b/beta/shared-skill"},
	}))

	plan, err := NewService(configFile, baseDir).PlanApply("go-dev", ApplyReq{
		Agent:   "universal",
		Scope:   "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Len(t, plan.Items, 1)
	assert.Eq(t, "error", plan.Items[0].Action)
	assert.Contains(t, plan.Items[0].Reason, "ambiguous skill target")
}

func TestService_PlanApplyResolvesAgentAlias(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")

	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{
		Dirname:    ".claude",
		Aliases:    []string{"claude"},
		ProjectDir: filepath.Join(baseDir, ".claude"),
	}
	config.Profiles = map[string]profile.Profile{
		"go-dev": {Targets: []profile.Target{{Source: "gstack", Skill: "review"}}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "review", SourceID: "gstack"},
	}))

	plan, err := NewService(configFile, baseDir).PlanApply("go-dev", ApplyReq{
		Agent:   "claude",
		Scope:   "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Eq(t, "claude-code", plan.Agent)
	assert.Eq(t, "project", plan.Scope)
}

func TestService_ApplyRefusesPlanWithErrors(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	sourceDir := filepath.Join(baseDir, "source", "review")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Review"), 0o644))

	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.LockFile = lockFile
	config.Profiles = map[string]profile.Profile{
		"go-dev": {Targets: []profile.Target{
			{Source: "gstack", Skill: "review"},
			{Source: "gstack", Skill: "missing"},
		}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "review", SourceID: "gstack", InstallEntry: ".", Path: sourceDir},
	}))

	_, err := NewService(configFile, baseDir).Apply("go-dev", ApplyReq{
		Agent:   "universal",
		Scope:   "project",
		WorkDir: baseDir,
	})

	if err == nil {
		t.Fatalf("Apply() error = nil, want plan error")
	}
	assert.Contains(t, err.Error(), "profile apply plan has errors")
	_, statErr := os.Stat(filepath.Join(baseDir, ".agents", "skills", "review"))
	assert.True(t, os.IsNotExist(statErr))
	records, loadErr := lockstore.NewStore().Load(lockFile)
	if loadErr == nil {
		assert.Len(t, records, 0)
		return
	}
	assert.True(t, os.IsNotExist(loadErr))
}

func TestService_ApplyInstallsMissingSkillsWithProfile(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	sourceDir := filepath.Join(baseDir, "source", "review")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Review"), 0o644))

	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.LockFile = lockFile
	config.Profiles = map[string]profile.Profile{
		"go-dev": {Targets: []profile.Target{{Source: "gstack", Skill: "review"}}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "review", SourceID: "gstack", InstallEntry: ".", Path: sourceDir},
	}))

	result, err := NewService(configFile, baseDir).Apply("go-dev", ApplyReq{
		Agent:   "universal",
		Scope:   "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Len(t, result.Installed, 1)
	records, err := lockstore.NewStore().Load(lockFile)
	assert.NoErr(t, err)
	assert.Eq(t, "go-dev", records[filepath.Clean(baseDir)][0].Profile)
}

func TestService_ApplyUsesProfileInstallMode(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	sourceDir := filepath.Join(baseDir, "source", "review")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Review"), 0o644))

	config := cfg.DefaultConfig()
	config.InstallMode = "junction"
	config.IndexFile = indexFile
	config.LockFile = lockFile
	config.Profiles = map[string]profile.Profile{
		"go-dev": {
			InstallMode: "copy",
			Targets:     []profile.Target{{Source: "gstack", Skill: "review"}},
		},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "review", SourceID: "gstack", InstallEntry: ".", Path: sourceDir},
	}))

	_, err := NewService(configFile, baseDir).Apply("go-dev", ApplyReq{
		Agent:   "universal",
		Scope:   "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Changed"), 0o644))
	data, err := os.ReadFile(filepath.Join(baseDir, ".agents", "skills", "review", "SKILL.md"))
	assert.NoErr(t, err)
	assert.Eq(t, "# Review", string(data))
}

func TestService_ApplyReportsInstallFailures(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	sourceDir := filepath.Join(baseDir, "source", "review")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Review"), 0o644))
	missingDir := filepath.Join(baseDir, "source", "missing")

	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.LockFile = lockFile
	config.Profiles = map[string]profile.Profile{
		"go-dev": {Targets: []profile.Target{
			{Source: "gstack", Skill: "broken"},
			{Source: "gstack", Skill: "review"},
		}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "review", SourceID: "gstack", InstallEntry: ".", Path: sourceDir},
		{ID: "broken", SourceID: "gstack", InstallEntry: ".", Path: missingDir},
	}))

	result, err := NewService(configFile, baseDir).Apply("go-dev", ApplyReq{
		Agent:   "universal",
		Scope:   "project",
		WorkDir: baseDir,
	})

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "profile apply failed")
	assert.Len(t, result.Installed, 1)
	assert.Len(t, result.InstallFailed, 1)
	assert.Eq(t, "broken", result.InstallFailed[0].SkillID)
}

func TestService_ApplyRejectsInvalidProfileInstallMode(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	sourceDir := filepath.Join(baseDir, "source", "review")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Review"), 0o644))

	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.LockFile = lockFile
	config.Profiles = map[string]profile.Profile{
		"go-dev": {
			InstallMode: "bad",
			Targets:     []profile.Target{{Source: "gstack", Skill: "review"}},
		},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "review", SourceID: "gstack", InstallEntry: ".", Path: sourceDir},
	}))

	_, err := NewService(configFile, baseDir).Apply("go-dev", ApplyReq{
		Agent:   "universal",
		Scope:   "project",
		WorkDir: baseDir,
	})

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "invalid profile install_mode")
	_, statErr := os.Stat(filepath.Join(baseDir, ".agents", "skills", "review"))
	assert.True(t, os.IsNotExist(statErr))
}
