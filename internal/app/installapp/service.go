package installapp

import (
	"errors"
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

type RestoreResolver func(record lockpkg.Record) (skill.Skill, bool, error)

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

type UninstallReq struct {
	Skills  []string
	Agent   string
	Scope   string
	WorkDir string
}

type UninstallPlan struct {
	Agent string              `json:"agent"`
	Scope string              `json:"scope"`
	Items []UninstallPlanItem `json:"items"`
}

type UninstallPlanItem struct {
	Action              string `json:"action"`
	SkillID             string `json:"skill_id"`
	QualifiedName       string `json:"qualified_name,omitempty"`
	SourceQualifiedName string `json:"source_qualified_name,omitempty"`
	SourceID            string `json:"source_id,omitempty"`
	Version             string `json:"version,omitempty"`
	Agent               string `json:"agent"`
	Scope               string `json:"scope"`
	InstalledPath       string `json:"installed_path,omitempty"`
	Reason              string `json:"reason,omitempty"`
}

type UninstallResult struct {
	Plan    UninstallPlan       `json:"plan"`
	Removed []UninstallPlanItem `json:"removed"`
	Failed  []InstallItemError  `json:"failed,omitempty"`
}

type Service struct {
	lockFile          string
	store             *lockstore.Store
	installer         *agentfs.Installer
	now               func() time.Time
	config            cfg.Config
	workDir           string
	installerExplicit bool
	fallbackNotifier  agentfs.FallbackNotifier
	restoreResolver   RestoreResolver
	force             bool
}

func (s *Service) WithForce(force bool) *Service {
	clone := *s
	clone.force = force
	return &clone
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
	if !clone.installerExplicit {
		mode := agentfs.NormalizeMode(strings.TrimSpace(config.InstallMode))
		installer := agentfs.NewInstallerWithMode(mode)
		installer.OnSymlinkFallback = clone.fallbackNotifier
		clone.installer = installer
	}
	return &clone
}

// WithInstallMode 返回一个使用指定安装模式的 Service 副本。
// 用于 CLI 的 --copy / --install-mode 标志覆盖 config.InstallMode。
// 该设置在后续 WithRuntime 中不会被 config 覆盖。
func (s *Service) WithInstallMode(mode agentfs.Mode) *Service {
	clone := *s
	installer := agentfs.NewInstallerWithMode(mode)
	installer.OnSymlinkFallback = clone.fallbackNotifier
	clone.installer = installer
	clone.installerExplicit = true
	return &clone
}

// WithSymlinkFallbackNotifier 注入 symlink 失败回退到 copy 时的回调（用于打印提示）
func (s *Service) WithSymlinkFallbackNotifier(notifier agentfs.FallbackNotifier) *Service {
	clone := *s
	clone.fallbackNotifier = notifier
	if clone.installer == nil {
		clone.installer = agentfs.NewInstaller()
	}
	cloneInstaller := *clone.installer
	cloneInstaller.OnSymlinkFallback = notifier
	clone.installer = &cloneInstaller
	return &clone
}

func (s *Service) WithRestoreResolver(resolver RestoreResolver) *Service {
	clone := *s
	clone.restoreResolver = resolver
	return &clone
}

type InstallReq struct {
	SkillID string
	Agent   string
	Scope   string
	WorkDir string
	Profile string
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

	installResult, err := runtimeSvc.InstallMulti(items, req.Agent, scope, scopeKey, targetRoot, req.Profile)
	if err != nil {
		return CommandResult{}, err
	}
	return CommandResult{
		Installed:     installResult.Installed,
		ResolveFailed: resolveFailed,
		InstallFailed: installResult.Failed,
	}, nil
}

// InstallMulti installs multiple skills with a single lock file load/save.
func (s *Service) InstallMulti(items []skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetRoot string, profileName string) (BatchInstallResult, error) {
	result := BatchInstallResult{
		Installed: make([]RuntimeRecord, 0, len(items)),
		Failed:    make([]InstallItemError, 0),
	}

	locks, err := s.loadLockFile()
	if err != nil {
		return result, err
	}

	for _, item := range items {
		record, err := s.installInto(item, agentName, scope, scopeKey, targetRoot, locks, profileName)
		if err != nil {
			result.Failed = append(result.Failed, InstallItemError{SkillID: item.ID, Reason: err.Error()})
			continue
		}
		result.Installed = append(result.Installed, record)
	}

	if len(result.Installed) > 0 {
		if err := s.store.Save(s.lockFile, locks); err != nil {
			return result, err
		}
	}
	return result, nil
}

// Install installs a single skill.
func (s *Service) Install(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetRoot string) (RuntimeRecord, error) {
	locks, err := s.loadLockFile()
	if err != nil {
		return RuntimeRecord{}, err
	}
	record, err := s.installInto(item, agentName, scope, scopeKey, targetRoot, locks, "")
	if err != nil {
		return RuntimeRecord{}, err
	}
	if err := s.store.Save(s.lockFile, locks); err != nil {
		return RuntimeRecord{}, err
	}
	return record, nil
}

// installInto installs a skill into the provided locks map without saving.
func (s *Service) installInto(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetRoot string, locks lockpkg.File, profileName string) (RuntimeRecord, error) {
	records := append([]lockpkg.Record(nil), locks[scopeKey]...)
	now := s.now()
	record := newLockRecord(item, agentName, profileName, now)
	conflict, hasConflict := findConflictingSkillSource(records, record)
	if hasConflict && !s.force {
		return RuntimeRecord{}, fmt.Errorf("skill already installed from another source: %s (%s)", item.ID, conflict.SourceQualifiedName)
	}

	targetPath := installTargetPath(item, targetRoot)
	if err := s.installer.Install(filepath.Join(item.Path, item.InstallEntry), targetPath); err != nil {
		return RuntimeRecord{}, err
	}

	if hasConflict {
		records = removeConflictingAgent(records, record)
	}
	records, record = upsertRecord(records, record)
	if len(records) == 0 {
		delete(locks, scopeKey)
	} else {
		locks[scopeKey] = records
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
	record := newLockRecord(item, agentName, "", now)

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
	var errs []error
	for _, skillID := range skillIDs {
		if err := s.Uninstall(skillID, agentName, scope); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", skillID, err))
		}
	}
	return errors.Join(errs...)
}

func (s *Service) PlanUninstall(req UninstallReq) (UninstallPlan, error) {
	scope, err := parseScope(req.Scope)
	if err != nil {
		return UninstallPlan{}, err
	}
	agentName := strings.TrimSpace(req.Agent)
	if agentName == "" {
		agentName = agent.DefaultAgentName
	}
	runtimeSvc := s.WithRuntime(s.runtimeConfig(), firstNonEmpty(req.WorkDir, s.runtimeWorkDir()))
	locks, err := runtimeSvc.loadLockFile()
	if err != nil {
		return UninstallPlan{}, err
	}

	plan := UninstallPlan{Agent: agentName, Scope: string(scope)}
	for _, target := range req.Skills {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		found := false
		for _, scopeKey := range runtimeSvc.matchScopeKeys(locks, scope) {
			for _, record := range locks[scopeKey] {
				if !matchesSkillTarget(record, target) || !containsAgent(record.Agents, agentName) {
					continue
				}
				path, err := runtimeSvc.resolveInstalledPath(scopeKey, scope, agentName, record)
				if err != nil {
					return UninstallPlan{}, err
				}
				action := "remove_agent"
				if len(record.Agents) == 1 {
					action = "remove_record"
				}
				plan.Items = append(plan.Items, UninstallPlanItem{
					Action:              action,
					SkillID:             record.SkillID,
					QualifiedName:       record.QualifiedName,
					SourceQualifiedName: record.SourceQualifiedName,
					SourceID:            record.SourceID,
					Version:             record.Version,
					Agent:               agentName,
					Scope:               string(scope),
					InstalledPath:       path,
				})
				found = true
			}
		}
		if !found {
			plan.Items = append(plan.Items, UninstallPlanItem{
				Action:  "error",
				SkillID: target,
				Agent:   agentName,
				Scope:   string(scope),
				Reason:  "skill not found",
			})
		}
	}
	return plan, nil
}

func (s *Service) RunUninstall(req UninstallReq) (UninstallResult, error) {
	plan, err := s.PlanUninstall(req)
	if err != nil {
		return UninstallResult{}, err
	}
	result := UninstallResult{Plan: plan}
	for _, item := range plan.Items {
		if item.Action == "error" {
			result.Failed = append(result.Failed, InstallItemError{SkillID: item.SkillID, Reason: item.Reason})
		}
	}
	if len(result.Failed) > 0 {
		return result, fmt.Errorf("uninstall plan has errors")
	}
	scope, err := parseScope(req.Scope)
	if err != nil {
		return result, err
	}
	runtimeSvc := s.WithRuntime(s.runtimeConfig(), firstNonEmpty(req.WorkDir, s.runtimeWorkDir()))
	if err := runtimeSvc.UninstallMulti(req.Skills, plan.Agent, scope); err != nil {
		return result, err
	}
	result.Removed = append(result.Removed, plan.Items...)
	return result, nil
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
			sourcePath, err := s.restoreSourcePath(record, sourcePaths)
			if err != nil {
				return nil, err
			}
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

func (s *Service) restoreSourcePath(record lockpkg.Record, sourcePaths map[string]string) (string, error) {
	if s.restoreResolver != nil {
		if item, handled, err := s.restoreResolver(record); err != nil {
			return "", err
		} else if handled {
			return filepath.Join(item.Path, item.InstallEntry), nil
		}
	}
	sourceRoot, ok := sourcePaths[record.SourceID]
	if !ok {
		return "", fmt.Errorf("source path not found for %s", record.SourceID)
	}
	return filepath.Join(sourceRoot, record.InstallEntry), nil
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
	return filepath.Join(targetRoot, record.SkillID), nil
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

func newLockRecord(item skill.Skill, agentName string, profileName string, now time.Time) lockpkg.Record {
	return lockpkg.Record{
		SkillID:             item.ID,
		QualifiedName:       item.QualifiedName,
		SourceQualifiedName: item.SourceQualifiedName,
		Version:             item.Version,
		SourceID:            item.SourceID,
		SourceType:          string(item.SourceType),
		SourceResolvedRef:   item.SourceResolvedRef,
		Profile:             profileName,
		InstallEntry:        item.InstallEntry,
		Agents:              []string{agentName},
		Checksum:            item.Checksum,
		RegistryEntryID:     item.RegistryEntryID,
		RegistryURL:         item.RegistryURL,
		DownloadURL:         item.DownloadURL,
		SourceURL:           item.SourceURL,
		SourceRef:           item.SourceRef,
		InstalledAt:         now,
		UpdatedAt:           now,
	}
}

func upsertRecord(records []lockpkg.Record, next lockpkg.Record) ([]lockpkg.Record, lockpkg.Record) {
	for i, record := range records {
		if sameInstallIdentity(record, next) {
			next.InstalledAt = record.InstalledAt
			next.Agents = mergeAgents(record.Agents, next.Agents)
			if next.Profile == "" {
				next.Profile = record.Profile
			}
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
		if record.SkillID != next.SkillID || !containsAgent(record.Agents, next.Agents[0]) {
			continue
		}
		if sameInstallIdentity(record, next) {
			return lockpkg.Record{}, false
		}
		return record, true
	}
	return lockpkg.Record{}, false
}

func removeConflictingAgent(records []lockpkg.Record, next lockpkg.Record) []lockpkg.Record {
	result := make([]lockpkg.Record, 0, len(records))
	for _, record := range records {
		if record.SkillID == next.SkillID && !sameInstallIdentity(record, next) {
			record.Agents = removeAgent(record.Agents, next.Agents[0])
			if len(record.Agents) == 0 {
				continue
			}
		}
		result = append(result, record)
	}
	return result
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
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
