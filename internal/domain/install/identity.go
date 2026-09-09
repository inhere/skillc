package install

import (
	"github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
)

// SameIdentity compares an installed record with an indexed skill using the
// strongest stable source identity available, with compatibility fallbacks.
func SameIdentity(record lock.Record, item skill.Skill) bool {
	if record.SkillID != item.ID {
		return false
	}
	if record.SourceID != "" || item.SourceID != "" {
		return record.SourceID != "" && record.SourceID == item.SourceID
	}
	if record.SourceQualifiedName != "" || item.SourceQualifiedName != "" {
		return record.SourceQualifiedName != "" && record.SourceQualifiedName == item.SourceQualifiedName
	}
	return record.QualifiedName != "" && record.QualifiedName == item.QualifiedName
}
