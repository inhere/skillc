package install

import "github.com/inhere/skillc/internal/domain/agent"

type ConflictMode string

const (
	ConflictModeOverwrite ConflictMode = "overwrite"
)

type Plan struct {
	SkillID      string
	Agent        string
	Scope        agent.Scope
	SourcePath   string
	InstallEntry string
	TargetRoot   string
	TargetPath   string
	ConflictMode ConflictMode
}

type Conflict struct {
	Exists bool
	Mode   ConflictMode
}
