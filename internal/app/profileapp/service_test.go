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

func TestService_CreateFromCollectionRejectsInvalidSelector(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, cfg.DefaultConfig(), baseDir))

	_, err := NewService(configFile, baseDir).CreateFromCollection("go-dev", "go")

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "collection selector must be <source>/<collection>")
}
