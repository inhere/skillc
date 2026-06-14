# Skillc v0 Phase 2 Status And Update Check Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a read-only status/update-check slice so users can see current project skill health, missing installs, outdated versions, unmanaged directories, and update candidates before mutating install directories or lock files.

**Architecture:** Add `internal/app/statusapp` as the shared query layer for status and update-check behavior. It will reuse config loading, lock records, `listapp` runtime path/status checks, repo index lookup, source sync, and profile attribution from Phase 1; CLI commands remain thin wrappers around app services. `update --check` will call the status/check path and never reinstall skills or write lock records.

**Tech Stack:** Go, `gookit/gcli`, existing YAML config store, JSON lock store, repo index, source sync service, `listapp`, Go unit tests with `github.com/gookit/goutil/testutil/assert`.

---

| 修订时间 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-14 | v0.1 | Codex | 基于 Phase 1 完成状态和增强设计文档输出 Phase 2 status/update-check 开发计划 |
| 2026-06-14 | v0.2 | Codex | 复审计划示例代码和 Markdown 结构，补齐设计/TODO 互链要求 |

相关文档：

- 设计文档：`docs/design/skillc-v0-enhance-design.md`
- Phase 1 计划：`docs/superpowers/plans/2026-06-13-skillc-v0-phase1-profile.md`
- 任务入口：`docs/TODO.md`

## Phase 2 Scope

本期做：

- 新增 `statusapp`，输出当前项目/agent/scope/profile 维度的状态项。
- 新增 `skillc status` 命令。
- 新增 `skillc update --check`，只检查并输出候选，不修改安装目录和 lock。
- 支持状态：`installed`、`missing`、`outdated`、`orphan`、`unmanaged`、`source-error`。
- `outdated` 的 v0 判定只比较非空 `Version`：lock/current version 与 index/latest version 都非空且不同才算 outdated。
- `orphan` 表示 lock 中有记录，但当前 index 找不到对应 skill identity。
- `unmanaged` 表示安装目录下存在 skill 目录，但 lock 没有记录。
- `source-error` 表示 update check 的 source sync 失败，相关 source 下记录不做 outdated 判定。

本期不做：

- 不做跨项目 install map。
- 不做 checksum drift。
- 不做 git commit/resolved_ref drift。
- 不做 Web。
- 不做 update 执行逻辑重写。
- 不做自动 prune/uninstall。
- 不改变现有 `update` 默认执行行为。

## File Structure

新增文件：

- `internal/app/statusapp/service.go`
  - 定义 status/update-check 查询模型。
  - 加载 config、lock、index。
  - 复用 `listapp` 判断安装目录是否存在。
  - 生成 `StatusItem` 列表和 summary。
  - 可选执行 source sync，并把 sync 失败转成 `source-error` 项。

- `internal/app/statusapp/service_test.go`
  - 覆盖 installed/missing/outdated/orphan/unmanaged/source-error/profile filter/agent filter。

修改文件：

- `internal/app/listapp/service.go`
  - 修复 `ScanUnrecorded(agentName, scope)` 在指定 agent 时仍扫描所有 agent 的问题。

- `internal/app/listapp/service_test.go`
  - 增加指定 agent 时只返回该 agent unrecorded skills 的测试。

- `internal/cli/app.go`
  - 注册 `status` 命令。

- `internal/cli/manage_cmd.go`
  - 新增 `buildStatusCommand()`。
  - 为 `update` 新增 `--check` flag；`--check` 路径调用 statusapp，不调用 updateapp reinstall。

- `internal/cli/app_test.go`
  - 增加 `status` 命令注册、status 输出、update --check 输出、update --check 不调用 updateapp 的 CLI 测试。

- `README.md`
  - 增加 status/update --check 命令说明。

- `README.zh-CN.md`
  - 增加 status/update --check 命令说明。

- `docs/design/skillc-v0-enhance-design.md`
  - 增加本计划链接，记录 Phase 2 实施边界。

- `docs/TODO.md`
  - 增加 Phase 2 计划链接。

## Status Model

本期新增的 app service 类型建议如下：

```go
package statusapp

type Req struct {
	Agent   string
	Scope   string
	Profile string
	WorkDir string
	Sync    bool
}

type Result struct {
	Items       []Item
	SyncFailed []SourceSyncError
	Summary     Summary
}

type Item struct {
	SkillID        string
	QualifiedName  string
	SourceID       string
	Agent          string
	Scope          string
	Profile        string
	Status         string
	CurrentVersion string
	LatestVersion  string
	InstalledPath  string
	Reason         string
}

type Summary struct {
	Installed   int
	Missing     int
	Outdated    int
	Orphan      int
	Unmanaged   int
	SourceError int
}

type SourceSyncError struct {
	SourceID string
	Reason   string
}
```

状态判定顺序：

1. source sync 失败且 item.SourceID 属于失败 source：`source-error`。
2. listapp 返回 `Status == "missing"`：`missing`。
3. index 找不到同 identity skill：`orphan`。
4. 当前版本和最新版本都非空且不同：`outdated`。
5. 其他 lock 记录且安装目录存在：`installed`。
6. 安装目录存在但无 lock 记录：`unmanaged`。

## Task 1: Fix Agent-Scoped Unrecorded Scan

**Files:**
- Modify: `internal/app/listapp/service.go`
- Modify: `internal/app/listapp/service_test.go`

- [x] **Step 1: Write failing test for agent-scoped unrecorded scan**

Append to `internal/app/listapp/service_test.go`:

```go
func TestService_ScanUnrecordedFiltersByRequestedAgent(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	conf := config.DefaultConfig()
	conf.AgentTools["claude-code"] = config.AgentToolConfig{Dirname: ".claude", ProjectDir: filepath.Join(baseDir, ".claude")}
	conf.AgentTools["codex"] = config.AgentToolConfig{Dirname: ".codex", ProjectDir: filepath.Join(baseDir, ".codex")}
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".claude", "skills", "claude-only"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".codex", "skills", "codex-only"), 0o755))

	groups, err := NewService(lockFile).WithRuntime(conf, baseDir).ScanUnrecorded("claude-code", agent.ScopeProject)

	assert.NoErr(t, err)
	assert.Len(t, groups, 1)
	assert.Eq(t, "claude-code", groups[0].AgentName)
	assert.Eq(t, []string{"claude-only"}, groups[0].Skills)
}
```

Add import if needed:

```go
import "github.com/inhere/skillc/internal/domain/agent"
```

- [x] **Step 2: Run listapp tests to verify failure**

Run:

```bash
go test ./internal/app/listapp -run TestService_ScanUnrecordedFiltersByRequestedAgent -count=1
```

Expected: FAIL because `ScanUnrecorded("claude-code", ...)` still scans codex.

- [x] **Step 3: Implement agent filter**

Modify `internal/app/listapp/service.go` inside `ScanUnrecorded` loop:

```go
for name, tool := range rc.AgentTools {
	if agentName != "" {
		canonicalName, _, ok := rc.ResolveAgentTool(agentName)
		if !ok || canonicalName != name {
			continue
		}
	}
	skillsDir, err := resolveSkillsDir(rc, workDir, name, tool, scope)
	if err != nil || skillsDir == "" {
		continue
	}
	// existing body...
}
```

- [x] **Step 4: Run listapp tests**

Run:

```bash
go test ./internal/app/listapp -count=1
```

Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/app/listapp/service.go internal/app/listapp/service_test.go
git commit -m "fix(list): filter unrecorded scan by agent"
```

## Task 2: Add Status App Service

**Files:**
- Create: `internal/app/statusapp/service.go`
- Create: `internal/app/statusapp/service_test.go`

- [ ] **Step 1: Write failing statusapp tests**

Create `internal/app/statusapp/service_test.go`:

```go
package statusapp

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

type syncerStub struct {
	syncFn func(id string) error
}

func (s syncerStub) Sync(id string) error {
	return s.syncFn(id)
}

func TestService_RunClassifiesInstalledMissingOutdatedAndOrphan(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	installedPath := filepath.Join(baseDir, ".agents", "skills", "installed")
	outdatedPath := filepath.Join(baseDir, ".agents", "skills", "outdated")
	orphanPath := filepath.Join(baseDir, ".agents", "skills", "orphan")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))
	assert.NoErr(t, os.MkdirAll(outdatedPath, 0o755))
	assert.NoErr(t, os.MkdirAll(orphanPath, 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	config.Sources = []sourcepkg.Source{{ID: "gstack", Type: sourcepkg.TypeLocal, Path: filepath.Join(baseDir, "source")}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{SkillID: "installed", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}},
			{SkillID: "missing", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}},
			{SkillID: "outdated", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}},
			{SkillID: "orphan", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "installed", SourceID: "gstack", Version: "1.0.0"},
		{ID: "missing", SourceID: "gstack", Version: "1.0.0"},
		{ID: "outdated", SourceID: "gstack", Version: "2.0.0"},
	}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	got := statusBySkill(result.Items)
	assert.Eq(t, "installed", got["installed"])
	assert.Eq(t, "missing", got["missing"])
	assert.Eq(t, "outdated", got["outdated"])
	assert.Eq(t, "orphan", got["orphan"])
	assert.Eq(t, 1, result.Summary.Installed)
	assert.Eq(t, 1, result.Summary.Missing)
	assert.Eq(t, 1, result.Summary.Outdated)
	assert.Eq(t, 1, result.Summary.Orphan)
}

func TestService_RunIncludesUnmanagedInstalledDirectories(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "manual"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Len(t, result.Items, 1)
	assert.Eq(t, "manual", result.Items[0].SkillID)
	assert.Eq(t, "unmanaged", result.Items[0].Status)
	assert.Eq(t, 1, result.Summary.Unmanaged)
}

func TestService_RunFiltersByProfile(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "review"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", Profile: "go-dev", Agents: []string{"universal"}},
			{SkillID: "review", SourceID: "gstack", Version: "1.0.0", Profile: "review", Agents: []string{"universal"}},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Version: "1.0.0"},
		{ID: "review", SourceID: "gstack", Version: "1.0.0"},
	}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", Profile: "go-dev", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Len(t, result.Items, 1)
	assert.Eq(t, "go-pro", result.Items[0].SkillID)
}

func TestService_RunReportsSourceSyncErrorsWithoutUpdatingLock(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	config.Sources = []sourcepkg.Source{{ID: "gstack", Type: sourcepkg.TypeLocal, Path: filepath.Join(baseDir, "source")}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{{ID: "go-pro", SourceID: "gstack", Version: "2.0.0", SourceType: sourcepkg.TypeLocal}}))
	svc := NewService(configFile, baseDir)
	svc.syncer = syncerStub{syncFn: func(id string) error {
		if id == "gstack" {
			return errors.New("sync failed")
		}
		return nil
	}}

	result, err := svc.Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir, Sync: true})

	assert.NoErr(t, err)
	assert.Len(t, result.SyncFailed, 1)
	assert.Eq(t, "source-error", result.Items[0].Status)
	assert.Eq(t, "sync failed", result.Items[0].Reason)
}

func statusBySkill(items []Item) map[string]string {
	out := map[string]string{}
	for _, item := range items {
		out[item.SkillID] = item.Status
	}
	return out
}
```

- [ ] **Step 2: Run statusapp tests to verify failure**

Run:

```bash
go test ./internal/app/statusapp -count=1
```

Expected: FAIL because `internal/app/statusapp` does not exist.

- [ ] **Step 3: Implement statusapp service**

Create `internal/app/statusapp/service.go`:

```go
package statusapp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/inhere/skillc/internal/app/apputil"
	"github.com/inhere/skillc/internal/app/configapp"
	"github.com/inhere/skillc/internal/app/listapp"
	"github.com/inhere/skillc/internal/app/sourceapp"
	"github.com/inhere/skillc/internal/domain/agent"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

const (
	StatusInstalled   = "installed"
	StatusMissing     = "missing"
	StatusOutdated    = "outdated"
	StatusOrphan      = "orphan"
	StatusUnmanaged   = "unmanaged"
	StatusSourceError = "source-error"
)

type sourceSyncer interface {
	Sync(id string) error
}

type Req struct {
	Agent   string
	Scope   string
	Profile string
	WorkDir string
	Sync    bool
}

type Result struct {
	Items       []Item
	SyncFailed []SourceSyncError
	Summary     Summary
}

type Item struct {
	SkillID        string
	QualifiedName  string
	SourceID       string
	Agent          string
	Scope          string
	Profile        string
	Status         string
	CurrentVersion string
	LatestVersion  string
	InstalledPath  string
	Reason         string
}

type Summary struct {
	Installed   int
	Missing     int
	Outdated    int
	Orphan      int
	Unmanaged   int
	SourceError int
}

type SourceSyncError struct {
	SourceID string
	Reason   string
}

type Service struct {
	configFile    string
	baseDir       string
	configService *configapp.Service
	indexStore    *repoindex.Store
	syncer        sourceSyncer
}

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile:    configFile,
		baseDir:       baseDir,
		configService: configapp.NewService(configFile, baseDir),
		indexStore:    repoindex.NewStore(),
		syncer:        sourceapp.NewService(configFile, baseDir),
	}
}

func (s *Service) Run(req Req) (Result, error) {
	config, err := s.configService.Show()
	if err != nil {
		return Result{}, err
	}
	if req.WorkDir == "" {
		req.WorkDir = s.baseDir
	}
	scope, err := apputil.ParseScope(defaultString(req.Scope, string(agent.ScopeProject)))
	if err != nil {
		return Result{}, err
	}
	agentName := defaultString(req.Agent, agent.DefaultAgentName)
	canonicalAgent, _, ok := config.ResolveAgentTool(agentName)
	if !ok {
		return Result{}, fmt.Errorf("unsupported agent: %s", agentName)
	}

	listSvc := listapp.NewService(config.LockFile).WithRuntime(config, req.WorkDir)
	listItems, err := listSvc.List(canonicalAgent, string(scope))
	if err != nil {
		return Result{}, err
	}
	listItems = filterByProfile(listItems, req.Profile)

	result := Result{}
	syncFailed := map[string]string{}
	if req.Sync {
		result.SyncFailed = s.syncSources(config.Sources)
		for _, failed := range result.SyncFailed {
			syncFailed[failed.SourceID] = failed.Reason
		}
	}

	indexItems, err := s.loadIndex(config.IndexFile)
	if err != nil {
		return Result{}, err
	}
	for _, current := range listItems {
		result.Items = append(result.Items, classifyListItem(current, indexItems, syncFailed))
	}
	unmanaged, err := listSvc.ScanUnrecorded(canonicalAgent, scope)
	if err != nil {
		return Result{}, err
	}
	for _, group := range unmanaged {
		for _, skillID := range group.Skills {
			result.Items = append(result.Items, Item{
				SkillID: skillID,
				Agent:   group.AgentName,
				Scope:   string(scope),
				Status:  StatusUnmanaged,
				Reason:  "installed directory has no lock record",
			})
		}
	}
	sortItems(result.Items)
	result.Summary = summarize(result.Items)
	return result, nil
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

func (s *Service) syncSources(sources []sourcepkg.Source) []SourceSyncError {
	out := make([]SourceSyncError, 0)
	for _, source := range sources {
		id := source.ID
		if id == "" {
			continue
		}
		if err := s.syncer.Sync(id); err != nil {
			out = append(out, SourceSyncError{SourceID: id, Reason: err.Error()})
		}
	}
	return out
}

func classifyListItem(current listapp.Item, indexItems []skill.Skill, syncFailed map[string]string) Item {
	item := Item{
		SkillID:        current.SkillID,
		QualifiedName:  current.QualifiedName,
		SourceID:       current.SourceID,
		Agent:          current.Agent,
		Scope:          current.Scope,
		Profile:        current.Profile,
		CurrentVersion: current.Version,
		InstalledPath:  current.InstalledPath,
	}
	if reason, ok := syncFailed[current.SourceID]; ok {
		item.Status = StatusSourceError
		item.Reason = reason
		return item
	}
	if current.Status == StatusMissing {
		item.Status = StatusMissing
		item.Reason = "installed path is missing"
		return item
	}
	latest, ok := findLatest(indexItems, current)
	if !ok {
		item.Status = StatusOrphan
		item.Reason = "skill not found in source index"
		return item
	}
	item.LatestVersion = latest.Version
	if current.Version != "" && latest.Version != "" && current.Version != latest.Version {
		item.Status = StatusOutdated
		item.Reason = fmt.Sprintf("version %s -> %s", current.Version, latest.Version)
		return item
	}
	item.Status = StatusInstalled
	return item
}

func findLatest(items []skill.Skill, current listapp.Item) (skill.Skill, bool) {
	for _, item := range items {
		if current.SkillID != item.ID {
			continue
		}
		if current.SourceID != "" || item.SourceID != "" {
			if current.SourceID != "" && current.SourceID == item.SourceID {
				return item, true
			}
			continue
		}
		if current.SourceQualifiedName != "" || item.SourceQualifiedName != "" {
			if current.SourceQualifiedName != "" && current.SourceQualifiedName == item.SourceQualifiedName {
				return item, true
			}
			continue
		}
		if current.QualifiedName != "" || item.QualifiedName != "" {
			if current.QualifiedName != "" && current.QualifiedName == item.QualifiedName {
				return item, true
			}
			continue
		}
	}
	return skill.Skill{}, false
}

func filterByProfile(items []listapp.Item, profileName string) []listapp.Item {
	if profileName == "" {
		return items
	}
	out := make([]listapp.Item, 0, len(items))
	for _, item := range items {
		if item.Profile == profileName {
			out = append(out, item)
		}
	}
	return out
}

func sortItems(items []Item) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Status != items[j].Status {
			return items[i].Status < items[j].Status
		}
		if items[i].SkillID != items[j].SkillID {
			return items[i].SkillID < items[j].SkillID
		}
		return filepath.Clean(items[i].InstalledPath) < filepath.Clean(items[j].InstalledPath)
	})
}

func summarize(items []Item) Summary {
	var summary Summary
	for _, item := range items {
		switch item.Status {
		case StatusInstalled:
			summary.Installed++
		case StatusMissing:
			summary.Missing++
		case StatusOutdated:
			summary.Outdated++
		case StatusOrphan:
			summary.Orphan++
		case StatusUnmanaged:
			summary.Unmanaged++
		case StatusSourceError:
			summary.SourceError++
		}
	}
	return summary
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
```

- [ ] **Step 4: Run statusapp tests**

Run:

```bash
go test ./internal/app/statusapp -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/statusapp/service.go internal/app/statusapp/service_test.go
git commit -m "feat(status): add status query service"
```

## Task 3: Add Status CLI

**Files:**
- Modify: `internal/cli/app.go`
- Modify: `internal/cli/manage_cmd.go`
- Modify: `internal/cli/app_test.go`

- [ ] **Step 1: Add failing CLI tests**

Add to `internal/cli/app_test.go`:

```go
func TestNewApp_RegistersStatusCommand(t *testing.T) {
	app := newTestApp()

	status := findCommandByName(app, "status")
	assert.NotNil(t, status)
	if status == nil {
		return
	}
	assert.Eq(t, "Show skill status", status.Desc)
}

func TestStatusCommand_PrintsSkillHealth(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", Profile: "go-dev", Agents: []string{"universal"}}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{{ID: "go-pro", SourceID: "gstack", Version: "2.0.0"}}))

	output := runAppInDirWithStdout(t, baseDir, []string{"status", "--agent", "universal", "--scope", "project", "--profile", "go-dev"})

	assert.Contains(t, output, "Skill Status")
	assert.Contains(t, output, "outdated")
	assert.Contains(t, output, "go-pro")
	assert.Contains(t, output, "go-dev")
	assert.Contains(t, output, "1.0.0")
	assert.Contains(t, output, "2.0.0")
}
```

- [ ] **Step 2: Run CLI tests to verify failure**

Run:

```bash
go test ./internal/cli -run "TestNewApp_RegistersStatusCommand|TestStatusCommand_PrintsSkillHealth" -count=1
```

Expected: FAIL because `status` command is not registered.

- [ ] **Step 3: Register status command**

Modify `internal/cli/app.go`:

```go
app.Add(buildStatusCommand())
```

Place it after `buildListCommand()` and before `buildDoctorCommand()`.

- [ ] **Step 4: Implement buildStatusCommand**

Modify imports in `internal/cli/manage_cmd.go`:

```go
import (
	// existing imports...
	"github.com/inhere/skillc/internal/app/statusapp"
)
```

Add:

```go
func buildStatusCommand() *gcli.Command {
	var opts ManageOptions
	var profileName string
	return &gcli.Command{
		Name: "status",
		Desc: "Show skill status",
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
			c.StrOpt(&profileName, "profile", "p", "", "profile name")
		},
		Func: func(c *gcli.Command, _ []string) error {
			config, cwd, err := loadConfig()
			if err != nil {
				return err
			}
			scope, err := parseScope(opts.Scope)
			if err != nil {
				return err
			}
			result, err := statusapp.NewService(defaultConfigFile(cwd), cwd).Run(statusapp.Req{
				Agent:   opts.Agent,
				Scope:   string(scope),
				Profile: profileName,
				WorkDir: cwd,
			})
			if err != nil {
				return err
			}
			return printStatusResult(result, config)
		},
	}
}

func printStatusResult(result statusapp.Result, _ cfg.Config) error {
	if len(result.Items) == 0 {
		ccolor.Warnln("no skills found")
		return nil
	}
	tb := table.New("Skill Status").SetHeads("Status", "Skill", "Source", "Agent", "Scope", "Profile", "Current", "Latest", "Reason")
	for _, item := range result.Items {
		tb.AddRow(item.Status, item.SkillID, item.SourceID, item.Agent, item.Scope, item.Profile, item.CurrentVersion, item.LatestVersion, item.Reason)
	}
	if _, err := fmt.Fprint(os.Stdout, tb.Render()); err != nil {
		return err
	}
	ccolor.Infof("summary installed=%d missing=%d outdated=%d orphan=%d unmanaged=%d source_error=%d\n",
		result.Summary.Installed,
		result.Summary.Missing,
		result.Summary.Outdated,
		result.Summary.Orphan,
		result.Summary.Unmanaged,
		result.Summary.SourceError,
	)
	return nil
}
```

- [ ] **Step 5: Run CLI tests**

Run:

```bash
go test ./internal/cli -run "TestNewApp_RegistersStatusCommand|TestStatusCommand_PrintsSkillHealth" -count=1
go test ./internal/cli -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/app.go internal/cli/manage_cmd.go internal/cli/app_test.go
git commit -m "feat(cli): add status command"
```

## Task 4: Add Update Check CLI

**Files:**
- Modify: `internal/cli/manage_cmd.go`
- Modify: `internal/cli/app_test.go`

- [ ] **Step 1: Add failing update --check CLI tests**

Add to `internal/cli/app_test.go`:

```go
func TestUpdateCommand_CheckPrintsCandidatesWithoutCallingUpdateRunner(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{{ID: "go-pro", SourceID: "gstack", Version: "2.0.0"}}))

	calledUpdateRunner := false
	prevFactory := newUpdateService
	newUpdateService = func(configFile string, baseDir string) updateRunner {
		return updateRunnerStub{runFn: func(req updateapp.Req) (updateapp.Result, error) {
			calledUpdateRunner = true
			return updateapp.Result{}, nil
		}}
	}
	defer func() {
		newUpdateService = prevFactory
	}()

	output := runAppInDirWithStdout(t, baseDir, []string{"update", "--check", "--agent", "universal"})

	assert.Contains(t, output, "Update Check")
	assert.Contains(t, output, "outdated")
	assert.Contains(t, output, "go-pro")
	assert.False(t, calledUpdateRunner)
}
```

- [ ] **Step 2: Run CLI tests to verify failure**

Run:

```bash
go test ./internal/cli -run TestUpdateCommand_CheckPrintsCandidatesWithoutCallingUpdateRunner -count=1
```

Expected: FAIL because `--check` flag does not exist.

- [ ] **Step 3: Add --check option and branch**

Modify `buildUpdateCommand()` in `internal/cli/manage_cmd.go`:

```go
func buildUpdateCommand() *gcli.Command {
	var opts ManageOptions
	var target string
	var checkOnly bool
	return &gcli.Command{
		Name:    "update",
		Desc:    "Update installed skills",
		Aliases: []string{"up"},
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
			c.StrOpt(&target, "target", "t", "", "skill id to update (default: update all)")
			c.BoolOpt(&checkOnly, "check", "", false, "check update candidates without installing")
			c.AddArg("skill", "skill id to update (same as --target)")
		},
		Func: func(c *gcli.Command, _ []string) error {
			_, cwd, err := loadConfig()
			if err != nil {
				return err
			}
			if target == "" {
				target = c.Arg("skill").String()
			}
			if checkOnly {
				result, err := statusapp.NewService(defaultConfigFile(cwd), cwd).Run(statusapp.Req{
					Agent:   opts.Agent,
					Scope:   opts.Scope,
					WorkDir: cwd,
					Sync:    true,
				})
				if err != nil {
					return err
				}
				return printUpdateCheckResult(result, target)
			}

			// existing update execution path...
		},
	}
}
```

Add helper:

```go
func printUpdateCheckResult(result statusapp.Result, target string) error {
	items := make([]statusapp.Item, 0, len(result.Items))
	for _, item := range result.Items {
		if target != "" && item.SkillID != target && item.QualifiedName != target {
			continue
		}
		if item.Status == statusapp.StatusInstalled {
			continue
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		ccolor.Successln("no update candidates")
		return nil
	}
	tb := table.New("Update Check").SetHeads("Status", "Skill", "Source", "Agent", "Current", "Latest", "Reason")
	for _, item := range items {
		tb.AddRow(item.Status, item.SkillID, item.SourceID, item.Agent, item.CurrentVersion, item.LatestVersion, item.Reason)
	}
	_, err := fmt.Fprint(os.Stdout, tb.Render())
	return err
}
```

- [ ] **Step 4: Run update check CLI tests**

Run:

```bash
go test ./internal/cli -run TestUpdateCommand_CheckPrintsCandidatesWithoutCallingUpdateRunner -count=1
go test ./internal/cli -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/manage_cmd.go internal/cli/app_test.go
git commit -m "feat(cli): add update check"
```

## Task 5: Update Documentation

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/design/skillc-v0-enhance-design.md`
- Modify: `docs/TODO.md`
- Modify: `docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md`

规划阶段已补齐 `docs/design/skillc-v0-enhance-design.md` 与 `docs/TODO.md` 到本计划的互链。实现阶段仍需更新 README 命令手册，并在最终验收后复核这些链接和状态描述。

- [ ] **Step 1: Update README command reference**

In `README.md`, add:

````markdown
### `status` — Skill health

```bash
skillc status                         # show current project skill status
skillc status --profile go-dev        # filter by profile
skillc status --agent claude-code     # filter by agent
```
````

In the `update` section, add:

```markdown
skillc update --check                 # preview update candidates without installing
```

- [ ] **Step 2: Update Chinese README command reference**

In `README.zh-CN.md`, add equivalent Chinese text:

````markdown
### `status` — Skill 状态

```bash
skillc status                         # 查看当前项目 Skill 状态
skillc status --profile go-dev        # 按 Profile 过滤
skillc status --agent claude-code     # 按 Agent 过滤
```
````

In update section, add:

```markdown
skillc update --check                 # 只预览更新候选，不安装
```

- [ ] **Step 3: Update design document links**

Add revision row to `docs/design/skillc-v0-enhance-design.md`:

```markdown
| 2026-06-14 | v0.7 | Codex | 增加 Phase 2 status/update-check 实施计划链接 |
```

In related docs section, add:

```markdown
- `docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md`
```

Under Phase 2 section, add:

```markdown
实施计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md`
```

- [ ] **Step 4: Update TODO links**

In `docs/TODO.md`, add:

```markdown
二期计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md`
```

- [ ] **Step 5: Mark this plan's documentation task complete**

Update this Task 5 checklist to checked when the docs are updated.

- [ ] **Step 6: Commit**

```bash
git add README.md README.zh-CN.md docs/design/skillc-v0-enhance-design.md docs/TODO.md docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md
git commit -m "docs: add skillc phase 2 status plan"
```

## Task 6: Full Regression And Plan Review

**Files:**
- Modify: `docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md`

- [ ] **Step 1: Run full tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 2: Run docs self-check**

Run:

```bash
rg -n -- "status|update --check|Phase 2|二期计划" README.md README.zh-CN.md docs/TODO.md docs/design/skillc-v0-enhance-design.md docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md
```

Expected: docs contain the new status/update-check references.

- [ ] **Step 3: Update final verification note**

Append to this plan:

```markdown
**Verification note (2026-06-14):** `go test ./...` passes. README, README.zh-CN, TODO, and design docs reference Phase 2 status/update-check behavior and implementation plan.
```

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md
git commit -m "docs: complete skillc phase 2 status plan review"
```

## Self-Review

Spec coverage:

- `status` command: Task 3.
- `update --check`: Task 4.
- installed/missing/outdated/orphan/unmanaged/source-error statuses: Task 2.
- profile/agent/scope filters: Task 2 and Task 3.
- no install/lock mutation in update check: Task 4 tests that `newUpdateService` is not called; statusapp only reads lock/index and optionally syncs sources.
- docs and plan links: Task 5.
- full regression: Task 6.

Out of scope:

- Cross-project install map.
- Web.
- Registry.
- Git resolved ref drift.
- Checksum drift.
- Automatic update execution changes.
- Prune/uninstall.

Failure modes checked:

- Missing installed directory becomes `missing`.
- Latest version mismatch becomes `outdated`.
- Missing index identity becomes `orphan`.
- Directory without lock record becomes `unmanaged`.
- Source sync failure becomes `source-error`.
- Agent filter prevents unmanaged entries from other agent directories.
- `update --check` does not call the mutating update runner.

Placeholder scan:

- No unresolved placeholders.
- Each task has concrete files, tests, commands, and commit steps.

Type consistency:

- `statusapp.Req`, `statusapp.Result`, `statusapp.Item`, `statusapp.Summary`, and `statusapp.SourceSyncError` are defined before CLI tasks use them.
- CLI helpers use `statusapp.StatusInstalled` constants defined in Task 2.
