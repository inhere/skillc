package lock

import "time"

const GlobalKey = "__global__"

type File map[string][]Record

type Record struct {
	SkillID             string    `json:"skill_id"`
	QualifiedName       string    `json:"qualified_name"`
	SourceQualifiedName string    `json:"source_qualified_name"`
	Version             string    `json:"version"`
	SourceID            string    `json:"source_id"`
	SourceType          string    `json:"source_type"`
	Profile             string    `json:"profile,omitempty"`
	InstallEntry        string    `json:"install_entry"`
	Agents              []string  `json:"agents"`
	Checksum            string    `json:"checksum"`
	InstalledAt         time.Time `json:"installed_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	Pinned              bool      `json:"pinned"`
}
