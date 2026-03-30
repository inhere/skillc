package installapp

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/inhere/skillc/internal/domain/agent"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	"github.com/inhere/skillc/internal/infra/agentfs"
	"github.com/inhere/skillc/internal/infra/lockstore"
)

type Service struct {
	lockFile  string
	store     *lockstore.Store
	installer *agentfs.Installer
	now       func() time.Time
}

func NewService(lockFile string) *Service {
	return &Service{
		lockFile:  lockFile,
		store:     lockstore.NewStore(),
		installer: agentfs.NewInstaller(),
		now:       time.Now,
	}
}

func (s *Service) Install(item skill.Skill, agentName string, scope agent.Scope, targetRoot string) (lockpkg.Record, error) {
	targetPath := filepath.Join(targetRoot, item.ID)
	if err := s.installer.Install(filepath.Join(item.Path, item.InstallEntry), targetPath); err != nil {
		return lockpkg.Record{}, err
	}
	now := s.now()
	record := lockpkg.Record{
		SkillID:       item.ID,
		Agent:         agentName,
		Scope:         string(scope),
		Version:       item.Version,
		SourceID:      item.SourceID,
		SourceType:    string(item.SourceType),
		InstalledPath: targetPath,
		InstalledAt:   now,
		UpdatedAt:     now,
	}
	if err := s.store.Save(s.lockFile, []lockpkg.Record{record}); err != nil {
		return lockpkg.Record{}, err
	}
	return record, nil
}

func (s *Service) Uninstall(skillID string, agentName string, scope agent.Scope) error {
	records, err := s.store.Load(s.lockFile)
	if err != nil {
		return err
	}

	kept := make([]lockpkg.Record, 0, len(records))
	for _, record := range records {
		if record.SkillID == skillID && record.Agent == agentName && record.Scope == string(scope) {
			if err := s.installer.Remove(record.InstalledPath); err != nil {
				return err
			}
			continue
		}
		kept = append(kept, record)
	}
	return s.store.Save(s.lockFile, kept)
}

func (s *Service) Restore(sourcePaths map[string]string) ([]lockpkg.Record, error) {
	records, err := s.store.Load(s.lockFile)
	if err != nil {
		return nil, err
	}

	restored := make([]lockpkg.Record, 0, len(records))
	for _, record := range records {
		sourcePath, ok := sourcePaths[record.SourceID]
		if !ok {
			return nil, fmt.Errorf("source not found for restore: %s", record.SourceID)
		}
		if err := s.installer.Install(sourcePath, record.InstalledPath); err != nil {
			return nil, err
		}
		restored = append(restored, record)
	}
	return restored, nil
}
