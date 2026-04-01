package skill

import sourcepkg "github.com/inhere/skillc/internal/domain/source"

type Skill struct {
	ID                  string
	Name                string
	Description         string
	Version             string
	SupportedAgents     []string
	SourceID            string
	SourceName          string
	SourceType          sourcepkg.Type
	Collection          string
	QualifiedName       string
	SourceQualifiedName string
	InstallEntry        string
	Path                string
}
