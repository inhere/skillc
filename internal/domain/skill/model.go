package skill

import sourcepkg "github.com/inhere/skillc/internal/domain/source"

type Skill struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	Version             string `json:"version"`
	SupportedAgents     []string `json:"supported_agents,omitempty"`
	SourceID            string `json:"source_id,omitempty"`
	SourceName          string `json:"source_name,omitempty"`
	SourceType          sourcepkg.Type `json:"source_type,omitempty"`
	Collection          string `json:"collection,omitempty"`
	QualifiedName       string `json:"qualified_name,omitempty"`
	SourceQualifiedName string `json:"source_qualified_name,omitempty"`
	InstallEntry        string `json:"install_entry,omitempty"`
	Path                string `json:"path,omitempty"`
}
