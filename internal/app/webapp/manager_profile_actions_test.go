package webapp

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/profile"
)

func TestManager_PlanProfileSaveCreatesPlan(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)

	plan, err := NewManager(configFile, baseDir).PlanProfileSave(profileSaveReq{
		Name:    "ops",
		Targets: []profile.Target{{Source: "gstack", Skill: "review"}},
	})

	assert.NoErr(t, err)
	assert.Eq(t, "ops", plan.Name)
	assert.Eq(t, "create", plan.Mode)
	assert.Len(t, plan.Added, 1)
}

func TestManager_RunProfileSavePersistsProfile(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)

	result, err := NewManager(configFile, baseDir).RunProfileSave(profileSaveReq{
		Name:    "ops",
		Targets: []profile.Target{{Source: "gstack", Skill: "review"}},
	})

	assert.NoErr(t, err)
	assert.Eq(t, true, result.Saved)
	got, err := NewManager(configFile, baseDir).Profiles()
	assert.NoErr(t, err)
	assert.Len(t, got, 2)
}
