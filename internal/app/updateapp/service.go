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
	ReinstallAtPath(item skill.Skill, agentName string, scope agent.Scope, targetPath string) (lockpkg.Record, error)
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

type Candidate struct {
	Installed lockpkg.Record
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
	Candidates   []Candidate
	Updated      []lockpkg.Record
	SyncFailed   []SourceSyncError
	UpdateFailed []UpdateItemError
	Failed       []FailedItem
	Skipped      []SkippedItem
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

	installer := s.newInstaller
	if installer == nil {
		installer = s.installFactory
	}
	worker := installer(config.LockFile)
	for _, candidate := range result.Candidates {
		record, err := worker.ReinstallAtPath(candidate.Latest, candidate.Installed.Agent, agent.Scope(candidate.Installed.Scope), candidate.Installed.InstalledPath)
		if err != nil {
			result.UpdateFailed = append(result.UpdateFailed, UpdateItemError{SkillID: candidate.Installed.SkillID, Reason: err.Error()})
			result.Failed = append(result.Failed, FailedItem{SkillID: candidate.Installed.SkillID, Reason: err.Error()})
			continue
		}
		result.Updated = append(result.Updated, record)
	}
	return result, nil
}

func (s *Service) collectSelected(config cfg.Config, req UpdateReq, scope agent.Scope) ([]lockpkg.Record, []SkippedItem, error) {
	records, err := s.loadRecords(config.LockFile)
	if err != nil {
		return nil, nil, err
	}
	if len(records) > 0 {
		return selectRecords(records, req.Target, req.Agent, string(scope), req.All)
	}
	return s.collectFromInstalledDirs(config, req.WorkDir, req.Agent, scope)
}

func (s *Service) loadRecords(path string) ([]lockpkg.Record, error) {
	records, err := s.lockStore.Load(path)
	if err == nil {
		return records, nil
	}
	if os.IsNotExist(err) {
		return nil, nil
	}
	return nil, err
}

func selectRecords(records []lockpkg.Record, target string, agentName string, scope string, all bool) ([]lockpkg.Record, []SkippedItem, error) {
	selected := make([]lockpkg.Record, 0, len(records))
	skipped := make([]SkippedItem, 0)
	for _, record := range records {
		if agentName != "" && record.Agent != agentName {
			continue
		}
		if scope != "" && record.Scope != scope {
			continue
		}
		if !all && target != "" && !matchesRecordTarget(record, target) {
			continue
		}
		if record.Pinned {
			skipped = append(skipped, SkippedItem{SkillID: record.SkillID, Reason: "pinned"})
			continue
		}
		selected = append(selected, record)
	}
	if !all && target != "" && len(selected) == 0 && len(skipped) == 0 {
		return nil, nil, fmt.Errorf("skill not found: %s", target)
	}
	return selected, skipped, nil
}

func matchesRecordTarget(record lockpkg.Record, target string) bool {
	return record.SkillID == target || record.QualifiedName == target || record.SourceQualifiedName == target
}

func (s *Service) collectFromInstalledDirs(config cfg.Config, workDir, agentName string, scope agent.Scope) ([]lockpkg.Record, []SkippedItem, error) {
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
	entryNames := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			entryNames[entry.Name()] = struct{}{}
		}
	}

	selected := make([]lockpkg.Record, 0, len(entries))
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
			selected = append(selected, lockpkg.Record{
				SkillID:             matches[0].ID,
				QualifiedName:       matches[0].QualifiedName,
				SourceQualifiedName: matches[0].SourceQualifiedName,
				Agent:               agentName,
				Scope:               string(scope),
				SourceID:            matches[0].SourceID,
				SourceType:          string(matches[0].SourceType),
				InstallEntry:        matches[0].InstallEntry,
				InstalledPath:       filepath.Join(targetRoot, entry.Name()),
			})
		default:
			skipped = append(skipped, SkippedItem{SkillID: entry.Name(), Reason: "ambiguous index match"})
		}
	}
	return selected, skipped, nil
}

func (s *Service) syncSources(records []lockpkg.Record) []SourceSyncError {
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


func uniqueSourceIDs(records []lockpkg.Record) []string {
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

func collectCandidates(records []lockpkg.Record, items []skill.Skill, syncFailed []SourceSyncError) ([]Candidate, []UpdateItemError) {
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
		latest, ok := findLatest(items, record)
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
	if record.SourceQualifiedName != "" || item.SourceQualifiedName != "" {
		return record.SourceQualifiedName != "" && record.SourceQualifiedName == item.SourceQualifiedName
	}
	if record.SourceID != "" || item.SourceID != "" {
		return record.SourceID != "" && record.SourceID == item.SourceID
	}
	if record.QualifiedName != "" || item.QualifiedName != "" {
		return record.QualifiedName != "" && record.QualifiedName == item.QualifiedName
	}
	return false
}

func matchInstalledDir(items []skill.Skill, dirName string, entryNames map[string]struct{}) []skill.Skill {
	matches := make([]skill.Skill, 0)
	for _, item := range items {
		if sourceScopedInstallDir(item) == dirName {
			matches = append(matches, item)
			continue
		}
		if item.ID != dirName {
			continue
		}
		if shouldMatchPlainInstalledDir(item, items, entryNames) {
			matches = append(matches, item)
		}
	}
	return matches
}

func shouldMatchPlainInstalledDir(item skill.Skill, items []skill.Skill, entryNames map[string]struct{}) bool {
	ownScopedDir := sourceScopedInstallDir(item)
	if _, ok := entryNames[ownScopedDir]; ok {
		return false
	}
	for _, other := range items {
		if other.ID != item.ID || sameSkillSource(item, other) {
			continue
		}
		if _, ok := entryNames[sourceScopedInstallDir(other)]; ok {
			return true
		}
	}
	return true
}

func sameSkillSource(a, b skill.Skill) bool {
	if a.SourceQualifiedName != "" || b.SourceQualifiedName != "" {
		return a.SourceQualifiedName != "" && a.SourceQualifiedName == b.SourceQualifiedName
	}
	if a.SourceID != "" || b.SourceID != "" {
		return a.SourceID != "" && a.SourceID == b.SourceID
	}
	if a.QualifiedName != "" || b.QualifiedName != "" {
		return a.QualifiedName != "" && a.QualifiedName == b.QualifiedName
	}
	return a.ID == b.ID
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

func failedFromSync(records []lockpkg.Record, syncFailed []SourceSyncError) []FailedItem {
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
