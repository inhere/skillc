package webapp

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestManager_PlanUninstall(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeWebActionFixture(t, baseDir)

	plan, err := NewManager(configFile, baseDir).PlanUninstall(uninstallActionReq{
		Skills: []string{"go-pro"},
		Agent:  "universal",
		Scope:  "project",
	})

	assert.NoErr(t, err)
	assert.Eq(t, "universal", plan.Agent)
	assert.Eq(t, "project", plan.Scope)
	assert.Len(t, plan.Items, 1)
	assert.Eq(t, "go-pro", plan.Items[0].SkillID)
}

func TestManager_RunUninstallReturnsResult(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeWebActionFixture(t, baseDir)

	result, err := NewManager(configFile, baseDir).RunUninstall(uninstallActionReq{
		Skills: []string{"go-pro"},
		Agent:  "universal",
		Scope:  "project",
	})

	assert.NoErr(t, err)
	assert.Len(t, result.Removed, 1)
	assert.Eq(t, "go-pro", result.Removed[0].SkillID)
}
