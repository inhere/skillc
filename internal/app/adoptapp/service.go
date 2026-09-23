// Package adoptapp 把安装目录里的本地改动写回 skill 源目录。
package adoptapp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/inhere/skillc/internal/app/apputil"
	"github.com/inhere/skillc/internal/app/configapp"
	"github.com/inhere/skillc/internal/app/listapp"
	"github.com/inhere/skillc/internal/app/sourceapp"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	installpkg "github.com/inhere/skillc/internal/domain/install"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/fsx"
	"github.com/inhere/skillc/internal/infra/hashx"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

// Action 表示单个文件的回写动作。
type Action string

const (
	// ActionWrite 把安装目录的版本写回源目录。
	ActionWrite Action = "write"
	// ActionDelete 本地删除了该文件；源文件需要用户自行删除。
	ActionDelete Action = "delete"
	// ActionUnchanged 内容一致，无需处理。
	ActionUnchanged Action = "unchanged"
)

// Item 是单个文件的回写项。
type Item struct {
	Path          string `json:"path"`
	Action        Action `json:"action"`
	SourcePath    string `json:"source_path,omitempty"`
	InstalledPath string `json:"installed_path,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

// Plan 是一次 adopt 的计划。
type Plan struct {
	SkillID       string `json:"skill_id"`
	SourceID      string `json:"source_id,omitempty"`
	SourceType    string `json:"source_type,omitempty"`
	SourcePath    string `json:"source_path,omitempty"`
	InstalledPath string `json:"installed_path,omitempty"`
	// Writable 表示源目录可写回；false 时 Reason 说明原因。
	Writable   bool   `json:"writable"`
	Reason     string `json:"reason,omitempty"`
	BackupPath string `json:"backup_path,omitempty"`
	Items      []Item `json:"items"`
}

// Req 是 adopt 的请求参数。
type Req struct {
	Target  string
	Agent   string
	Scope   string
	WorkDir string
	// DryRun 只输出计划，不写源目录。
	DryRun bool
}

type Service struct {
	configFile    string
	baseDir       string
	configService *configapp.Service
	lockStore     *lockstore.Store
	indexStore    *repoindex.Store
	syncer        *sourceapp.Service
	now           func() time.Time
}

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile:    configFile,
		baseDir:       baseDir,
		configService: configapp.NewService(configFile, baseDir),
		lockStore:     lockstore.NewStore(),
		indexStore:    repoindex.NewStore(),
		syncer:        sourceapp.NewService(configFile, baseDir),
		now:           time.Now,
	}
}

// Plan 计算需要写回源目录的文件清单。
func (s *Service) Plan(req Req) (Plan, error) {
	config, err := s.configService.Show()
	if err != nil {
		return Plan{}, err
	}
	if req.WorkDir == "" {
		req.WorkDir = s.baseDir
	}
	scope, err := apputil.ParseScope(defaultString(req.Scope, string(agent.ScopeProject)))
	if err != nil {
		return Plan{}, err
	}
	agentName := config.CanonicalAgentName(defaultString(req.Agent, agent.DefaultAgentName))

	records, err := s.lockStore.WithAgentResolver(config.CanonicalAgentName).Load(config.LockFile)
	if err != nil {
		if os.IsNotExist(err) {
			return Plan{}, fmt.Errorf("skill not found: %s", req.Target)
		}
		return Plan{}, err
	}
	scopeKey, err := apputil.ResolveScopeKey(scope, req.WorkDir)
	if err != nil {
		return Plan{}, err
	}

	record, installedPath, err := findInstalledRecord(config, records[scopeKey], scope, scopeKey, req.WorkDir, agentName, req.Target)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{
		SkillID:       record.SkillID,
		SourceID:      record.SourceID,
		SourceType:    record.SourceType,
		InstalledPath: installedPath,
	}

	if record.SourceType == string(sourcepkg.TypeRegistry) {
		plan.Reason = "registry skill: adopt the change in its upstream repository instead"
		return plan, nil
	}
	if record.SourceType == string(sourcepkg.TypeGit) {
		plan.Reason = "git source: the source directory is a cache clone; copy the change into your skills repository and run 'skillc source sync'"
		return plan, nil
	}
	if len(record.InstalledFiles) == 0 {
		plan.Reason = "no file manifest for this install; reinstall the skill once to enable adopt"
		return plan, nil
	}

	item, ok := s.findIndexedSkill(config, record)
	if !ok {
		plan.Reason = fmt.Sprintf("skill not found in source index: %s (run 'skillc source sync %s')", record.SkillID, record.SourceID)
		return plan, nil
	}
	sourceDir := filepath.Join(item.Path, item.InstallEntry)
	plan.SourcePath = sourceDir

	current, err := hashx.Deployed(installedPath)
	if err != nil {
		return Plan{}, err
	}
	plan.Items = planAdopt(record.InstalledFiles, current.Files, sourceDir, installedPath)
	plan.Writable = true
	return plan, nil
}

// Run 执行 adopt：先备份源目录，再把本地改动写回，最后重建索引并刷新 lock 基线。
func (s *Service) Run(req Req) (Plan, error) {
	plan, err := s.Plan(req)
	if err != nil {
		return Plan{}, err
	}
	if !plan.Writable {
		return plan, fmt.Errorf("cannot adopt %s: %s", plan.SkillID, plan.Reason)
	}
	writable := make([]Item, 0, len(plan.Items))
	for _, item := range plan.Items {
		if item.Action == ActionWrite {
			writable = append(writable, item)
		}
	}
	if len(writable) == 0 {
		return plan, nil
	}
	if req.DryRun {
		return plan, nil
	}

	config, err := s.configService.Show()
	if err != nil {
		return plan, err
	}
	scope, err := apputil.ParseScope(defaultString(req.Scope, string(agent.ScopeProject)))
	if err != nil {
		return plan, err
	}
	workDir := req.WorkDir
	if workDir == "" {
		workDir = s.baseDir
	}
	scopeKey, err := apputil.ResolveScopeKey(scope, workDir)
	if err != nil {
		return plan, err
	}

	guard := apputil.OverwriteGuard{BackupRoot: config.BackupDir, Force: true, Now: s.now}
	backupPath, err := guard.Backup(scopeKey, plan.SkillID, plan.SourcePath)
	if err != nil {
		return plan, err
	}
	plan.BackupPath = backupPath

	for _, item := range writable {
		if err := fsx.CopyFile(item.InstalledPath, item.SourcePath); err != nil {
			return plan, fmt.Errorf("adopt %s: %w", item.Path, err)
		}
	}

	// 源内容已变化：重建索引，并把新的源内容记为部署基线
	if err := s.syncer.Reindex(plan.SourceID); err != nil {
		return plan, err
	}
	if err := s.refreshRecord(config, scope, scopeKey, plan.SkillID); err != nil {
		return plan, err
	}
	return plan, nil
}

// refreshRecord 用新的源内容刷新 lock 记录（版本与部署基线）。
func (s *Service) refreshRecord(config cfg.Config, scope agent.Scope, scopeKey string, skillID string) error {
	records, err := s.lockStore.Load(config.LockFile)
	if err != nil {
		return err
	}
	group := records[scopeKey]
	updated := false
	for i := range group {
		if group[i].SkillID != skillID {
			continue
		}
		item, ok := s.findIndexedSkill(config, group[i])
		if !ok {
			continue
		}
		info, err := hashx.Deployed(filepath.Join(item.Path, item.InstallEntry))
		if err != nil {
			return err
		}
		group[i].Version = item.Version
		group[i].Checksum = item.Checksum
		group[i].SourceResolvedRef = item.SourceResolvedRef
		group[i].InstalledChecksum = info.Sum
		group[i].InstalledFiles = info.Files
		group[i].UpdatedAt = s.now()
		updated = true
	}
	if !updated {
		return nil
	}
	records[scopeKey] = group
	return s.lockStore.WithAgentResolver(config.CanonicalAgentName).Save(config.LockFile, records)
}

// findIndexedSkill 按来源身份在索引里定位 skill。
func (s *Service) findIndexedSkill(config cfg.Config, record lockpkg.Record) (skill.Skill, bool) {
	items, err := s.indexStore.Load(config.IndexFile)
	if err != nil {
		return skill.Skill{}, false
	}
	for _, item := range items {
		if item.Path != "" && installpkg.SameIdentity(record, item) {
			return item, true
		}
	}
	return skill.Skill{}, false
}

// planAdopt 比较部署基线与安装目录现状，得出需要写回源目录的文件。
func planAdopt(baseline map[string]string, current map[string]string, sourceDir string, installedDir string) []Item {
	paths := make([]string, 0, len(baseline)+len(current))
	seen := make(map[string]struct{}, len(baseline)+len(current))
	for _, group := range []map[string]string{baseline, current} {
		for path := range group {
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)

	items := make([]Item, 0, len(paths))
	for _, path := range paths {
		base, baseOK := baseline[path]
		cur, curOK := current[path]
		item := Item{
			Path:          path,
			SourcePath:    filepath.Join(sourceDir, filepath.FromSlash(path)),
			InstalledPath: filepath.Join(installedDir, filepath.FromSlash(path)),
		}
		switch {
		case !curOK && baseOK:
			item.Action = ActionDelete
			item.Reason = "deleted locally; delete it in the source yourself"
		case curOK && baseOK && cur == base:
			item.Action = ActionUnchanged
		default:
			item.Action = ActionWrite
		}
		items = append(items, item)
	}
	return items
}

// findInstalledRecord 在当前项目里定位目标 skill 的安装记录与安装目录。
func findInstalledRecord(config cfg.Config, records []lockpkg.Record, scope agent.Scope, scopeKey string, workDir string, agentName string, target string) (lockpkg.Record, string, error) {
	for _, record := range records {
		if !matchesTarget(record, target) {
			continue
		}
		for _, current := range record.Agents {
			if config.CanonicalAgentName(current) != agentName {
				continue
			}
			installedPath, err := listapp.ResolveInstalledPath(config, workDir, scopeKey, scope, agentName, record.SkillID)
			if err != nil {
				return lockpkg.Record{}, "", err
			}
			return record, installedPath, nil
		}
	}
	return lockpkg.Record{}, "", fmt.Errorf("skill not found: %s", target)
}

func matchesTarget(record lockpkg.Record, target string) bool {
	return record.SkillID == target || record.QualifiedName == target || record.SourceQualifiedName == target
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
