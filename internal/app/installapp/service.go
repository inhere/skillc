package installapp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/inhere/skillc/internal/app/searchapp"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	"github.com/inhere/skillc/internal/infra/agentfs"
	"github.com/inhere/skillc/internal/infra/lockstore"
)

type skillLookup interface {
	Resolve(target string) ([]skill.Skill, error)
}

type CommandResult struct {
	Installed     []lockpkg.Record
	Restored      []lockpkg.Record
	ResolveFailed []searchapp.TargetError
	InstallFailed []InstallItemError
}

type InstallItemError struct {
	SkillID string
	Reason  string
}

type BatchInstallResult struct {
	Installed []lockpkg.Record
	Failed    []InstallItemError
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

type InstallReq struct {
	SkillID string
	Agent   string
	Scope   string
	WorkDir string
}

// Run installs skills. 通过 skill id 搜索并安装技能
func (s *Service) Run(config cfg.Config, req InstallReq, lookup skillLookup) (CommandResult, error) {
	// 从 lock file 恢复所有技能
	if req.SkillID == "" {
		restored, err := s.Restore(sourcePathMap(config))
		if err != nil {
			return CommandResult{}, err
		}
		return CommandResult{Restored: restored}, nil
	}

	if lookup == nil {
		return CommandResult{}, fmt.Errorf("skill lookup is required")
	}

	items, err := lookup.Resolve(req.SkillID)
	if err != nil {
		return CommandResult{}, err
	}
	return s.RunResolved(config, req, items, nil)
}

func (s *Service) RunResolved(config cfg.Config, req InstallReq, items []skill.Skill, resolveFailed []searchapp.TargetError) (CommandResult, error) {
	scope, err := parseScope(req.Scope)
	if err != nil {
		return CommandResult{}, err
	}
	targetRoot, err := agent.ResolveInstallPath(config, req.WorkDir, req.Agent, scope)
	if err != nil {
		return CommandResult{}, err
	}

	installResult, err := s.InstallMulti(items, req.Agent, scope, targetRoot)
	if err != nil {
		return CommandResult{}, err
	}
	return CommandResult{
		Installed:     installResult.Installed,
		ResolveFailed: resolveFailed,
		InstallFailed: installResult.Failed,
	}, nil
}

// InstallMulti installs multiple skills.
func (s *Service) InstallMulti(items []skill.Skill, agentName string, scope agent.Scope, targetRoot string) (BatchInstallResult, error) {
	result := BatchInstallResult{
		Installed: make([]lockpkg.Record, 0, len(items)),
		Failed:    make([]InstallItemError, 0),
	}
	for _, item := range items {
		record, err := s.Install(item, agentName, scope, targetRoot)
		if err != nil {
			result.Failed = append(result.Failed, InstallItemError{SkillID: item.ID, Reason: err.Error()})
			continue
		}
		result.Installed = append(result.Installed, record)
	}
	return result, nil
}

// Install installs a skill.
func (s *Service) Install(item skill.Skill, agentName string, scope agent.Scope, targetRoot string) (lockpkg.Record, error) {
	records, err := s.loadRecords()
	if err != nil {
		return lockpkg.Record{}, err
	}

	targetPath := installTargetPath(records, item, agentName, scope, targetRoot)
	if err := s.installer.Install(filepath.Join(item.Path, item.InstallEntry), targetPath); err != nil {
		return lockpkg.Record{}, err
	}

	now := s.now()
	record := lockpkg.Record{
		SkillID:             item.ID,
		QualifiedName:       item.QualifiedName,
		SourceQualifiedName: item.SourceQualifiedName,
		Agent:               agentName,
		Scope:               string(scope),
		Version:             item.Version,
		SourceID:            item.SourceID,
		SourceType:          string(item.SourceType),
		InstallEntry:        item.InstallEntry,
		InstalledPath:       targetPath,
		InstalledAt:         now,
		UpdatedAt:           now,
	}
	record = preserveInstalledAt(records, record)

	records = upsertRecord(records, record)
	if err := s.store.Save(s.lockFile, records); err != nil {
		return lockpkg.Record{}, err
	}
	return record, nil
}

func (s *Service) ReinstallAtPath(item skill.Skill, agentName string, scope agent.Scope, targetPath string) (lockpkg.Record, error) {
	records, err := s.loadRecords()
	if err != nil {
		return lockpkg.Record{}, err
	}

	if err := s.installer.Install(filepath.Join(item.Path, item.InstallEntry), targetPath); err != nil {
		return lockpkg.Record{}, err
	}

	now := s.now()
	record := lockpkg.Record{
		SkillID:             item.ID,
		QualifiedName:       item.QualifiedName,
		SourceQualifiedName: item.SourceQualifiedName,
		Agent:               agentName,
		Scope:               string(scope),
		Version:             item.Version,
		SourceID:            item.SourceID,
		SourceType:          string(item.SourceType),
		InstallEntry:        item.InstallEntry,
		InstalledPath:       targetPath,
		InstalledAt:         now,
		UpdatedAt:           now,
	}
	record = preserveInstalledAt(records, record)

	records = upsertRecord(records, record)
	if err := s.store.Save(s.lockFile, records); err != nil {
		return lockpkg.Record{}, err
	}
	return record, nil
}


// UninstallMulti uninstalls multiple skills.
func (s *Service) UninstallMulti(skillIDs []string, agentName string, scope agent.Scope) error {
	for _, skillID := range skillIDs {
		if err := s.Uninstall(skillID, agentName, scope); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Uninstall(skillID string, agentName string, scope agent.Scope) error {
	records, err := s.store.Load(s.lockFile)
	if err != nil {
		return err
	}

	matches, err := matchingRecords(records, skillID, agentName, string(scope))
	if err != nil {
		return err
	}

	kept := make([]lockpkg.Record, 0, len(records))
	for i, record := range records {
		if _, ok := matches[i]; ok {
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

func preserveInstalledAt(records []lockpkg.Record, next lockpkg.Record) lockpkg.Record {
	for _, record := range records {
		if sameInstallIdentity(record, next) {
			next.InstalledAt = record.InstalledAt
			return next
		}
	}
	return next
}

func upsertRecord(records []lockpkg.Record, next lockpkg.Record) []lockpkg.Record {
	for i, record := range records {
		if sameInstallIdentity(record, next) {
			next.InstalledAt = record.InstalledAt
			records[i] = next
			return records
		}
	}
	return append(records, next)
}

func sameInstallIdentity(current lockpkg.Record, next lockpkg.Record) bool {
	if current.Agent != next.Agent || current.Scope != next.Scope || current.SkillID != next.SkillID {
		return false
	}
	if current.SourceQualifiedName != "" || next.SourceQualifiedName != "" {
		return current.SourceQualifiedName == next.SourceQualifiedName
	}
	if current.SourceID != "" || next.SourceID != "" {
		return current.SourceID == next.SourceID
	}
	return current.QualifiedName == next.QualifiedName
}

func installTargetPath(records []lockpkg.Record, item skill.Skill, agentName string, scope agent.Scope, targetRoot string) string {
	fallback := filepath.Join(targetRoot, item.ID)
	for _, record := range records {
		if sameInstallIdentity(record, lockpkg.Record{
			SkillID:             item.ID,
			QualifiedName:       item.QualifiedName,
			SourceQualifiedName: item.SourceQualifiedName,
			SourceID:            item.SourceID,
			Agent:               agentName,
			Scope:               string(scope),
		}) {
			if record.InstalledPath != "" {
				return record.InstalledPath
			}
			break
		}
	}
	if needsSourceScopedPath(records, item, agentName, scope) {
		return filepath.Join(targetRoot, sourceScopedInstallDir(item))
	}
	return fallback
}

func needsSourceScopedPath(records []lockpkg.Record, item skill.Skill, agentName string, scope agent.Scope) bool {
	for _, record := range records {
		if record.Agent != agentName || record.Scope != string(scope) || record.SkillID != item.ID {
			continue
		}
		if !sameInstallIdentity(record, lockpkg.Record{
			SkillID:             item.ID,
			QualifiedName:       item.QualifiedName,
			SourceQualifiedName: item.SourceQualifiedName,
			SourceID:            item.SourceID,
			Agent:               agentName,
			Scope:               string(scope),
		}) {
			return true
		}
	}
	return false
}

func sourceScopedInstallDir(item skill.Skill) string {
	if item.SourceQualifiedName != "" {
		return strings.ReplaceAll(item.SourceQualifiedName, "/", "--")
	}
	if item.SourceID != "" {
		return item.SourceID + "--" + item.ID
	}
	return item.ID
}

func matchingRecords(records []lockpkg.Record, target string, agentName string, scope string) (map[int]struct{}, error) {
	exact := make(map[int]struct{})
	for i, record := range records {
		if record.Agent != agentName || record.Scope != scope {
			continue
		}
		if record.SkillID == target || record.QualifiedName == target || record.SourceQualifiedName == target {
			exact[i] = struct{}{}
		}
	}
	if len(exact) > 0 {
		return exact, nil
	}

	matches := make(map[int]struct{})
	if strings.Contains(target, "/") {
		prefix := target + "/"
		for i, record := range records {
			if record.Agent != agentName || record.Scope != scope {
				continue
			}
			if strings.HasPrefix(record.SourceQualifiedName, prefix) {
				matches[i] = struct{}{}
			}
		}
	} else {
		sources := make(map[string]struct{})
		for i, record := range records {
			if record.Agent != agentName || record.Scope != scope {
				continue
			}
			if strings.HasPrefix(record.QualifiedName, target+"/") {
				matches[i] = struct{}{}
				sourceKey := record.SourceID
				if sourceKey == "" && record.SourceQualifiedName != "" {
					sourceKey = strings.SplitN(record.SourceQualifiedName, "/", 2)[0]
				}
				sources[sourceKey] = struct{}{}
			}
		}
		if len(matches) > 0 && len(sources) > 1 {
			return nil, fmt.Errorf("ambiguous collection target: %s; use source/collection", target)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("skill not found: %s", target)
	}
	return matches, nil
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
