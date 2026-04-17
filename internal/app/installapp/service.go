package installapp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/inhere/skillc/internal/app/apputil"
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

type RuntimeRecord struct {
	lockpkg.Record
	Agent         string
	Scope         string
	InstalledPath string
}

type CommandResult struct {
	Installed     []RuntimeRecord
	Restored      []RuntimeRecord
	ResolveFailed []searchapp.TargetError
	InstallFailed []InstallItemError
}

type InstallItemError struct {
	SkillID string
	Reason  string
}

type BatchInstallResult struct {
	Installed []RuntimeRecord
	Failed    []InstallItemError
}

type Service struct {
	lockFile  string
	store     *lockstore.Store
	installer *agentfs.Installer
	now       func() time.Time
	config    cfg.Config
	workDir   string
}

func NewService(lockFile string) *Service {
	return &Service{
		lockFile:  lockFile,
		store:     lockstore.NewStore(),
		installer: agentfs.NewInstaller(),
		now:       time.Now,
	}
}

func (s *Service) WithRuntime(config cfg.Config, workDir string) *Service {
	clone := *s
	clone.config = config
	clone.workDir = workDir
	return &clone
}

type InstallReq struct {
	SkillID string
	Agent   string
	Scope   string
	WorkDir string
}

// Run installs skills. 通过 skill id 搜索并安装技能
func (s *Service) Run(config cfg.Config, req InstallReq, lookup skillLookup) (CommandResult, error) {
	runtimeSvc := s.WithRuntime(config, req.WorkDir)
	if req.SkillID == "" {
		restored, err := runtimeSvc.Restore(sourcePathMap(config))
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
	return runtimeSvc.RunResolved(config, req, items, nil)
}

func (s *Service) RunResolved(config cfg.Config, req InstallReq, items []skill.Skill, resolveFailed []searchapp.TargetError) (CommandResult, error) {
	runtimeSvc := s.WithRuntime(config, req.WorkDir)
	scope, err := parseScope(req.Scope)
	if err != nil {
		return CommandResult{}, err
	}
	scopeKey, err := resolveScopeKey(scope, req.WorkDir)
	if err != nil {
		return CommandResult{}, err
	}
	targetRoot, err := agent.ResolveInstallPath(config, req.WorkDir, req.Agent, scope)
	if err != nil {
		return CommandResult{}, err
	}

	installResult, err := runtimeSvc.InstallMulti(items, req.Agent, scope, scopeKey, targetRoot)
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
func (s *Service) InstallMulti(items []skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetRoot string) (BatchInstallResult, error) {
	result := BatchInstallResult{
		Installed: make([]RuntimeRecord, 0, len(items)),
		Failed:    make([]InstallItemError, 0),
	}
	for _, item := range items {
		record, err := s.Install(item, agentName, scope, scopeKey, targetRoot)
		if err != nil {
			result.Failed = append(result.Failed, InstallItemError{SkillID: item.ID, Reason: err.Error()})
			continue
		}
		result.Installed = append(result.Installed, record)
	}
	return result, nil
}

// Install installs a skill.
func (s *Service) Install(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetRoot string) (RuntimeRecord, error) {
	locks, err := s.loadLockFile()
	if err != nil {
		return RuntimeRecord{}, err
	}

	records := append([]lockpkg.Record(nil), locks[scopeKey]...)
	now := s.now()
	record := lockpkg.Record{
		SkillID:             item.ID,
		QualifiedName:       item.QualifiedName,
		SourceQualifiedName: item.SourceQualifiedName,
		Version:             item.Version,
		SourceID:            item.SourceID,
		SourceType:          string(item.SourceType),
		InstallEntry:        item.InstallEntry,
		Agents:              []string{agentName},
		InstalledAt:         now,
		UpdatedAt:           now,
	}
	if conflict, ok := findConflictingSkillSource(records, record); ok {
		return RuntimeRecord{}, fmt.Errorf("skill already installed from another source: %s (%s)", item.ID, conflict.SourceQualifiedName)
	}

	targetPath := installTargetPath(item, targetRoot)
	if err := s.installer.Install(filepath.Join(item.Path, item.InstallEntry), targetPath); err != nil {
		return RuntimeRecord{}, err
	}

	records, record = upsertRecord(records, record)
	if len(records) == 0 {
		delete(locks, scopeKey)
	} else {
		locks[scopeKey] = records
	}
	if err := s.store.Save(s.lockFile, locks); err != nil {
		return RuntimeRecord{}, err
	}
	return newRuntimeRecord(record, agentName, scope, targetPath), nil
}

func (s *Service) ReinstallAtPath(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (RuntimeRecord, error) {
	locks, err := s.loadLockFile()
	if err != nil {
		return RuntimeRecord{}, err
	}

	records := append([]lockpkg.Record(nil), locks[scopeKey]...)
	if err := s.installer.Install(filepath.Join(item.Path, item.InstallEntry), targetPath); err != nil {
		return RuntimeRecord{}, err
	}

	now := s.now()
	record := lockpkg.Record{
		SkillID:             item.ID,
		QualifiedName:       item.QualifiedName,
		SourceQualifiedName: item.SourceQualifiedName,
		Version:             item.Version,
		SourceID:            item.SourceID,
		SourceType:          string(item.SourceType),
		InstallEntry:        item.InstallEntry,
		Agents:              []string{agentName},
		InstalledAt:         now,
		UpdatedAt:           now,
	}

	records, record = upsertRecord(records, record)
	if len(records) == 0 {
		delete(locks, scopeKey)
	} else {
		locks[scopeKey] = records
	}
	if err := s.store.Save(s.lockFile, locks); err != nil {
		return RuntimeRecord{}, err
	}
	return newRuntimeRecord(record, agentName, scope, targetPath), nil
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
	locks, err := s.loadLockFile()
	if err != nil {
		return err
	}
	if len(locks) == 0 {
		return fmt.Errorf("skill not found: %s", skillID)
	}

	scopeKeys := s.matchScopeKeys(locks, scope)
	updated := false
	for _, scopeKey := range scopeKeys {
		records := locks[scopeKey]
		nextRecords := make([]lockpkg.Record, 0, len(records))
		for _, record := range records {
			if !matchesSkillTarget(record, skillID) {
				nextRecords = append(nextRecords, record)
				continue
			}
			if !containsAgent(record.Agents, agentName) {
				nextRecords = append(nextRecords, record)
				continue
			}

			targetPath, err := s.resolveInstalledPath(scopeKey, scope, agentName, record)
			if err != nil {
				return err
			}
			if err := s.installer.Remove(targetPath); err != nil {
				return err
			}

			updated = true
			record.Agents = removeAgent(record.Agents, agentName)
			if len(record.Agents) == 0 {
				continue
			}
			record.UpdatedAt = s.now()
			nextRecords = append(nextRecords, record)
		}
		if len(nextRecords) == 0 {
			delete(locks, scopeKey)
			continue
		}
		locks[scopeKey] = nextRecords
	}
	if !updated {
		return fmt.Errorf("skill not found: %s", skillID)
	}
	return s.store.Save(s.lockFile, locks)
}

func (s *Service) Restore(sourcePaths map[string]string) ([]RuntimeRecord, error) {
	locks, err := s.loadLockFile()
	if err != nil {
		return nil, err
	}
	if len(locks) == 0 {
		return []RuntimeRecord{}, nil
	}

	scopeKeys := make([]string, 0, len(locks))
	for scopeKey := range locks {
		scopeKeys = append(scopeKeys, scopeKey)
	}
	sort.Strings(scopeKeys)

	restored := make([]RuntimeRecord, 0)
	for _, scopeKey := range scopeKeys {
		scope := scopeFromKey(scopeKey)
		for _, record := range locks[scopeKey] {
			sourceRoot, ok := sourcePaths[record.SourceID]
			if !ok {
				return nil, fmt.Errorf("source path not found for %s", record.SourceID)
			}
			sourcePath := filepath.Join(sourceRoot, record.InstallEntry)
			for _, agentName := range record.Agents {
				targetPath, err := s.resolveInstalledPath(scopeKey, scope, agentName, record)
				if err != nil {
					return nil, err
				}
				if err := s.installer.Install(sourcePath, targetPath); err != nil {
					return nil, err
				}
				restored = append(restored, newRuntimeRecord(record, agentName, scope, targetPath))
			}
		}
	}
	return restored, nil
}

func (s *Service) loadLockFile() (lockpkg.File, error) {
	locks, err := s.store.Load(s.lockFile)
	if err == nil {
		if locks == nil {
			return lockpkg.File{}, nil
		}
		return locks, nil
	}
	if os.IsNotExist(err) {
		return lockpkg.File{}, nil
	}
	return nil, err
}

func (s *Service) matchScopeKeys(locks lockpkg.File, scope agent.Scope) []string {
	if scope == agent.ScopeUser {
		if _, ok := locks[lockpkg.GlobalKey]; ok {
			return []string{lockpkg.GlobalKey}
		}
		return nil
	}
	workDir := s.runtimeWorkDir()
	if workDir == "" {
		return nil
	}
	scopeKey, err := resolveScopeKey(scope, workDir)
	if err != nil {
		return nil
	}
	if _, ok := locks[scopeKey]; !ok {
		return nil
	}
	return []string{scopeKey}
}


func (s *Service) resolveInstalledPath(scopeKey string, scope agent.Scope, agentName string, record lockpkg.Record) (string, error) {
	baseDir := s.runtimeWorkDir()
	if scope == agent.ScopeProject {
		baseDir = scopeKey
	}
	targetRoot, err := agent.ResolveInstallPath(s.runtimeConfig(), baseDir, agentName, scope)
	if err != nil {
		return "", err
	}
	flatPath := filepath.Join(targetRoot, record.SkillID)
	legacyPath := filepath.Join(targetRoot, apputil.LegacyInstallDir(record.SkillID, record.SourceQualifiedName, record.SourceID))
	return apputil.PreferExistingInstallPath(flatPath, legacyPath)
}

func (s *Service) runtimeConfig() cfg.Config {
	if len(s.config.AgentTools) != 0 {
		return s.config
	}
	return cfg.DefaultConfig()
}

func (s *Service) runtimeWorkDir() string {
	if strings.TrimSpace(s.workDir) != "" {
		return s.workDir
	}
	cwd, err := os.Getwd()
	if err == nil && strings.TrimSpace(cwd) != "" {
		return cwd
	}
	return filepath.Dir(s.lockFile)
}

func newRuntimeRecord(record lockpkg.Record, agentName string, scope agent.Scope, targetPath string) RuntimeRecord {
	return RuntimeRecord{
		Record:        record,
		Agent:         agentName,
		Scope:         string(scope),
		InstalledPath: targetPath,
	}
}

func upsertRecord(records []lockpkg.Record, next lockpkg.Record) ([]lockpkg.Record, lockpkg.Record) {
	for i, record := range records {
		if sameInstallIdentity(record, next) {
			next.InstalledAt = record.InstalledAt
			next.Agents = mergeAgents(record.Agents, next.Agents)
			records[i] = next
			return records, next
		}
	}
	next.Agents = mergeAgents(nil, next.Agents)
	return append(records, next), next
}

func sameInstallIdentity(current lockpkg.Record, next lockpkg.Record) bool {
	if current.SkillID != next.SkillID {
		return false
	}
	if current.SourceID != "" || next.SourceID != "" {
		return current.SourceID != "" && current.SourceID == next.SourceID
	}
	if current.SourceQualifiedName != "" || next.SourceQualifiedName != "" {
		return current.SourceQualifiedName != "" && current.SourceQualifiedName == next.SourceQualifiedName
	}
	return current.QualifiedName == next.QualifiedName
}

func installTargetPath(item skill.Skill, targetRoot string) string {
	return filepath.Join(targetRoot, item.ID)
}

func findConflictingSkillSource(records []lockpkg.Record, next lockpkg.Record) (lockpkg.Record, bool) {
	for _, record := range records {
		if record.SkillID != next.SkillID {
			continue
		}
		if sameInstallIdentity(record, next) {
			return lockpkg.Record{}, false
		}
		return record, true
	}
	return lockpkg.Record{}, false
}

func legacyInstallDir(skillID string, sourceQualifiedName string, sourceID string) string {
	return apputil.LegacyInstallDir(skillID, sourceQualifiedName, sourceID)
}

func preferExistingInstallPath(flatPath string, legacyPath string) (string, error) {
	return apputil.PreferExistingInstallPath(flatPath, legacyPath)
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
	return apputil.ParseScope(value)
}

func resolveScopeKey(scope agent.Scope, workDir string) (string, error) {
	return apputil.ResolveScopeKey(scope, workDir)
}

func scopeFromKey(scopeKey string) agent.Scope {
	return apputil.ScopeFromKey(scopeKey)
}

func matchesSkillTarget(record lockpkg.Record, target string) bool {
	return record.SkillID == target || record.QualifiedName == target || record.SourceQualifiedName == target
}

func containsAgent(agents []string, agentName string) bool {
	for _, current := range agents {
		if current == agentName {
			return true
		}
	}
	return false
}

func removeAgent(agents []string, agentName string) []string {
	next := make([]string, 0, len(agents))
	for _, current := range agents {
		if current == agentName {
			continue
		}
		next = append(next, current)
	}
	return next
}

func mergeAgents(existing []string, incoming []string) []string {
	set := make(map[string]struct{}, len(existing)+len(incoming))
	merged := make([]string, 0, len(existing)+len(incoming))
	for _, agentName := range append(append([]string(nil), existing...), incoming...) {
		if agentName == "" {
			continue
		}
		if _, ok := set[agentName]; ok {
			continue
		}
		set[agentName] = struct{}{}
		merged = append(merged, agentName)
	}
	sort.Strings(merged)
	return merged
}
