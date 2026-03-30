package installapp

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	"github.com/inhere/skillc/internal/infra/agentfs"
	"github.com/inhere/skillc/internal/infra/lockstore"
)

type skillLookup interface {
	Show(id string) (skill.Skill, error)
}

type CommandResult struct {
	Installed *lockpkg.Record
	Restored  []lockpkg.Record
}

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

func (s *Service) Run(config cfg.Config, workingDir string, args []string, lookup skillLookup) (CommandResult, error) {
	if len(args) == 0 {
		restored, err := s.Restore(sourcePathMap(config))
		if err != nil {
			return CommandResult{}, err
		}
		return CommandResult{Restored: restored}, nil
	}
	if len(args) < 3 {
		return CommandResult{}, fmt.Errorf("skill id, agent, and scope are required")
	}
	if lookup == nil {
		return CommandResult{}, fmt.Errorf("skill lookup is required")
	}

	scope, err := parseScope(args[2])
	if err != nil {
		return CommandResult{}, err
	}
	item, err := lookup.Show(args[0])
	if err != nil {
		return CommandResult{}, err
	}
	targetRoot, err := agent.ResolveInstallPath(config, workingDir, args[1], scope)
	if err != nil {
		return CommandResult{}, err
	}
	record, err := s.Install(item, args[1], scope, targetRoot)
	if err != nil {
		return CommandResult{}, err
	}
	return CommandResult{Installed: &record}, nil
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
		InstallEntry:  item.InstallEntry,
		InstalledPath: targetPath,
		InstalledAt:   now,
		UpdatedAt:     now,
	}

	records, err := s.loadRecords()
	if err != nil {
		return lockpkg.Record{}, err
	}
	records = upsertRecord(records, record)
	if err := s.store.Save(s.lockFile, records); err != nil {
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
		installSourcePath := sourcePath
		if record.InstallEntry != "" {
			installSourcePath = filepath.Join(sourcePath, record.InstallEntry)
		}
		if err := s.installer.Install(installSourcePath, record.InstalledPath); err != nil {
			return nil, err
		}
		restored = append(restored, record)
	}
	return restored, nil
}

func (s *Service) loadRecords() ([]lockpkg.Record, error) {
	records, err := s.store.Load(s.lockFile)
	if err == nil {
		return records, nil
	}
	if os.IsNotExist(err) {
		return nil, nil
	}
	return nil, err
}

func upsertRecord(records []lockpkg.Record, next lockpkg.Record) []lockpkg.Record {
	for i, record := range records {
		if record.SkillID == next.SkillID && record.Agent == next.Agent && record.Scope == next.Scope {
			next.InstalledAt = record.InstalledAt
			records[i] = next
			return records
		}
	}
	return append(records, next)
}

func sourcePathMap(config cfg.Config) map[string]string {
	paths := make(map[string]string, len(config.Sources))
	for _, src := range config.Sources {
		if src.Path == "" {
			continue
		}
		paths[src.ID] = src.Path
	}
	return paths
}

func parseScope(value string) (agent.Scope, error) {
	scope := agent.Scope(value)
	switch scope {
	case agent.ScopeUser, agent.ScopeProject:
		return scope, nil
	default:
		return "", fmt.Errorf("unsupported scope: %s", value)
	}
}
