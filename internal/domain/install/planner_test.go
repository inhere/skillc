package install

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/agent"
	"github.com/inhere/skillc/internal/domain/skill"
)

func TestBuildPlan_CreatesTargetPathFromSkillAndResolverResult(t *testing.T) {
	baseDir := t.TempDir()
	targetRoot := filepath.Join(baseDir, ".claude", "skills")
	s := skill.Skill{
		ID:           "hello-skill",
		Path:         filepath.Join(baseDir, "hello-skill"),
		InstallEntry: "commands",
	}

	plan, err := BuildPlan(s, "claude-code", agent.ScopeProject, targetRoot)
	assert.NoErr(t, err)
	assert.Eq(t, "hello-skill", plan.SkillID)
	assert.Eq(t, filepath.Join(s.Path, "commands"), plan.SourcePath)
	assert.Eq(t, targetRoot, plan.TargetRoot)
	assert.Eq(t, filepath.Join(targetRoot, "hello-skill"), plan.TargetPath)
	assert.Eq(t, ConflictModeOverwrite, plan.ConflictMode)
}
