# Skillc v0 Phase 6 Current-Project Web Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 补齐 `skillc web` 在当前项目范围内的 source 管理、profile 管理、skill 卸载和最小操作历史，让 Web 从“能谨慎 apply/update”推进到“能完成当前项目常用管理闭环”。

**Architecture:** 延续 Phase 4/5 的 thin handler 模式，HTTP handler 只负责请求解析、确认校验、JSON 转换和 history 记录，实际写操作继续通过 `sourceapp`、`profileapp`、`installapp` 等 app service 完成。所有 Web 写操作坚持 plan-first：先返回稳定 plan JSON，再由 run endpoint 要求 `confirm:true` 后执行。第一轮只操作 `skillc web` 启动目录对应的当前项目，不引入跨项目批量更新、Registry、远程权限模型或后台队列。

**Tech Stack:** Go, `net/http`, embedded static HTML/CSS/JS via Go string, existing app/domain/infra packages, JSONL history file, Go unit tests with `github.com/gookit/goutil/testutil/assert`, HTTP handler tests with `httptest`, full verification via `go test ./...`.

---

| 修订时间 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-15 | v0.1 | Codex | 基于 Phase 5 完成状态输出 Phase 6 当前项目 Web 管理补齐实施计划 |

相关文档：

- 设计文档：`docs/design/skillc-v0-enhance-design.md`
- 参考分析：`docs/design/skillc-reference-projects-analysis.md`
- Phase 1 计划：`docs/superpowers/plans/2026-06-13-skillc-v0-phase1-profile.md`
- Phase 2 计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md`
- Phase 3 计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase3-interactive-selection.md`
- Phase 4 计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase4-web-management.md`
- Phase 5 计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase5-web-execution.md`
- 任务入口：`docs/TODO.md`

## Phase 6 Scope

本期做：

- Web source 管理：
  - 新增 source add plan/run，支持 local path 或 git URL，可选 `ref` 和 `sync`。
  - 新增 source sync plan/run，支持单个 source 和 all sources。
  - 新增 source remove plan/run，删除前展示影响：已安装记录、关联 profile targets、索引 skills/collections 数量。
- Web profile 管理：
  - 新增 profile save plan/run，覆盖创建和编辑。
  - 新增 profile from-installed plan/run，把当前项目已安装 skills 保存为 profile。
  - 新增 profile from-collection plan/run，把 `<source>/<collection>` 保存为 profile。
  - Web UI 提供 profile 表单、targets 文本编辑和 from-installed/from-collection 快捷入口。
- Web uninstall：
  - 新增 uninstall plan/run，先展示将删除的 agent/scope/path/lock record，再要求 `confirm:true`。
  - 当前只支持当前项目或当前 user scope，不做跨项目卸载。
- Web history：
  - 新增最小 JSONL history store。
  - 记录 Web 写操作的时间、action、agent、scope、workdir、request、plan、result/error。
  - 新增 `GET /api/history` 和 Web History 视图。
- 更新 README、README.zh-CN、docs/TODO.md、设计文档和本计划验证记录。

本期不做：

- 不做跨项目批量更新所有下游项目。
- 不做 project registry / project selection / per-project confirmation。
- 不做 Registry。
- 不做远程访问、登录、token、多用户权限。
- 不做后台 job 队列。
- 不做 source ID 去掉 `local-` / `git-` 前缀；该项单独放到后续 Source UX 阶段。
- 不做 checksum / git commit drift 的精确校验。
- 不改变现有 CLI 命令语义，除非新增 app service 方法需要 CLI 继续兼容。

## User-Facing Behavior

### Source Add

在 Sources 页输入 local path 或 git URL：

```json
{
  "value": "../team-skills",
  "ref": "",
  "sync": true
}
```

点击 `Plan add` 后，Web 调用：

```http
POST /api/sources/add/plan
Content-Type: application/json
```

返回：

```json
{
  "action": "add_local",
  "source": {
    "id": "local-team-skills",
    "type": "local",
    "name": "team-skills",
    "path": "<absolute-path-to-team-skills>"
  },
  "items": [
    {"action":"add","target":"local-team-skills","reason":"source is not configured"},
    {"action":"sync","target":"local-team-skills","reason":"sync requested"}
  ]
}
```

用户确认后点击 `Run source action`，Web 调用：

```json
{
  "confirm": true,
  "value": "../team-skills",
  "sync": true
}
```

### Source Sync / Remove

Source 表格每行提供 `Plan sync` 和 `Plan remove`。

Remove plan 必须展示影响：

```json
{
  "action": "remove",
  "source_id": "gstack",
  "impact": {
    "installed_count": 2,
    "profile_target_count": 1,
    "indexed_skill_count": 12,
    "collection_count": 3
  },
  "warnings": [
    "2 installed lock record(s) reference this source",
    "1 profile target(s) reference this source"
  ]
}
```

### Profile Save

Profiles 页提供表单：

- Name
- Description
- Default Agent
- Default Scope
- Install Mode
- Targets textarea

Targets textarea 每行一种格式：

```text
gstack go-pro
gstack review
team source-qualified/skill
```

保存前调用 `POST /api/profiles/save/plan`，返回 added/removed/kept 目标差异。确认后调用 `POST /api/profiles/save/run`。

### Profile From Installed / Collection

快捷入口：

```json
{"name":"go-dev","agent":"universal","scope":"project"}
```

或：

```json
{"name":"go-dev","selector":"gstack/go"}
```

先 plan，确认后保存。若 profile 已存在，run endpoint 返回 `409`，用户需要改名或使用 profile save 编辑。

### Uninstall

Projects / Install Map 和 Status 视图里的 installed skill 提供 `Plan uninstall`。计划展示：

```json
{
  "agent": "universal",
  "scope": "project",
  "items": [
    {
      "action": "remove_record",
      "skill_id": "go-pro",
      "source_id": "gstack",
      "version": "1.0.0",
      "agent": "universal",
      "scope": "project",
      "installed_path": ".agents/skills/go-pro"
    }
  ]
}
```

用户确认后调用 `POST /api/uninstall/run`，请求体必须包含 `confirm:true`。

### History

History 页显示最近 100 条 Web 写操作：

- time
- action
- agent
- scope
- status
- target summary
- error

History 用于当前本地管理排查，不作为安全审计或多用户审计系统。

## File Structure

新增文件：

- `internal/app/webapp/manager_source_actions.go`
  - Web source add/sync/remove 的 plan/run service。
  - 计算 source remove impact。

- `internal/app/webapp/manager_source_actions_test.go`
  - 覆盖 source add plan、sync plan、remove impact、run action 返回稳定 JSON 所需字段。

- `internal/app/webapp/manager_profile_actions.go`
  - Web profile save/from-installed/from-collection 的 plan/run service。
  - 复用 `profileapp` 新增 plan/build 方法。

- `internal/app/webapp/manager_profile_actions_test.go`
  - 覆盖 profile save plan、from-installed plan、from-collection plan、重复 profile 冲突。

- `internal/app/webapp/manager_uninstall_actions.go`
  - Web uninstall plan/run service。
  - 复用 `installapp.PlanUninstall` 和 `installapp.RunUninstall`。

- `internal/app/webapp/manager_uninstall_actions_test.go`
  - 覆盖 uninstall plan、run result、partial/no target 行为。

- `internal/app/webapp/history.go`
  - JSONL history store。
  - 提供 append/list 最近记录。

- `internal/app/webapp/history_test.go`
  - 覆盖 append、list order、invalid line tolerant read。

修改文件：

- `internal/app/profileapp/service.go`
  - 新增 profile save plan、build from installed、build from collection 方法。

- `internal/app/profileapp/service_test.go`
  - 覆盖新增 profile planning/build 方法。

- `internal/app/installapp/service.go`
  - 新增 uninstall plan/run 结果模型。
  - 现有 `UninstallMulti` 保持兼容。

- `internal/app/installapp/service_test.go`
  - 覆盖 uninstall plan 和 run result。

- `internal/app/webapp/manager.go`
  - 增加 history path 和 action service helper。

- `internal/app/webapp/manager_server.go`
  - 新增 source/profile/uninstall/history routes。
  - run endpoints 统一要求 `confirm:true`。

- `internal/app/webapp/manager_server_test.go`
  - 覆盖新 API、method validation、confirm guard、history route。

- `internal/app/webapp/manager_static.go`
  - 增加 Sources 管理控件、Profiles 表单、Uninstall 按钮、History 视图。

- `README.md`
  - 更新 Web 当前项目管理能力说明。

- `README.zh-CN.md`
  - 更新中文 Web 能力边界和安全确认说明。

- `docs/TODO.md`
  - 增加 Phase 6 计划链接和状态。

- `docs/design/skillc-v0-enhance-design.md`
  - 增加 Phase 6 计划链接、修订记录和阶段说明。

- `docs/superpowers/plans/2026-06-15-skillc-v0-phase6-web-current-project-management.md`
  - 实施中持续更新 checkbox 和验证记录。

## API Model

新增 routes：

```text
POST /api/sources/add/plan
POST /api/sources/add/run
POST /api/sources/sync/plan
POST /api/sources/sync/run
POST /api/sources/remove/plan
POST /api/sources/remove/run

POST /api/profiles/save/plan
POST /api/profiles/save/run
POST /api/profiles/from-installed/plan
POST /api/profiles/from-installed/run
POST /api/profiles/from-collection/plan
POST /api/profiles/from-collection/run

POST /api/uninstall/plan
POST /api/uninstall/run

GET /api/history
```

Run endpoints 的通用规则：

- 请求体必须是 JSON。
- `confirm != true` 返回 `400 {"error":"confirmation required"}`。
- JSON 解析失败返回 `400 {"error":"invalid json body"}`。
- GET 写操作返回 `405 {"error":"method not allowed"}`。
- run 成功后刷新 UI 数据，并写入 history。
- run 失败但已有部分 plan/result 时，返回 `200`，body 中包含 `error` 字段；完全无法解析或无法执行时返回 4xx/5xx。

建议新增核心类型：

```go
package webapp

type sourceActionReq struct {
	Confirm bool   `json:"confirm,omitempty"`
	ID      string `json:"id,omitempty"`
	Value   string `json:"value,omitempty"`
	Ref     string `json:"ref,omitempty"`
	Sync    bool   `json:"sync,omitempty"`
}

type sourceActionPlan struct {
	Action   string             `json:"action"`
	SourceID string             `json:"source_id,omitempty"`
	Source   sourcepkg.Source   `json:"source,omitempty"`
	Existing bool               `json:"existing,omitempty"`
	Impact   sourceRemoveImpact `json:"impact,omitempty"`
	Items    []sourcePlanItem   `json:"items"`
	Warnings []string           `json:"warnings,omitempty"`
}

type sourcePlanItem struct {
	Action string `json:"action"`
	Target string `json:"target"`
	Reason string `json:"reason,omitempty"`
}

type sourceRemoveImpact struct {
	InstalledCount     int `json:"installed_count"`
	ProfileTargetCount int `json:"profile_target_count"`
	IndexedSkillCount  int `json:"indexed_skill_count"`
	CollectionCount    int `json:"collection_count"`
}

type profileSaveReq struct {
	Confirm      bool             `json:"confirm,omitempty"`
	Name         string           `json:"name"`
	Description  string           `json:"description,omitempty"`
	DefaultAgent string           `json:"default_agent,omitempty"`
	DefaultScope string           `json:"default_scope,omitempty"`
	InstallMode  string           `json:"install_mode,omitempty"`
	Targets      []profile.Target `json:"targets"`
}

type profileSavePlan struct {
	Name    string           `json:"name"`
	Mode    string           `json:"mode"`
	Profile profile.Profile  `json:"profile"`
	Added   []profile.Target `json:"added,omitempty"`
	Removed []profile.Target `json:"removed,omitempty"`
	Kept    []profile.Target `json:"kept,omitempty"`
}

type profileFromInstalledReq struct {
	Confirm bool   `json:"confirm,omitempty"`
	Name    string `json:"name"`
	Agent   string `json:"agent,omitempty"`
	Scope   string `json:"scope,omitempty"`
}

type profileFromCollectionReq struct {
	Confirm  bool   `json:"confirm,omitempty"`
	Name     string `json:"name"`
	Selector string `json:"selector"`
}

type uninstallActionReq struct {
	Confirm bool     `json:"confirm,omitempty"`
	Skills  []string `json:"skills"`
	Agent   string   `json:"agent,omitempty"`
	Scope   string   `json:"scope,omitempty"`
}
```

## Task 1: Add Source Action Manager Service

**Files:**
- Create: `internal/app/webapp/manager_source_actions.go`
- Create: `internal/app/webapp/manager_source_actions_test.go`
- Modify: `internal/app/webapp/manager.go`

- [x] **Step 1: Write failing tests for source plans**

Add tests to `internal/app/webapp/manager_source_actions_test.go`:

```go
package webapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/profile"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

func TestManager_PlanSourceAddLocal(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	sourceDir := filepath.Join(baseDir, "team-skills")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "go-pro"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "go-pro", "SKILL.md"), []byte("# Go Pro\n"), 0o644))

	plan, err := NewManager(configFile, baseDir).PlanSourceAdd(sourceActionReq{Value: sourceDir, Sync: true})

	assert.NoErr(t, err)
	assert.Eq(t, "add_local", plan.Action)
	assert.Eq(t, sourcepkg.TypeLocal, plan.Source.Type)
	assert.Eq(t, false, plan.Existing)
	assert.Eq(t, "add", plan.Items[0].Action)
	assert.Eq(t, "sync", plan.Items[1].Action)
}

func TestManager_PlanSourceAddExistingLocal(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	config, err := configstore.NewYAMLStore().Load(configFile, baseDir)
	assert.NoErr(t, err)
	existingPath := config.Sources[0].Path

	plan, err := NewManager(configFile, baseDir).PlanSourceAdd(sourceActionReq{Value: existingPath})

	assert.NoErr(t, err)
	assert.Eq(t, "exists", plan.Action)
	assert.Eq(t, true, plan.Existing)
	assert.Eq(t, config.Sources[0].ID, plan.Source.ID)
}

func TestManager_PlanSourceRemoveIncludesImpact(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	config, err := configstore.NewYAMLStore().Load(configFile, baseDir)
	assert.NoErr(t, err)
	config.Profiles["ops"] = profile.Profile{Targets: []profile.Target{{Source: "gstack", Skill: "review"}}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(config.LockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID:  "review",
			SourceID: "gstack",
			Version:  "1.0.0",
			Agents:   []string{"universal"},
		}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(config.IndexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Collection: "go"},
		{ID: "review", SourceID: "gstack", Collection: "ops"},
	}))

	plan, err := NewManager(configFile, baseDir).PlanSourceRemove(sourceActionReq{ID: "gstack"})

	assert.NoErr(t, err)
	assert.Eq(t, "remove", plan.Action)
	assert.Eq(t, "gstack", plan.SourceID)
	assert.Eq(t, 1, plan.Impact.InstalledCount)
	assert.Eq(t, 1, plan.Impact.ProfileTargetCount)
	assert.Eq(t, 2, plan.Impact.IndexedSkillCount)
	assert.Eq(t, 2, plan.Impact.CollectionCount)
	assert.Contains(t, plan.Warnings[0], "installed lock")
}
```

- [x] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/app/webapp -run 'TestManager_PlanSource(Add|Remove)' -v
```

Expected: FAIL because `PlanSourceAdd`, `PlanSourceRemove`, `sourceActionReq`, and `sourceActionPlan` are not defined.

- [x] **Step 3: Implement source action planning**

Create `internal/app/webapp/manager_source_actions.go`:

```go
package webapp

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/inhere/skillc/internal/app/sourceapp"
	"github.com/inhere/skillc/internal/domain/profile"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
)

type sourceActionReq struct {
	Confirm bool   `json:"confirm,omitempty"`
	ID      string `json:"id,omitempty"`
	Value   string `json:"value,omitempty"`
	Ref     string `json:"ref,omitempty"`
	Sync    bool   `json:"sync,omitempty"`
}

type sourceActionPlan struct {
	Action   string             `json:"action"`
	SourceID string             `json:"source_id,omitempty"`
	Source   sourcepkg.Source   `json:"source,omitempty"`
	Existing bool               `json:"existing,omitempty"`
	Impact   sourceRemoveImpact `json:"impact,omitempty"`
	Items    []sourcePlanItem   `json:"items"`
	Warnings []string           `json:"warnings,omitempty"`
}

type sourcePlanItem struct {
	Action string `json:"action"`
	Target string `json:"target"`
	Reason string `json:"reason,omitempty"`
}

type sourceRemoveImpact struct {
	InstalledCount     int `json:"installed_count"`
	ProfileTargetCount int `json:"profile_target_count"`
	IndexedSkillCount  int `json:"indexed_skill_count"`
	CollectionCount    int `json:"collection_count"`
}

func (m *Manager) PlanSourceAdd(req sourceActionReq) (sourceActionPlan, error) {
	value := strings.TrimSpace(req.Value)
	if value == "" {
		return sourceActionPlan{}, fmt.Errorf("source value is required")
	}
	config, err := m.config()
	if err != nil {
		return sourceActionPlan{}, err
	}
	if sourceapp.IsGitURL(value) {
		for _, src := range config.Sources {
			if src.Type == sourcepkg.TypeGit && src.URL == value {
				return sourceActionPlan{Action: "exists", SourceID: src.ID, Source: src, Existing: true, Items: []sourcePlanItem{{Action: "skip", Target: src.ID, Reason: "source already exists"}}}, nil
			}
		}
		src, err := sourcepkg.NewGitSource(value, req.Ref)
		if err != nil {
			return sourceActionPlan{}, err
		}
		return sourceAddPlan("add_git", src, req.Sync), nil
	}
	src, err := sourcepkg.NewLocalSource(value)
	if err != nil {
		return sourceActionPlan{}, err
	}
	for _, current := range config.Sources {
		if current.Type == sourcepkg.TypeLocal && filepath.Clean(current.Path) == filepath.Clean(src.Path) {
			return sourceActionPlan{Action: "exists", SourceID: current.ID, Source: current, Existing: true, Items: []sourcePlanItem{{Action: "skip", Target: current.ID, Reason: "source already exists"}}}, nil
		}
	}
	return sourceAddPlan("add_local", src, req.Sync), nil
}

func sourceAddPlan(action string, src sourcepkg.Source, sync bool) sourceActionPlan {
	items := []sourcePlanItem{{Action: "add", Target: src.ID, Reason: "source is not configured"}}
	if sync {
		items = append(items, sourcePlanItem{Action: "sync", Target: src.ID, Reason: "sync requested"})
	}
	return sourceActionPlan{Action: action, SourceID: src.ID, Source: src, Items: items}
}

func (m *Manager) PlanSourceSync(req sourceActionReq) (sourceActionPlan, error) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		return sourceActionPlan{}, fmt.Errorf("source id is required")
	}
	if id == "__all__" || id == "all" {
		return sourceActionPlan{Action: "sync_all", Items: []sourcePlanItem{{Action: "sync", Target: "all", Reason: "sync all configured sources"}}}, nil
	}
	src, err := m.findSource(id)
	if err != nil {
		return sourceActionPlan{}, err
	}
	return sourceActionPlan{Action: "sync", SourceID: src.ID, Source: src, Items: []sourcePlanItem{{Action: "sync", Target: src.ID}}}, nil
}

func (m *Manager) PlanSourceRemove(req sourceActionReq) (sourceActionPlan, error) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		return sourceActionPlan{}, fmt.Errorf("source id is required")
	}
	src, err := m.findSource(id)
	if err != nil {
		return sourceActionPlan{}, err
	}
	impact, warnings, err := m.sourceRemoveImpact(src.ID)
	if err != nil {
		return sourceActionPlan{}, err
	}
	return sourceActionPlan{
		Action:   "remove",
		SourceID: src.ID,
		Source:   src,
		Impact:   impact,
		Warnings: warnings,
		Items:    []sourcePlanItem{{Action: "remove", Target: src.ID, Reason: "source config and indexed skills will be removed"}},
	}, nil
}

func (m *Manager) findSource(id string) (sourcepkg.Source, error) {
	items, err := m.Sources()
	if err != nil {
		return sourcepkg.Source{}, err
	}
	for _, src := range items {
		if src.ID == id {
			return src, nil
		}
	}
	return sourcepkg.Source{}, fmt.Errorf("source not found: %s", id)
}

func (m *Manager) sourceRemoveImpact(sourceID string) (sourceRemoveImpact, []string, error) {
	config, err := m.config()
	if err != nil {
		return sourceRemoveImpact{}, nil, err
	}
	records, err := loadLock(config.LockFile)
	if err != nil {
		return sourceRemoveImpact{}, nil, err
	}
	indexItems, err := loadIndex(config.IndexFile)
	if err != nil {
		return sourceRemoveImpact{}, nil, err
	}
	impact := sourceRemoveImpact{}
	for _, recordsInScope := range records {
		for _, record := range recordsInScope {
			if record.SourceID == sourceID {
				impact.InstalledCount++
			}
		}
	}
	for _, item := range config.Profiles {
		for _, target := range item.Targets {
			if target.Source == sourceID {
				impact.ProfileTargetCount++
			}
		}
	}
	collections := map[string]struct{}{}
	for _, item := range indexItems {
		if item.SourceID != sourceID {
			continue
		}
		impact.IndexedSkillCount++
		if item.Collection != "" {
			collections[item.Collection] = struct{}{}
		}
	}
	impact.CollectionCount = len(collections)
	warnings := make([]string, 0, 2)
	if impact.InstalledCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d installed lock record(s) reference this source", impact.InstalledCount))
	}
	if impact.ProfileTargetCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d profile target(s) reference this source", impact.ProfileTargetCount))
	}
	return impact, warnings, nil
}

func targetSet(targets []profile.Target) map[string]struct{} {
	out := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		out[target.Source+"\x00"+target.Skill] = struct{}{}
	}
	return out
}

func skillSourceSet(items []skill.Skill) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		out[item.SourceID] = struct{}{}
	}
	return out
}
```

Also add this exported helper to `internal/app/sourceapp/service.go` so Web can reuse the existing URL rule without duplicating string checks:

```go
func IsGitURL(value string) bool {
	return isGitURL(value)
}
```

- [x] **Step 4: Run source plan tests**

Run:

```bash
go test ./internal/app/webapp -run 'TestManager_PlanSource(Add|Remove)' -v
```

Expected: PASS.

- [x] **Step 5: Add and test source run methods**

Append tests to `manager_source_actions_test.go`:

```go
func TestManager_RunSourceAddAddsAndSyncsLocalSource(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	sourceDir := filepath.Join(baseDir, "team-skills")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "hello"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "hello", "SKILL.md"), []byte("# Hello\n"), 0o644))

	result, err := NewManager(configFile, baseDir).RunSourceAdd(sourceActionReq{Value: sourceDir, Sync: true})

	assert.NoErr(t, err)
	assert.Eq(t, "add_local", result.Plan.Action)
	assert.Eq(t, true, result.Synced)
	assert.Eq(t, "", result.Error)
}

func TestManager_RunSourceRemoveRemovesSourceAndReturnsPlan(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)

	result, err := NewManager(configFile, baseDir).RunSourceRemove(sourceActionReq{ID: "gstack"})

	assert.NoErr(t, err)
	assert.Eq(t, "remove", result.Plan.Action)
	assert.Eq(t, "gstack", result.Plan.SourceID)
	sources, err := NewManager(configFile, baseDir).Sources()
	assert.NoErr(t, err)
	assert.Len(t, sources, 0)
}
```

Implement run result and methods:

```go
type sourceActionResult struct {
	Error  string           `json:"error,omitempty"`
	Plan   sourceActionPlan `json:"plan"`
	Added  bool             `json:"added,omitempty"`
	Synced bool             `json:"synced,omitempty"`
	Removed bool            `json:"removed,omitempty"`
}

func (m *Manager) RunSourceAdd(req sourceActionReq) (sourceActionResult, error) {
	plan, err := m.PlanSourceAdd(req)
	if err != nil {
		return sourceActionResult{}, err
	}
	if plan.Existing {
		return sourceActionResult{Plan: plan}, nil
	}
	src, _, err := sourceapp.NewService(m.configFile, m.baseDir).EnsureSource(req.Value, req.Ref)
	if err != nil {
		return sourceActionResult{Plan: plan, Error: err.Error()}, nil
	}
	result := sourceActionResult{Plan: plan, Added: true}
	if req.Sync {
		if err := sourceapp.NewService(m.configFile, m.baseDir).Sync(src.ID); err != nil {
			result.Error = err.Error()
			return result, nil
		}
		result.Synced = true
	}
	return result, nil
}

func (m *Manager) RunSourceSync(req sourceActionReq) (sourceActionResult, error) {
	plan, err := m.PlanSourceSync(req)
	if err != nil {
		return sourceActionResult{}, err
	}
	svc := sourceapp.NewService(m.configFile, m.baseDir)
	if plan.Action == "sync_all" {
		err = svc.SyncAll()
	} else {
		err = svc.Sync(plan.SourceID)
	}
	result := sourceActionResult{Plan: plan}
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}
	result.Synced = true
	return result, nil
}

func (m *Manager) RunSourceRemove(req sourceActionReq) (sourceActionResult, error) {
	plan, err := m.PlanSourceRemove(req)
	if err != nil {
		return sourceActionResult{}, err
	}
	result := sourceActionResult{Plan: plan}
	if err := sourceapp.NewService(m.configFile, m.baseDir).Remove(plan.SourceID); err != nil {
		result.Error = err.Error()
		return result, nil
	}
	result.Removed = true
	return result, nil
}
```

Run:

```bash
go test ./internal/app/webapp -run 'TestManager_RunSource' -v
```

Expected: PASS.

- [x] **Step 6: Commit source action manager**

Run:

```bash
git add internal/app/sourceapp/service.go internal/app/webapp/manager_source_actions.go internal/app/webapp/manager_source_actions_test.go
git commit -m "feat(skillc): add web source action planning"
```

## Task 2: Add Source HTTP Routes and UI Controls

**Files:**
- Modify: `internal/app/webapp/manager_server.go`
- Modify: `internal/app/webapp/manager_server_test.go`
- Modify: `internal/app/webapp/manager_static.go`

- [x] **Step 1: Write failing HTTP tests for source routes**

Append to `manager_server_test.go`:

```go
func TestManagerServerSourceAddPlanEndpoint(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	sourceDir := filepath.Join(baseDir, "team-skills")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "hello"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "hello", "SKILL.md"), []byte("# Hello\n"), 0o644))
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequestWithBody(server, http.MethodPost, "/api/sources/add/plan", strings.NewReader(`{"value":"`+strings.ReplaceAll(sourceDir, `\`, `\\`)+`","sync":true}`))

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"action":"add_local"`)
	assert.Contains(t, rec.Body.String(), `"action":"sync"`)
}

func TestManagerServerSourceAddRunRequiresConfirmation(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequestWithBody(server, http.MethodPost, "/api/sources/add/run", strings.NewReader(`{"value":"./skills"}`))

	assert.Eq(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), `"confirmation required"`)
}

func TestManagerServerSourceRemoveRunEndpoint(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequestWithBody(server, http.MethodPost, "/api/sources/remove/run", strings.NewReader(`{"confirm":true,"id":"gstack"}`))

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"removed":true`)
	assert.Contains(t, rec.Body.String(), `"source_id":"gstack"`)
}
```

If `manager_server_test.go` does not already import `os`, add it to the import block because the test writes a temporary `SKILL.md` file:

```go
import (
	"os"
)
```

- [x] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/app/webapp -run 'TestManagerServerSource(Add|Remove)' -v
```

Expected: FAIL with 404 because source action routes are not registered.

- [x] **Step 3: Register source routes and handlers**

Modify `ManagerServer.Handler()`:

```go
mux.HandleFunc("/api/sources/add/plan", s.handleSourceAddPlan)
mux.HandleFunc("/api/sources/add/run", s.handleSourceAddRun)
mux.HandleFunc("/api/sources/sync/plan", s.handleSourceSyncPlan)
mux.HandleFunc("/api/sources/sync/run", s.handleSourceSyncRun)
mux.HandleFunc("/api/sources/remove/plan", s.handleSourceRemovePlan)
mux.HandleFunc("/api/sources/remove/run", s.handleSourceRemoveRun)
```

Add helpers and handlers to `manager_server.go`:

```go
func readJSONReq[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	if r.Body == nil || r.Body == http.NoBody {
		return req, true
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid json body"))
		return req, false
	}
	return req, true
}

func requireConfirmedSourceReq(w http.ResponseWriter, r *http.Request) (sourceActionReq, bool) {
	req, ok := readJSONReq[sourceActionReq](w, r)
	if !ok {
		return sourceActionReq{}, false
	}
	if !req.Confirm {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "confirmation required"})
		return sourceActionReq{}, false
	}
	return req, true
}

func (s *ManagerServer) handleSourceAddPlan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := readJSONReq[sourceActionReq](w, r)
	if !ok {
		return
	}
	writeResult(w, s.manager.PlanSourceAdd(req))
}

func (s *ManagerServer) handleSourceAddRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := requireConfirmedSourceReq(w, r)
	if !ok {
		return
	}
	result, err := s.manager.RunSourceAdd(req)
	s.recordHistory(r, "source.add", req, result, err)
	writeResult(w, result, err)
}
```

Add the remaining source handlers:

```go
func (s *ManagerServer) handleSourceSyncPlan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := readJSONReq[sourceActionReq](w, r)
	if !ok {
		return
	}
	writeResult(w, s.manager.PlanSourceSync(req))
}

func (s *ManagerServer) handleSourceSyncRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := requireConfirmedSourceReq(w, r)
	if !ok {
		return
	}
	result, err := s.manager.RunSourceSync(req)
	s.recordHistory(r, "source.sync", req, result, err)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleSourceRemovePlan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := readJSONReq[sourceActionReq](w, r)
	if !ok {
		return
	}
	writeResult(w, s.manager.PlanSourceRemove(req))
}

func (s *ManagerServer) handleSourceRemoveRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := requireConfirmedSourceReq(w, r)
	if !ok {
		return
	}
	result, err := s.manager.RunSourceRemove(req)
	s.recordHistory(r, "source.remove", req, result, err)
	writeResult(w, result, err)
}
```

- [x] **Step 4: Run source HTTP tests**

Run:

```bash
go test ./internal/app/webapp -run 'TestManagerServerSource(Add|Remove)' -v
```

Expected: PASS.

- [x] **Step 5: Add source controls to static page**

In `manager_static.go`, update Sources section:

```html
<section id="view-sources" class="view">
  <div class="section-head"><h3>Sources</h3><span class="hint" id="sources-count"></span></div>
  <div class="panel section">
    <div class="toolbar-row">
      <input id="source-value-input" placeholder="Local path or git URL" autocomplete="off">
      <input id="source-ref-input" placeholder="Git ref" autocomplete="off">
      <label class="inline-check"><input id="source-sync-input" type="checkbox"> sync</label>
      <button id="plan-source-add-btn">Plan add</button>
    </div>
  </div>
  <div id="sources-table"></div>
</section>
```

Update `renderSources()` row action column:

```js
return '<tr><td>' + esc(s.ID || s.id) + '</td><td>' + esc(s.Name || s.name) +
  '</td><td>' + esc(s.Type || s.type) + '</td><td>' + statusPill(s.Status || s.status || 'configured') +
  '</td><td class="wrap mono">' + esc(s.Path || s.path || s.URL || s.url) +
  '</td><td>' + esc(s.Ref || s.ref || '') + '</td><td class="wrap">' + esc(s.ErrorMessage || s.error_message || '') +
  '</td><td><button data-source-sync="' + esc(s.ID || s.id) + '">Plan sync</button> ' +
  '<button class="danger" data-source-remove="' + esc(s.ID || s.id) + '">Plan remove</button></td></tr>';
```

Add functions:

```js
function planSourceAdd() {
  clearError();
  var payload = {
    value: byId('source-value-input').value.trim(),
    ref: byId('source-ref-input').value.trim(),
    sync: byId('source-sync-input').checked
  };
  postJSON('/api/sources/add/plan', payload).then(function (plan) {
    setPendingAction({ type: 'source-add', payload: payload });
    byId('plan-output').textContent = JSON.stringify(plan, null, 2);
  }).catch(showError);
}

function planSourceSync(id) {
  clearError();
  postJSON('/api/sources/sync/plan', { id: id }).then(function (plan) {
    setPendingAction({ type: 'source-sync', payload: { id: id } });
    byId('plan-output').textContent = JSON.stringify(plan, null, 2);
  }).catch(showError);
}

function planSourceRemove(id) {
  clearError();
  postJSON('/api/sources/remove/plan', { id: id }).then(function (plan) {
    setPendingAction({ type: 'source-remove', payload: { id: id } });
    byId('plan-output').textContent = JSON.stringify(plan, null, 2);
  }).catch(showError);
}
```

Extend the action bar with:

```html
<button class="danger" id="run-source-action-btn" disabled>Run source action</button>
```

Add run handler:

```js
function runSourceAction() {
  var action = state.pendingAction;
  if (!action || action.type.indexOf('source-') !== 0) return;
  if (!window.confirm('Run this source action for the current project?')) return;
  var route = {
    'source-add': '/api/sources/add/run',
    'source-sync': '/api/sources/sync/run',
    'source-remove': '/api/sources/remove/run'
  }[action.type];
  var payload = Object.assign({ confirm: true }, action.payload || {});
  postJSON(route, payload).then(function (result) {
    byId('plan-output').textContent = JSON.stringify(result, null, 2);
    setPendingAction(null);
    loadAll();
  }).catch(showError);
}
```

Update `setPendingAction()`:

```js
byId('run-source-action-btn').disabled = !(action && action.type.indexOf('source-') === 0);
```

- [x] **Step 6: Add static page smoke tests**

Append to `manager_server_test.go`:

```go
func TestManagerServerStaticPageContainsSourceManagementControls(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequest(server, http.MethodGet, "/")
	body := rec.Body.String()

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, body, `id="source-value-input"`)
	assert.Contains(t, body, "/api/sources/add/plan")
	assert.Contains(t, body, "/api/sources/remove/run")
	assert.Contains(t, body, `id="run-source-action-btn"`)
}
```

Run:

```bash
go test ./internal/app/webapp -run 'TestManagerServerStaticPageContainsSourceManagementControls|TestManagerServerSource' -v
```

Expected: PASS.

- [x] **Step 7: Commit source Web routes and UI**

Run:

```bash
git add internal/app/webapp/manager_server.go internal/app/webapp/manager_server_test.go internal/app/webapp/manager_static.go
git commit -m "feat(skillc): add web source management endpoints"
```

## Task 3: Add Profile Planning Support in profileapp

**Files:**
- Modify: `internal/app/profileapp/service.go`
- Modify: `internal/app/profileapp/service_test.go`

- [x] **Step 1: Write failing profileapp tests**

Append to `internal/app/profileapp/service_test.go`:

```go
func TestService_BuildFromCollectionReturnsUnsavedProfile(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeProfileFixture(t, baseDir)
	assert.NoErr(t, repoindex.NewStore().Save(config.IndexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Collection: "go"},
		{ID: "review", SourceID: "gstack", Collection: "ops"},
	}))

	got, err := NewService(configFile, baseDir).BuildFromCollection("gstack/go")

	assert.NoErr(t, err)
	assert.Len(t, got.Targets, 1)
	assert.Eq(t, "gstack", got.Targets[0].Source)
	assert.Eq(t, "go-pro", got.Targets[0].Skill)
}

func TestService_PlanSaveReportsAddedRemovedKeptTargets(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeProfileFixture(t, baseDir)
	config.Profiles = map[string]profile.Profile{
		"go-dev": {Targets: []profile.Target{{Source: "gstack", Skill: "go-pro"}, {Source: "gstack", Skill: "old"}}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))

	plan, err := NewService(configFile, baseDir).PlanSave("go-dev", profile.Profile{
		Targets: []profile.Target{{Source: "gstack", Skill: "go-pro"}, {Source: "gstack", Skill: "review"}},
	})

	assert.NoErr(t, err)
	assert.Eq(t, "edit", plan.Mode)
	assert.Len(t, plan.Added, 1)
	assert.Eq(t, "review", plan.Added[0].Skill)
	assert.Len(t, plan.Removed, 1)
	assert.Eq(t, "old", plan.Removed[0].Skill)
	assert.Len(t, plan.Kept, 1)
}
```

- [x] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/app/profileapp -run 'TestService_(BuildFromCollectionReturnsUnsavedProfile|PlanSaveReportsAddedRemovedKeptTargets)' -v
```

Expected: FAIL because `BuildFromCollection` and `PlanSave` are not defined.

- [x] **Step 3: Add profile save plan types and builders**

Add to `profileapp/service.go`:

```go
type SavePlan struct {
	Name    string
	Mode    string
	Profile profile.Profile
	Added   []profile.Target
	Removed []profile.Target
	Kept    []profile.Target
}

func (s *Service) BuildFromInstalled(req CreateFromInstalledReq) (profile.Profile, error) {
	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return profile.Profile{}, err
	}
	agentName := firstNonEmpty(req.Agent, agent.DefaultAgentName)
	canonicalAgent, _, ok := config.ResolveAgentTool(agentName)
	if !ok {
		return profile.Profile{}, fmt.Errorf("unsupported agent: %s", agentName)
	}
	scope, err := apputil.ParseScope(firstNonEmpty(req.Scope, string(agent.ScopeProject)))
	if err != nil {
		return profile.Profile{}, err
	}
	workDir := firstNonEmpty(req.WorkDir, s.baseDir)
	items, err := listapp.NewService(config.LockFile).WithRuntime(config, workDir).List(canonicalAgent, string(scope))
	if err != nil {
		return profile.Profile{}, err
	}
	targets := make([]profile.Target, 0, len(items))
	for _, item := range items {
		if item.Status != "installed" {
			continue
		}
		targets = append(targets, profile.Target{Source: item.SourceID, Skill: item.SkillID})
	}
	targets, err = profile.NormalizeTargets(targets)
	if err != nil {
		return profile.Profile{}, err
	}
	return profile.Profile{DefaultAgent: canonicalAgent, DefaultScope: string(scope), Targets: targets}, nil
}

func (s *Service) BuildFromCollection(selector string) (profile.Profile, error) {
	sourceID, collection, ok := strings.Cut(selector, "/")
	if !ok || strings.TrimSpace(sourceID) == "" || strings.TrimSpace(collection) == "" || strings.Contains(collection, "/") {
		return profile.Profile{}, fmt.Errorf("collection selector must be <source>/<collection>: %s", selector)
	}
	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return profile.Profile{}, err
	}
	items, err := searchapp.NewService(config.IndexFile).ListSourceSkills(strings.TrimSpace(sourceID), strings.TrimSpace(collection))
	if err != nil {
		return profile.Profile{}, err
	}
	targets := make([]profile.Target, 0, len(items))
	for _, item := range items {
		targets = append(targets, profile.Target{Source: item.SourceID, Skill: item.ID})
	}
	targets, err = profile.NormalizeTargets(targets)
	if err != nil {
		return profile.Profile{}, err
	}
	return profile.Profile{Targets: targets}, nil
}

func (s *Service) PlanSave(name string, item profile.Profile) (SavePlan, error) {
	if err := profile.ValidateName(name); err != nil {
		return SavePlan{}, err
	}
	targets, err := profile.NormalizeTargets(item.Targets)
	if err != nil {
		return SavePlan{}, err
	}
	item.Targets = targets
	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return SavePlan{}, err
	}
	mode := "create"
	var current profile.Profile
	if existing, ok := config.Profiles[name]; ok {
		mode = "edit"
		current = existing
	}
	added, removed, kept := diffTargets(current.Targets, item.Targets)
	return SavePlan{Name: name, Mode: mode, Profile: item, Added: added, Removed: removed, Kept: kept}, nil
}

func diffTargets(before []profile.Target, after []profile.Target) ([]profile.Target, []profile.Target, []profile.Target) {
	beforeSet := targetKeyMap(before)
	afterSet := targetKeyMap(after)
	added := make([]profile.Target, 0)
	removed := make([]profile.Target, 0)
	kept := make([]profile.Target, 0)
	for _, target := range after {
		if _, ok := beforeSet[target.Source+"\x00"+target.Skill]; ok {
			kept = append(kept, target)
			continue
		}
		added = append(added, target)
	}
	for _, target := range before {
		if _, ok := afterSet[target.Source+"\x00"+target.Skill]; !ok {
			removed = append(removed, target)
		}
	}
	return added, removed, kept
}

func targetKeyMap(targets []profile.Target) map[string]struct{} {
	out := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		out[target.Source+"\x00"+target.Skill] = struct{}{}
	}
	return out
}
```

Refactor existing methods:

```go
func (s *Service) CreateFromInstalled(name string, req CreateFromInstalledReq) (profile.Profile, error) {
	if err := profile.ValidateName(name); err != nil {
		return profile.Profile{}, err
	}
	item, err := s.BuildFromInstalled(req)
	if err != nil {
		return profile.Profile{}, err
	}
	return s.Create(name, item)
}

func (s *Service) CreateFromCollection(name string, selector string) (profile.Profile, error) {
	if err := profile.ValidateName(name); err != nil {
		return profile.Profile{}, err
	}
	item, err := s.BuildFromCollection(selector)
	if err != nil {
		return profile.Profile{}, err
	}
	return s.Create(name, item)
}
```

- [x] **Step 4: Run profileapp tests**

Run:

```bash
go test ./internal/app/profileapp -run 'TestService_(BuildFromCollectionReturnsUnsavedProfile|PlanSaveReportsAddedRemovedKeptTargets|CreateFrom)' -v
```

Expected: PASS.

- [x] **Step 5: Commit profile planning support**

Run:

```bash
git add internal/app/profileapp/service.go internal/app/profileapp/service_test.go
git commit -m "feat(skillc): add profile save planning"
```

## Task 4: Add Web Profile Management Endpoints and UI

**Files:**
- Create: `internal/app/webapp/manager_profile_actions.go`
- Create: `internal/app/webapp/manager_profile_actions_test.go`
- Modify: `internal/app/webapp/manager_server.go`
- Modify: `internal/app/webapp/manager_server_test.go`
- Modify: `internal/app/webapp/manager_static.go`

- [x] **Step 1: Write failing manager tests for profile actions**

Create `manager_profile_actions_test.go`:

```go
package webapp

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/profile"
)

func TestManager_PlanProfileSaveCreatesPlan(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)

	plan, err := NewManager(configFile, baseDir).PlanProfileSave(profileSaveReq{
		Name: "ops",
		Targets: []profile.Target{{Source: "gstack", Skill: "review"}},
	})

	assert.NoErr(t, err)
	assert.Eq(t, "ops", plan.Name)
	assert.Eq(t, "create", plan.Mode)
	assert.Len(t, plan.Added, 1)
}

func TestManager_RunProfileSavePersistsProfile(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)

	result, err := NewManager(configFile, baseDir).RunProfileSave(profileSaveReq{
		Name: "ops",
		Targets: []profile.Target{{Source: "gstack", Skill: "review"}},
	})

	assert.NoErr(t, err)
	assert.Eq(t, true, result.Saved)
	got, err := NewManager(configFile, baseDir).Profiles()
	assert.NoErr(t, err)
	assert.Len(t, got, 2)
}
```

- [x] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/app/webapp -run 'TestManager_(PlanProfileSave|RunProfileSave)' -v
```

Expected: FAIL because profile action methods and types are not defined.

- [x] **Step 3: Implement manager profile action service**

Create `manager_profile_actions.go`:

```go
package webapp

import (
	"github.com/inhere/skillc/internal/app/profileapp"
	"github.com/inhere/skillc/internal/domain/profile"
)

type profileSaveReq struct {
	Confirm      bool             `json:"confirm,omitempty"`
	Name         string           `json:"name"`
	Description  string           `json:"description,omitempty"`
	DefaultAgent string           `json:"default_agent,omitempty"`
	DefaultScope string           `json:"default_scope,omitempty"`
	InstallMode  string           `json:"install_mode,omitempty"`
	Targets      []profile.Target `json:"targets"`
}

type profileSavePlan struct {
	Name    string           `json:"name"`
	Mode    string           `json:"mode"`
	Profile profile.Profile  `json:"profile"`
	Added   []profile.Target `json:"added,omitempty"`
	Removed []profile.Target `json:"removed,omitempty"`
	Kept    []profile.Target `json:"kept,omitempty"`
}

type profileSaveResult struct {
	Error string          `json:"error,omitempty"`
	Plan  profileSavePlan `json:"plan"`
	Saved bool            `json:"saved"`
}

type profileFromInstalledReq struct {
	Confirm bool   `json:"confirm,omitempty"`
	Name    string `json:"name"`
	Agent   string `json:"agent,omitempty"`
	Scope   string `json:"scope,omitempty"`
}

type profileFromCollectionReq struct {
	Confirm  bool   `json:"confirm,omitempty"`
	Name     string `json:"name"`
	Selector string `json:"selector"`
}

func (m *Manager) PlanProfileSave(req profileSaveReq) (profileSavePlan, error) {
	plan, err := profileapp.NewService(m.configFile, m.baseDir).PlanSave(req.Name, profile.Profile{
		Description:  req.Description,
		DefaultAgent: req.DefaultAgent,
		DefaultScope: req.DefaultScope,
		InstallMode:  req.InstallMode,
		Targets:      req.Targets,
	})
	return toProfileSavePlan(plan), err
}

func (m *Manager) RunProfileSave(req profileSaveReq) (profileSaveResult, error) {
	plan, err := m.PlanProfileSave(req)
	if err != nil {
		return profileSaveResult{}, err
	}
	err = profileapp.NewService(m.configFile, m.baseDir).Save(req.Name, plan.Profile)
	result := profileSaveResult{Plan: plan, Saved: err == nil}
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}
	return result, nil
}

func (m *Manager) PlanProfileFromInstalled(req profileFromInstalledReq) (profileSavePlan, error) {
	item, err := profileapp.NewService(m.configFile, m.baseDir).BuildFromInstalled(profileapp.CreateFromInstalledReq{
		Agent: req.Agent,
		Scope: req.Scope,
		WorkDir: m.baseDir,
	})
	if err != nil {
		return profileSavePlan{}, err
	}
	plan, err := profileapp.NewService(m.configFile, m.baseDir).PlanSave(req.Name, item)
	return toProfileSavePlan(plan), err
}

func (m *Manager) RunProfileFromInstalled(req profileFromInstalledReq) (profileSaveResult, error) {
	plan, err := m.PlanProfileFromInstalled(req)
	if err != nil {
		return profileSaveResult{}, err
	}
	if plan.Mode != "create" {
		return profileSaveResult{}, conflictError("profile already exists: " + req.Name)
	}
	err = profileapp.NewService(m.configFile, m.baseDir).Save(req.Name, plan.Profile)
	result := profileSaveResult{Plan: plan, Saved: err == nil}
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}
	return result, nil
}

func (m *Manager) PlanProfileFromCollection(req profileFromCollectionReq) (profileSavePlan, error) {
	item, err := profileapp.NewService(m.configFile, m.baseDir).BuildFromCollection(req.Selector)
	if err != nil {
		return profileSavePlan{}, err
	}
	plan, err := profileapp.NewService(m.configFile, m.baseDir).PlanSave(req.Name, item)
	return toProfileSavePlan(plan), err
}

func (m *Manager) RunProfileFromCollection(req profileFromCollectionReq) (profileSaveResult, error) {
	plan, err := m.PlanProfileFromCollection(req)
	if err != nil {
		return profileSaveResult{}, err
	}
	if plan.Mode != "create" {
		return profileSaveResult{}, conflictError("profile already exists: " + req.Name)
	}
	err = profileapp.NewService(m.configFile, m.baseDir).Save(req.Name, plan.Profile)
	result := profileSaveResult{Plan: plan, Saved: err == nil}
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}
	return result, nil
}

func toProfileSavePlan(plan profileapp.SavePlan) profileSavePlan {
	return profileSavePlan{
		Name: plan.Name,
		Mode: plan.Mode,
		Profile: plan.Profile,
		Added: plan.Added,
		Removed: plan.Removed,
		Kept: plan.Kept,
	}
}
```

Add conflict support to `manager_server.go`:

```go
type httpStatusError struct {
	status int
	msg    string
}

func (e httpStatusError) Error() string { return e.msg }

func conflictError(msg string) error {
	return httpStatusError{status: http.StatusConflict, msg: msg}
}
```

Update `writeResult`:

```go
func writeResult(w http.ResponseWriter, result any, err error) {
	if err != nil {
		var statusErr httpStatusError
		if errors.As(err, &statusErr) {
			writeJSON(w, statusErr.status, errorResp{Error: statusErr.msg})
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
```

- [x] **Step 4: Run manager profile action tests**

Run:

```bash
go test ./internal/app/webapp -run 'TestManager_(PlanProfileSave|RunProfileSave)' -v
```

Expected: PASS.

- [x] **Step 5: Add profile HTTP route tests**

Append to `manager_server_test.go`:

```go
func TestManagerServerProfileSavePlanEndpoint(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequestWithBody(server, http.MethodPost, "/api/profiles/save/plan", strings.NewReader(`{"name":"ops","targets":[{"source":"gstack","skill":"review"}]}`))

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"name":"ops"`)
	assert.Contains(t, rec.Body.String(), `"mode":"create"`)
}

func TestManagerServerProfileSaveRunRequiresConfirmation(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequestWithBody(server, http.MethodPost, "/api/profiles/save/run", strings.NewReader(`{"name":"ops","targets":[{"source":"gstack","skill":"review"}]}`))

	assert.Eq(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), `"confirmation required"`)
}

func TestManagerServerProfileFromCollectionRunConflictsOnExistingProfile(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequestWithBody(server, http.MethodPost, "/api/profiles/from-collection/run", strings.NewReader(`{"confirm":true,"name":"go-dev","selector":"gstack/tools"}`))

	assert.Eq(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), `"profile already exists: go-dev"`)
}
```

- [x] **Step 6: Register profile action routes and handlers**

Add routes:

```go
mux.HandleFunc("/api/profiles/save/plan", s.handleProfileSavePlan)
mux.HandleFunc("/api/profiles/save/run", s.handleProfileSaveRun)
mux.HandleFunc("/api/profiles/from-installed/plan", s.handleProfileFromInstalledPlan)
mux.HandleFunc("/api/profiles/from-installed/run", s.handleProfileFromInstalledRun)
mux.HandleFunc("/api/profiles/from-collection/plan", s.handleProfileFromCollectionPlan)
mux.HandleFunc("/api/profiles/from-collection/run", s.handleProfileFromCollectionRun)
```

Add confirmation helpers:

```go
func requireConfirmedProfileSaveReq(w http.ResponseWriter, r *http.Request) (profileSaveReq, bool) {
	req, ok := readJSONReq[profileSaveReq](w, r)
	if !ok {
		return profileSaveReq{}, false
	}
	if !req.Confirm {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "confirmation required"})
		return profileSaveReq{}, false
	}
	return req, true
}
```

Add confirmation helpers for the two shortcut request types:

```go
func requireConfirmedProfileFromInstalledReq(w http.ResponseWriter, r *http.Request) (profileFromInstalledReq, bool) {
	req, ok := readJSONReq[profileFromInstalledReq](w, r)
	if !ok {
		return profileFromInstalledReq{}, false
	}
	if !req.Confirm {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "confirmation required"})
		return profileFromInstalledReq{}, false
	}
	return req, true
}

func requireConfirmedProfileFromCollectionReq(w http.ResponseWriter, r *http.Request) (profileFromCollectionReq, bool) {
	req, ok := readJSONReq[profileFromCollectionReq](w, r)
	if !ok {
		return profileFromCollectionReq{}, false
	}
	if !req.Confirm {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "confirmation required"})
		return profileFromCollectionReq{}, false
	}
	return req, true
}
```

Add the profile action handlers:

```go
func (s *ManagerServer) handleProfileSavePlan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := readJSONReq[profileSaveReq](w, r)
	if !ok {
		return
	}
	writeResult(w, s.manager.PlanProfileSave(req))
}

func (s *ManagerServer) handleProfileSaveRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := requireConfirmedProfileSaveReq(w, r)
	if !ok {
		return
	}
	result, err := s.manager.RunProfileSave(req)
	s.recordHistory(r, "profile.save", req, result, err)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleProfileFromInstalledPlan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := readJSONReq[profileFromInstalledReq](w, r)
	if !ok {
		return
	}
	writeResult(w, s.manager.PlanProfileFromInstalled(req))
}

func (s *ManagerServer) handleProfileFromInstalledRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := requireConfirmedProfileFromInstalledReq(w, r)
	if !ok {
		return
	}
	result, err := s.manager.RunProfileFromInstalled(req)
	s.recordHistory(r, "profile.from-installed", req, result, err)
	writeResult(w, result, err)
}

func (s *ManagerServer) handleProfileFromCollectionPlan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := readJSONReq[profileFromCollectionReq](w, r)
	if !ok {
		return
	}
	writeResult(w, s.manager.PlanProfileFromCollection(req))
}

func (s *ManagerServer) handleProfileFromCollectionRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := requireConfirmedProfileFromCollectionReq(w, r)
	if !ok {
		return
	}
	result, err := s.manager.RunProfileFromCollection(req)
	s.recordHistory(r, "profile.from-collection", req, result, err)
	writeResult(w, result, err)
}
```

- [x] **Step 7: Add profile form UI**

Add profile editor panel before `profiles-table`:

```html
<div class="panel section">
  <div class="toolbar-row">
    <input id="profile-name-input" placeholder="Profile name" autocomplete="off">
    <input id="profile-description-input" placeholder="Description" autocomplete="off">
    <input id="profile-agent-input" placeholder="Default agent" autocomplete="off">
    <select id="profile-scope-input">
      <option value="">default scope</option>
      <option value="project">project</option>
      <option value="user">user</option>
    </select>
    <input id="profile-install-mode-input" placeholder="Install mode" autocomplete="off">
  </div>
  <textarea id="profile-targets-input" rows="5" placeholder="source skill"></textarea>
  <div class="toolbar-row">
    <button id="plan-profile-save-btn">Plan save</button>
    <button id="plan-profile-installed-btn">Plan from installed</button>
    <input id="profile-collection-input" placeholder="source/collection" autocomplete="off">
    <button id="plan-profile-collection-btn">Plan from collection</button>
  </div>
</div>
```

Add target parser:

```js
function profileTargetsFromText() {
  return byId('profile-targets-input').value.split('\n').map(function (line) {
    var trimmed = line.trim();
    if (!trimmed) return null;
    var parts = trimmed.split(/\s+/);
    if (parts.length === 1) return { skill: parts[0] };
    return { source: parts[0], skill: parts.slice(1).join(' ') };
  }).filter(Boolean);
}
```

Add `Run profile action` to the action bar:

```html
<button class="danger" id="run-profile-action-btn" disabled>Run profile action</button>
```

Add profile action planning functions:

```js
function profileSavePayload() {
  return {
    name: byId('profile-name-input').value.trim(),
    description: byId('profile-description-input').value.trim(),
    default_agent: byId('profile-agent-input').value.trim(),
    default_scope: byId('profile-scope-input').value,
    install_mode: byId('profile-install-mode-input').value.trim(),
    targets: profileTargetsFromText()
  };
}

function planProfileSave() {
  clearError();
  var payload = profileSavePayload();
  postJSON('/api/profiles/save/plan', payload).then(function (plan) {
    setPendingAction({ type: 'profile-save', payload: payload });
    byId('plan-output').textContent = JSON.stringify(plan, null, 2);
  }).catch(showError);
}

function planProfileFromInstalled() {
  clearError();
  var payload = {
    name: byId('profile-name-input').value.trim(),
    agent: byId('agent-input').value.trim(),
    scope: byId('scope-input').value
  };
  postJSON('/api/profiles/from-installed/plan', payload).then(function (plan) {
    setPendingAction({ type: 'profile-installed', payload: payload });
    byId('plan-output').textContent = JSON.stringify(plan, null, 2);
  }).catch(showError);
}

function planProfileFromCollection() {
  clearError();
  var payload = {
    name: byId('profile-name-input').value.trim(),
    selector: byId('profile-collection-input').value.trim()
  };
  postJSON('/api/profiles/from-collection/plan', payload).then(function (plan) {
    setPendingAction({ type: 'profile-collection', payload: payload });
    byId('plan-output').textContent = JSON.stringify(plan, null, 2);
  }).catch(showError);
}
```

Add profile action execution:

```js
function runProfileAction() {
  var action = state.pendingAction;
  if (!action || action.type.indexOf('profile-') !== 0) return;
  if (!window.confirm('Run this profile action for the current project?')) return;
  var route = {
    'profile-save': '/api/profiles/save/run',
    'profile-installed': '/api/profiles/from-installed/run',
    'profile-collection': '/api/profiles/from-collection/run'
  }[action.type];
  var payload = Object.assign({ confirm: true }, action.payload || {});
  postJSON(route, payload).then(function (result) {
    byId('plan-output').textContent = JSON.stringify(result, null, 2);
    setPendingAction(null);
    loadAll();
  }).catch(showError);
}
```

Extend `setPendingAction()`:

```js
byId('run-profile-action-btn').disabled = !(action && action.type.indexOf('profile-') === 0);
```

Register button events:

```js
byId('plan-profile-save-btn').addEventListener('click', planProfileSave);
byId('plan-profile-installed-btn').addEventListener('click', planProfileFromInstalled);
byId('plan-profile-collection-btn').addEventListener('click', planProfileFromCollection);
byId('run-profile-action-btn').addEventListener('click', runProfileAction);
```

- [x] **Step 8: Run profile route and static tests**

Run:

```bash
go test ./internal/app/webapp -run 'TestManagerServerProfile(Save|From)|TestManager_(PlanProfileSave|RunProfileSave)' -v
```

Expected: PASS.

- [x] **Step 9: Commit Web profile management**

Run:

```bash
git add internal/app/webapp/manager_profile_actions.go internal/app/webapp/manager_profile_actions_test.go internal/app/webapp/manager_server.go internal/app/webapp/manager_server_test.go internal/app/webapp/manager_static.go
git commit -m "feat(skillc): add web profile management"
```

## Task 5: Add Uninstall Plan and Web Uninstall

**Files:**
- Modify: `internal/app/installapp/service.go`
- Modify: `internal/app/installapp/service_test.go`
- Create: `internal/app/webapp/manager_uninstall_actions.go`
- Create: `internal/app/webapp/manager_uninstall_actions_test.go`
- Modify: `internal/app/webapp/manager_server.go`
- Modify: `internal/app/webapp/manager_server_test.go`
- Modify: `internal/app/webapp/manager_static.go`

- [x] **Step 1: Write failing installapp uninstall plan tests**

Append to `internal/app/installapp/service_test.go`:

```go
func TestService_PlanUninstallReturnsTargetPath(t *testing.T) {
	baseDir := t.TempDir()
	config := testConfig(baseDir)
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	service := NewService(lockFile).WithRuntime(config, baseDir)
	item := skill.Skill{ID: "go-pro", SourceID: "gstack", Version: "1.0.0", InstallEntry: ".", Path: filepath.Join(baseDir, "source", "go-pro")}
	assert.NoErr(t, os.MkdirAll(item.Path, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(item.Path, "SKILL.md"), []byte("# Go\n"), 0o644))
	_, err := service.RunResolved(config, InstallReq{Agent: "universal", Scope: "project", WorkDir: baseDir}, []skill.Skill{item}, nil)
	assert.NoErr(t, err)

	plan, err := service.PlanUninstall(UninstallReq{Skills: []string{"go-pro"}, Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Eq(t, "universal", plan.Agent)
	assert.Eq(t, "project", plan.Scope)
	assert.Len(t, plan.Items, 1)
	assert.Eq(t, "remove_record", plan.Items[0].Action)
	assert.Eq(t, "go-pro", plan.Items[0].SkillID)
	assert.Contains(t, plan.Items[0].InstalledPath, "go-pro")
}

func TestService_RunUninstallReturnsRemovedItems(t *testing.T) {
	baseDir := t.TempDir()
	config := testConfig(baseDir)
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	service := NewService(lockFile).WithRuntime(config, baseDir)
	item := skill.Skill{ID: "go-pro", SourceID: "gstack", Version: "1.0.0", InstallEntry: ".", Path: filepath.Join(baseDir, "source", "go-pro")}
	assert.NoErr(t, os.MkdirAll(item.Path, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(item.Path, "SKILL.md"), []byte("# Go\n"), 0o644))
	_, err := service.RunResolved(config, InstallReq{Agent: "universal", Scope: "project", WorkDir: baseDir}, []skill.Skill{item}, nil)
	assert.NoErr(t, err)

	result, err := service.RunUninstall(UninstallReq{Skills: []string{"go-pro"}, Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Len(t, result.Removed, 1)
	assert.Eq(t, "go-pro", result.Removed[0].SkillID)
}
```

- [x] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/app/installapp -run 'TestService_(PlanUninstall|RunUninstall)' -v
```

Expected: FAIL because `UninstallReq`, `PlanUninstall`, and `RunUninstall` are not defined.

- [x] **Step 3: Implement uninstall plan/result in installapp**

Add types to `service.go`:

```go
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
```

Add methods:

```go
func (s *Service) PlanUninstall(req UninstallReq) (UninstallPlan, error) {
	scope, err := parseScope(req.Scope)
	if err != nil {
		return UninstallPlan{}, err
	}
	agentName := strings.TrimSpace(req.Agent)
	if agentName == "" {
		agentName = agent.DefaultAgentName
	}
	runtimeSvc := s.WithRuntime(s.runtimeConfig(), req.WorkDir)
	locks, err := runtimeSvc.loadLockFile()
	if err != nil {
		return UninstallPlan{}, err
	}
	plan := UninstallPlan{Agent: agentName, Scope: string(scope)}
	for _, target := range req.Skills {
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
					Action: action, SkillID: record.SkillID, QualifiedName: record.QualifiedName,
					SourceQualifiedName: record.SourceQualifiedName, SourceID: record.SourceID,
					Version: record.Version, Agent: agentName, Scope: string(scope), InstalledPath: path,
				})
				found = true
			}
		}
		if !found {
			plan.Items = append(plan.Items, UninstallPlanItem{Action: "error", SkillID: target, Agent: agentName, Scope: string(scope), Reason: "skill not found"})
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
	runtimeSvc := s.WithRuntime(s.runtimeConfig(), req.WorkDir)
	if err := runtimeSvc.UninstallMulti(req.Skills, plan.Agent, scope); err != nil {
		return result, err
	}
	for _, item := range plan.Items {
		result.Removed = append(result.Removed, item)
	}
	return result, nil
}
```

- [x] **Step 4: Run installapp uninstall tests**

Run:

```bash
go test ./internal/app/installapp -run 'TestService_(PlanUninstall|RunUninstall)' -v
```

Expected: PASS.

- [x] **Step 5: Add Web uninstall manager tests and service**

Create `manager_uninstall_actions_test.go`:

```go
package webapp

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestManager_PlanUninstall(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeWebActionFixture(t, baseDir)

	plan, err := NewManager(configFile, baseDir).PlanUninstall(uninstallActionReq{Skills: []string{"go-pro"}, Agent: "universal", Scope: "project"})

	assert.NoErr(t, err)
	assert.Eq(t, "universal", plan.Agent)
	assert.Len(t, plan.Items, 1)
	assert.Eq(t, "go-pro", plan.Items[0].SkillID)
}
```

Create `manager_uninstall_actions.go`:

```go
package webapp

import (
	"github.com/inhere/skillc/internal/app/installapp"
)

type uninstallActionReq struct {
	Confirm bool     `json:"confirm,omitempty"`
	Skills  []string `json:"skills"`
	Agent   string   `json:"agent,omitempty"`
	Scope   string   `json:"scope,omitempty"`
}

type uninstallActionResult struct {
	Error   string                         `json:"error,omitempty"`
	Plan    installapp.UninstallPlan       `json:"plan"`
	Removed []installapp.UninstallPlanItem `json:"removed"`
	Failed  []actionErrorItem              `json:"failed,omitempty"`
}

func (m *Manager) PlanUninstall(req uninstallActionReq) (installapp.UninstallPlan, error) {
	config, err := m.config()
	if err != nil {
		return installapp.UninstallPlan{}, err
	}
	return installapp.NewService(config.LockFile).WithRuntime(config, m.baseDir).PlanUninstall(installapp.UninstallReq{
		Skills: req.Skills,
		Agent: req.Agent,
		Scope: req.Scope,
		WorkDir: m.baseDir,
	})
}

func (m *Manager) RunUninstall(req uninstallActionReq) (uninstallActionResult, error) {
	config, err := m.config()
	if err != nil {
		return uninstallActionResult{}, err
	}
	result, err := installapp.NewService(config.LockFile).WithRuntime(config, m.baseDir).RunUninstall(installapp.UninstallReq{
		Skills: req.Skills,
		Agent: req.Agent,
		Scope: req.Scope,
		WorkDir: m.baseDir,
	})
	out := uninstallActionResult{Plan: result.Plan, Removed: result.Removed, Failed: installErrors(result.Failed)}
	if err != nil {
		if len(out.Plan.Items) > 0 || len(out.Failed) > 0 {
			out.Error = err.Error()
			return out, nil
		}
		return out, err
	}
	return out, nil
}
```

Run:

```bash
go test ./internal/app/webapp -run 'TestManager_PlanUninstall' -v
```

Expected: PASS.

- [x] **Step 6: Add uninstall HTTP endpoints and UI**

Add routes:

```go
mux.HandleFunc("/api/uninstall/plan", s.handleUninstallPlan)
mux.HandleFunc("/api/uninstall/run", s.handleUninstallRun)
```

Add tests:

```go
func TestManagerServerUninstallRunRequiresConfirmation(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeWebActionFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequestWithBody(server, http.MethodPost, "/api/uninstall/run", strings.NewReader(`{"skills":["go-pro"]}`))

	assert.Eq(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), `"confirmation required"`)
}

func TestManagerServerUninstallPlanEndpoint(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeWebActionFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequestWithBody(server, http.MethodPost, "/api/uninstall/plan", strings.NewReader(`{"skills":["go-pro"],"agent":"universal","scope":"project"}`))

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"skill_id":"go-pro"`)
	assert.Contains(t, rec.Body.String(), `"installed_path"`)
}
```

In `manager_static.go`, add uninstall buttons to install map rows:

```js
'</td><td>' + esc(item.version || '') + '</td><td><button class="danger" data-uninstall="' +
esc(item.source_qualified_name || item.qualified_name || item.skill_id) + '">Plan uninstall</button></td></tr>';
```

Add functions `planUninstall(skill)` and `runUninstall()` using `/api/uninstall/plan` and `/api/uninstall/run`, and extend `setPendingAction()` for action type `uninstall`.

- [x] **Step 7: Run uninstall tests**

Run:

```bash
go test ./internal/app/installapp -run 'TestService_(PlanUninstall|RunUninstall)' -v
go test ./internal/app/webapp -run 'TestManager(Server)?Uninstall' -v
```

Expected: PASS.

- [x] **Step 8: Commit Web uninstall**

Run:

```bash
git add internal/app/installapp/service.go internal/app/installapp/service_test.go internal/app/webapp/manager_uninstall_actions.go internal/app/webapp/manager_uninstall_actions_test.go internal/app/webapp/manager_server.go internal/app/webapp/manager_server_test.go internal/app/webapp/manager_static.go
git commit -m "feat(skillc): add web uninstall planning"
```

## Task 6: Add Web Action History

**Files:**
- Create: `internal/app/webapp/history.go`
- Create: `internal/app/webapp/history_test.go`
- Modify: `internal/app/webapp/manager.go`
- Modify: `internal/app/webapp/manager_server.go`
- Modify: `internal/app/webapp/manager_server_test.go`
- Modify: `internal/app/webapp/manager_static.go`

- [x] **Step 1: Write failing history store tests**

Create `history_test.go`:

```go
package webapp

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
)

func TestHistoryStoreAppendAndListRecent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	store := newHistoryStore(path)
	store.now = func() time.Time { return time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC) }

	assert.NoErr(t, store.Append(HistoryRecord{Action: "source.add", Status: "ok"}))
	items, err := store.List(10)

	assert.NoErr(t, err)
	assert.Len(t, items, 1)
	assert.Eq(t, "source.add", items[0].Action)
	assert.Eq(t, "ok", items[0].Status)
	assert.Eq(t, "2026-06-15T10:00:00Z", items[0].Time)
}
```

- [x] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/app/webapp -run TestHistoryStoreAppendAndListRecent -v
```

Expected: FAIL because `newHistoryStore` and `HistoryRecord` are not defined.

- [x] **Step 3: Implement JSONL history store**

Create `history.go`:

```go
package webapp

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"time"
)

type HistoryRecord struct {
	Time    string `json:"time"`
	Action  string `json:"action"`
	Agent   string `json:"agent,omitempty"`
	Scope   string `json:"scope,omitempty"`
	WorkDir string `json:"workdir,omitempty"`
	Status  string `json:"status"`
	Request any    `json:"request,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

type historyStore struct {
	path string
	now  func() time.Time
}

func newHistoryStore(path string) *historyStore {
	return &historyStore{path: path, now: time.Now}
}

func (s *historyStore) Append(record HistoryRecord) error {
	if record.Time == "" {
		record.Time = s.now().UTC().Format(time.RFC3339)
	}
	if record.Status == "" {
		record.Status = "ok"
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (s *historyStore) List(limit int) ([]HistoryRecord, error) {
	f, err := os.Open(s.path)
	if os.IsNotExist(err) {
		return []HistoryRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	items := make([]HistoryRecord, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var item HistoryRecord
		if err := json.Unmarshal(scanner.Bytes(), &item); err != nil {
			continue
		}
		items = append(items, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	slices.Reverse(items)
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}
```

- [x] **Step 4: Add manager history path and server integration**

Add to `manager.go`:

```go
func (m *Manager) History(limit int) ([]HistoryRecord, error) {
	return newHistoryStore(m.historyFile()).List(limit)
}

func (m *Manager) historyFile() string {
	return filepath.Join(filepath.Dir(m.configFile), "skillc-web-history.jsonl")
}
```

Add `path/filepath` import to `manager.go`.

Add to `manager_server.go`:

```go
func (s *ManagerServer) handleHistory(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	items, err := s.manager.History(100)
	writeResult(w, items, err)
}

func (s *ManagerServer) recordHistory(r *http.Request, action string, req any, result any, err error) {
	status := "ok"
	errMsg := ""
	if err != nil {
		status = "error"
		errMsg = err.Error()
	}
	managerReq := managerReqFromQuery(r)
	_ = newHistoryStore(s.manager.historyFile()).Append(HistoryRecord{
		Action: action,
		Agent: managerReq.Agent,
		Scope: managerReq.Scope,
		WorkDir: managerReq.WorkDir,
		Status: status,
		Request: req,
		Result: result,
		Error: errMsg,
	})
}
```

Register route:

```go
mux.HandleFunc("/api/history", s.handleHistory)
```

- [x] **Step 5: Add history endpoint tests**

Append to `manager_server_test.go`:

```go
func TestManagerServerHistoryEndpointReturnsRecords(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeWebActionFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)
	_ = newHistoryStore(server.manager.historyFile()).Append(HistoryRecord{Action: "source.add", Status: "ok"})

	rec := performManagerRequest(server, http.MethodGet, "/api/history")

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"action":"source.add"`)
}
```

Run:

```bash
go test ./internal/app/webapp -run 'TestHistoryStore|TestManagerServerHistory' -v
```

Expected: PASS.

- [x] **Step 6: Add History view to static page**

Add nav button:

```html
<button data-view="history">History</button>
```

Add view:

```html
<section id="view-history" class="view">
  <div class="section-head"><h3>History</h3><span class="hint">Last 100 Web write actions</span></div>
  <div id="history-table"></div>
</section>
```

Extend state:

```js
history: []
```

Add `/api/history` to `loadAll()` Promise list and assign `state.history`.

Add renderer:

```js
function renderHistory() {
  var rows = state.history.map(function (item) {
    return '<tr><td>' + esc(item.time) + '</td><td>' + esc(item.action) +
      '</td><td>' + esc(item.status) + '</td><td>' + esc(item.agent || '') +
      '</td><td>' + esc(item.scope || '') + '</td><td class="wrap">' + esc(item.error || '') + '</td></tr>';
  });
  byId('history-table').innerHTML = table(['Time', 'Action', 'Status', 'Agent', 'Scope', 'Error'], rows, 'No Web write actions recorded.');
}
```

Call `renderHistory()` from `renderAll()`.

- [x] **Step 7: Commit history support**

Run:

```bash
git add internal/app/webapp/history.go internal/app/webapp/history_test.go internal/app/webapp/manager.go internal/app/webapp/manager_server.go internal/app/webapp/manager_server_test.go internal/app/webapp/manager_static.go
git commit -m "feat(skillc): add web action history"
```

## Task 7: Documentation and Final Verification

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/TODO.md`
- Modify: `docs/design/skillc-v0-enhance-design.md`
- Modify: `docs/superpowers/plans/2026-06-15-skillc-v0-phase6-web-current-project-management.md`

- [x] **Step 1: Update README files**

Add to `README.zh-CN.md` Web section:

```markdown
`skillc web` 当前支持本地当前项目管理：查看 source/profile/status/install-map/version-drift，计划后确认执行 profile apply、update、source add/sync/remove、profile save/from-installed/from-collection 和 uninstall。所有 Web 写操作都会先展示 plan，执行请求必须包含 `confirm:true`，并写入本地 `skillc-web-history.jsonl` 历史记录。
```

Add this paragraph to `README.md`:

```markdown
`skillc web` supports current-project management: source/profile/status/install-map/version-drift views, guarded profile apply and update, source add/sync/remove, profile save/from-installed/from-collection, and uninstall. Every Web write action is plan-first, requires `confirm:true` on the run request, and appends a local `skillc-web-history.jsonl` record.
```

- [x] **Step 2: Update design and task docs**

Update `docs/design/skillc-v0-enhance-design.md`:

- Add a revision row for Phase 6 completion.
- Update Phase 6 status from planned to completed.
- Keep cross-project batch update, Registry, precise checksum/commit drift, remote permissions, and source ID cleanup listed as remaining work.

Update `docs/TODO.md`:

- Mark the Web management sub-bullet for current-project source/profile/uninstall/history as completed.
- Keep top-level Web management unchecked because cross-project batch update and full install/search detail workflows remain.

- [x] **Step 3: Run focused tests**

Run:

```bash
go test ./internal/app/profileapp ./internal/app/installapp ./internal/app/webapp
```

Expected: PASS.

- [x] **Step 4: Run full verification**

Run:

```bash
go test ./...
```

Expected: PASS.

- [x] **Step 5: Update this plan checkbox statuses and verification record**

Add a verification section at the end of this file:

```markdown
## Verification

- `go test ./internal/app/profileapp ./internal/app/installapp ./internal/app/webapp`
- `go test ./...`
```

Check off all completed steps in this plan.

- [x] **Step 6: Commit documentation and verification**

Run:

```bash
git add README.md README.zh-CN.md docs/TODO.md docs/design/skillc-v0-enhance-design.md docs/superpowers/plans/2026-06-15-skillc-v0-phase6-web-current-project-management.md
git commit -m "docs(skillc): document phase 6 web management"
```

## Self-Review Checklist

- [x] Source add/sync/remove has plan and confirm run endpoints.
- [x] Source remove plan exposes installed/profile/index/collection impact.
- [x] Profile save/from-installed/from-collection has plan and confirm run endpoints.
- [x] Uninstall has app-level plan/run and Web endpoints.
- [x] Every Web run endpoint requires `confirm:true`.
- [x] Every Web run endpoint records history.
- [x] Static UI exposes source, profile, uninstall, and history controls without external assets.
- [x] Cross-project update and Registry remain out of Phase 6 scope.
- [x] `go test ./...` passes before claiming implementation complete.

## Remaining After Phase 6

- Cross-project project registry, project selection, per-project confirmation, and `update --all-projects`.
- Registry discovery model and `skillc registry ...` commands.
- Source UX cleanup: custom `--id/--name`, no forced `local-` / `git-` prefix for new source IDs, `source info <id>`.
- Precise drift detection using Git commit / resolved ref and local checksum.
- Project manifest / `skillc.profile.yaml` export/import flow.
- Remote Web access and multi-user permission model.

## Verification

- `go test ./internal/app/profileapp ./internal/app/installapp ./internal/app/webapp`
- `go test ./...`
