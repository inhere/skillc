package registryapp

import (
	"strings"

	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
)

type RecordResolver interface {
	Resolve(record lockpkg.Record) (skill.Skill, bool, error)
	Latest(record lockpkg.Record) (skill.Skill, bool, error)
}

type LockedResolver struct {
	service *Service
}

func NewLockedResolver(configFile string, baseDir string) *LockedResolver {
	return &LockedResolver{service: NewService(configFile, baseDir)}
}

func (r *LockedResolver) Resolve(record lockpkg.Record) (skill.Skill, bool, error) {
	if record.SourceType != string(sourcepkg.TypeRegistry) {
		return skill.Skill{}, false, nil
	}
	item, err := r.service.MaterializeSkill(recordSelector(record))
	return item, true, err
}

func (r *LockedResolver) Latest(record lockpkg.Record) (skill.Skill, bool, error) {
	if record.SourceType != string(sourcepkg.TypeRegistry) {
		return skill.Skill{}, false, nil
	}
	entry, err := r.service.InfoSkill(recordSelector(record))
	if err != nil {
		return skill.Skill{}, true, err
	}
	item, err := newMaterializer(nil).skillFromEntry(entry, "")
	return item, true, err
}

func recordSelector(record lockpkg.Record) string {
	entryID := strings.TrimSpace(record.RegistryEntryID)
	if entryID == "" {
		entryID = strings.TrimSpace(record.SkillID)
	}
	sourceID := strings.TrimSpace(record.SourceID)
	if sourceID == "" {
		return entryID
	}
	return sourceID + "/" + entryID
}
