package webapp

import (
	"encoding/json"
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

func TestManager_ApplyProfileInstallsMissingProfileSkills(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeWebActionFixture(t, baseDir)

	result, err := NewManager(configFile, baseDir).ApplyProfile("go-dev", ManagerReq{
		Agent:   "universal",
		Scope:   "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Eq(t, "go-dev", result.Plan.Profile)
	assert.Len(t, result.Installed, 1)
	assert.Eq(t, "review", result.Installed[0].SkillID)
	assert.Eq(t, "gstack", result.Installed[0].SourceID)
	assert.Eq(t, "universal", result.Installed[0].Agent)
	assert.Eq(t, "project", result.Installed[0].Scope)
	_, err = os.Stat(filepath.Join(baseDir, ".agents", "skills", "review"))
	assert.NoErr(t, err)
}

func TestManager_RunUpdateUpdatesOutdatedSkill(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeWebActionFixture(t, baseDir)

	result, err := NewManager(configFile, baseDir).RunUpdate(WebUpdateReq{
		ManagerReq: ManagerReq{Agent: "universal", Scope: "project", WorkDir: baseDir},
		Target:     "go-pro",
	})

	assert.NoErr(t, err)
	assert.Len(t, result.Updated, 1)
	assert.Eq(t, "go-pro", result.Updated[0].SkillID)
	assert.Eq(t, "2.0.0", result.Updated[0].Version)
	assert.Eq(t, "gstack", result.Updated[0].SourceID)
	data, err := os.ReadFile(filepath.Join(baseDir, ".agents", "skills", "go-pro", "SKILL.md"))
	assert.NoErr(t, err)
	assert.Contains(t, string(data), "updated")
}

func TestActionRuntimeRecordUsesStableJSONFields(t *testing.T) {
	data, err := json.Marshal(updateRunActionResult{
		Updated: []actionRuntimeRecord{{
			SkillID:       "go-pro",
			SourceID:      "gstack",
			Version:       "2.0.0",
			Agent:         "universal",
			Scope:         "project",
			InstalledPath: "project/.agents/skills/go-pro",
		}},
		SyncFailed: []actionSourceErrorItem{{SourceID: "gstack", Reason: "sync failed"}},
	})

	assert.NoErr(t, err)
	body := string(data)
	assert.Contains(t, body, `"skill_id":"go-pro"`)
	assert.Contains(t, body, `"source_id":"gstack"`)
	assert.Contains(t, body, `"installed_path":"project/.agents/skills/go-pro"`)
	assert.Contains(t, body, `"sync_failed":[{"source_id":"gstack","reason":"sync failed"}]`)
	assert.NotContains(t, body, `"SkillID"`)
	assert.NotContains(t, body, `"SourceID"`)
	assert.NotContains(t, body, `"InstalledPath"`)
}

func writeWebActionFixture(t *testing.T, baseDir string) string {
	t.Helper()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	sourceRoot := filepath.Join(baseDir, "source")

	goSource := createWebActionSkillSource(t, sourceRoot, "go-pro", "2.0.0", "updated")
	reviewSource := createWebActionSkillSource(t, sourceRoot, "review", "1.0.0", "review")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.RegistryCacheDir = filepath.Join(baseDir, "cache", "registry")
	config.InstallMode = "copy"
	config.AgentTools["universal"] = cfg.AgentToolConfig{
		Dirname:    ".agents",
		ProjectDir: filepath.Join(baseDir, ".agents"),
	}
	config.Sources = []sourcepkg.Source{{
		ID:     "gstack",
		Name:   "gstack",
		Type:   sourcepkg.TypeLocal,
		Path:   sourceRoot,
		Status: "ready",
	}}
	config.Profiles = map[string]profile.Profile{
		"go-dev": {
			Description:  "Go dev",
			DefaultAgent: "universal",
			DefaultScope: "project",
			Targets:     []profile.Target{{Source: "gstack", Skill: "review"}},
		},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))

	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(baseDir, ".agents", "skills", "go-pro", "SKILL.md"), []byte("# old"), 0o644))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID:             "go-pro",
			SourceID:            "gstack",
			QualifiedName:       "tools/go-pro",
			SourceQualifiedName: "gstack/tools/go-pro",
			Version:             "1.0.0",
			Agents:              []string{"universal"},
		}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Collection: "tools", QualifiedName: "tools/go-pro", SourceQualifiedName: "gstack/tools/go-pro", Version: "2.0.0", SourceType: sourcepkg.TypeLocal, InstallEntry: ".", Path: goSource},
		{ID: "review", SourceID: "gstack", Collection: "tools", QualifiedName: "tools/review", SourceQualifiedName: "gstack/tools/review", Version: "1.0.0", SourceType: sourcepkg.TypeLocal, InstallEntry: ".", Path: reviewSource},
	}))
	return configFile
}

func createWebActionSkillSource(t *testing.T, sourceRoot string, id string, version string, content string) string {
	t.Helper()
	root := filepath.Join(sourceRoot, id)
	assert.NoErr(t, os.MkdirAll(root, 0o755))
	body := "---\n" +
		"id: " + id + "\n" +
		"name: " + id + "\n" +
		"version: " + version + "\n" +
		"---\n\n" +
		"# " + id + "\n" + content
	assert.NoErr(t, os.WriteFile(filepath.Join(root, "SKILL.md"), []byte(body), 0o644))
	return root
}
