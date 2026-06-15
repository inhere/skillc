# Skillc v0 Phase 5 Web Execution Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. This plan is written for inline execution and does not require subagents.

**Goal:** 在 Phase 4 Web 管理界面的 plan-first 基础上，新增当前项目范围内的 profile apply 和 update 执行闭环，让 Web 能先生成计划、再显式确认执行、最后刷新状态。

**Architecture:** 继续保持 HTTP handler 很薄，写操作通过 `webapp.Manager` 调用现有 `profileapp.Apply` 和 `updateapp.Run`。所有 Web 执行入口都必须要求请求体包含 `confirm:true`，并返回 Web 专用稳定 JSON 模型，不直接暴露 Go struct 的大写字段名。第一轮只允许操作当前 `skillc web` 启动目录对应的 workdir / agent / scope，不做跨项目批量更新、卸载、source 删除或远程 Web 权限模型。

**Tech Stack:** Go, `net/http`, embedded static HTML/CSS/JS via Go string, existing `profileapp` / `updateapp` services, Go unit tests with `github.com/gookit/goutil/testutil/assert`, HTTP handler tests with `httptest`, full verification via `go test ./...`.

---

| 修订时间 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-15 | v0.1 | Codex | 基于 Phase 4 Web 管理 MVP 输出 Phase 5 Web 执行闭环实施计划 |

相关文档：

- 设计文档：`docs/design/skillc-v0-enhance-design.md`
- 参考分析：`docs/design/skillc-reference-projects-analysis.md`
- Phase 1 计划：`docs/superpowers/plans/2026-06-13-skillc-v0-phase1-profile.md`
- Phase 2 计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md`
- Phase 3 计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase3-interactive-selection.md`
- Phase 4 计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase4-web-management.md`
- 任务入口：`docs/TODO.md`

## Phase 5 Scope

本期做：

- 新增 Web profile apply 执行服务：
  - `POST /api/profiles/{name}/apply`
  - 请求体必须包含 `{"confirm":true}`。
  - 执行前仍由用户通过 UI 先请求 `/api/profiles/{name}/plan`。
  - 后端执行调用 `profileapp.NewService(...).Apply(...)`。
  - 返回 plan、installed、install_failed 等稳定 JSON 字段。
- 新增 Web update 执行服务：
  - `POST /api/update/run`
  - 请求体必须包含 `{"confirm":true}`，可选 `{"target":"..."}`。
  - 执行前 UI 仍先请求 `/api/update/plan`。
  - 后端执行调用 `updateapp.NewService(...).Run(...)`。
  - 返回 updated、skipped、failed、sync_failed、cleanup_failed 等稳定 JSON 字段。
- Web UI 增加执行确认控件：
  - Profile 行继续有 `Plan`，计划生成后出现 `Apply profile`。
  - Dashboard / Version Drift 继续有 `Plan update`，计划生成后出现 `Run update`。
  - 执行按钮必须带浏览器确认或显式确认控件，并发送 `confirm:true`。
  - 执行完成后刷新 summary/status/install-map/version-drift。
- 补充 README、README.zh-CN、TODO、设计文档和本计划验证记录。

本期不做：

- 不做 Web 卸载 skill。
- 不做 Web 删除 source。
- 不做跨项目批量更新所有下游项目。
- 不做后台 job 队列、操作历史持久化或审计日志。
- 不做远程访问、登录、token 或多用户权限。
- 不做 Registry。
- 不改现有 CLI 写操作语义。

## User-Facing Behavior

### Profile Apply

用户打开：

```bash
skillc web
```

在 Profiles 页：

1. 点击某个 profile 的 `Plan`。
2. Plan Output 显示 apply plan。
3. 若用户确认，点击 `Apply profile`。
4. Web 调用 `POST /api/profiles/{name}/apply`，请求体：

```json
{"confirm": true}
```

5. 执行结果显示已安装项和失败项，并刷新 Dashboard / Status。

若缺少确认：

```http
POST /api/profiles/go-dev/apply
Content-Type: application/json

{}
```

返回：

```json
{"error":"confirmation required"}
```

### Update Run

在 Dashboard 或 Version Drift 页：

1. 点击 `Plan update`。
2. Plan Output 显示 missing/outdated 候选。
3. 若用户确认，点击 `Run update`。
4. Web 调用 `POST /api/update/run`，请求体：

```json
{"confirm": true}
```

或指定目标：

```json
{"confirm": true, "target": "gstack/tools/go-pro"}
```

5. 执行结果显示 updated/skipped/failed，并刷新 Dashboard / Status / Drift。

## File Structure

新增文件：

- `internal/app/webapp/manager_actions.go`
  - 定义 Web 写操作请求、结果和转换模型。
  - 封装 `Manager.ApplyProfile(name, req)` 和 `Manager.RunUpdate(req)`。

- `internal/app/webapp/manager_actions_test.go`
  - 覆盖 profile apply 执行、update 执行、结果 JSON 模型所需字段。

修改文件：

- `internal/app/webapp/manager_server.go`
  - 新增执行路由和请求体解析。
  - 扩展 profile path action 解析，使 `/plan` 和 `/apply` 分流。
  - 新增 `POST /api/update/run`。

- `internal/app/webapp/manager_server_test.go`
  - 覆盖 confirm guard、method validation、profile apply endpoint、update run endpoint。

- `internal/app/webapp/manager_static.go`
  - 增加执行按钮、确认逻辑、结果展示和执行后刷新。

- `internal/cli/app_test.go`
  - 可选：不直接修改 CLI 行为；仅当 `skillc web` smoke 需要更稳定测试时补充。

- `README.md`
  - 更新 `web` 命令说明：Web 当前支持计划后确认执行 profile apply 和 update。

- `README.zh-CN.md`
  - 更新中文 Web 能力边界。

- `docs/TODO.md`
  - 增加 Phase 5 计划链接和状态。

- `docs/design/skillc-v0-enhance-design.md`
  - 增加修订记录、Phase 5 计划链接和下一步建议。

- `docs/superpowers/plans/2026-06-15-skillc-v0-phase5-web-execution.md`
  - 实施中持续更新 checkbox 和验证记录。

## API Model

建议新增 Web 请求和响应模型：

```go
package webapp

type actionConfirmReq struct {
	Confirm bool   `json:"confirm"`
	Target  string `json:"target,omitempty"`
}

type actionRuntimeRecord struct {
	SkillID       string `json:"skill_id"`
	SourceID      string `json:"source_id,omitempty"`
	Version       string `json:"version,omitempty"`
	Agent         string `json:"agent,omitempty"`
	Scope         string `json:"scope,omitempty"`
	InstalledPath string `json:"installed_path,omitempty"`
}

type actionErrorItem struct {
	SkillID string `json:"skill_id"`
	Reason  string `json:"reason"`
}

type actionSourceErrorItem struct {
	SourceID string `json:"source_id"`
	Reason   string `json:"reason"`
}

type profileApplyActionResult struct {
	Plan          profile.ApplyPlan     `json:"plan"`
	Installed     []actionRuntimeRecord `json:"installed"`
	InstallFailed []actionErrorItem     `json:"install_failed,omitempty"`
}

type updateRunActionResult struct {
	Updated       []actionRuntimeRecord `json:"updated"`
	Skipped       []actionErrorItem     `json:"skipped,omitempty"`
	Failed        []actionErrorItem     `json:"failed,omitempty"`
	SyncFailed    []actionSourceErrorItem `json:"sync_failed,omitempty"`
	CleanupFailed []actionErrorItem     `json:"cleanup_failed,omitempty"`
}
```

确认规则：

- `POST /api/profiles/{name}/apply` 和 `POST /api/update/run` 都必须解析 JSON body。
- `confirm != true` 时返回 `400` 和 `{"error":"confirmation required"}`。
- JSON 解析失败返回 `400` 和 `{"error":"invalid json body"}`。
- 不允许 GET 执行写操作；返回 `405`。
- profile action path 只允许：
  - `/api/profiles/{name}/plan`
  - `/api/profiles/{name}/apply`

## Task 1: Add Web Action Manager Service

**Files:**
- Create: `internal/app/webapp/manager_actions.go`
- Create: `internal/app/webapp/manager_actions_test.go`

- [x] **Step 1: Write failing manager action tests**

Create `internal/app/webapp/manager_actions_test.go` with tests:

- `TestManager_ApplyProfileInstallsMissingProfileSkills`
- `TestManager_RunUpdateUpdatesOutdatedSkill`
- `TestActionRuntimeRecordUsesStableJSONFields`

Test fixture guidance:

```go
func writeWebActionFixture(t *testing.T, baseDir string) string {
	t.Helper()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	sourceRoot := filepath.Join(baseDir, "source")

	goSource := createWebActionSkillSource(t, sourceRoot, "go-pro", "2.0.0", "updated")
	reviewSource := createWebActionSkillSource(t, sourceRoot, "review", "1.0.0", "review")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{
		Dirname:    ".agents",
		ProjectDir: filepath.Join(baseDir, ".agents"),
	}
	config.Sources = []sourcepkg.Source{{
		ID:     "gstack",
		Name:   "gstack",
		Type:   sourcepkg.TypeLocal,
		Path:   sourceRoot,
		Status: "ready",
	}}
	config.Profiles = map[string]profile.Profile{
		"go-dev": {
			Description:  "Go dev",
			DefaultAgent: "universal",
			DefaultScope: "project",
			Targets:     []profile.Target{{Source: "gstack", Skill: "review"}},
		},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))

	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID:             "go-pro",
			SourceID:            "gstack",
			QualifiedName:       "tools/go-pro",
			SourceQualifiedName: "gstack/tools/go-pro",
			Version:             "1.0.0",
			Agents:              []string{"universal"},
		}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Collection: "tools", QualifiedName: "tools/go-pro", SourceQualifiedName: "gstack/tools/go-pro", Version: "2.0.0", SourceType: sourcepkg.TypeLocal, InstallEntry: ".", Path: goSource},
		{ID: "review", SourceID: "gstack", Collection: "tools", QualifiedName: "tools/review", SourceQualifiedName: "gstack/tools/review", Version: "1.0.0", SourceType: sourcepkg.TypeLocal, InstallEntry: ".", Path: reviewSource},
	}))
	return configFile
}
```

Assertions:

```go
result, err := NewManager(configFile, baseDir).ApplyProfile("go-dev", ManagerReq{
	Agent: "universal", Scope: "project", WorkDir: baseDir,
})
assert.NoErr(t, err)
assert.Eq(t, "go-dev", result.Plan.Profile)
assert.Len(t, result.Installed, 1)
assert.Eq(t, "review", result.Installed[0].SkillID)
_, err = os.Stat(filepath.Join(baseDir, ".agents", "skills", "review"))
assert.NoErr(t, err)
```

Update assertions:

```go
result, err := NewManager(configFile, baseDir).RunUpdate(WebUpdateReq{
	ManagerReq: ManagerReq{Agent: "universal", Scope: "project", WorkDir: baseDir},
	Target: "go-pro",
})
assert.NoErr(t, err)
assert.Len(t, result.Updated, 1)
assert.Eq(t, "go-pro", result.Updated[0].SkillID)
assert.Eq(t, "2.0.0", result.Updated[0].Version)
```

- [x] **Step 2: Run manager action tests to verify failure**

Run:

```bash
go test ./internal/app/webapp -run 'TestManager_(ApplyProfile|RunUpdate)|TestActionRuntimeRecord' -count=1
```

Expected: FAIL because manager action methods and result models do not exist.

- [x] **Step 3: Implement manager action service**

Create `internal/app/webapp/manager_actions.go`.

Add imports:

```go
import (
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/profileapp"
	"github.com/inhere/skillc/internal/app/updateapp"
	"github.com/inhere/skillc/internal/domain/profile"
)
```

Add types:

```go
type WebUpdateReq struct {
	ManagerReq
	Target string
}

type actionRuntimeRecord struct {
	SkillID       string `json:"skill_id"`
	SourceID      string `json:"source_id,omitempty"`
	Version       string `json:"version,omitempty"`
	Agent         string `json:"agent,omitempty"`
	Scope         string `json:"scope,omitempty"`
	InstalledPath string `json:"installed_path,omitempty"`
}

type actionErrorItem struct {
	SkillID string `json:"skill_id"`
	Reason  string `json:"reason"`
}

type actionSourceErrorItem struct {
	SourceID string `json:"source_id"`
	Reason   string `json:"reason"`
}

type profileApplyActionResult struct {
	Plan          profile.ApplyPlan     `json:"plan"`
	Installed     []actionRuntimeRecord `json:"installed"`
	InstallFailed []actionErrorItem     `json:"install_failed,omitempty"`
}

type updateRunActionResult struct {
	Updated       []actionRuntimeRecord `json:"updated"`
	Skipped       []actionErrorItem     `json:"skipped,omitempty"`
	Failed        []actionErrorItem     `json:"failed,omitempty"`
	SyncFailed    []actionSourceErrorItem `json:"sync_failed,omitempty"`
	CleanupFailed []actionErrorItem     `json:"cleanup_failed,omitempty"`
}
```

Add methods:

```go
func (m *Manager) ApplyProfile(name string, req ManagerReq) (profileApplyActionResult, error) {
	result, err := profileapp.NewService(m.configFile, m.baseDir).Apply(name, profileapp.ApplyReq{
		Agent: req.Agent, Scope: req.Scope, WorkDir: req.WorkDir,
	})
	out := profileApplyActionResult{
		Plan:          result.Plan,
		Installed:     runtimeRecords(result.Installed),
		InstallFailed: installErrors(result.InstallFailed),
	}
	if err != nil {
		return out, err
	}
	return out, nil
}

func (m *Manager) RunUpdate(req WebUpdateReq) (updateRunActionResult, error) {
	result, err := updateapp.NewService(m.configFile, m.baseDir).Run(updateapp.Req{
		Target: req.Target,
		Agent: req.Agent,
		Scope: req.Scope,
		WorkDir: req.WorkDir,
	})
	out := updateRunActionResult{
		Updated:       runtimeRecords(result.Updated),
		Skipped:       skippedErrors(result.Skipped),
		Failed:        failedErrors(result.Failed),
		SyncFailed:    sourceSyncErrors(result.SyncFailed),
		CleanupFailed: failedErrors(result.CleanupFailed),
	}
	if err != nil {
		return out, err
	}
	return out, nil
}
```

Add small converter helpers for `installapp.RuntimeRecord`, `installapp.InstallItemError`, `updateapp.SkippedItem`, `updateapp.FailedItem`, and `updateapp.SourceSyncError`. `SourceSyncError` must convert to `actionSourceErrorItem` with `source_id`, not to a skill-level `skill_id` error.

- [x] **Step 4: Run manager action tests**

Run:

```bash
go test ./internal/app/webapp -run 'TestManager_(ApplyProfile|RunUpdate)|TestActionRuntimeRecord' -count=1
```

Expected: PASS.

- [x] **Step 5: Commit Task 1**

```bash
git add internal/app/webapp/manager_actions.go internal/app/webapp/manager_actions_test.go
git commit -m "feat(web): add execution manager actions"
```

## Task 2: Add Web Execute HTTP Endpoints

**Files:**
- Modify: `internal/app/webapp/manager_server.go`
- Modify: `internal/app/webapp/manager_server_test.go`

- [x] **Step 1: Write failing HTTP execute tests**

Add tests:

- `TestManagerServerProfileApplyRequiresConfirmation`
- `TestManagerServerProfileApplyEndpointExecutesConfirmedApply`
- `TestManagerServerUpdateRunRequiresConfirmation`
- `TestManagerServerUpdateRunEndpointExecutesConfirmedUpdate`
- `TestManagerServerRejectsInvalidProfileActionPath`

Example shape:

```go
func TestManagerServerProfileApplyRequiresConfirmation(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeWebActionFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	req := httptest.NewRequest(http.MethodPost, "/api/profiles/go-dev/apply", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	assert.Eq(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), `"confirmation required"`)
}
```

Confirmed apply:

```go
req := httptest.NewRequest(http.MethodPost, "/api/profiles/go-dev/apply?agent=universal&scope=project", strings.NewReader(`{"confirm":true}`))
req.Header.Set("Content-Type", "application/json")
rec := httptest.NewRecorder()
server.Handler().ServeHTTP(rec, req)

assert.Eq(t, http.StatusOK, rec.Code)
assert.Contains(t, rec.Body.String(), `"installed"`)
assert.Contains(t, rec.Body.String(), `"skill_id":"review"`)
```

Confirmed update:

```go
req := httptest.NewRequest(http.MethodPost, "/api/update/run?agent=universal&scope=project", strings.NewReader(`{"confirm":true,"target":"go-pro"}`))
```

- [x] **Step 2: Run HTTP execute tests to verify failure**

Run:

```bash
go test ./internal/app/webapp -run 'TestManagerServer(ProfileApply|UpdateRun|RejectsInvalidProfileActionPath)' -count=1
```

Expected: FAIL because execute endpoints and confirm parsing do not exist.

- [x] **Step 3: Implement execute route parsing and confirm guard**

In `manager_server.go`:

- Add route:

```go
mux.HandleFunc("/api/update/run", s.handleUpdateRun)
```

- Replace `profilePlanName(path string)` with:

```go
type profileAction struct {
	Name   string
	Action string
}

func parseProfileAction(path string) (profileAction, bool) {
	rest := strings.TrimPrefix(path, "/api/profiles/")
	if rest == path || rest == "" {
		return profileAction{}, false
	}
	name, tail, ok := strings.Cut(rest, "/")
	if !ok || name == "" || strings.Contains(name, "/") {
		return profileAction{}, false
	}
	switch tail {
	case "plan", "apply":
		return profileAction{Name: name, Action: tail}, true
	default:
		return profileAction{}, false
	}
}
```

- Add request parser:

```go
type actionConfirmReq struct {
	Confirm bool   `json:"confirm"`
	Target  string `json:"target,omitempty"`
}

func readActionConfirmReq(r *http.Request) (actionConfirmReq, error) {
	defer r.Body.Close()
	if r.Body == http.NoBody {
		return actionConfirmReq{}, nil
	}
	var req actionConfirmReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return actionConfirmReq{}, fmt.Errorf("invalid json body")
	}
	return req, nil
}
```

- Add guard:

```go
func requireConfirm(w http.ResponseWriter, r *http.Request) (actionConfirmReq, bool) {
	req, err := readActionConfirmReq(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return actionConfirmReq{}, false
	}
	if !req.Confirm {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "confirmation required"})
		return actionConfirmReq{}, false
	}
	return req, true
}
```

- Extend `handleProfileAction`:

```go
action, ok := parseProfileAction(r.URL.Path)
if !ok { http.NotFound(w, r); return }

switch action.Action {
case "plan":
	if !allowMethod(w, r, http.MethodPost) { return }
	result, err := s.manager.PlanProfileApply(action.Name, managerReqFromQuery(r))
	writeResult(w, result, err)
case "apply":
	if !allowMethod(w, r, http.MethodPost) { return }
	if _, ok := requireConfirm(w, r); !ok { return }
	result, err := s.manager.ApplyProfile(action.Name, managerReqFromQuery(r))
	writeResult(w, result, err)
}
```

- Add `handleUpdateRun`:

```go
func (s *ManagerServer) handleUpdateRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) { return }
	body, ok := requireConfirm(w, r)
	if !ok { return }
	req := managerReqFromQuery(r)
	result, err := s.manager.RunUpdate(WebUpdateReq{ManagerReq: req, Target: body.Target})
	writeResult(w, result, err)
}
```

- [x] **Step 4: Run HTTP execute tests**

Run:

```bash
go test ./internal/app/webapp -run 'TestManagerServer(ProfileApply|UpdateRun|RejectsInvalidProfileActionPath)' -count=1
```

Expected: PASS.

- [x] **Step 5: Run full webapp tests**

Run:

```bash
go test ./internal/app/webapp -count=1
```

Expected: PASS.

- [x] **Step 6: Commit Task 2**

```bash
git add internal/app/webapp/manager_server.go internal/app/webapp/manager_server_test.go
git commit -m "feat(web): add execution API endpoints"
```

## Task 3: Add Static UI Execute Controls

**Files:**
- Modify: `internal/app/webapp/manager_static.go`
- Modify: `internal/app/webapp/manager_server_test.go`

- [x] **Step 1: Add failing static UI tests**

Add tests:

- `TestManagerServerIndexPageContainsExecutionControls`
- `TestManagerServerStaticPagePostsConfirmedActions`

Assertions:

```go
assert.Contains(t, body, "Apply profile")
assert.Contains(t, body, "Run update")
assert.Contains(t, body, "/api/profiles/")
assert.Contains(t, body, "/apply")
assert.Contains(t, body, "/api/update/run")
assert.Contains(t, body, "confirm")
```

Keep existing no external asset test passing:

```go
assert.NotContains(t, body, "https://")
assert.NotContains(t, body, "http://")
```

- [x] **Step 2: Run static UI execute tests to verify failure**

Run:

```bash
go test ./internal/app/webapp -run 'TestManagerServerIndexPageContainsExecutionControls|TestManagerServerStaticPagePostsConfirmedActions|TestManagerServerStaticPageDoesNotUseExternalAssets' -count=1
```

Expected: FAIL because execution controls do not exist.

- [x] **Step 3: Add action bar markup**

In `manager_static.go`, replace the Plan Output section with:

```html
<section class="section">
  <div class="section-head">
    <h3>Plan Output</h3>
    <div class="actions" id="action-bar">
      <button id="apply-profile-btn" disabled>Apply profile</button>
      <button id="run-update-btn" disabled>Run update</button>
    </div>
  </div>
  <pre id="plan-output" class="plan">No plan requested.</pre>
</section>
```

Add CSS:

```css
.actions { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
button:disabled { cursor: not-allowed; opacity: .45; }
button.danger { border-color: #c77b72; color: #a93535; }
button.danger:hover { background: var(--bad-soft); }
```

- [x] **Step 4: Add pending action state and execute JS**

In the existing JS state:

```js
var state = {
  summary: null,
  sources: [],
  profiles: [],
  status: { items: [], summary: {} },
  installs: [],
  drift: [],
  skills: [],
  pendingAction: null
};
```

Add helpers:

```js
function setPendingAction(action) {
  state.pendingAction = action;
  byId('apply-profile-btn').disabled = !(action && action.type === 'profile');
  byId('run-update-btn').disabled = !(action && action.type === 'update');
}

function postJSON(path, payload) {
  return api(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload || {})
  });
}
```

Update `planProfile(name)`:

```js
function planProfile(name) {
  clearError();
  postJSON('/api/profiles/' + encodeURIComponent(name) + '/plan', {})
    .then(function (plan) {
      setPendingAction({ type: 'profile', name: name });
      byId('plan-output').textContent = JSON.stringify(plan, null, 2);
    })
    .catch(showError);
}
```

Update `planUpdate()`:

```js
function planUpdate() {
  clearError();
  postJSON('/api/update/plan', {})
    .then(function (plan) {
      setPendingAction({ type: 'update' });
      byId('plan-output').textContent = JSON.stringify(plan, null, 2);
    })
    .catch(showError);
}
```

Add execute handlers:

```js
function applyProfile() {
  if (!state.pendingAction || state.pendingAction.type !== 'profile') return;
  if (!window.confirm('Apply this profile to the current project?')) return;
  postJSON('/api/profiles/' + encodeURIComponent(state.pendingAction.name) + '/apply', { confirm: true })
    .then(function (result) {
      byId('plan-output').textContent = JSON.stringify(result, null, 2);
      setPendingAction(null);
      loadAll();
    })
    .catch(showError);
}

function runUpdate() {
  if (!state.pendingAction || state.pendingAction.type !== 'update') return;
  if (!window.confirm('Run update for the current project?')) return;
  postJSON('/api/update/run', { confirm: true })
    .then(function (result) {
      byId('plan-output').textContent = JSON.stringify(result, null, 2);
      setPendingAction(null);
      loadAll();
    })
    .catch(showError);
}
```

Wire buttons:

```js
byId('apply-profile-btn').addEventListener('click', applyProfile);
byId('run-update-btn').addEventListener('click', runUpdate);
setPendingAction(null);
```

- [x] **Step 5: Run static UI execute tests**

Run:

```bash
go test ./internal/app/webapp -run 'TestManagerServerIndexPageContainsExecutionControls|TestManagerServerStaticPagePostsConfirmedActions|TestManagerServerStaticPageDoesNotUseExternalAssets' -count=1
```

Expected: PASS.

- [x] **Step 6: Commit Task 3**

```bash
git add internal/app/webapp/manager_static.go internal/app/webapp/manager_server_test.go
git commit -m "feat(web): add execution controls"
```

## Task 4: Documentation and Plan Links

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/TODO.md`
- Modify: `docs/design/skillc-v0-enhance-design.md`
- Modify: `docs/superpowers/plans/2026-06-15-skillc-v0-phase5-web-execution.md`

- [ ] **Step 1: Update English README**

Update the `web` section to say:

```markdown
The web manager runs on `127.0.0.1` by default and shows sources, profiles, current status, project install map, and version drift. Web writes are still guarded: profile apply and update require a plan-first flow and an explicit confirmation, and only operate on the current project/agent/scope.
```

- [ ] **Step 2: Update Chinese README**

Update the `web` section to say:

```markdown
Web 管理界面默认监听 `127.0.0.1`，用于查看 sources、profiles、当前项目状态、项目安装分布和版本差异。Web 写操作仍有保护：profile apply 和 update 必须先生成计划，再显式确认执行，并且第一轮只操作当前项目、agent 和 scope。
```

- [ ] **Step 3: Update TODO and design docs**

In `docs/TODO.md`:

- Keep the broad Web item unchecked because uninstall/delete/source management are still not done.
- Add Phase 5 plan link and state.

Example:

```markdown
  - 五期计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase5-web-execution.md`
  - 五期目标：在 Web 中补齐当前项目 profile apply / update 的计划、确认、执行闭环；卸载、source 删除和跨项目批量更新继续后置。
```

In `docs/design/skillc-v0-enhance-design.md`:

- Add revision row.
- Add Phase 5 plan link near earlier phase links.
- Replace "下一步建议" with Phase 5 Web execution MVP.

- [ ] **Step 4: Commit docs**

```bash
git add README.md README.zh-CN.md docs/TODO.md docs/design/skillc-v0-enhance-design.md docs/superpowers/plans/2026-06-15-skillc-v0-phase5-web-execution.md
git commit -m "docs: add skillc phase 5 web execution plan"
```

## Task 5: Full Verification and Review

**Files:**
- Modify: `docs/superpowers/plans/2026-06-15-skillc-v0-phase5-web-execution.md`

- [ ] **Step 1: Run focused tests**

Run:

```bash
go test ./internal/app/webapp -count=1
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
rg -n -- "Phase 5|五期|skillc web|Apply profile|Run update|confirmation|确认|Web 执行" README.md README.zh-CN.md docs/TODO.md docs/design/skillc-v0-enhance-design.md docs/superpowers/plans/2026-06-15-skillc-v0-phase5-web-execution.md
```

Expected: docs mention Phase 5 and do not claim uninstall/source deletion/cross-project execution support.

- [ ] **Step 4: Manual smoke test**

Build and start local server:

```bash
go build -o ./tmp/skillc-web-smoke.exe ./cmd/skillc
./tmp/skillc-web-smoke.exe web --port 18080
```

Open or request:

```text
http://127.0.0.1:18080
```

Expected:

- Page loads without external assets.
- Page contains `Apply profile` and `Run update` controls.
- `/api/summary` returns JSON.
- `/api/status` returns JSON with `items` and `summary`.
- `POST /api/update/run` without `{"confirm":true}` returns `400`.
- Server process is stopped before final commit.

- [ ] **Step 5: Update final verification note**

Append:

```markdown
**Verification note (2026-06-15):** `go test ./internal/app/webapp -count=1`, `go test ./...`, docs sanity check, and local smoke test pass. Manual smoke confirms execute endpoints reject missing confirmation and the static UI exposes plan-first execution controls.
```

- [ ] **Step 6: Commit final checkbox updates**

```bash
git add docs/superpowers/plans/2026-06-15-skillc-v0-phase5-web-execution.md
git commit -m "docs: complete skillc phase 5 web execution plan review"
```

## Self-Review Checklist

- [ ] Web execution only runs after an explicit confirmation flag reaches the server.
- [ ] Profile apply execution delegates to `profileapp.Apply`.
- [ ] Update execution delegates to `updateapp.Run`.
- [ ] HTTP handlers do not duplicate install/update/profile business rules.
- [ ] Execute result JSON uses stable lowercase field names.
- [ ] UI remains self-contained and uses no external CDN/assets.
- [ ] UI refreshes summary/status/install-map/version-drift after execution.
- [ ] Web still defaults to `127.0.0.1`.
- [ ] `show --web` skill viewer remains compatible.
- [ ] Tests cover manager service, HTTP confirm guards, execute endpoints and static page controls.
- [ ] `go test ./...` passes before marking complete.

## Acceptance Criteria

- Web Profiles page can generate a profile apply plan and then execute it after confirmation.
- Web Dashboard / Version Drift can generate an update plan and then execute update after confirmation.
- Execute endpoints reject missing or false confirmation.
- Execute endpoints reject invalid methods and invalid profile action paths.
- Execute result payloads show installed/updated/skipped/failed items with stable JSON fields, and source sync failures use `source_id`.
- Dashboard/status data refreshes after execution.
- No Web endpoint in this phase performs uninstall, source deletion or cross-project batch update.
- README, README.zh-CN, TODO and design docs reference Phase 5.

## Risks and Guardrails

- Web write operations can mutate local files. Keep the server-side `confirm:true` guard even if the UI has a browser confirmation.
- Update execution may sync sources and reinstall files. Scope it to the current workdir/agent/scope passed through `ManagerReq`.
- Do not invent new install/update semantics in Web handlers. Use existing app services and convert results only at the boundary.
- Do not add a frontend build chain for this phase.
- Do not broaden to cross-project update until project registry / project selection / per-project confirmation are designed.
- Keep Web output honest: if execution partially fails, return the partial result and error message instead of hiding failures.
