package install

import (
	"path/filepath"

	"github.com/inhere/skillc/internal/domain/agent"
	"github.com/inhere/skillc/internal/domain/skill"
)

func BuildPlan(item skill.Skill, agentName string, scope agent.Scope, targetRoot string) (Plan, error) {
	return Plan{
		SkillID:      item.ID,
		Agent:        agentName,
		Scope:        scope,
		SourcePath:   filepath.Join(item.Path, item.InstallEntry),
		InstallEntry: item.InstallEntry,
		TargetRoot:   targetRoot,
		TargetPath:   filepath.Join(targetRoot, item.ID),
		ConflictMode: ConflictModeOverwrite,
	}, nil
}
