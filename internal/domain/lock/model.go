package lock

import "time"

type Record struct {
	SkillID       string    `json:"skill_id"`
	Agent         string    `json:"agent"`
	Scope         string    `json:"scope"`
	Version       string    `json:"version"`
	SourceID      string    `json:"source_id"`
	SourceType    string    `json:"source_type"`
	InstalledPath string    `json:"installed_path"`
	Checksum      string    `json:"checksum"`
	InstalledAt   time.Time `json:"installed_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Pinned        bool      `json:"pinned"`
}
