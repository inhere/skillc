package projectupdateapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/project"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

func TestService_PlanUsesRegisteredProjectsOnly(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeProjectUpdateFixture(t, baseDir)

	plan, err := NewService(configFile, baseDir).Plan(Req{Agent: "universal", Scope: "project", Sync: false})

	assert.NoErr(t, err)
	assert.Eq(t, "universal", plan.Agent)
	assert.Eq(t, "project", plan.Scope)
	assert.Len(t, plan.Projects, 2)
	assert.Eq(t, config.Projects[0].ID, plan.Projects[0].ProjectID)
	assert.Len(t, plan.Projects[0].Items, 1)
	assert.Eq(t, "go-pro", plan.Projects[0].Items[0].SkillID)
	assert.Eq(t, "outdated", plan.Projects[0].Items[0].Status)
}

func TestService_PlanFiltersSelectedProjectsAndTarget(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeProjectUpdateFixture(t, baseDir)

	plan, err := NewService(configFile, baseDir).Plan(Req{Agent: "universal", Scope: "project", ProjectIDs: []string{"demo"}, Target: "review", Sync: false})

	assert.NoErr(t, err)
	assert.Len(t, plan.Projects, 1)
	assert.Eq(t, "demo", plan.Projects[0].ProjectID)
	assert.Len(t, plan.Projects[0].Items, 1)
	assert.Eq(t, "review", plan.Projects[0].Items[0].SkillID)
}

func TestService_RunRequiresConfirmation(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeProjectUpdateFixture(t, baseDir)

	_, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", Sync: false})

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "confirmation required")
}

func TestService_RunExecutesConfirmedCandidateUpdates(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeProjectUpdateFixture(t, baseDir)

	result, err := NewService(configFile, baseDir).Run(Req{
		Agent:      "universal",
		Scope:      "project",
		ProjectIDs: []string{"skillc"},
		Confirm:    true,
		Sync:       true,
	})

	assert.NoErr(t, err)
	assert.Len(t, result.Results, 1)
	assert.Eq(t, "skillc", result.Results[0].ProjectID)
	assert.Len(t, result.Results[0].Updated, 1)
	assert.Eq(t, "go-pro", result.Results[0].Updated[0].SkillID)
	assert.Eq(t, "2.0.0", result.Results[0].Updated[0].Version)
}

func writeProjectUpdateFixture(t *testing.T, baseDir string) (string, cfg.Config) {
	t.Helper()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	sourceDir := filepath.Join(baseDir, "source")
	projectA := filepath.Join(baseDir, "skillc")
	projectB := filepath.Join(baseDir, "demo")
	assert.NoErr(t, os.MkdirAll(filepath.Join(projectA, ".agents", "skills", "go-pro"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(projectB, ".agents", "skills", "review"), 0o755))
	writeProjectUpdateSkill(t, sourceDir, "go-pro", "Go Pro", "2.0.0")
	writeProjectUpdateSkill(t, sourceDir, "review", "Review", "2.0.0")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.Sources = []sourcepkg.Source{{
		ID:     "gstack",
		Name:   "gstack",
		Type:   sourcepkg.TypeLocal,
		Path:   sourceDir,
		Status: "ready",
	}}
	config.Projects = []project.Project{
		{ID: "skillc", Name: "Skillc", Path: projectA},
		{ID: "demo", Name: "Demo", Path: projectB},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectA: {{SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}}},
		projectB: {{SkillID: "review", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Version: "2.0.0", SourceType: sourcepkg.TypeLocal, Path: filepath.Join(sourceDir, "go-pro")},
		{ID: "review", SourceID: "gstack", Version: "2.0.0", SourceType: sourcepkg.TypeLocal, Path: filepath.Join(sourceDir, "review")},
	}))
	return configFile, config
}

func writeProjectUpdateSkill(t *testing.T, sourceDir string, id string, name string, version string) {
	t.Helper()
	skillDir := filepath.Join(sourceDir, id)
	assert.NoErr(t, os.MkdirAll(skillDir, 0o755))
	content := "---\nid: " + id + "\nname: " + name + "\nversion: " + version + "\n---\n# " + name + "\n"
	assert.NoErr(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644))
}
