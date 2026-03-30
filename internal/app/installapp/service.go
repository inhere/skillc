package installapp

import (
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
