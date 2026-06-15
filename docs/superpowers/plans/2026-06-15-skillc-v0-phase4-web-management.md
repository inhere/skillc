# Skillc v0 Phase 4 Web Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增 `skillc web` 本地管理界面，让用户能查看 source、collection、profile、当前项目状态、跨项目安装分布和版本差异，并通过 plan-first 的方式预览 profile apply / update 操作。

**Architecture:** 在现有 `sourceapp`、`searchapp`、`profileapp`、`statusapp`、`updateapp` 之上新增 Web 管理查询层，不在 HTTP handler 中复制业务规则。第一轮新增轻量 project registry / install map 能力，基于当前 JSON lock 中的 project scope key 聚合项目安装记录；Web 写入口只返回 plan 或调用已有 dry-run 语义，不直接执行 destructive 操作。现有 `show --web` skill 文件查看器保留，并把可复用的文件浏览能力收敛为 Web 管理的 skill detail 子能力。

**Tech Stack:** Go, `net/http`, `html/template`, embedded static HTML/CSS/JS via Go string or `embed`, existing app/domain/infra packages, Go unit tests with `github.com/gookit/goutil/testutil/assert`, HTTP handler tests with `httptest`.

---

| 修订时间 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-15 | v0.1 | Codex | 基于 Phase 1/2/3 完成状态输出 Phase 4 Web 管理 MVP 实施计划 |

相关文档：

- 设计文档：`docs/design/skillc-v0-enhance-design.md`
- 参考分析：`docs/design/skillc-reference-projects-analysis.md`
- Phase 1 计划：`docs/superpowers/plans/2026-06-13-skillc-v0-phase1-profile.md`
- Phase 2 计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md`
- Phase 3 计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase3-interactive-selection.md`
- 任务入口：`docs/TODO.md`

## Phase 4 Scope

本期做：

- 新增 `skillc web [--host 127.0.0.1] [--port 8080]`，作为本地管理入口。
- 新增 Web management service，提供 dashboard、sources、collections、skills、profiles、current status、install map、version drift 的查询模型。
- 新增轻量 project registry / install map：从 lock 的 project scope key 聚合项目、agent、scope、profile、skill、source、version，不引入数据库。
- Web API 提供只读 JSON endpoints：
  - `GET /api/summary`
  - `GET /api/sources`
  - `GET /api/collections`
  - `GET /api/skills`
  - `GET /api/profiles`
  - `GET /api/status`
  - `GET /api/install-map`
  - `GET /api/version-drift`
- Web API 提供 plan-first endpoints：
  - `POST /api/profiles/{name}/plan`
  - `POST /api/update/plan`
- Web 页面第一轮做本地单页管理 UI：Dashboard、Sources、Profiles、Skills、Projects、Version Drift。
- 所有写操作按钮第一轮只展示计划结果，不直接改文件；真正执行 install/update/apply 留到 Phase 4 后续小阶段。
- 默认只监听 `127.0.0.1`，不做鉴权、远程管理、多用户。
- 更新 README、README.zh-CN、TODO 和设计文档。

本期不做：

- 不做远程 Web 访问、登录、账号或 token。
- 不做 Web 中直接执行安装、卸载、更新、删除 source。
- 不做数据库、后台 daemon 或常驻项目扫描。
- 不做完整操作日志。
- 不做 deployed file hash / checksum drift 精确检测。
- 不做 Git source commit drift 的完整 compare，仅展示 lock/index/source 上已有 version/resolved_ref/checksum 字段能支持的差异。
- 不替换 `show --web`，只保留并为后续复用预留接口。

## User-Facing Behavior

### Start Web

```bash
skillc web
skillc web --port 8090
skillc web --host 127.0.0.1 --port 8090
```

启动输出：

```text
Skillc web manager started: http://127.0.0.1:8080
Project: <current working directory>
Press Ctrl+C to stop
```

### Web Screens

Dashboard：

- 当前项目路径、agent、scope。
- source 数量、profile 数量、indexed skills 数量。
- 当前项目 status summary：installed/missing/outdated/orphan/unmanaged/source-error。
- update candidates 摘要。

Sources：

- source ID、name、type、status、path/url、ref、resolved_ref、last_sync_at、error。
- 每个 source 下的 collections 和 skill count。

Profiles：

- profile name、description、default agent/scope、target count。
- profile apply plan 预览，调用 `profileapp.PlanApply`。

Skills：

- indexed skills 搜索与列表。
- source、collection、version、installed projects count。
- 第一轮不内嵌完整 skill 文件查看器，只提供 skill metadata；后续可接入 `show --web` 文件浏览。

Projects / Install Map：

- project path、agent、scope、profile、skill count、outdated/missing count。
- skill -> projects/agents/scopes/profile/version 的安装分布。

Version Drift：

- 按 source-qualified skill 聚合跨项目版本。
- 显示 current versions、latest index version、affected projects。
- `POST /api/update/plan` 返回将会更新哪些项目/agent/scope 的计划，不直接执行。

## File Structure

新增文件：

- `internal/app/webapp/manager.go`
  - 定义 Web 管理查询 service。
  - 汇总 config、source、index、profiles、status、install map、version drift。

- `internal/app/webapp/manager_test.go`
  - 覆盖 summary、sources、profiles、install map、version drift。

- `internal/app/webapp/project_index.go`
  - 从 lock records 构建项目安装索引。
  - 只读取 lock，不扫描磁盘，不引入 DB。

- `internal/app/webapp/project_index_test.go`
  - 覆盖 project/global scope 聚合、agent 展开、profile 聚合、版本差异分组。

- `internal/app/webapp/manager_server.go`
  - 新增 Web 管理 server、API routes、static page handler。
  - 与现有 skill viewer server 分离，避免 `server.go` 继续膨胀。

- `internal/app/webapp/manager_server_test.go`
  - 使用 `httptest` 覆盖 JSON API、method validation、path validation。

- `internal/app/webapp/manager_static.go`
  - 存放第一轮单页 HTML/CSS/JS 模板。
  - 不引入 Node/Vite/React，保持单二进制。

- `internal/cli/web_cmd.go`
  - 新增 `skillc web` 命令注册与 host/port 参数。

修改文件：

- `internal/cli/app.go`
  - 注册 `buildWebCommand()`。

- `internal/cli/app_test.go`
  - 覆盖 web 命令注册、host/port 参数传递。

- `internal/app/webapp/server.go`
  - 只做小范围整理：保留 `show --web` viewer；必要时抽出 `safeFilePath` 等可复用 helper。

- `README.md`
  - 增加 `skillc web` 命令说明和本期 Web 能力边界。

- `README.zh-CN.md`
  - 增加中文命令说明。

- `docs/TODO.md`
  - 增加 Phase 4 计划链接和状态。

- `docs/design/skillc-v0-enhance-design.md`
  - 增加 Phase 4 计划链接、修订记录和下一步建议。

- `docs/superpowers/plans/2026-06-15-skillc-v0-phase4-web-management.md`
  - 实施中持续更新 checkbox 和验证记录。

## API Model

建议新增核心类型：

```go
package webapp

type ManagerReq struct {
	Agent   string
	Scope   string
	WorkDir string
}

type Summary struct {
	ProjectPath  string
	SourceCount  int
	ProfileCount int
	SkillCount   int
	Status       StatusSummary
}

type StatusSummary struct {
	Installed   int
	Missing     int
	Outdated    int
	Orphan      int
	Unmanaged   int
	SourceError int
}

type ProjectInstall struct {
	ProjectPath         string `json:"project_path"`
	Scope               string `json:"scope"`
	Agent               string `json:"agent"`
	Profile             string `json:"profile,omitempty"`
	SkillID             string `json:"skill_id"`
	QualifiedName       string `json:"qualified_name,omitempty"`
	SourceQualifiedName string `json:"source_qualified_name,omitempty"`
	SourceID            string `json:"source_id,omitempty"`
	Version             string `json:"version,omitempty"`
}

type VersionDriftGroup struct {
	SkillID             string           `json:"skill_id"`
	SourceQualifiedName string           `json:"source_qualified_name,omitempty"`
	SourceID            string           `json:"source_id,omitempty"`
	LatestVersion       string           `json:"latest_version,omitempty"`
	Versions            []VersionBucket  `json:"versions"`
}

type VersionBucket struct {
	Version  string           `json:"version"`
	Projects []ProjectInstall `json:"projects"`
}
```

Identity rules：

- install map / drift canonical identity 优先级：`SourceQualifiedName -> QualifiedName -> SourceID + "\x00" + SkillID -> SkillID`。
- version drift 只在同一个 grouping key 下比较。
- latest version 来自 repo index；lookup 对旧 lock metadata 使用 alias lookup，同一个 index skill 至少支持 `SourceID + "\x00" + ID`、`SourceQualifiedName`、`QualifiedName`、`ID`。
- drift 分组优先使用 installed record 上的精确 `SourceQualifiedName` / `QualifiedName`；`SourceID + SkillID` 与 bare `SkillID` 只作为非 ambiguous alias 处理 rename 或旧 lock metadata。
- alias 若映射到多个不同 canonical identity，则视为 ambiguous，不用于自动分组或 latest 解析。
- drift group 的 `SkillID` / `SourceID` / `SourceQualifiedName` 优先使用 latest index 的 canonical metadata；只有拿不到 latest 时才 fallback 到 installed record。
- 如果 index 没有对应项，latest 为空，drift group 仍展示 current versions。
- global scope 记录使用 project path `__global__` 展示，不伪装成当前项目。

## Task 1: Add Project Install Index

**Files:**
- Create: `internal/app/webapp/project_index.go`
- Create: `internal/app/webapp/project_index_test.go`

- [x] **Step 1: Write failing project index tests**

Create `internal/app/webapp/project_index_test.go`:

```go
package webapp

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
)

func TestBuildProjectInstallIndexExpandsAgentsAndProfiles(t *testing.T) {
	projectA := filepath.Clean("/work/project-a")
	records := lockpkg.File{
		projectA: {
			{
				SkillID:             "go-pro",
				QualifiedName:       "tools/go-pro",
				SourceQualifiedName: "gstack/tools/go-pro",
				SourceID:            "gstack",
				Version:             "1.0.0",
				Profile:             "go-dev",
				Agents:              []string{"universal", "codex"},
			},
		},
	}

	items := BuildProjectInstallIndex(records)

	assert.Len(t, items, 2)
	assert.Eq(t, projectA, items[0].ProjectPath)
	assert.Eq(t, "project", items[0].Scope)
	assert.Eq(t, "go-dev", items[0].Profile)
	assert.Eq(t, "gstack/tools/go-pro", items[0].SourceQualifiedName)
}

func TestBuildProjectInstallIndexKeepsGlobalScopeSeparate(t *testing.T) {
	records := lockpkg.File{
		lockpkg.GlobalKey: {
			{SkillID: "review", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}},
		},
	}

	items := BuildProjectInstallIndex(records)

	assert.Len(t, items, 1)
	assert.Eq(t, lockpkg.GlobalKey, items[0].ProjectPath)
	assert.Eq(t, "global", items[0].Scope)
}
```

Also add `TestBuildVersionDriftGroupsBySourceQualifiedIdentity`, where two projects install the same source-qualified skill with versions `1.0.0` and `2.0.0`, and assert one drift group with two version buckets.

- [x] **Step 2: Run tests to verify failure**

Run:

```bash
go test ./internal/app/webapp -run 'TestBuildProjectInstallIndex|TestBuildVersionDrift' -count=1
```

Expected: FAIL because project index helpers do not exist.

- [x] **Step 3: Implement project install index**

Create `internal/app/webapp/project_index.go` with:

```go
package webapp

import (
	"sort"

	"github.com/inhere/skillc/internal/app/apputil"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
)

type ProjectInstall struct {
	ProjectPath         string `json:"project_path"`
	Scope               string `json:"scope"`
	Agent               string `json:"agent"`
	Profile             string `json:"profile,omitempty"`
	SkillID             string `json:"skill_id"`
	QualifiedName       string `json:"qualified_name,omitempty"`
	SourceQualifiedName string `json:"source_qualified_name,omitempty"`
	SourceID            string `json:"source_id,omitempty"`
	Version             string `json:"version,omitempty"`
}

type VersionDriftGroup struct {
	SkillID             string          `json:"skill_id"`
	SourceQualifiedName string          `json:"source_qualified_name,omitempty"`
	SourceID            string          `json:"source_id,omitempty"`
	LatestVersion       string          `json:"latest_version,omitempty"`
	Versions            []VersionBucket `json:"versions"`
}

type VersionBucket struct {
	Version  string           `json:"version"`
	Projects []ProjectInstall `json:"projects"`
}
```

Add functions:

- `BuildProjectInstallIndex(records lockpkg.File) []ProjectInstall`
- `BuildVersionDrift(items []ProjectInstall, index []skill.Skill) []VersionDriftGroup`
- `projectInstallKey(item ProjectInstall) string`
- `latestVersionByInstallKey(index []skill.Skill) map[string]string`

Implementation rules:

- Expand every lock record agent into one `ProjectInstall`.
- Records without agents are ignored because install map is agent-attributed; lock writers should persist agent names.
- For normal scope keys, `Scope` comes from `apputil.ScopeFromKey(scopeKey)`.
- For `lockpkg.GlobalKey`, keep `ProjectPath` as the original scope key and expose `Scope` as `global` for Web install map display.
- `ProjectPath` is the original scope key.
- Sort by project path, skill ID, agent.
- Version buckets sorted by version.
- Only return drift groups where either:
  - more than one current version exists, or
  - latest version is non-empty and at least one current version differs from latest.

- [x] **Step 4: Run webapp project index tests**

Run:

```bash
go test ./internal/app/webapp -run 'TestBuildProjectInstallIndex|TestBuildVersionDrift' -count=1
```

Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/app/webapp/project_index.go internal/app/webapp/project_index_test.go
git commit -m "feat(web): add project install index"
```

**Verification note (2026-06-15):** `go test ./internal/app/webapp -run 'TestBuildProjectInstallIndex|TestBuildVersionDrift' -count=1` and `go test ./internal/app/webapp -count=1` pass. Added drift coverage to verify numeric dotted version selection chooses `1.10.0` over `1.9.0`, version buckets sort numerically so `1.9.0` appears before `1.10.0`, rename scenarios can still resolve through non-ambiguous aliases, same source same skill ID across different collections does not merge, latest lookup supports old lock metadata aliases while keeping ambiguous aliases unresolved, drift group identity metadata prefers latest index canonical fields over stale installed record names, and records without agent attribution are skipped from the install index.

## Task 2: Add Web Manager Query Service

**Files:**
- Create: `internal/app/webapp/manager.go`
- Create: `internal/app/webapp/manager_test.go`

- [x] **Step 1: Write failing manager tests**

Create tests for:

- `TestManager_SummaryCountsSourcesProfilesSkillsAndStatus`
- `TestManager_ListSourcesReturnsConfiguredSources`
- `TestManager_ListProfilesReturnsSavedProfiles`
- `TestManager_InstallMapReadsLockRecords`
- `TestManager_VersionDriftUsesIndexLatestVersion`

Use temp config, lock, and index files. Use `configstore.NewYAMLStore().Save`, `lockstore.NewStore().Save`, and `repoindex.NewStore().Save`.

Minimum assertion shape:

```go
func TestManager_SummaryCountsSourcesProfilesSkillsAndStatus(t *testing.T) {
	baseDir := t.TempDir()
	configFile, conf := writeWebManagerFixture(t, baseDir)

	result, err := NewManager(configFile, baseDir).Summary(ManagerReq{
		Agent: "universal",
		Scope: "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Eq(t, baseDir, result.ProjectPath)
	assert.Eq(t, 1, result.SourceCount)
	assert.Eq(t, 1, result.ProfileCount)
	assert.Eq(t, 2, result.SkillCount)
	assert.Eq(t, 1, result.Status.Installed)
	_ = conf
}
```

- [x] **Step 2: Run manager tests to verify failure**

Run:

```bash
go test ./internal/app/webapp -run 'TestManager_' -count=1
```

Expected: FAIL because `Manager` does not exist.

- [x] **Step 3: Implement manager service**

Create `internal/app/webapp/manager.go`:

```go
package webapp

import (
	"os"

	"github.com/inhere/skillc/internal/app/configapp"
	"github.com/inhere/skillc/internal/app/profileapp"
	"github.com/inhere/skillc/internal/app/searchapp"
	"github.com/inhere/skillc/internal/app/statusapp"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/profile"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

type ManagerReq struct {
	Agent   string
	Scope   string
	WorkDir string
}

type Summary struct {
	ProjectPath  string        `json:"project_path"`
	SourceCount  int           `json:"source_count"`
	ProfileCount int           `json:"profile_count"`
	SkillCount   int           `json:"skill_count"`
	Status       StatusSummary `json:"status"`
}

type StatusSummary struct {
	Installed   int `json:"installed"`
	Missing     int `json:"missing"`
	Outdated    int `json:"outdated"`
	Orphan      int `json:"orphan"`
	Unmanaged   int `json:"unmanaged"`
	SourceError int `json:"source_error"`
}

type Manager struct {
	configFile string
	baseDir    string
}
```

Add:

- `NewManager(configFile, baseDir string) *Manager`
- `Summary(req ManagerReq) (Summary, error)`
- `Sources() ([]sourcepkg.Source, error)`
- `Collections(sourceID string) ([]repoindex.SourceCollectionSummary, error)`
- `Skills(keyword string) ([]skill.Skill, error)`
- `Profiles() ([]profile.NamedProfile, error)`
- `Status(req ManagerReq) (statusapp.Result, error)`
- `InstallMap() ([]ProjectInstall, error)`
- `VersionDrift() ([]VersionDriftGroup, error)`
- `PlanProfileApply(name string, req ManagerReq) (profile.ApplyPlan, error)`

Use existing services:

- `configapp.NewService(s.configFile, s.baseDir).Show()`
- `profileapp.NewService(s.configFile, s.baseDir).List()` / `PlanApply`
- `searchapp.NewService(config.IndexFile)`
- `statusapp.NewService(s.configFile, s.baseDir).Run()`
- `lockstore.NewStore().Load(config.LockFile)`
- `repoindex.NewStore().Load(config.IndexFile)`

Handle missing lock/index files as empty results where appropriate.

- [x] **Step 4: Run manager tests**

Run:

```bash
go test ./internal/app/webapp -run 'TestManager_' -count=1
```

Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/app/webapp/manager.go internal/app/webapp/manager_test.go
git commit -m "feat(web): add management query service"
```

**Verification note (2026-06-15):** `go test ./internal/app/webapp -run 'TestManager_' -count=1` passes. Manager service delegates to existing config/source/search/profile/status services, treats missing lock/index as empty for derived views, and exposes summary, source/profile lists, install map, version drift, skill search, collections and profile apply plan queries.

## Task 3: Add Web Manager HTTP API

**Files:**
- Create: `internal/app/webapp/manager_server.go`
- Create: `internal/app/webapp/manager_server_test.go`

- [x] **Step 1: Write failing HTTP API tests**

Create tests using `httptest.NewServer` or `httptest.NewRecorder`:

- `TestManagerServerSummaryEndpoint`
- `TestManagerServerProfilesEndpoint`
- `TestManagerServerInstallMapEndpoint`
- `TestManagerServerProfilePlanEndpoint`
- `TestManagerServerRejectsInvalidMethods`

Example:

```go
func TestManagerServerSummaryEndpoint(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	req := httptest.NewRequest(http.MethodGet, "/api/summary?agent=universal&scope=project", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"source_count":1`)
	assert.Contains(t, rec.Body.String(), `"profile_count":1`)
}
```

- [x] **Step 2: Run HTTP tests to verify failure**

Run:

```bash
go test ./internal/app/webapp -run 'TestManagerServer' -count=1
```

Expected: FAIL because manager server does not exist.

- [x] **Step 3: Implement API server**

Create `internal/app/webapp/manager_server.go` with:

- `type ManagerServer struct`
- `func NewManagerServer(configFile, baseDir string) *ManagerServer`
- `func (s *ManagerServer) Handler() http.Handler`
- `func (s *ManagerServer) Serve(host string, port int) error`

Routes:

```text
GET  /
GET  /api/summary
GET  /api/sources
GET  /api/collections
GET  /api/skills
GET  /api/profiles
GET  /api/status
GET  /api/install-map
GET  /api/version-drift
POST /api/profiles/{name}/plan
POST /api/update/plan
```

Rules:

- JSON response helper must set `Content-Type: application/json; charset=utf-8` before `WriteHeader`.
- Non-GET on read endpoints returns `405`.
- Invalid profile plan path returns `404`.
- Query params:
  - `agent`, `scope`, `keyword`, `source`.
- `POST /api/update/plan` first round returns `statusapp` update candidates (`outdated` and `missing`) as a plan-like JSON object; it does not call `updateapp.Run`.

- [x] **Step 4: Run HTTP API tests**

Run:

```bash
go test ./internal/app/webapp -run 'TestManagerServer' -count=1
```

Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/app/webapp/manager_server.go internal/app/webapp/manager_server_test.go
git commit -m "feat(web): add management API server"
```

**Verification note (2026-06-15):** `go test ./internal/app/webapp -run 'TestManagerServer' -count=1` passes. API routes expose summary, sources, collections, skills, profiles, status, install map, version drift, profile apply plan and update plan endpoints with method validation; update plan only returns missing/outdated status items and does not mutate files.

## Task 4: Add Static Web Management UI

**Files:**
- Create: `internal/app/webapp/manager_static.go`
- Modify: `internal/app/webapp/manager_server.go`
- Modify: `internal/app/webapp/manager_server_test.go`

- [x] **Step 1: Add failing static page tests**

Add tests:

- `TestManagerServerIndexPageContainsAppShell`
- `TestManagerServerStaticPageDoesNotUseExternalAssets`

Assertions:

```go
assert.Contains(t, body, "Skillc")
assert.Contains(t, body, "Dashboard")
assert.Contains(t, body, "Sources")
assert.Contains(t, body, "Profiles")
assert.Contains(t, body, "Version Drift")
assert.NotContains(t, body, "https://")
assert.NotContains(t, body, "http://")
```

- [x] **Step 2: Run static page tests to verify failure**

Run:

```bash
go test ./internal/app/webapp -run 'TestManagerServerIndexPage|TestManagerServerStaticPage' -count=1
```

Expected: FAIL until the page exists.

- [x] **Step 3: Implement single-page UI**

Create `internal/app/webapp/manager_static.go`:

```go
package webapp

const managerHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Skillc Web Manager</title>
<style>
/* Place the complete self-contained management UI CSS here. */
</style>
</head>
<body>
<aside id="nav"></aside>
<main id="app"></main>
<script>
/* Place the complete self-contained management UI JavaScript here. */
</script>
</body>
</html>`
```

UI requirements:

- First screen is the actual management dashboard, not a landing page.
- Quiet operational tool style: dense, readable, low decoration.
- No nested cards.
- No external JS/CSS/CDN.
- Use native HTML/CSS/JS only.
- Use tabs or sidebar for Dashboard, Sources, Profiles, Skills, Projects, Version Drift.
- Use tables for scan-heavy data.
- Use icon-like text symbols only where stable and ASCII-compatible; avoid emoji in generated code.
- Keep border radius <= 8px.
- No purple-blue gradient, decorative orbs, or marketing hero.
- Text must not depend on viewport-scaled font sizes.

Client behavior:

- On load, fetch `/api/summary`, `/api/sources`, `/api/profiles`, `/api/status`, `/api/install-map`, `/api/version-drift`.
- Render empty states for no sources/profiles/install map.
- Provide search input for skills that calls `/api/skills?keyword=<value>`.
- Profile row has a `Plan` button that calls `/api/profiles/<name>/plan` and renders returned plan.
- Version Drift row has a `Plan update` button that calls `/api/update/plan`.

- [x] **Step 4: Wire page handler**

In `ManagerServer.Handler()`, route `/` to `managerHTML` with `Content-Type: text/html; charset=utf-8`.

- [x] **Step 5: Run static page tests**

Run:

```bash
go test ./internal/app/webapp -run 'TestManagerServerIndexPage|TestManagerServerStaticPage' -count=1
```

Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add internal/app/webapp/manager_static.go internal/app/webapp/manager_server.go internal/app/webapp/manager_server_test.go
git commit -m "feat(web): add management UI shell"
```

**Verification note (2026-06-15):** `go test ./internal/app/webapp -run 'TestManagerServerIndexPage|TestManagerServerStaticPage' -count=1` and `go test ./internal/app/webapp -count=1` pass. The management page is a self-contained operational UI with Dashboard, Sources, Profiles, Skills, Projects and Version Drift views, no external assets, and plan-only profile/update actions.

## Task 5: Add `skillc web` CLI

**Files:**
- Create: `internal/cli/web_cmd.go`
- Modify: `internal/cli/app.go`
- Modify: `internal/cli/app_test.go`

- [ ] **Step 1: Write failing CLI tests**

Add tests:

- `TestNewApp_RegistersWebCommand`
- `TestWebCommandPassesHostAndPort`

Use a server factory injection to avoid binding a real port:

```go
type webServerStub struct {
	host string
	port int
}

func (s *webServerStub) Serve(host string, port int) error {
	s.host = host
	s.port = port
	return nil
}
```

- [ ] **Step 2: Run CLI tests to verify failure**

Run:

```bash
go test ./internal/cli -run 'TestNewApp_RegistersWebCommand|TestWebCommand' -count=1
```

Expected: FAIL because `web` command is not registered.

- [ ] **Step 3: Implement CLI command**

Create `internal/cli/web_cmd.go`:

```go
package cli

import (
	"github.com/gookit/gcli/v3"
	"github.com/inhere/skillc/internal/app/webapp"
)

type webManagerServer interface {
	Serve(host string, port int) error
}

var newWebManagerServer = func(configFile string, baseDir string) webManagerServer {
	return webapp.NewManagerServer(configFile, baseDir)
}

func buildWebCommand() *gcli.Command {
	var host string
	var port int
	return &gcli.Command{
		Name: "web",
		Desc: "Start local web manager",
		Config: func(c *gcli.Command) {
			c.StrOpt(&host, "host", "", "127.0.0.1", "web server host")
			c.IntOpt(&port, "port", "p", 8080, "web server port")
		},
		Func: func(c *gcli.Command, _ []string) error {
			cwd := getWorkdir()
			return newWebManagerServer(defaultConfigFile(cwd), cwd).Serve(host, port)
		},
	}
}
```

Register it in `NewApp()` after `show` or before `install`.

- [ ] **Step 4: Run CLI web tests**

Run:

```bash
go test ./internal/cli -run 'TestNewApp_RegistersWebCommand|TestWebCommand' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/web_cmd.go internal/cli/app.go internal/cli/app_test.go
git commit -m "feat(cli): add web manager command"
```

## Task 6: Documentation and Plan Links

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/TODO.md`
- Modify: `docs/design/skillc-v0-enhance-design.md`
- Modify: `docs/superpowers/plans/2026-06-15-skillc-v0-phase4-web-management.md`

- [ ] **Step 1: Update README command reference**

Add:

````markdown
### `web` - Local management UI

```bash
skillc web
skillc web --port 8090
```

The web manager runs on `127.0.0.1` by default and shows sources, profiles, current status, project install map, and version drift. The first v0 slice is plan-first: profile apply and update actions show plans before later execution support is added.
````

- [ ] **Step 2: Update Chinese README command reference**

Add equivalent Chinese section:

````markdown
### `web` - 本地管理界面

```bash
skillc web
skillc web --port 8090
```

Web 管理界面默认监听 `127.0.0.1`，用于查看 sources、profiles、当前项目状态、项目安装分布和版本差异。第一轮 Web 写操作只展示计划，不直接执行安装或更新。
````

- [ ] **Step 3: Update TODO and design docs**

Update:

- `docs/TODO.md`
  - Add Phase 4 plan link.
  - Mark Web management item as planned, not complete.

- `docs/design/skillc-v0-enhance-design.md`
  - Add revision row.
  - Add related plan link.
  - Add "四期开发计划" near previous phase links.
  - Replace outdated "下一步建议" with Phase 4 Web management MVP.

- [ ] **Step 4: Commit docs**

```bash
git add README.md README.zh-CN.md docs/TODO.md docs/design/skillc-v0-enhance-design.md docs/superpowers/plans/2026-06-15-skillc-v0-phase4-web-management.md
git commit -m "docs: add skillc phase 4 web plan"
```

## Task 7: Full Verification and Review

**Files:**
- Modify: `docs/superpowers/plans/2026-06-15-skillc-v0-phase4-web-management.md`

- [ ] **Step 1: Run focused tests**

Run:

```bash
go test ./internal/app/webapp ./internal/cli -count=1
```

Expected: PASS.

- [ ] **Step 2: Run full test suite**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 3: Run docs sanity check**

Run:

```bash
rg -n -- "Phase 4|四期|skillc web|web manager|install map|version drift|Web 管理" README.md README.zh-CN.md docs/TODO.md docs/design/skillc-v0-enhance-design.md docs/superpowers/plans/2026-06-15-skillc-v0-phase4-web-management.md
```

Expected: docs contain Phase 4 references and do not claim Web execution support before it exists.

- [ ] **Step 4: Manual smoke test**

Run:

```bash
go run ./cmd/skillc web --port 18080
```

Open:

```text
http://127.0.0.1:18080
```

Expected:

- Page loads without external assets.
- Dashboard data is not blank.
- `/api/summary` returns JSON.
- `/api/install-map` returns JSON.
- `/api/version-drift` returns JSON.

Stop the server before committing final plan completion.

- [ ] **Step 5: Update final verification note**

Append after replacing the date with the actual verification date:

```markdown
**Verification note (2026-06-15):** `go test ./internal/app/webapp ./internal/cli -count=1` and `go test ./...` pass. Manual smoke test of `skillc web --port 18080` confirms the local page and JSON APIs load.
```

- [ ] **Step 6: Commit final checkbox updates**

```bash
git add docs/superpowers/plans/2026-06-15-skillc-v0-phase4-web-management.md
git commit -m "docs: complete skillc phase 4 web plan review"
```

## Self-Review Checklist

- [ ] Web management is a local operational UI, not a landing page.
- [ ] API handlers delegate to app services and do not duplicate install/update/profile rules.
- [ ] First Web slice is plan-first and does not execute writes directly.
- [ ] `show --web` skill viewer remains compatible.
- [ ] Install map is derived from lock records and does not require a DB.
- [ ] Version drift grouping uses stable skill identity and does not merge same IDs from unrelated sources.
- [ ] Web UI uses no external CDN/assets.
- [ ] Web defaults to `127.0.0.1`.
- [ ] Go tests cover service, API and CLI command registration.
- [ ] `go test ./...` passes before marking the plan complete.

## Acceptance Criteria

- `skillc web` starts a local Web manager at `127.0.0.1:8080` by default.
- Web Dashboard shows source/profile/skill counts and current project status summary.
- Web Sources page shows configured sources and source collections.
- Web Profiles page lists profiles and can request an apply plan.
- Web Skills page lists/searches indexed skills.
- Web Projects / Install Map page shows which projects/agents/scopes have installed skills based on lock records.
- Web Version Drift page groups version differences across projects by stable source-qualified identity.
- Web update/profile endpoints return plans and do not mutate files.
- README, README.zh-CN, TODO and design docs reference Phase 4.

## Risks and Guardrails

- Existing `webapp/server.go` is already large because it embeds the skill viewer HTML. Keep management server and static page in separate files.
- Cross-project install map is only as complete as the lock file. Do not claim it discovers projects outside lock records.
- Same skill ID can exist in multiple sources. Always group by source-qualified identity before falling back to ID.
- Web UI must not bypass CLI/app service confirmation semantics. First slice only returns plans.
- Avoid pulling in a frontend build chain for v0. A static embedded page is enough and keeps install simple.
- Avoid dark, decorative, one-hue UI. This is an operational tool; use restrained, scan-friendly layout and tables.
