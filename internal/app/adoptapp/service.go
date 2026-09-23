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
	"github.com/inhere/skillc/internal/infra/gitx"
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
	git           fileDiffer
	now           func() time.Time
}

// fileDiffer 生成两个文件的统一 diff（默认为 git diff --no-index）。
type fileDiffer interface {
	DiffNoIndex(left string, right string) (string, error)
}

// incomingSuffix 是合并冲突时保留的上游文件后缀。
const incomingSuffix = ".incoming"

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile:    configFile,
		baseDir:       baseDir,
		configService: configapp.NewService(configFile, baseDir),
		lockStore:     lockstore.NewStore(),
		indexStore:    repoindex.NewStore(),
		syncer:        sourceapp.NewService(configFile, baseDir),
		git:           gitx.New(""),
		now:           time.Now,
	}
}

// resolvedTarget 是一次 adopt/diff 解析出的目标与三份文件清单。
type resolvedTarget struct {
	record        lockpkg.Record
	installedPath string
	sourcePath    string
	baseline      map[string]string
	current       map[string]string
	incoming      map[string]string
	incomingSum   string
	writable      bool
	reason        string
}

// Plan 计算需要写回源目录的文件清单。
func (s *Service) Plan(req Req) (Plan, error) {
	target, err := s.resolveTarget(req)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{
		SkillID:       target.record.SkillID,
		SourceID:      target.record.SourceID,
		SourceType:    target.record.SourceType,
		InstalledPath: target.installedPath,
		SourcePath:    target.sourcePath,
		Writable:      target.writable,
		Reason:        target.reason,
	}
	if !target.writable {
		return plan, nil
	}
	plan.Items = planAdopt(target.baseline, target.current, target.sourcePath, target.installedPath)
	return plan, nil
}

// FileState 表示单个文件在某一侧（本地/上游）的状态。
type FileState string

const (
	StateSame     FileState = "same"
	StateModified FileState = "modified"
	StateAdded    FileState = "added"
	StateDeleted  FileState = "deleted"
	// StateAbsent 表示该侧既没有基线也没有文件（例如本地新增、上游从来没有）。
	StateAbsent FileState = "absent"
)

// FileDiff 是单个文件的差异条目。
type FileDiff struct {
	Path     string    `json:"path"`
	Local    FileState `json:"local"`
	Upstream FileState `json:"upstream"`
	// Conflict 表示本地与上游都相对安装基线发生了变化。
	Conflict bool `json:"conflict,omitempty"`
	// HasIncoming 表示目录里存在待处理的 <path>.incoming。
	HasIncoming bool `json:"has_incoming,omitempty"`
	// Patch 是「安装目录 vs 源目录」的统一 diff，双方都存在且不同时才有值。
	Patch string `json:"patch,omitempty"`
	// Note 说明无法生成 patch 的原因（例如一侧缺失）。
	Note string `json:"note,omitempty"`
}

// DiffResult 是一次 diff 的结果。
type DiffResult struct {
	SkillID       string     `json:"skill_id"`
	SourceID      string     `json:"source_id,omitempty"`
	SourceType    string     `json:"source_type,omitempty"`
	InstalledPath string     `json:"installed_path,omitempty"`
	SourcePath    string     `json:"source_path,omitempty"`
	Writable      bool       `json:"writable"`
	Reason        string     `json:"reason,omitempty"`
	Files         []FileDiff `json:"files"`
}

// Diff 列出安装目录与源目录的逐文件差异，并为两侧都存在的文件生成统一 diff。
func (s *Service) Diff(req Req) (DiffResult, error) {
	target, err := s.resolveTarget(req)
	if err != nil {
		return DiffResult{}, err
	}
	result := DiffResult{
		SkillID:       target.record.SkillID,
		SourceID:      target.record.SourceID,
		SourceType:    target.record.SourceType,
		InstalledPath: target.installedPath,
		SourcePath:    target.sourcePath,
		Writable:      target.writable,
		Reason:        target.reason,
	}
	if !target.writable {
		return result, nil
	}

	plan := installpkg.PlanMerge(target.baseline, target.current, target.incoming)
	files := make([]FileDiff, 0, len(plan))
	for _, item := range plan {
		base, baseOK := target.baseline[item.Path]
		local, localOK := target.current[item.Path]
		upstream, upstreamOK := target.incoming[item.Path]
		entry := FileDiff{
			Path:     item.Path,
			Local:    fileState(baseOK, base, localOK, local),
			Upstream: fileState(baseOK, base, upstreamOK, upstream),
		}
		// 冲突判定与合并引擎一致：合并会保留本地并写出 <file>.incoming
		entry.Conflict = item.Action == installpkg.MergeConflict
		entry.Note = mergeNote(item.Action)
		installedFile := filepath.Join(target.installedPath, filepath.FromSlash(item.Path))
		sourceFile := filepath.Join(target.sourcePath, filepath.FromSlash(item.Path))
		if _, err := os.Stat(installedFile + incomingSuffix); err == nil {
			entry.HasIncoming = true
		}
		switch {
		case local == upstream:
			// 内容一致，无需 diff
		case !localOK || !upstreamOK:
			// 只有一侧存在，没有可对比的文本
		default:
			patch, err := s.git.DiffNoIndex(installedFile, sourceFile)
			if err != nil {
				return DiffResult{}, err
			}
			entry.Patch = rewritePatchPaths(patch, item.Path)
		}
		files = append(files, entry)
	}
	result.Files = files
	return result, nil
}

// mergeNote 把合并动作翻译成一句提示，普通前进不提示。
func mergeNote(action installpkg.MergeAction) string {
	switch action {
	case installpkg.MergeConflict:
		return "conflict: merge keeps local and writes <file>.incoming"
	case installpkg.MergeKeepRemoved:
		return "upstream deleted, local edit kept"
	case installpkg.MergeLocalDeleted:
		return "deleted locally"
	case installpkg.MergeAddLocal:
		return "local only"
	case installpkg.MergeAddIncoming:
		return "upstream only"
	case installpkg.MergeRemove:
		return "upstream deleted"
	}
	return ""
}

// fileState 根据基线哈希与某一侧的哈希判断状态。
func fileState(baseOK bool, base string, sideOK bool, side string) FileState {
	switch {
	case sideOK && !baseOK:
		return StateAdded
	case sideOK && side == base:
		return StateSame
	case sideOK:
		return StateModified
	case baseOK:
		return StateDeleted
	default:
		return StateAbsent
	}
}

// rewritePatchPaths 把 git diff 头部的绝对路径换成 installed/<rel> 与 source/<rel>。
func rewritePatchPaths(patch string, rel string) string {
	lines := strings.Split(patch, "\n")
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			lines[i] = "diff --git installed/" + rel + " source/" + rel
		case strings.HasPrefix(line, "--- "):
			lines[i] = "--- installed/" + rel
		case strings.HasPrefix(line, "+++ "):
			lines[i] = "+++ source/" + rel
		}
	}
	return strings.Join(lines, "\n")
}

// resolveTarget 解析目标 skill 的安装目录、源目录与三份文件清单（基线/本地/上游）。
func (s *Service) resolveTarget(req Req) (resolvedTarget, error) {
	config, err := s.configService.Show()
	if err != nil {
		return resolvedTarget{}, err
	}
	if req.WorkDir == "" {
		req.WorkDir = s.baseDir
	}
	scope, err := apputil.ParseScope(defaultString(req.Scope, string(agent.ScopeProject)))
	if err != nil {
		return resolvedTarget{}, err
	}
	agentName := config.CanonicalAgentName(defaultString(req.Agent, agent.DefaultAgentName))

	records, err := s.lockStore.WithAgentResolver(config.CanonicalAgentName).Load(config.LockFile)
	if err != nil {
		if os.IsNotExist(err) {
			return resolvedTarget{}, fmt.Errorf("skill not found: %s", req.Target)
		}
		return resolvedTarget{}, err
	}
	scopeKey, err := apputil.ResolveScopeKey(scope, req.WorkDir)
	if err != nil {
		return resolvedTarget{}, err
	}
	record, installedPath, err := findInstalledRecord(config, records[scopeKey], scope, scopeKey, req.WorkDir, agentName, req.Target)
	if err != nil {
		return resolvedTarget{}, err
	}
	target := resolvedTarget{record: record, installedPath: installedPath}

	switch {
	case record.SourceType == string(sourcepkg.TypeRegistry):
		target.reason = "registry skill: adopt the change in its upstream repository instead"
		return target, nil
	case record.SourceType == string(sourcepkg.TypeGit):
		target.reason = "git source: the source directory is a cache clone; copy the change into your skills repository and run 'skillc source sync'"
		return target, nil
	case len(record.InstalledFiles) == 0:
		target.reason = "no file manifest for this install; reinstall the skill once to enable adopt"
		return target, nil
	}

	item, ok := s.findIndexedSkill(config, record)
	if !ok {
		target.reason = fmt.Sprintf("skill not found in source index: %s (run 'skillc source sync %s')", record.SkillID, record.SourceID)
		return target, nil
	}
	target.sourcePath = filepath.Join(item.Path, item.InstallEntry)

	current, err := hashx.Deployed(installedPath)
	if err != nil {
		return resolvedTarget{}, err
	}
	incoming, err := hashx.Deployed(target.sourcePath)
	if err != nil {
		return resolvedTarget{}, err
	}
	target.baseline = record.InstalledFiles
	target.current = current.Files
	target.incoming = incoming.Files
	target.incomingSum = incoming.Sum
	target.writable = true
	return target, nil
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
