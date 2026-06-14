# Skillc v0 Phase 3 Interactive Selection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 基于已引入的 `gookit/cliui`，为 `install`、`update`、`profile create` 增加可过滤、多选、可测试的交互式选择入口。

**Architecture:** CLI 命令保持薄封装，业务仍复用现有 `searchapp`、`installapp`、`statusapp`、`updateapp`、`profileapp`。新增 `internal/infra/termselect` 作为 `github.com/gookit/cliui/interact` / `interact/ui` / `interact/backend` 的 adapter，并在 CLI 层通过 `newMultiSelector` 注入，单测使用 fake selector 或 `interact.NewUIFakeBackend`。

**Tech Stack:** Go, `gookit/gcli`, `gookit/cliui v0.3.1` (`interact`, `interact/ui`, `interact/backend`), existing app/domain/infra packages, Go unit tests with `github.com/gookit/goutil/testutil/assert`.

---

| 修订时间 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-14 | v0.1 | Codex | 基于 v0 增强设计文档和 Phase 2 完成状态输出 Phase 3 交互式选择实施计划 |
| 2026-06-14 | v0.2 | Codex | 采纳 `gookit/cliui` 已引入的事实，改为 cliui adapter 方案，移除自研 selector/parser 设计 |

相关文档：

- 设计文档：`docs/design/skillc-v0-enhance-design.md`
- Phase 1 计划：`docs/superpowers/plans/2026-06-13-skillc-v0-phase1-profile.md`
- Phase 2 计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md`
- 任务入口：`docs/TODO.md`

## Phase 3 Scope

本期做：

- 复用已引入的 `gookit/cliui/interact`、`interact/ui`、`interact/backend`，封装一个很薄的终端多选 adapter。
- `skillc install --interactive [keyword]`：从 index 搜索出候选项，在 TUI 中继续过滤并多选，最终复用现有 install 执行。
- `skillc update --interactive [skill]` / `--target`：从当前项目的 updateable 状态中多选，再逐项调用现有 update 执行。
- `skillc profile create <name> --interactive`：从 index 多选 skill 并保存为 profile targets。
- 扩展 index keyword 匹配字段，让交互入口能按 ID、qualified name、source-qualified name、source、collection 搜索。
- 保持非交互命令完整行为不变，所有写操作仍有摘要和确认，`--yes` 只跳过确认。

本期不做：

- 不做 Web 管理。
- 不做跨项目 install map。
- 不做批量更新所有下游项目。
- 不引入新的 TUI 依赖。
- 不重写 `installapp` / `updateapp` / `profileapp` 主链路。
- 不改变 `update --check` 当前输出语义。
- 不做 checksum / git commit drift 精确判断。

## User-Facing Behavior

### Install

```bash
skillc install --interactive
skillc install --interactive go
skillc install --interactive --agent codex --scope project
```

行为：

1. 加载 index。
2. 如果命令提供 `skill` 参数，将它作为初始搜索关键字，而不是直接安装 target。
3. 候选项展示 target、version、source、collection。
4. 用户在 `cliui` TUI 中输入关键字过滤，使用方向键移动、空格切换多选、回车确认。
5. 选择后打印安装计划摘要。
6. 未传 `--yes` 时继续使用确认提示。
7. 执行仍调用 `installapp.RunResolved`。

### Update

```bash
skillc update --interactive
skillc update --interactive --target go-pro
skillc update --interactive --agent claude-code --scope project
```

行为：

1. 调用 `statusapp`，`Sync: true`。
2. 候选只包含可通过 `update` 修复或更新的状态：
   - `outdated`
   - `missing`
3. 排除：
   - `installed`
   - `orphan`
   - `unmanaged`
   - `source-error`
4. 若提供 `--target` 或 positional `skill`，先按 target 过滤候选，再进入多选。
5. 对每个选中项调用现有 `updateapp.Run`，target 优先使用 `SourceQualifiedName`，其次 `QualifiedName`，最后 `SkillID`。
6. 汇总输出 updated/skipped/failed/cleanup failed。

### Profile Create

```bash
skillc profile create go-dev --interactive
skillc profile create go-dev --interactive --agent codex --scope project
```

行为：

1. 从 index 中列出候选 skills。
2. 用户过滤并多选。
3. 保存为 profile targets，target 使用 `{Source: item.SourceID, Skill: item.ID}`。
4. `--interactive` 与 `--from-installed`、`--from-collection` 互斥。
5. 若传入 `--agent` / `--scope`，写入 profile default agent/scope，与后续 `profile apply` 行为一致。

## File Structure

新增文件：

- `internal/infra/termselect/selector.go`
  - 定义 `Item`、`Options`、`Selector`、`CliUISelector`。
  - 只封装 `gookit/cliui` 组件，不自研过滤、方向键、多选、解析器。

- `internal/infra/termselect/selector_test.go`
  - 使用 `interact.NewUIFakeBackend` 和 `interact/backend.Event` 覆盖过滤、多选、空候选。

- `internal/cli/interactive_cmd.go`
  - CLI 层选择器注入点和 skill/status item 到 termselect item 的转换。
  - 只放 CLI 适配，不放业务规则。

修改文件：

- `internal/infra/repoindex/search.go`
  - 扩展 keyword 匹配字段。

- `internal/infra/repoindex/search_test.go`
  - 覆盖 ID、qualified name、source-qualified name、source id/name、collection 匹配。

- `internal/app/statusapp/service.go`
  - `Item` 增加 `SourceQualifiedName`，供交互式 update 生成精确 target。

- `internal/app/statusapp/service_test.go`
  - 补充 `SourceQualifiedName` 输出测试。

- `internal/cli/manage_cmd.go`
  - `install` 增加 `--interactive`。
  - `update` 增加 `--interactive`。
  - 交互路径复用现有服务和输出函数。

- `internal/cli/profile_cmd.go`
  - `profile create` 增加 `--interactive`。
  - 与 `--from-installed` / `--from-collection` 做互斥校验。

- `internal/cli/app_test.go`
  - 覆盖 install/update/profile create 三条交互路径。

- `README.md`
  - 增加 interactive 命令示例。

- `README.zh-CN.md`
  - 增加 interactive 命令示例。

- `docs/design/skillc-v0-enhance-design.md`
  - 标注 Phase 3 使用 `gookit/cliui`。

- `docs/TODO.md`
  - 增加 Phase 3 计划链接。

- `docs/superpowers/plans/2026-06-14-skillc-v0-phase3-interactive-selection.md`
  - 实施中持续更新 checkbox 和验证记录。

## Task 1: Add `gookit/cliui` Terminal Selector Adapter

**Files:**
- Create: `internal/infra/termselect/selector.go`
- Create: `internal/infra/termselect/selector_test.go`

- [x] **Step 1: Write failing adapter tests**

Create `internal/infra/termselect/selector_test.go`:

```go
package termselect

import (
	"context"
	"testing"

	"github.com/gookit/cliui/interact"
	"github.com/gookit/cliui/interact/backend"
	"github.com/gookit/goutil/testutil/assert"
)

func TestCliUISelectorSelectMultiWithFilter(t *testing.T) {
	be := interact.NewUIFakeBackend(
		backend.Event{Type: backend.EventKey, Text: "go"},
		backend.Event{Type: backend.EventKey, Key: backend.KeySpace},
		backend.Event{Type: backend.EventKey, Key: backend.KeyCtrlU},
		backend.Event{Type: backend.EventKey, Key: backend.KeyEnter},
	)
	selector := NewCliUISelectorWithBackend(be)

	got, err := selector.SelectMulti(context.Background(), Options{
		Title: "Choose skills",
		Items: []Item{
			{Key: "1", Label: "Go Pro repo-a/tools v1.0.0", Value: "repo-a/tools/go-pro"},
			{Key: "2", Label: "Review repo-a/tools v1.0.0", Value: "repo-a/tools/review"},
		},
	})

	assert.NoErr(t, err)
	assert.Len(t, got, 1)
	assert.Eq(t, "repo-a/tools/go-pro", got[0].Value)
}
```

Also add `TestCliUISelectorSelectMultiWithTypedKeys` and `TestCliUISelectorReturnsEmptyForNoItems`.

- [x] **Step 2: Run adapter tests to verify failure**

Run:

```bash
go test ./internal/infra/termselect -count=1
```

Expected: FAIL because `internal/infra/termselect` does not exist.

- [x] **Step 3: Implement minimal cliui adapter**

Create `internal/infra/termselect/selector.go`:

```go
package termselect

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gookit/cliui/interact"
	"github.com/gookit/cliui/interact/ui"
)

type Item struct {
	Key    string
	Label  string
	Value  string
	Detail string
}

type Options struct {
	Title        string
	Items        []Item
	PageSize     int
	FilterPrompt string
}

type Selector interface {
	SelectMulti(ctx context.Context, opts Options) ([]Item, error)
}

type CliUISelector struct {
	backend interact.UIBackend
}

func NewCliUISelector() *CliUISelector {
	return NewCliUISelectorWithBackend(interact.NewUIReadlineBackend())
}

func NewCliUISelectorWithBackend(be interact.UIBackend) *CliUISelector {
	if be == nil {
		be = interact.NewUIReadlineBackend()
	}
	return &CliUISelector{backend: be}
}

func (s *CliUISelector) SelectMulti(ctx context.Context, opts Options) ([]Item, error) {
	if len(opts.Items) == 0 {
		return nil, nil
	}

	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = "Select items"
	}

	uiItems := make([]ui.Item, 0, len(opts.Items))
	for idx, item := range opts.Items {
		item = normalizeItem(idx, item)
		uiItems = append(uiItems, ui.Item{
			Key:   item.Key,
			Label: itemLabel(item),
			Value: item,
		})
	}

	component := interact.NewUIMultiSelect(title, uiItems)
	component.Filterable = true
	component.FilterPrompt = opts.FilterPrompt
	component.PageSize = opts.PageSize
	if component.PageSize <= 0 {
		component.PageSize = 12
	}

	result, err := component.Run(ctx, s.backend)
	if err != nil {
		return nil, err
	}

	selected := make([]Item, 0, len(result.Values))
	for _, value := range result.Values {
		item, ok := value.(Item)
		if !ok {
			return nil, fmt.Errorf("unexpected selected item type %T", value)
		}
		selected = append(selected, item)
	}
	return selected, nil
}

func normalizeItem(idx int, item Item) Item {
	if strings.TrimSpace(item.Key) == "" {
		item.Key = strconv.Itoa(idx + 1)
	}
	if strings.TrimSpace(item.Label) == "" {
		item.Label = item.Value
	}
	return item
}

func itemLabel(item Item) string {
	if item.Detail == "" {
		return item.Label
	}
	return item.Label + " " + item.Detail
}
```

This adapter must return an empty slice without opening the TUI when `opts.Items` is empty, because `cliui` correctly treats empty item lists as invalid component state.

Mapping rules:

- `termselect.Item.Key` defaults to 1-based index if empty.
- `termselect.Item.Label` is shown in TUI.
- `termselect.Item.Value` stores the stable business target, usually source-qualified target.
- `ui.Item.Value` stores the original `termselect.Item` so selected results can be mapped back without parsing UI text.

- [x] **Step 4: Run adapter tests**

Run:

```bash
go test ./internal/infra/termselect -count=1
```

Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/infra/termselect
git commit -m "feat(termselect): add cliui multi selector adapter"
```

**Verification note (2026-06-15):** `go test ./internal/infra/termselect -count=1` and `go test ./...` pass. The adapter delegates filtering and multi-select to `gookit/cliui`; empty item lists return an empty result before opening the TUI. Submitted key text is covered as a cliui backend path, not as the primary TUI user flow.

## Task 2: Expand Index Keyword Matching for Interactive Search

**Files:**
- Modify: `internal/infra/repoindex/search.go`
- Modify: `internal/infra/repoindex/search_test.go`

- [x] **Step 1: Write failing keyword match tests**

Add a test to `internal/infra/repoindex/search_test.go`:

```go
func TestFilter_MatchesIdentitySourceAndCollectionFields(t *testing.T) {
	items := []skill.Skill{{
		ID:                  "go-pro",
		Name:                "Go Pro",
		Collection:          "tools",
		QualifiedName:       "tools/go-pro",
		SourceQualifiedName: "repo-a/tools/go-pro",
		SourceID:            "source-a",
		SourceName:          "workflow-repo",
	}}

	tests := []struct {
		name    string
		keyword string
	}{
		{name: "id", keyword: "go-pro"},
		{name: "collection", keyword: "tools"},
		{name: "qualified name", keyword: "tools/go-pro"},
		{name: "source qualified name", keyword: "repo-a/tools"},
		{name: "source id", keyword: "source-a"},
		{name: "source name", keyword: "workflow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Filter(items, Query{Keyword: tt.keyword})
			assert.Len(t, got, 1)
			assert.Eq(t, "go-pro", got[0].ID)
		})
	}
}
```

- [x] **Step 2: Run repoindex tests to verify failure**

Run:

```bash
go test ./internal/infra/repoindex -count=1
```

Expected: FAIL for at least source/qualified-field matching.

- [x] **Step 3: Implement `matchesKeyword` helper**

In `internal/infra/repoindex/search.go`, replace the current name/description-only condition with:

```go
if query.Keyword != "" && !matchesKeyword(item, query.Keyword) {
	continue
}
```

Add:

```go
func matchesKeyword(item skill.Skill, keyword string) bool {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return true
	}
	fields := []string{
		item.ID,
		item.Name,
		item.Description,
		item.Collection,
		item.QualifiedName,
		item.SourceQualifiedName,
		item.SourceID,
		item.SourceName,
		string(item.SourceType),
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), keyword) {
			return true
		}
	}
	return false
}
```

- [x] **Step 4: Run repoindex tests**

Run:

```bash
go test ./internal/infra/repoindex -count=1
```

Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/infra/repoindex/search.go internal/infra/repoindex/search_test.go
git commit -m "feat(search): match skill identity fields"
```

**Verification note (2026-06-15):** Added RED coverage for ID, collection, qualified name, source-qualified name, source ID, source name, and source type keyword matching. `go test ./internal/infra/repoindex -count=1` and `go test ./...` pass.

## Task 3: Add CLI Interactive Helper Boundary

**Files:**
- Create: `internal/cli/interactive_cmd.go`
- Modify: `internal/cli/app_test.go`
- Modify: `internal/app/statusapp/service.go`

- [x] **Step 1: Write failing helper tests**

Add tests near other CLI helper tests in `internal/cli/app_test.go`:

```go
func TestSkillSelectItemsUseStableSourceQualifiedTargets(t *testing.T) {
	items := skillSelectItems([]skill.Skill{{
		ID:                  "go-pro",
		Name:                "Go Pro",
		Version:             "1.2.3",
		SourceID:            "repo-a",
		Collection:          "tools",
		QualifiedName:       "tools/go-pro",
		SourceQualifiedName: "repo-a/tools/go-pro",
	}})

	assert.Len(t, items, 1)
	assert.Eq(t, "1", items[0].Key)
	assert.Eq(t, "repo-a/tools/go-pro", items[0].Value)
	assert.Contains(t, items[0].Label, "Go Pro")
	assert.Contains(t, items[0].Label, "1.2.3")
}
```

Also add:

- `TestSelectedSkillsMapsSelectedTargetsBackToSkills`
- `TestUpdateSelectItemsOnlyIncludesUpdateableStatuses`
- `TestUpdateTargetPrefersSourceQualifiedName`

- [x] **Step 2: Run CLI tests to verify failure**

Run:

```bash
go test ./internal/cli -run 'TestSkillSelectItems|TestSelectedSkills|TestUpdateSelectItems|TestUpdateTarget' -count=1
```

Expected: FAIL because helpers do not exist.

- [x] **Step 3: Implement helper boundary**

Create `internal/cli/interactive_cmd.go`:

```go
package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/inhere/skillc/internal/app/statusapp"
	"github.com/inhere/skillc/internal/domain/skill"
	"github.com/inhere/skillc/internal/infra/termselect"
)

type multiSelector interface {
	SelectMulti(ctx context.Context, opts termselect.Options) ([]termselect.Item, error)
}

var newMultiSelector = func() multiSelector {
	return termselect.NewCliUISelector()
}
```

Add helper functions:

- `skillSelectItems(skills []skill.Skill) []termselect.Item`
- `selectedSkills(skills []skill.Skill, selected []termselect.Item) []skill.Skill`
- `updateSelectItems(items []statusapp.Item) []termselect.Item`
- `selectedUpdateTargets(items []statusapp.Item, selected []termselect.Item) []string`
- `skillTarget(item skill.Skill) string`
- `statusTarget(item statusapp.Item) string`

Rules:

- Key is a 1-based numeric string for easy typed selection.
- Value is the stable target.
- Label includes user-facing name/id, source, collection, version/status.
- Only `outdated` and `missing` status become update candidates.
- `statusapp.Item` carries `SourceQualifiedName` through from `listapp.Item`, so update selection can prefer source-qualified targets.

- [x] **Step 4: Run helper tests**

Run:

```bash
go test ./internal/cli -run 'TestSkillSelectItems|TestSelectedSkills|TestUpdateSelectItems|TestUpdateTarget' -count=1
```

Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/cli/interactive_cmd.go internal/cli/app_test.go
git commit -m "feat(cli): add interactive selection helpers"
```

**Verification note (2026-06-15):** Added RED coverage for skill select item targets, selected skill mapping, update candidate filtering, and update target precedence. Initial targeted CLI test failed on missing helpers and missing `statusapp.Item.SourceQualifiedName`; after implementation, `go test ./internal/cli -run 'TestSkillSelectItems|TestSelectedSkills|TestUpdateSelectItems|TestUpdateTarget' -count=1`, `go test ./internal/app/statusapp ./internal/cli -count=1`, and `go test ./...` pass.

**Spec review fix note (2026-06-15):** Fixed Task 3 helper boundary issues without implementing install/update/profile interactive commands: select labels keep title text separate from detail metadata to avoid duplicate rendered text; skill/status targets preserve the documented precedence `SourceQualifiedName -> QualifiedName -> SkillID` without synthesizing source-qualified names from `SourceID`; and missing status items now fill `QualifiedName`, `SourceQualifiedName`, and `LatestVersion` from the index before returning `missing`. `go test ./internal/cli -run 'TestSkillSelectItems|TestSkillTarget|TestSelectedSkills|TestUpdateSelectItems|TestStatusTarget|TestUpdateTarget' -count=1`, `go test ./internal/app/statusapp ./internal/cli -count=1`, and `go test ./...` pass.

## Task 4: Add `install --interactive`

**Files:**
- Modify: `internal/cli/manage_cmd.go`
- Modify: `internal/cli/app_test.go`

- [ ] **Step 1: Write failing install interactive CLI test**

Add a fake selector in `internal/cli/app_test.go`:

```go
type selectorStub struct {
	items []termselect.Item
	got   termselect.Options
}

func (s *selectorStub) SelectMulti(ctx context.Context, opts termselect.Options) ([]termselect.Item, error) {
	s.got = opts
	return s.items, nil
}
```

Add test:

```go
func TestInstallCommandInteractiveSelectsAndInstallsSkills(t *testing.T) {
	baseDir := t.TempDir()
	// create config, index, and a source skill directory like existing install tests

	stub := &selectorStub{items: []termselect.Item{{Value: "repo-a/tools/go-pro"}}}
	prevSelector := newMultiSelector
	newMultiSelector = func() multiSelector { return stub }
	defer func() { newMultiSelector = prevSelector }()

	output := runAppInDirWithStdout(t, baseDir, []string{"install", "--interactive", "--yes", "--agent", "universal"})

	assert.Contains(t, output, "Will install skills: go-pro")
	assert.Contains(t, output, "installed go-pro")
	assert.Contains(t, stub.got.Title, "Install")
}
```

Also add `TestInstallCommandInteractiveUsesSkillArgAsSearchKeyword`, verifying `install --interactive go` passes only matching candidates to selector and does not treat `go` as direct target.

- [ ] **Step 2: Run install interactive tests to verify failure**

Run:

```bash
go test ./internal/cli -run 'TestInstallCommandInteractive' -count=1
```

Expected: FAIL because `--interactive` does not exist.

- [ ] **Step 3: Implement install interactive flag and branch**

In `buildInstallCommand`:

- add `var interactive bool`
- register `c.BoolOpt(&interactive, "interactive", "i", false, "interactively select skills to install")`
- reset any command-level state if needed, matching existing patterns
- after config/source/mode validation, branch before the current restore/direct-target logic:

```go
if interactive {
	keyword := c.Arg("skill").String()
	skills, err := newSearchService().Search(keyword, opts.Agent, "")
	if err != nil {
		return err
	}
	selected, err := newMultiSelector().SelectMulti(context.Background(), termselect.Options{
		Title: "Install skills",
		Items: skillSelectItems(skills),
	})
	if err != nil {
		return err
	}
	resolved := selectedSkills(skills, selected)
	// print the same install plan summary and call RunResolved
}
```

Reuse the existing install mode, fallback notifier, confirmation prompt, and result printing. If no candidates or no selected skills, print a warning and return nil without modifying the lock.

- [ ] **Step 4: Run install interactive tests**

Run:

```bash
go test ./internal/cli -run 'TestInstallCommandInteractive|TestInstallCommand_' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/manage_cmd.go internal/cli/app_test.go
git commit -m "feat(cli): add interactive install"
```

## Task 5: Add `update --interactive`

**Files:**
- Modify: `internal/app/statusapp/service.go`
- Modify: `internal/app/statusapp/service_test.go`
- Modify: `internal/cli/manage_cmd.go`
- Modify: `internal/cli/app_test.go`

- [ ] **Step 1: Write failing status `SourceQualifiedName` test**

Add or extend status tests so an indexed skill with `SourceQualifiedName: "repo-a/tools/go-pro"` produces `statusapp.Item.SourceQualifiedName == "repo-a/tools/go-pro"` for installed/outdated/missing classifications.

- [ ] **Step 2: Run status tests to verify failure**

Run:

```bash
go test ./internal/app/statusapp -run SourceQualifiedName -count=1
```

Expected: FAIL because `statusapp.Item` does not expose `SourceQualifiedName`.

- [ ] **Step 3: Implement status target propagation**

In `statusapp.Item`, add:

```go
SourceQualifiedName string
```

In `classifyListItem`, copy `current.SourceQualifiedName` initially, and after `findLatest`, fill it from `latest.SourceQualifiedName` when missing.

- [ ] **Step 4: Write failing update interactive CLI test**

Add test with:

- config + lock record for `go-pro`
- index version newer than lock version
- fake selector selecting `repo-a/tools/go-pro`
- stub `newUpdateService` records called targets

Expected assertions:

- selector title includes `Update`
- update runner called once
- update req target equals `repo-a/tools/go-pro`
- output includes runner result lines

- [ ] **Step 5: Run update interactive tests to verify failure**

Run:

```bash
go test ./internal/cli -run 'TestUpdateCommandInteractive' -count=1
```

Expected: FAIL because `--interactive` update path does not exist.

- [ ] **Step 6: Implement update interactive flag and branch**

In `buildUpdateCommand`:

- add `var interactive bool`
- register `c.BoolOpt(&interactive, "interactive", "i", false, "interactively select update candidates")`
- preserve `--check` behavior; if both `--check` and `--interactive` are set, return a clear error such as `--check and --interactive are mutually exclusive`
- resolve `target` from `--target` or positional `skill` before branching
- interactive branch:
  - call `statusapp.NewService(...).Run(statusapp.Req{Sync: true, Agent: opts.Agent, Scope: opts.Scope, WorkDir: cwd})`
  - filter candidates through helper
  - apply target filter when target is set
  - run selector
  - call `newUpdateService(...).Run(updateapp.Req{Target: selectedTarget, Agent: opts.Agent, Scope: opts.Scope, WorkDir: cwd})` for each selected target
  - reuse existing update result printing

- [ ] **Step 7: Run update/status tests**

Run:

```bash
go test ./internal/app/statusapp ./internal/cli -run 'Test.*Interactive|TestService_.*SourceQualifiedName|TestUpdateCommand_' -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/app/statusapp/service.go internal/app/statusapp/service_test.go internal/cli/manage_cmd.go internal/cli/app_test.go
git commit -m "feat(cli): add interactive update"
```

## Task 6: Add `profile create --interactive`

**Files:**
- Modify: `internal/cli/profile_cmd.go`
- Modify: `internal/cli/app_test.go`

- [ ] **Step 1: Write failing profile create interactive tests**

Add:

- `TestProfileCreateInteractiveSelectsSkills`
- `TestProfileCreateInteractiveIsMutuallyExclusiveWithFromInstalled`
- `TestProfileCreateInteractiveIsMutuallyExclusiveWithFromCollection`

Expected behavior:

- selected skill becomes `profile.Target{Source: selected.SourceID, Skill: selected.ID}`
- command prints `profile created: <name>`
- no profile is written when mutually exclusive flags are combined

- [ ] **Step 2: Run profile interactive tests to verify failure**

Run:

```bash
go test ./internal/cli -run 'TestProfileCreateInteractive' -count=1
```

Expected: FAIL because `--interactive` is not supported.

- [ ] **Step 3: Implement profile create interactive branch**

In `buildProfileCreateCommand`:

- add `var interactive bool`
- register `c.BoolOpt(&interactive, "interactive", "i", false, "interactively select skills")`
- update deferred reset and `fillProfileCreateOptions` to include `--interactive`
- include `interactive` in the source mode count:

```go
sourceCount := 0
if interactive { sourceCount++ }
if fromInstalled { sourceCount++ }
if fromCollection != "" { sourceCount++ }
if sourceCount != 1 {
	return fmt.Errorf("use exactly one of --interactive, --from-installed or --from-collection")
}
```

Interactive branch:

```go
skills, err := newSearchService().Search("", agentName, "")
selected, err := newMultiSelector().SelectMulti(context.Background(), termselect.Options{
	Title: "Create profile " + name,
	Items: skillSelectItems(skills),
})
selectedSkills := selectedSkills(skills, selected)
targets := make([]profile.Target, 0, len(selectedSkills))
for _, item := range selectedSkills {
	targets = append(targets, profile.Target{Source: item.SourceID, Skill: item.ID})
}
_, err = svc.Create(name, profile.Profile{
	DefaultAgent: agentName,
	DefaultScope: scope,
	Targets: targets,
})
```

If no candidates or no selected skills, return a clear error and do not create an empty profile.

- [ ] **Step 4: Run profile interactive tests**

Run:

```bash
go test ./internal/cli -run 'TestProfileCreateInteractive|TestProfileCreateCommandRequiresExactlyOneSource' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/profile_cmd.go internal/cli/app_test.go
git commit -m "feat(cli): add interactive profile create"
```

## Task 7: Documentation and Full Verification

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/design/skillc-v0-enhance-design.md`
- Modify: `docs/TODO.md`
- Modify: `docs/superpowers/plans/2026-06-14-skillc-v0-phase3-interactive-selection.md`

- [ ] **Step 1: Update README command examples**

Add examples:

```bash
skillc install --interactive [keyword]
skillc update --interactive
skillc profile create go-dev --interactive
```

Mention that interactive selection is based on `gookit/cliui` and supports type-to-filter + multi-select.

- [ ] **Step 2: Update Chinese README**

Add equivalent Chinese examples and one sentence explaining filter/multi-select behavior.

- [ ] **Step 3: Update design and TODO docs**

Ensure these docs link to this plan:

- `docs/design/skillc-v0-enhance-design.md`
- `docs/TODO.md`

Do not mark the TODO interactive item complete until implementation is actually done.

- [ ] **Step 4: Run focused tests**

Run:

```bash
go test ./internal/infra/termselect ./internal/infra/repoindex ./internal/app/statusapp ./internal/cli -count=1
```

Expected: PASS.

- [ ] **Step 5: Run full test suite**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 6: Run doc/search sanity checks**

Run:

```bash
rg -n -- "interactive|gookit/cliui|Phase 3|三期计划|termselect" README.md README.zh-CN.md docs/TODO.md docs/design/skillc-v0-enhance-design.md docs/superpowers/plans/2026-06-14-skillc-v0-phase3-interactive-selection.md
```

Expected: docs contain Phase 3 references and no stale self-built parser wording.

- [ ] **Step 7: Update plan verification note**

At the bottom of this file, add:

```markdown
**Verification note (YYYY-MM-DD):** `go test ./internal/infra/termselect ./internal/infra/repoindex ./internal/app/statusapp ./internal/cli -count=1` and `go test ./...` pass. README, README.zh-CN, TODO, and design docs reference Phase 3 interactive selection behavior.
```

- [ ] **Step 8: Commit docs and final checkbox updates**

```bash
git add README.md README.zh-CN.md docs/design/skillc-v0-enhance-design.md docs/TODO.md docs/superpowers/plans/2026-06-14-skillc-v0-phase3-interactive-selection.md
git commit -m "docs: document interactive selection commands"
```

## Self-Review Checklist

- [ ] `termselect` is a `gookit/cliui` adapter only; it does not implement its own terminal parser, range parser, or filtering engine.
- [ ] CLI tests use fake selector injection and do not require a real TTY.
- [ ] Adapter tests use `interact.NewUIFakeBackend`.
- [ ] Non-interactive install/update/profile commands keep their current behavior.
- [ ] `install --interactive` treats positional `skill` as a search keyword only.
- [ ] `update --interactive` only offers `outdated` and `missing`.
- [ ] `profile create --interactive` is mutually exclusive with `--from-installed` and `--from-collection`.
- [ ] Keyword search matches ID, description, qualified names, source id/name, collection.
- [ ] `go test ./...` passes before marking the plan complete.

## Acceptance Criteria

- `skillc install --interactive [keyword]` can filter and multi-select indexed skills, then install selected skills through existing install flow.
- `skillc update --interactive` can filter and multi-select update candidates and update each selected target through existing update flow.
- `skillc profile create <name> --interactive` can filter and multi-select indexed skills and persist a profile with explicit targets.
- All three commands remain script-friendly: existing non-interactive forms still work and are tested.
- Interactive behavior is covered without real terminal input.
- No new TUI dependency is introduced.

## Risks and Guardrails

- `gcli` command state can leak between app runs in tests. Keep existing raw-args fill/reset patterns and include `interactive` in resets.
- Long source-qualified targets are awkward as direct UI keys. Use short numeric keys and store the stable target in item value.
- `statusapp` currently lacks `SourceQualifiedName` on output. Add it before relying on update interactive target precision.
- `repoindex.Filter` name/description-only matching is too weak for interactive discovery. Expand it before wiring install/profile interactive.
- `cliui` handles filtering and key events. Do not duplicate that behavior in `skillc`.
