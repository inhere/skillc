package skill

import sourcepkg "github.com/inhere/skillc/internal/domain/source"

type Skill struct {
	ID              string
	Name            string
	Description     string
	Version         string
	SupportedAgents []string
	SourceID        string
	SourceType      sourcepkg.Type
	InstallEntry    string
	Path            string
}
