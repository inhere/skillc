package webapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/profile"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

func TestManager_PlanSourceAddLocal(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	sourceDir := filepath.Join(baseDir, "team-skills")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "go-pro"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "go-pro", "SKILL.md"), []byte("# Go Pro\n"), 0o644))

	plan, err := NewManager(configFile, baseDir).PlanSourceAdd(sourceActionReq{Value: sourceDir, Sync: true})

	assert.NoErr(t, err)
	assert.Eq(t, "add_local", plan.Action)
	assert.Eq(t, sourcepkg.TypeLocal, plan.Source.Type)
	assert.Eq(t, false, plan.Existing)
	assert.Eq(t, "add", plan.Items[0].Action)
	assert.Eq(t, "sync", plan.Items[1].Action)
}

func TestManager_PlanSourceAddExistingLocal(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	config, err := configstore.NewYAMLStore().Load(configFile, baseDir)
	assert.NoErr(t, err)
	existingPath := config.Sources[0].Path

	plan, err := NewManager(configFile, baseDir).PlanSourceAdd(sourceActionReq{Value: existingPath})

	assert.NoErr(t, err)
	assert.Eq(t, "exists", plan.Action)
	assert.Eq(t, true, plan.Existing)
	assert.Eq(t, config.Sources[0].ID, plan.Source.ID)
}

func TestManager_PlanSourceRemoveIncludesImpact(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	config, err := configstore.NewYAMLStore().Load(configFile, baseDir)
	assert.NoErr(t, err)
	config.Profiles["ops"] = profile.Profile{Targets: []profile.Target{{Source: "gstack", Skill: "review"}}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(config.LockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID:  "review",
			SourceID: "gstack",
			Version:  "1.0.0",
			Agents:   []string{"universal"},
		}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(config.IndexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Collection: "go"},
		{ID: "review", SourceID: "gstack", Collection: "ops"},
	}))

	plan, err := NewManager(configFile, baseDir).PlanSourceRemove(sourceActionReq{ID: "gstack"})

	assert.NoErr(t, err)
	assert.Eq(t, "remove", plan.Action)
	assert.Eq(t, "gstack", plan.SourceID)
	assert.Eq(t, 1, plan.Impact.InstalledCount)
	assert.Eq(t, 2, plan.Impact.ProfileTargetCount)
	assert.Eq(t, 2, plan.Impact.IndexedSkillCount)
	assert.Eq(t, 2, plan.Impact.CollectionCount)
	assert.Contains(t, plan.Warnings[0], "installed lock")
}

func TestManager_RunSourceAddAddsAndSyncsLocalSource(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, "source"), 0o755))
	sourceDir := filepath.Join(baseDir, "team-skills")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "hello"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "hello", "SKILL.md"), []byte("# Hello\n"), 0o644))

	result, err := NewManager(configFile, baseDir).RunSourceAdd(sourceActionReq{Value: sourceDir, Sync: true})

	assert.NoErr(t, err)
	assert.Eq(t, "add_local", result.Plan.Action)
	assert.Eq(t, true, result.Synced)
	assert.Eq(t, "", result.Error)
}

func TestManager_RunSourceRemoveRemovesSourceAndReturnsPlan(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)

	result, err := NewManager(configFile, baseDir).RunSourceRemove(sourceActionReq{ID: "gstack"})

	assert.NoErr(t, err)
	assert.Eq(t, "remove", result.Plan.Action)
	assert.Eq(t, "gstack", result.Plan.SourceID)
	sources, err := NewManager(configFile, baseDir).Sources()
	assert.NoErr(t, err)
	assert.Len(t, sources, 0)
}
