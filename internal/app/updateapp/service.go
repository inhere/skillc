package updateapp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/inhere/skillc/internal/app/configapp"
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/sourceapp"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

type sourceSyncer interface {
	Sync(id string) error
}

type reinstallService interface {
	ReinstallAtPath(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error)
}

type sourceServiceFactory func(configFile string, baseDir string) sourceSyncer

type installServiceFactory func(lockFile string) reinstallService

type UpdateReq struct {
	Target  string
	Agent   string
	Scope   string
	All     bool
	WorkDir string
}

type Req = UpdateReq

type InstalledItem struct {
	lockpkg.Record
	Agent         string
	Scope         string
	ScopeKey      string
	InstalledPath string
	FromLock      bool
}

type Candidate struct {
	Installed InstalledItem
	Latest    skill.Skill
}

type SourceSyncError struct {
	SourceID string
	Reason   string
}

type UpdateItemError struct {
	SkillID string
	Reason  string
}

type FailedItem struct {
	SkillID string
	Reason  string
}

type SkippedItem struct {
	SkillID string
	Reason  string
}

type Result struct {
	Candidates    []Candidate
	Updated       []installapp.RuntimeRecord
	SyncFailed    []SourceSyncError
	UpdateFailed  []UpdateItemError
	CleanupFailed []FailedItem
	Failed        []FailedItem
	Skipped       []SkippedItem
}

type Service struct {
	configFile     string
	baseDir        string
	configService  *configapp.Service
	lockStore      *lockstore.Store
	indexStore     *repoindex.Store
	syncer         sourceSyncer
	newInstaller   func(lockFile string) reinstallService
	sourceFactory  sourceServiceFactory
	installFactory installServiceFactory
	removeAll      func(path string) error
}

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile:    configFile,
		baseDir:       baseDir,
		configService: configapp.NewService(configFile, baseDir),
		lockStore:     lockstore.NewStore(),
		indexStore:    repoindex.NewStore(),
		syncer:        sourceapp.NewService(configFile, baseDir),
		newInstaller: func(lockFile string) reinstallService {
			return installapp.NewService(lockFile)
		},
		sourceFactory: func(configFile string, baseDir string) sourceSyncer {
			return sourceapp.NewService(configFile, baseDir)
		},
		installFactory: func(lockFile string) reinstallService {
			return installapp.NewService(lockFile)
		},
		removeAll: os.RemoveAll,
	}
}

func (s *Service) Run(req UpdateReq) (Result, error) {
	config, err := s.configService.Show()
	if err != nil {
		return Result{}, err
	}

	scope, err := parseScope(req.Scope)
	if err != nil {
		return Result{}, err
	}
	if req.WorkDir == "" {
		req.WorkDir = s.baseDir
	}

	selected, skipped, err := s.collectSelected(config, req, scope)
	if err != nil {
		return Result{}, err
	}
	if len(selected) == 0 {
		return Result{Skipped: skipped}, nil
	}

	result := Result{Skipped: skipped}
	result.SyncFailed = s.syncSources(selected)

	items, err := s.loadIndex(config.IndexFile)
	if err != nil {
		return Result{}, err
	}

	result.Candidates, result.UpdateFailed = collectCandidates(selected, items, result.SyncFailed)
	result.Failed = append(result.Failed, failedFromSync(selected, result.SyncFailed)...)
	for _, item := range result.UpdateFailed {
		result.Failed = append(result.Failed, FailedItem{SkillID: item.SkillID, Reason: item.Reason})
	}

		installer := s.installFactory
	if s.newInstaller != nil {
		installer = s.newInstaller
	}
	worker := installer(config.LockFile)
	removeAll := s.removeAll
	if removeAll == nil {
		removeAll = os.RemoveAll
	}
	for _, candidate := range result.Candidates {
		oldPath := candidate.Installed.InstalledPath
		targetPath, err := resolveLatestInstalledPath(config, req.WorkDir, candidate.Installed.ScopeKey, candidate.Installed.Agent, agent.Scope(candidate.Installed.Scope), candidate.Latest)
		if err != nil {
			result.UpdateFailed = append(result.UpdateFailed, UpdateItemError{SkillID: candidate.Installed.SkillID, Reason: err.Error()})
			result.Failed = append(result.Failed, FailedItem{SkillID: candidate.Installed.SkillID, Reason: err.Error()})
			continue
		}
		removeOldPath := oldPath != targetPath
		record, err := worker.ReinstallAtPath(
			candidate.Latest,
			candidate.Installed.Agent,
			agent.Scope(candidate.Installed.Scope),
			candidate.Installed.ScopeKey,
			targetPath,
		)
		if err != nil {
			result.UpdateFailed = append(result.UpdateFailed, UpdateItemError{SkillID: candidate.Installed.SkillID, Reason: err.Error()})
			result.Failed = append(result.Failed, FailedItem{SkillID: candidate.Installed.SkillID, Reason: err.Error()})
			continue
		}
		result.Updated = append(result.Updated, record)
		if removeOldPath {
			if err := removeAll(oldPath); err != nil {
				result.CleanupFailed = append(result.CleanupFailed, FailedItem{SkillID: candidate.Installed.SkillID, Reason: err.Error()})
			}
		}
	}
	return result, nil
}

func (s *Service) collectSelected(config cfg.Config, req UpdateReq, scope agent.Scope) ([]InstalledItem, []SkippedItem, error) {
	records, err := s.loadRecords(config.LockFile)
	if err != nil {
		return nil, nil, err
	}
	if len(records) > 0 {
		return selectRecords(config, req.WorkDir, records, req.Target, req.Agent, string(scope), req.All)
	}
	return s.collectFromInstalledDirs(config, req.WorkDir, req.Agent, scope)
}

func (s *Service) loadRecords(path string) (lockpkg.File, error) {
	records, err := s.lockStore.Load(path)
	if err == nil {
		return records, nil
	}
	if os.IsNotExist(err) {
		return nil, nil
	}
	return nil, err
}

func selectRecords(config cfg.Config, workDir string, records lockpkg.File, target string, agentName string, scope string, all bool) ([]InstalledItem, []SkippedItem, error) {
	selected := make([]InstalledItem, 0)
	skipped := make([]SkippedItem, 0)
	for _, scopeKey := range sortedScopeKeys(records) {
		recordScope := scopeFromKey(scopeKey)
		if scope != "" && string(recordScope) != scope {
			continue
		}
		for _, record := range records[scopeKey] {
			filteredAgents := filterAgents(record.Agents, agentName)
			if len(filteredAgents) == 0 {
				continue
			}
			if !all && target != "" && !matchesRecordTarget(record, target) {
				continue
			}
			if record.Pinned {
				skipped = append(skipped, SkippedItem{SkillID: record.SkillID, Reason: "pinned"})
				continue
			}
			for _, currentAgent := range filteredAgents {
				installedPath, err := resolveInstalledPath(config, workDir, scopeKey, currentAgent, recordScope, record)
				if err != nil {
					return nil, nil, err
				}
				selected = append(selected, InstalledItem{
					Record:        record,
					Agent:         currentAgent,
					Scope:         string(recordScope),
					ScopeKey:      scopeKey,
					InstalledPath: installedPath,
					FromLock:      true,
				})
			}
		}
	}
	if !all && target != "" && len(selected) == 0 && len(skipped) == 0 {
		return nil, nil, fmt.Errorf("skill not found: %s", target)
	}
	return selected, skipped, nil
}

func matchesRecordTarget(record lockpkg.Record, target string) bool {
	return record.SkillID == target || record.QualifiedName == target || record.SourceQualifiedName == target
}

func (s *Service) collectFromInstalledDirs(config cfg.Config, workDir, agentName string, scope agent.Scope) ([]InstalledItem, []SkippedItem, error) {
	targetRoot, err := agent.ResolveInstallPath(config, workDir, agentName, scope)
	if err != nil {
		return nil, nil, err
	}
	entries, err := os.ReadDir(targetRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	items, err := s.loadIndex(config.IndexFile)
	if err != nil {
		return nil, nil, err
	}
	scopeKey, err := resolveScopeKey(scope, workDir)
	if err != nil {
		return nil, nil, err
	}
	entryNames := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			entryNames[entry.Name()] = struct{}{}
		}
	}

	selected := make([]InstalledItem, 0, len(entries))
	skipped := make([]SkippedItem, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		matches := matchInstalledDir(items, entry.Name(), entryNames)
		switch len(matches) {
		case 0:
			skipped = append(skipped, SkippedItem{SkillID: entry.Name(), Reason: "index match not found"})
		case 1:
			selected = append(selected, InstalledItem{
				Record: lockpkg.Record{
					SkillID:             matches[0].ID,
					QualifiedName:       matches[0].QualifiedName,
					SourceQualifiedName: matches[0].SourceQualifiedName,
					SourceID:            matches[0].SourceID,
					SourceType:          string(matches[0].SourceType),
					InstallEntry:        matches[0].InstallEntry,
				},
				Agent:         agentName,
				Scope:         string(scope),
				ScopeKey:      scopeKey,
				InstalledPath: filepath.Join(targetRoot, entry.Name()),
				FromLock:      false,
			})
		default:
			skipped = append(skipped, SkippedItem{SkillID: entry.Name(), Reason: "ambiguous index match"})
		}
	}
	return selected, skipped, nil
}

func (s *Service) syncSources(records []InstalledItem) []SourceSyncError {
	svc := s.syncer
	if svc == nil {
		svc = s.sourceFactory(s.configFile, s.baseDir)
	}
	ids := uniqueSourceIDs(records)
	failed := make([]SourceSyncError, 0)
	for _, id := range ids {
		if err := svc.Sync(id); err != nil {
			failed = append(failed, SourceSyncError{SourceID: id, Reason: err.Error()})
		}
	}
	return failed
}

func uniqueSourceIDs(records []InstalledItem) []string {
	seen := make(map[string]struct{}, len(records))
	ids := make([]string, 0, len(records))
	for _, record := range records {
		if record.SourceID == "" {
			continue
		}
		if _, ok := seen[record.SourceID]; ok {
			continue
		}
		seen[record.SourceID] = struct{}{}
		ids = append(ids, record.SourceID)
	}
	sort.Strings(ids)
	return ids
}

func (s *Service) loadIndex(path string) ([]skill.Skill, error) {
	items, err := s.indexStore.Load(path)
	if err == nil {
		return items, nil
	}
	if os.IsNotExist(err) {
		return []skill.Skill{}, nil
	}
	return nil, err
}

func collectCandidates(records []InstalledItem, items []skill.Skill, syncFailed []SourceSyncError) ([]Candidate, []UpdateItemError) {
	failedSources := make(map[string]struct{}, len(syncFailed))
	for _, item := range syncFailed {
		failedSources[item.SourceID] = struct{}{}
	}

	candidates := make([]Candidate, 0, len(records))
	failed := make([]UpdateItemError, 0)
	for _, record := range records {
		if _, ok := failedSources[record.SourceID]; ok {
			continue
		}
		latest, ok := findLatest(items, record.Record)
		if !ok {
			failed = append(failed, UpdateItemError{SkillID: record.SkillID, Reason: fmt.Sprintf("installed skill not found in source index: %s", record.SkillID)})
			continue
		}
		candidates = append(candidates, Candidate{Installed: record, Latest: latest})
	}
	return candidates, failed
}

func findLatest(items []skill.Skill, record lockpkg.Record) (skill.Skill, bool) {
	for _, item := range items {
		if sameCandidateIdentity(record, item) {
			return item, true
		}
	}
	return skill.Skill{}, false
}

func sameCandidateIdentity(record lockpkg.Record, item skill.Skill) bool {
	if record.SkillID != item.ID {
		return false
	}
	if record.SourceID != "" || item.SourceID != "" {
		return record.SourceID != "" && record.SourceID == item.SourceID
	}
	if record.SourceQualifiedName != "" || item.SourceQualifiedName != "" {
		return record.SourceQualifiedName != "" && record.SourceQualifiedName == item.SourceQualifiedName
	}
	if record.QualifiedName != "" || item.QualifiedName != "" {
		return record.QualifiedName != "" && record.QualifiedName == item.QualifiedName
	}
	return false
}

func legacyInstallDir(skillID string, sourceQualifiedName string, sourceID string) string {
	if sourceQualifiedName != "" {
		return strings.ReplaceAll(sourceQualifiedName, "/", "--")
	}
	if sourceID != "" {
		return sourceID + "--" + skillID
	}
	return skillID
}

func matchInstalledDir(items []skill.Skill, dirName string, _ map[string]struct{}) []skill.Skill {
	matches := make([]skill.Skill, 0)
	for _, item := range items {
		if item.ID == dirName || legacyInstallDir(item.ID, item.SourceQualifiedName, item.SourceID) == dirName {
			matches = append(matches, item)
		}
	}
	return matches
}

func preferExistingInstallPath(flatPath string, legacyPath string) (string, error) {
	if _, err := os.Stat(flatPath); err == nil {
		return flatPath, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if legacyPath == flatPath {
		return flatPath, nil
	}
	if _, err := os.Stat(legacyPath); err == nil {
		return legacyPath, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return flatPath, nil
}

func failedFromSync(records []InstalledItem, syncFailed []SourceSyncError) []FailedItem {
	if len(syncFailed) == 0 {
		return nil
	}
	failedBySource := make(map[string]string, len(syncFailed))
	for _, item := range syncFailed {
		failedBySource[item.SourceID] = item.Reason
	}
	failed := make([]FailedItem, 0)
	for _, record := range records {
		if reason, ok := failedBySource[record.SourceID]; ok {
			failed = append(failed, FailedItem{SkillID: record.SkillID, Reason: fmt.Sprintf("source sync failed: %s", reason)})
		}
	}
	return failed
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

func resolveScopeKey(scope agent.Scope, workDir string) (string, error) {
	if scope == agent.ScopeUser {
		return lockpkg.GlobalKey, nil
	}
	if strings.TrimSpace(workDir) == "" {
		return "", fmt.Errorf("work dir is required for project scope")
	}
	absPath, err := filepath.Abs(workDir)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absPath), nil
}

func scopeFromKey(scopeKey string) agent.Scope {
	if scopeKey == lockpkg.GlobalKey {
		return agent.ScopeUser
	}
	return agent.ScopeProject
}

func resolveLatestInstalledPath(config cfg.Config, workDir string, scopeKey string, agentName string, scope agent.Scope, item skill.Skill) (string, error) {
	baseDir := workDir
	if scope == agent.ScopeProject {
		baseDir = scopeKey
	}
	targetRoot, err := agent.ResolveInstallPath(config, baseDir, agentName, scope)
	if err != nil {
		return "", err
	}
	return filepath.Join(targetRoot, item.ID), nil
}

func resolveInstalledPath(config cfg.Config, workDir string, scopeKey string, agentName string, scope agent.Scope, record lockpkg.Record) (string, error) {
	baseDir := workDir
	if scope == agent.ScopeProject {
		baseDir = scopeKey
	}
	targetRoot, err := agent.ResolveInstallPath(config, baseDir, agentName, scope)
	if err != nil {
		return "", err
	}
	flatPath := filepath.Join(targetRoot, record.SkillID)
	legacyPath := filepath.Join(targetRoot, legacyInstallDir(record.SkillID, record.SourceQualifiedName, record.SourceID))
	resolvedPath, err := preferExistingInstallPath(flatPath, legacyPath)
	if err != nil {
		return "", err
	}
	return resolvedPath, nil
}

func filterAgents(agents []string, agentName string) []string {
	if agentName == "" {
		return append([]string(nil), agents...)
	}
	for _, current := range agents {
		if current == agentName {
			return []string{current}
		}
	}
	return nil
}

func sortedScopeKeys(records lockpkg.File) []string {
	keys := make([]string, 0, len(records))
	for scopeKey := range records {
		keys = append(keys, scopeKey)
	}
	sort.Strings(keys)
	return keys
}
