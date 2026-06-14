# Skillc v0 Phase 1 Profile Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the first v0 enhancement slice: profile-based skill sets, profile apply dry-run/apply flow, source-scoped collection browsing, and removal of the top-level collection command from the recommended CLI surface.

**Architecture:** Keep the existing layered structure. Add a small `profile` domain model, persist profiles inside the existing YAML config, implement profile orchestration in `internal/app/profileapp`, reuse `searchapp` for index lookup and `installapp` for actual installation, and move collection browsing under `source` commands. This phase does not implement Web, Registry, cross-project status, or version drift.

**Tech Stack:** Go, `gookit/gcli`, existing YAML config store, existing JSON lock store, existing repo index, existing install/list services, Go unit tests with `github.com/gookit/goutil/x/assert`.

---

| 修订时间 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-13 | v0.1 | Codex | 基于 v0 增强设计文档输出 Phase 1 功能开发计划 |
| 2026-06-13 | v0.2 | Codex | 补充参考项目复审链接和一期边界校验 |
| 2026-06-14 | v0.3 | Codex | 二次复审一期计划，修正示例代码问题并补充工程复用和失败模式检查 |

相关文档：

- 设计文档：`docs/design/skillc-v0-enhance-design.md`
- 参考分析：`docs/design/skillc-reference-projects-analysis.md`
- 任务入口：`docs/TODO.md`

## 复审结论

复审 `docs/design/skillc-v0-enhance-design.md` 后，没有发现阻断一期计划的问题。

确认的一期边界：

- 做：`profile` 配置模型、`profile list/show/create --from-installed/create --from-collection/diff/apply --dry-run/--yes`。
- 做：lock record 增加 `profile` 归属字段。
- 做：`source collections`、`source skills <source-id> --collection <name>`，替代顶级 `collection` 命令。
- 做：从 collection 创建 profile 时展开为明确 `source + skill` targets。
- 做：移除 `install --collection` 推荐路径；实现层可先删除 CLI flag，内部解析函数可保留到后续清理。
- 不做：Web 管理界面。
- 不做：Registry。
- 不做：跨项目 install map / version drift。
- 不做：`status` 和 `update --check`。
- 不做：`profile export/import` 和 project manifest；仅保证 profile 数据模型不封死后续扩展。
- 不做：动态跟随 collection 的 `include_collection`。

参考项目复审后的约束：

- 不引入“当前激活 profile 自动吸收安装项”的隐式行为。
- `profile apply` 必须先生成 plan，plan 语义要为后续 Web/status/diff/outdated/check 复用留出空间。
- 本期不实现 `profile export/import` 和 project manifest，但数据模型不能封死后续项目内 manifest。
- 本期不增强 lock 的 deployed files/hash；只新增 `profile` 归属字段，完整 drift/version-diff 放到后续阶段。

### 2026-06-14 二次复审结论

结论：一期计划可以进入实现，没有发现需要扩大或收缩范围的阻断问题。计划仍按 `profile` 最小闭环推进，`collection` 只作为 source 下的浏览和导入输入，不恢复顶级命令。

本轮修正：

- 修正 Task 1 的 `ValidateName` 测试：空 profile 名称应为非法输入。
- 修正 Task 4 的 `SourceCollectionSummary` 排序示例：该结构没有 `ID` 字段，排序只使用 `SourceID` 和 `Name`。
- 明确 Phase 1 不实现 `profile export/import`，只避免模型设计阻塞后续项目 manifest。

已有能力复用检查：

- 复用 `configstore.YAMLStore` 持久化 profiles，不引入 SQLite 或中央库。
- 复用 `lockstore.Store` 读取安装事实，并只给 lock record 增加 `profile` 归属字段。
- 复用 `repoindex.Store` 和现有 index 数据解析 collection/source/skill，不新增第二套索引。
- 复用 `listapp.Service` 判断当前项目已安装项，避免 profileapp 直接扫描安装目录。
- 复用 `installapp.Service` 执行实际安装，profileapp 只负责 plan 和编排。
- 复用 CLI 现有 `confirmPrompt`、table 输出和 `defaultConfigFile/getWorkdir` 模式。

失败模式复审：

- profile 名称非法：domain 层拒绝，CLI/app service 直接返回错误。
- profile target 找不到：`PlanApply` 生成 `error` plan item，不应静默跳过。
- collection selector 不是 `<source>/<collection>`：`CreateFromCollection` 返回明确错误。
- 同名 skill 跨 source 歧义：profile target 优先保存 `source + skill`，apply 时校验 source mismatch。
- 重复 apply：已安装项应生成 `skip`，不重复写入 lock。
- apply 中安装失败：保留 `installapp` 的逐项失败结果和 lock 写入语义，不在 profileapp 自己写部分状态。
- top-level `collection` 命令移除后：必须补 `source collections` / `source skills --collection` CLI 测试，避免浏览能力丢失。

## File Structure

新增文件：

- `internal/domain/profile/model.go`
  - 定义 `Profile`、`Target`、`NamedProfile`、`ApplyPlan`、`ApplyPlanItem`。
  - 提供 `ValidateName`、`ValidateTarget`、`NormalizeTargets` 等纯规则函数。

- `internal/domain/profile/model_test.go`
  - 覆盖 profile 名称校验、target 校验、重复 target 去重、稳定排序。

- `internal/app/profileapp/service.go`
  - 编排 profile CRUD、从 lock 生成 profile、从 collection 生成 profile、生成 apply plan、执行 apply。
  - 复用 `configstore.YAMLStore`、`lockstore.Store`、`repoindex.Store`、`installapp.Service`。

- `internal/app/profileapp/service_test.go`
  - 覆盖 list/show/create/from-installed/from-collection/diff/apply dry-run/apply。

- `internal/cli/profile_cmd.go`
  - 注册 `skillc profile ...` 命令。
  - 只做参数解析、调用 app service、输出表格或计划结果。

修改文件：

- `internal/domain/config/model.go`
  - `Config` 增加 `Profiles map[string]profile.Profile`。

- `internal/infra/configstore/yaml_store.go`
  - `rawConfig` 增加 `Profiles`。
  - `Load` / `Save` / `cloneConfig` / `fromRawConfig` 支持 profiles。

- `internal/domain/lock/model.go`
  - `Record` 增加 `Profile string`。

- `internal/app/installapp/service.go`
  - `InstallReq` 增加 `Profile string`。
  - 安装写 lock record 时写入 `Profile`。

- `internal/app/listapp/service.go`
  - `Item` 增加 `Profile string`。
  - `List` 输出数据包含 profile 归属。

- `internal/infra/repoindex/collection.go`
  - 增加 source-scoped collection 列表和 source-scoped collection skills 查询。

- `internal/app/searchapp/service.go`
  - 增加 `ListSourceCollections(sourceID string)`。
  - 增加 `ListSourceSkills(sourceID, collection string)`。
  - 增加 profile target 解析辅助入口。

- `internal/cli/source_cmd.go`
  - 新增 `source collections [source-id]`。
  - 新增 `source skills <source-id> [--collection <name>]`。

- `internal/cli/manage_cmd.go`
  - 删除 `install --collection` flag 和 CLI 分支。
  - 保持普通 install、batch install、restore 不变。

- `internal/cli/app.go`
  - 注册 `profile`。
  - 移除顶级 `collection` 注册。

- `internal/cli/app_test.go`
  - 删除顶级 collection 注册测试。
  - 增加 profile/source collection/install flag 行为测试。

- `docs/design/skillc-v0-enhance-design.md`
  - 增加本计划链接。

- `docs/TODO.md`
  - 增加一期计划链接，不勾选完成。

删除文件：

- `internal/cli/collection_cmd.go`
  - 顶级 collection 命令不再保留。source 下的新命令替代其浏览能力。

## Task 1: Add Profile Domain Model

**Files:**
- Create: `internal/domain/profile/model.go`
- Test: `internal/domain/profile/model_test.go`

- [ ] **Step 1: Write failing profile domain tests**

Create `internal/domain/profile/model_test.go`:

```go
package profile

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "go-dev"},
		{name: "flutter_dev"},
		{name: "security.review"},
		{name: "", wantErr: true},
		{name: "has space", wantErr: true},
		{name: "../bad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.name)
			if tt.wantErr {
				assert.NotNil(t, err)
				return
			}
			assert.NoErr(t, err)
		})
	}
}

func TestNormalizeTargetsSortsAndDeduplicates(t *testing.T) {
	targets := []Target{
		{Source: "gstack", Skill: "review"},
		{Source: "gstack", Skill: "go-pro"},
		{Source: "gstack", Skill: "review"},
		{Skill: "local-only"},
	}

	got, err := NormalizeTargets(targets)

	assert.NoErr(t, err)
	assert.Len(t, got, 3)
	assert.Eq(t, "local-only", got[0].Skill)
	assert.Eq(t, "go-pro", got[1].Skill)
	assert.Eq(t, "review", got[2].Skill)
}

func TestNormalizeTargetsRejectsEmptySkill(t *testing.T) {
	_, err := NormalizeTargets([]Target{{Source: "gstack"}})

	assert.NotNil(t, err)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
go test ./internal/domain/profile -count=1
```

Expected: FAIL because `internal/domain/profile` does not exist.

- [ ] **Step 3: Implement profile model**

Create `internal/domain/profile/model.go`:

```go
package profile

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/inhere/skillc/internal/domain/skill"
)

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type Profile struct {
	Description  string   `yaml:"description,omitempty" json:"description,omitempty"`
	DefaultAgent string   `yaml:"default_agent,omitempty" json:"default_agent,omitempty"`
	DefaultScope string   `yaml:"default_scope,omitempty" json:"default_scope,omitempty"`
	InstallMode  string   `yaml:"install_mode,omitempty" json:"install_mode,omitempty"`
	Targets      []Target `yaml:"targets" json:"targets"`
}

type NamedProfile struct {
	Name string `json:"name"`
	Profile
}

type Target struct {
	Source string `yaml:"source,omitempty" json:"source,omitempty"`
	Skill  string `yaml:"skill" json:"skill"`
	Pinned bool   `yaml:"pinned,omitempty" json:"pinned,omitempty"`
}

type ApplyPlan struct {
	Profile string          `json:"profile"`
	Agent   string          `json:"agent"`
	Scope   string          `json:"scope"`
	Items   []ApplyPlanItem `json:"items"`
}

type ApplyPlanItem struct {
	Action string      `json:"action"`
	Target Target      `json:"target"`
	Skill  skill.Skill `json:"skill,omitempty"`
	Reason string      `json:"reason,omitempty"`
}

func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("profile name is required")
	}
	if !namePattern.MatchString(name) {
		return fmt.Errorf("invalid profile name: %s", name)
	}
	return nil
}

func NormalizeTargets(targets []Target) ([]Target, error) {
	seen := make(map[string]struct{}, len(targets))
	out := make([]Target, 0, len(targets))
	for _, target := range targets {
		target.Source = strings.TrimSpace(target.Source)
		target.Skill = strings.TrimSpace(target.Skill)
		if target.Skill == "" {
			return nil, fmt.Errorf("profile target skill is required")
		}
		key := target.Source + "\x00" + target.Skill
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, target)
	}
	slices.SortFunc(out, func(a, b Target) int {
		if a.Source != b.Source {
			if a.Source < b.Source {
				return -1
			}
			return 1
		}
		if a.Skill < b.Skill {
			return -1
		}
		if a.Skill > b.Skill {
			return 1
		}
		return 0
	})
	return out, nil
}
```

- [ ] **Step 4: Run profile domain tests**

Run:

```bash
go test ./internal/domain/profile -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/profile/model.go internal/domain/profile/model_test.go
git commit -m "feat(profile): add profile domain model"
```

## Task 2: Persist Profiles In Config

**Files:**
- Modify: `internal/domain/config/model.go`
- Modify: `internal/infra/configstore/yaml_store.go`
- Test: `internal/infra/configstore/yaml_store_test.go`

- [ ] **Step 1: Add failing config persistence test**

Append to `internal/infra/configstore/yaml_store_test.go`:

```go
func TestYAMLStore_LoadSaveProfiles(t *testing.T) {
	baseDir := t.TempDir()
	path := filepath.Join(baseDir, "skillc.yaml")

	data := cfg.DefaultConfig()
	data.Profiles = map[string]profile.Profile{
		"go-dev": {
			Description:  "Go development",
			DefaultAgent: "universal",
			DefaultScope: "project",
			Targets: []profile.Target{
				{Source: "gstack", Skill: "go-pro"},
				{Source: "gstack", Skill: "review"},
			},
		},
	}

	store := NewYAMLStore()
	assert.NoErr(t, store.Save(path, data, baseDir))

	got, err := store.Load(path, baseDir)
	assert.NoErr(t, err)
	assert.Len(t, got.Profiles, 1)
	assert.Eq(t, "Go development", got.Profiles["go-dev"].Description)
	assert.Len(t, got.Profiles["go-dev"].Targets, 2)
	assert.Eq(t, "review", got.Profiles["go-dev"].Targets[1].Skill)
}
```

Add imports if missing:

```go
import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/profile"
)
```

- [ ] **Step 2: Run configstore tests to verify failure**

Run:

```bash
go test ./internal/infra/configstore -count=1
```

Expected: FAIL because `cfg.Config` has no `Profiles`.

- [ ] **Step 3: Add profile field to config model**

Modify `internal/domain/config/model.go`:

```go
import (
	"slices"

	"github.com/inhere/skillc/internal/domain/profile"
	domainsource "github.com/inhere/skillc/internal/domain/source"
)
```

Add to `Config`:

```go
Profiles map[string]profile.Profile `yaml:"profiles,omitempty"`
```

- [ ] **Step 4: Persist profiles in YAML store**

Modify `internal/infra/configstore/yaml_store.go`:

```go
import (
	"os"
	"path/filepath"

	gkconfig "github.com/gookit/config/v2"
	gkyaml "github.com/gookit/config/v2/yaml"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/profile"
	domainsource "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/fsx"
)
```

Add to `rawConfig`:

```go
Profiles map[string]profile.Profile `yaml:"profiles,omitempty"`
```

Add to `Save` `loader.SetData` map:

```go
"profiles": persisted.Profiles,
```

Add to `cloneConfig`:

```go
if data.Profiles != nil {
	clone.Profiles = make(map[string]profile.Profile, len(data.Profiles))
	for name, item := range data.Profiles {
		targets := append([]profile.Target(nil), item.Targets...)
		item.Targets = targets
		clone.Profiles[name] = item
	}
}
```

Add to `fromRawConfig` return value:

```go
Profiles: raw.Profiles,
```

- [ ] **Step 5: Run configstore tests**

Run:

```bash
go test ./internal/infra/configstore -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/config/model.go internal/infra/configstore/yaml_store.go internal/infra/configstore/yaml_store_test.go
git commit -m "feat(profile): persist profiles in config"
```

## Task 3: Add Profile Attribution To Lock Records

**Files:**
- Modify: `internal/domain/lock/model.go`
- Modify: `internal/app/installapp/service.go`
- Modify: `internal/app/installapp/service_test.go`
- Modify: `internal/app/listapp/service.go`
- Modify: `internal/app/listapp/service_test.go`

- [ ] **Step 1: Add failing install profile attribution test**

Append to `internal/app/installapp/service_test.go`:

```go
func TestInstallMulti_RecordsProfileName(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	sourceDir := filepath.Join(baseDir, "source", "go-pro")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Go Pro"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}

	item := skill.Skill{
		ID:           "go-pro",
		SourceID:     "gstack",
		InstallEntry: ".",
		Path:         sourceDir,
	}

	_, err := NewService(lockFile).RunResolved(config, InstallReq{
		Agent:   "universal",
		Scope:   "project",
		WorkDir: baseDir,
		Profile: "go-dev",
	}, []skill.Skill{item}, nil)
	assert.NoErr(t, err)

	records, err := lockstore.NewStore().Load(lockFile)
	assert.NoErr(t, err)
	assert.Eq(t, "go-dev", records[filepath.Clean(baseDir)][0].Profile)
}
```

- [ ] **Step 2: Run focused tests to verify failure**

Run:

```bash
go test ./internal/app/installapp -count=1
```

Expected: FAIL because `InstallReq.Profile` and `lock.Record.Profile` do not exist.

- [ ] **Step 3: Add profile field to lock and install**

Modify `internal/domain/lock/model.go`:

```go
Profile string `json:"profile,omitempty"`
```

Add to `installapp.InstallReq` in `internal/app/installapp/service.go`:

```go
Profile string
```

In `RunResolved`, pass profile into `InstallMulti` explicitly:

```go
installResult, err := runtimeSvc.InstallMulti(items, req.Agent, scope, scopeKey, targetRoot, req.Profile)
```

Change `InstallMulti` signature:

```go
func (s *Service) InstallMulti(items []skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetRoot string, profileName string) (BatchInstallResult, error)
```

Inside `InstallMulti`, call `installInto` with the profile name:

```go
record, err := s.installInto(item, agentName, scope, scopeKey, targetRoot, locks, profileName)
```

Change `Install` and `ReinstallAtPath` to pass an empty profile name for now:

```go
record, err := s.installInto(item, agentName, scope, scopeKey, targetRoot, locks, "")
```

Change `installInto` signature:

```go
func (s *Service) installInto(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetRoot string, locks lockpkg.File, profileName string) (RuntimeRecord, error)
```

In `installInto`, set:

```go
Profile: profileName,
```

Keep `ReinstallAtPath` profile empty for this phase unless the update flow is later extended to preserve profile attribution. This avoids hidden service state and keeps profile attribution tied to explicit install requests.

- [ ] **Step 4: Add list item profile field**

Modify `internal/app/listapp/service.go` `Item`:

```go
Profile string
```

In `toItem`, set:

```go
Profile: record.Profile,
```

Add a focused test in `internal/app/listapp/service_test.go`:

```go
func TestList_IncludesProfileName(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	installedPath := filepath.Join(baseDir, ".agents", "skills", "go-pro")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID: "go-pro",
			SourceID: "gstack",
			Agents: []string{"universal"},
			Profile: "go-dev",
		}},
	}))

	items, err := NewService(lockFile).WithRuntime(config, baseDir).List("universal", "project")
	assert.NoErr(t, err)
	assert.Len(t, items, 1)
	assert.Eq(t, "go-dev", items[0].Profile)
}
```

- [ ] **Step 5: Run install/list tests**

Run:

```bash
go test ./internal/app/installapp ./internal/app/listapp -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/lock/model.go internal/app/installapp/service.go internal/app/installapp/service_test.go internal/app/listapp/service.go internal/app/listapp/service_test.go
git commit -m "feat(profile): record profile attribution in lock"
```

## Task 4: Move Collection Browsing Under Source

**Files:**
- Modify: `internal/infra/repoindex/collection.go`
- Modify: `internal/infra/repoindex/collection_test.go`
- Modify: `internal/app/searchapp/service.go`
- Modify: `internal/app/searchapp/service_test.go`
- Modify: `internal/cli/source_cmd.go`
- Modify: `internal/cli/app.go`
- Modify: `internal/cli/app_test.go`
- Delete: `internal/cli/collection_cmd.go`

- [ ] **Step 1: Add repoindex source collection tests**

Append to `internal/infra/repoindex/collection_test.go`:

```go
func TestListSourceCollectionsFiltersBySource(t *testing.T) {
	items := []skill.Skill{
		{ID: "go-pro", Collection: "go", SourceID: "gstack", SourceName: "GStack"},
		{ID: "go-test", Collection: "go", SourceID: "gstack", SourceName: "GStack"},
		{ID: "py-pro", Collection: "python", SourceID: "team", SourceName: "Team"},
	}

	got := ListSourceCollections(items, "gstack")

	assert.Len(t, got, 1)
	assert.Eq(t, "go", got[0].Name)
	assert.Eq(t, "gstack", got[0].SourceID)
	assert.Eq(t, 2, got[0].SkillCount)
}

func TestListSourceSkillsFiltersByCollection(t *testing.T) {
	items := []skill.Skill{
		{ID: "go-pro", Name: "Go Pro", Collection: "go", SourceID: "gstack"},
		{ID: "review", Name: "Review", Collection: "ops", SourceID: "gstack"},
		{ID: "other", Name: "Other", Collection: "go", SourceID: "team"},
	}

	got, err := ListSourceSkills(items, "gstack", "go")

	assert.NoErr(t, err)
	assert.Len(t, got, 1)
	assert.Eq(t, "go-pro", got[0].ID)
}
```

- [ ] **Step 2: Run repoindex tests to verify failure**

Run:

```bash
go test ./internal/infra/repoindex -count=1
```

Expected: FAIL because `ListSourceCollections` and `ListSourceSkills` do not exist.

- [ ] **Step 3: Implement source-scoped collection helpers**

Add to `internal/infra/repoindex/collection.go`:

```go
type SourceCollectionSummary struct {
	SourceID   string
	SourceName string
	Name       string
	SkillCount int
}

func ListSourceCollections(items []skill.Skill, sourceID string) []SourceCollectionSummary {
	groups := make(map[string]*SourceCollectionSummary)
	for _, item := range items {
		if item.Collection == "" {
			continue
		}
		if sourceID != "" && item.SourceID != sourceID && item.SourceName != sourceID {
			continue
		}
		key := item.SourceID + "\x00" + item.Collection
		summary := groups[key]
		if summary == nil {
			summary = &SourceCollectionSummary{
				SourceID:   item.SourceID,
				SourceName: item.SourceName,
				Name:       item.Collection,
			}
			groups[key] = summary
		}
		summary.SkillCount++
	}
	result := make([]SourceCollectionSummary, 0, len(groups))
	for _, summary := range groups {
		result = append(result, *summary)
	}
	slices.SortFunc(result, func(a, b SourceCollectionSummary) int {
		if a.SourceID != b.SourceID {
			if a.SourceID < b.SourceID {
				return -1
			}
			return 1
		}
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return result
}

func ListSourceSkills(items []skill.Skill, sourceID string, collection string) ([]skill.Skill, error) {
	matched := make([]skill.Skill, 0)
	for _, item := range items {
		if sourceID != "" && item.SourceID != sourceID && item.SourceName != sourceID {
			continue
		}
		if collection != "" && item.Collection != collection {
			continue
		}
		matched = append(matched, item)
	}
	if len(matched) == 0 {
		return nil, fmt.Errorf("source skills not found")
	}
	slices.SortFunc(matched, func(a, b skill.Skill) int {
		if a.Collection != b.Collection {
			if a.Collection < b.Collection {
				return -1
			}
			return 1
		}
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return matched, nil
}
```

- [ ] **Step 4: Add searchapp wrappers and tests**

Add to `internal/app/searchapp/service.go`:

```go
func (s *Service) ListSourceCollections(sourceID string) ([]repoindex.SourceCollectionSummary, error) {
	items, err := s.store.Load(s.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []repoindex.SourceCollectionSummary{}, nil
		}
		return nil, err
	}
	return repoindex.ListSourceCollections(items, sourceID), nil
}

func (s *Service) ListSourceSkills(sourceID string, collection string) ([]skill.Skill, error) {
	items, err := s.store.Load(s.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("source skills not found")
		}
		return nil, err
	}
	return repoindex.ListSourceSkills(items, sourceID, collection)
}
```

Add tests in `internal/app/searchapp/service_test.go` that save an index and assert `ListSourceCollections("gstack")` and `ListSourceSkills("gstack", "go")`.

- [ ] **Step 5: Add source CLI commands**

In `internal/cli/source_cmd.go`, add two commands to `buildSourceCommand()`:

```go
cmd.Add(buildSourceCollectionsCommand())
cmd.Add(buildSourceSkillsCommand())
```

Add command builders:

```go
func buildSourceCollectionsCommand() *gcli.Command {
	return &gcli.Command{
		Name: "collections",
		Desc: "List collections grouped under sources",
		Config: func(c *gcli.Command) {
			c.AddArg("source", "source id or name")
		},
		Func: func(c *gcli.Command, _ []string) error {
			sourceID := c.Arg("source").String()
			items, err := newSearchService().ListSourceCollections(sourceID)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				ccolor.Warnln("no collections found")
				return nil
			}
			tb := table.New("Source Collections").SetHeads("Source", "Collection", "Skills")
			for _, item := range items {
				sourceName := item.SourceID
				if item.SourceName != "" {
					sourceName = item.SourceName
				}
				tb.AddRow(sourceName, item.Name, item.SkillCount)
			}
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
		},
	}
}

func buildSourceSkillsCommand() *gcli.Command {
	var collection string
	return &gcli.Command{
		Name: "skills",
		Desc: "List skills under a source",
		Config: func(c *gcli.Command) {
			c.AddArg("source", "source id or name", true)
			c.StrOpt(&collection, "collection", "c", "", "filter by source collection")
		},
		Func: func(c *gcli.Command, _ []string) error {
			sourceID := c.Arg("source").String()
			items, err := newSearchService().ListSourceSkills(sourceID, collection)
			if err != nil {
				return err
			}
			tb := table.New("Source Skills").SetHeads("Collection", "Skill", "Description")
			for _, item := range items {
				tb.AddRow(item.Collection, item.ID, truncateDescription(item.Description, 60))
			}
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
		},
	}
}
```

Move `truncateStr` from deleted `collection_cmd.go` into `source_cmd.go` as `truncateDescription`.

- [ ] **Step 6: Remove top-level collection registration**

Modify `internal/cli/app.go`:

```go
// remove:
// app.Add(buildCollectionCommand())
```

Delete `internal/cli/collection_cmd.go`.

Update `internal/cli/app_test.go`:

```go
func TestNewApp_DoesNotRegisterTopLevelCollectionCommand(t *testing.T) {
	app := newTestApp()

	collection := findCommandByName(app, "collection")
	assert.Nil(t, collection)
}
```

Add CLI tests for:

```go
output := runAppInDirWithStdout(t, baseDir, []string{"source", "collections", "gstack"})
assert.Contains(t, output, "go")

output = runAppInDirWithStdout(t, baseDir, []string{"source", "skills", "gstack", "--collection", "go"})
assert.Contains(t, output, "go-pro")
```

- [ ] **Step 7: Run focused tests**

Run:

```bash
go test ./internal/infra/repoindex ./internal/app/searchapp ./internal/cli -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/infra/repoindex/collection.go internal/infra/repoindex/collection_test.go internal/app/searchapp/service.go internal/app/searchapp/service_test.go internal/cli/source_cmd.go internal/cli/app.go internal/cli/app_test.go
git rm internal/cli/collection_cmd.go
git commit -m "refactor(cli): move collection browsing under source"
```

## Task 5: Implement Profile Service Create/List/Show

**Files:**
- Create: `internal/app/profileapp/service.go`
- Test: `internal/app/profileapp/service_test.go`

- [ ] **Step 1: Add failing profile service tests**

Create `internal/app/profileapp/service_test.go` with:

```go
package profileapp

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/profile"
	"github.com/inhere/skillc/internal/infra/configstore"
)

func TestService_ListAndShowProfiles(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	config := cfg.DefaultConfig()
	config.Profiles = map[string]profile.Profile{
		"go-dev": {Description: "Go dev", Targets: []profile.Target{{Source: "gstack", Skill: "go-pro"}}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))

	svc := NewService(configFile, baseDir)
	list, err := svc.List()
	assert.NoErr(t, err)
	assert.Len(t, list, 1)
	assert.Eq(t, "go-dev", list[0].Name)

	got, err := svc.Show("go-dev")
	assert.NoErr(t, err)
	assert.Eq(t, "Go dev", got.Description)
}

func TestService_SaveProfileNormalizesTargets(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, cfg.DefaultConfig(), baseDir))

	svc := NewService(configFile, baseDir)
	err := svc.Save("go-dev", profile.Profile{
		Targets: []profile.Target{
			{Source: "gstack", Skill: "review"},
			{Source: "gstack", Skill: "go-pro"},
			{Source: "gstack", Skill: "review"},
		},
	})
	assert.NoErr(t, err)

	got, err := svc.Show("go-dev")
	assert.NoErr(t, err)
	assert.Len(t, got.Targets, 2)
	assert.Eq(t, "go-pro", got.Targets[0].Skill)
}
```

- [ ] **Step 2: Run profileapp tests to verify failure**

Run:

```bash
go test ./internal/app/profileapp -count=1
```

Expected: FAIL because package does not exist.

- [ ] **Step 3: Implement profile service CRUD**

Create `internal/app/profileapp/service.go`:

```go
package profileapp

import (
	"fmt"
	"sort"

	"github.com/inhere/skillc/internal/domain/profile"
	"github.com/inhere/skillc/internal/infra/configstore"
)

type Service struct {
	configFile string
	baseDir    string
	store      *configstore.YAMLStore
}

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile: configFile,
		baseDir:    baseDir,
		store:      configstore.NewYAMLStore(),
	}
}

func (s *Service) List() ([]profile.NamedProfile, error) {
	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(config.Profiles))
	for name := range config.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]profile.NamedProfile, 0, len(names))
	for _, name := range names {
		out = append(out, profile.NamedProfile{Name: name, Profile: config.Profiles[name]})
	}
	return out, nil
}

func (s *Service) Show(name string) (profile.Profile, error) {
	if err := profile.ValidateName(name); err != nil {
		return profile.Profile{}, err
	}
	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return profile.Profile{}, err
	}
	item, ok := config.Profiles[name]
	if !ok {
		return profile.Profile{}, fmt.Errorf("profile not found: %s", name)
	}
	return item, nil
}

func (s *Service) Save(name string, item profile.Profile) error {
	if err := profile.ValidateName(name); err != nil {
		return err
	}
	targets, err := profile.NormalizeTargets(item.Targets)
	if err != nil {
		return err
	}
	item.Targets = targets
	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return err
	}
	if config.Profiles == nil {
		config.Profiles = map[string]profile.Profile{}
	}
	config.Profiles[name] = item
	return s.store.Save(s.configFile, config, s.baseDir)
}
```

- [ ] **Step 4: Run profileapp tests**

Run:

```bash
go test ./internal/app/profileapp -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/profileapp/service.go internal/app/profileapp/service_test.go
git commit -m "feat(profile): add profile service"
```

## Task 6: Create Profiles From Installed Skills And Collections

**Files:**
- Modify: `internal/app/profileapp/service.go`
- Modify: `internal/app/profileapp/service_test.go`

- [ ] **Step 1: Add failing tests for profile creation sources**

Append tests:

```go
func TestService_CreateFromInstalled(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	installedPath := filepath.Join(baseDir, ".agents", "skills", "go-pro")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID: "go-pro",
			SourceID: "gstack",
			Agents: []string{"universal"},
		}},
	}))

	svc := NewService(configFile, baseDir)
	got, err := svc.CreateFromInstalled("go-dev", CreateFromInstalledReq{
		Agent: "universal",
		Scope: "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Len(t, got.Targets, 1)
	assert.Eq(t, "gstack", got.Targets[0].Source)
	assert.Eq(t, "go-pro", got.Targets[0].Skill)
}

func TestService_CreateFromCollection(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Collection: "go"},
		{ID: "review", SourceID: "gstack", Collection: "go"},
		{ID: "python", SourceID: "gstack", Collection: "python"},
	}))

	svc := NewService(configFile, baseDir)
	got, err := svc.CreateFromCollection("go-dev", "gstack/go")

	assert.NoErr(t, err)
	assert.Len(t, got.Targets, 2)
	assert.Eq(t, "go-pro", got.Targets[0].Skill)
	assert.Eq(t, "review", got.Targets[1].Skill)
}
```

Imports needed:

```go
import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/skillc/internal/domain/skill"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)
```

- [ ] **Step 2: Run profileapp tests to verify failure**

Run:

```bash
go test ./internal/app/profileapp -count=1
```

Expected: FAIL because methods do not exist.

- [ ] **Step 3: Implement create from installed**

Add request type and method to `internal/app/profileapp/service.go`:

```go
type CreateFromInstalledReq struct {
	Agent   string
	Scope   string
	WorkDir string
}

func (s *Service) CreateFromInstalled(name string, req CreateFromInstalledReq) (profile.Profile, error) {
	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return profile.Profile{}, err
	}
	scope := req.Scope
	if scope == "" {
		scope = "project"
	}
	agentName := req.Agent
	if agentName == "" {
		agentName = "universal"
	}
	workDir := req.WorkDir
	if workDir == "" {
		workDir = s.baseDir
	}
	items, err := listapp.NewService(config.LockFile).WithRuntime(config, workDir).List(agentName, scope)
	if err != nil {
		return profile.Profile{}, err
	}
	targets := make([]profile.Target, 0, len(items))
	for _, item := range items {
		targets = append(targets, profile.Target{
			Source: item.SourceID,
			Skill:  item.SkillID,
		})
	}
	out := profile.Profile{
		DefaultAgent: agentName,
		DefaultScope: scope,
		Targets:      targets,
	}
	if err := s.Save(name, out); err != nil {
		return profile.Profile{}, err
	}
	return s.Show(name)
}
```

Add imports:

```go
import "strings"

import (
	"github.com/inhere/skillc/internal/app/listapp"
	"github.com/inhere/skillc/internal/domain/skill"
	"github.com/inhere/skillc/internal/infra/repoindex"
)
```

- [ ] **Step 4: Implement create from collection**

Add:

```go
func (s *Service) CreateFromCollection(name string, selector string) (profile.Profile, error) {
	sourceID, collection, ok := strings.Cut(selector, "/")
	if !ok || sourceID == "" || collection == "" {
		return profile.Profile{}, fmt.Errorf("collection selector must be <source>/<collection>")
	}
	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return profile.Profile{}, err
	}
	items, err := repoindex.NewStore().Load(config.IndexFile)
	if err != nil {
		return profile.Profile{}, err
	}
	matched, err := repoindex.ListSourceSkills(items, sourceID, collection)
	if err != nil {
		return profile.Profile{}, err
	}
	targets := make([]profile.Target, 0, len(matched))
	for _, item := range matched {
		targets = append(targets, profile.Target{Source: item.SourceID, Skill: item.ID})
	}
	out := profile.Profile{Targets: targets}
	if err := s.Save(name, out); err != nil {
		return profile.Profile{}, err
	}
	return s.Show(name)
}
```

- [ ] **Step 5: Run profileapp tests**

Run:

```bash
go test ./internal/app/profileapp -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/app/profileapp/service.go internal/app/profileapp/service_test.go
git commit -m "feat(profile): create profiles from installed skills and collections"
```

## Task 7: Add Profile Apply Plan And Apply Execution

**Files:**
- Modify: `internal/app/profileapp/service.go`
- Modify: `internal/app/profileapp/service_test.go`

- [ ] **Step 1: Add failing apply plan tests**

Append:

```go
func TestService_PlanApplySkipsInstalledAndInstallsMissing(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	installedPath := filepath.Join(baseDir, ".agents", "skills", "go-pro")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))

	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.LockFile = lockFile
	config.Profiles = map[string]profile.Profile{
		"go-dev": {Targets: []profile.Target{
			{Source: "gstack", Skill: "go-pro"},
			{Source: "gstack", Skill: "review"},
		}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack"},
		{ID: "review", SourceID: "gstack"},
	}))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID: "go-pro",
			SourceID: "gstack",
			Agents: []string{"universal"},
		}},
	}))

	plan, err := NewService(configFile, baseDir).PlanApply("go-dev", ApplyReq{
		Agent: "universal",
		Scope: "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Len(t, plan.Items, 2)
	assert.Eq(t, "skip", plan.Items[0].Action)
	assert.Eq(t, "install", plan.Items[1].Action)
}
```

- [ ] **Step 2: Run profileapp tests to verify failure**

Run:

```bash
go test ./internal/app/profileapp -count=1
```

Expected: FAIL because `ApplyReq` and `PlanApply` do not exist.

- [ ] **Step 3: Implement apply planning**

Add to `internal/app/profileapp/service.go`:

```go
type ApplyReq struct {
	Agent  string
	Scope  string
	WorkDir string
}

type ApplyResult struct {
	Plan      profile.ApplyPlan
	Installed []installapp.RuntimeRecord
}

func (s *Service) PlanApply(name string, req ApplyReq) (profile.ApplyPlan, error) {
	item, err := s.Show(name)
	if err != nil {
		return profile.ApplyPlan{}, err
	}
	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return profile.ApplyPlan{}, err
	}
	agentName := firstNonEmpty(req.Agent, item.DefaultAgent, "universal")
	scope := firstNonEmpty(req.Scope, item.DefaultScope, "project")
	workDir := firstNonEmpty(req.WorkDir, s.baseDir)

	indexItems, err := repoindex.NewStore().Load(config.IndexFile)
	if err != nil {
		return profile.ApplyPlan{}, err
	}
	installed, err := listapp.NewService(config.LockFile).WithRuntime(config, workDir).List(agentName, scope)
	if err != nil {
		return profile.ApplyPlan{}, err
	}
	installedSet := make(map[string]struct{}, len(installed))
	for _, current := range installed {
		installedSet[current.SourceID+"\x00"+current.SkillID] = struct{}{}
		installedSet["\x00"+current.SkillID] = struct{}{}
	}

	plan := profile.ApplyPlan{Profile: name, Agent: agentName, Scope: scope}
	for _, target := range item.Targets {
		found, ok := findTargetSkill(indexItems, target)
		if !ok {
			plan.Items = append(plan.Items, profile.ApplyPlanItem{Action: "error", Target: target, Reason: "skill not found in index"})
			continue
		}
		installedKey := target.Source + "\x00" + target.Skill
		if _, ok := installedSet[installedKey]; ok {
			plan.Items = append(plan.Items, profile.ApplyPlanItem{Action: "skip", Target: target, Skill: found, Reason: "already installed"})
			continue
		}
		if target.Source == "" {
			if _, ok := installedSet["\x00"+target.Skill]; ok {
				plan.Items = append(plan.Items, profile.ApplyPlanItem{Action: "skip", Target: target, Skill: found, Reason: "already installed"})
				continue
			}
		}
		if target.Source != "" && found.SourceID != target.Source && found.SourceName != target.Source {
			plan.Items = append(plan.Items, profile.ApplyPlanItem{Action: "error", Target: target, Reason: "resolved skill source mismatch"})
			continue
		}
		if _, ok := installedSet[found.SourceID+"\x00"+found.ID]; ok {
			plan.Items = append(plan.Items, profile.ApplyPlanItem{Action: "skip", Target: target, Skill: found, Reason: "already installed"})
			continue
		}
		plan.Items = append(plan.Items, profile.ApplyPlanItem{Action: "install", Target: target, Skill: found})
	}
	return plan, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func findTargetSkill(items []skill.Skill, target profile.Target) (skill.Skill, bool) {
	for _, item := range items {
		if target.Source != "" && item.SourceID != target.Source && item.SourceName != target.Source {
			continue
		}
		if item.ID == target.Skill || item.QualifiedName == target.Skill || item.SourceQualifiedName == target.Skill {
			return item, true
		}
	}
	return skill.Skill{}, false
}
```

- [ ] **Step 4: Implement apply execution**

Add:

```go
func (s *Service) Apply(name string, req ApplyReq) (ApplyResult, error) {
	plan, err := s.PlanApply(name, req)
	if err != nil {
		return ApplyResult{}, err
	}
	config, err := s.store.Load(s.configFile, s.baseDir)
	if err != nil {
		return ApplyResult{}, err
	}
	toInstall := make([]skill.Skill, 0)
	for _, item := range plan.Items {
		if item.Action == "install" {
			toInstall = append(toInstall, item.Skill)
		}
	}
	if len(toInstall) == 0 {
		return ApplyResult{Plan: plan}, nil
	}
	workDir := firstNonEmpty(req.WorkDir, s.baseDir)
	result, err := installapp.NewService(config.LockFile).RunResolved(config, installapp.InstallReq{
		Agent:   plan.Agent,
		Scope:   plan.Scope,
		WorkDir: workDir,
		Profile: name,
	}, toInstall, nil)
	if err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{Plan: plan, Installed: result.Installed}, nil
}
```

- [ ] **Step 5: Add apply execution test**

Append:

```go
func TestService_ApplyInstallsMissingSkillsWithProfile(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	sourceDir := filepath.Join(baseDir, "source", "review")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Review"), 0o644))

	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.LockFile = lockFile
	config.Profiles = map[string]profile.Profile{
		"go-dev": {Targets: []profile.Target{{Source: "gstack", Skill: "review"}}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "review", SourceID: "gstack", InstallEntry: ".", Path: sourceDir},
	}))

	result, err := NewService(configFile, baseDir).Apply("go-dev", ApplyReq{
		Agent: "universal",
		Scope: "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Len(t, result.Installed, 1)
	records, err := lockstore.NewStore().Load(lockFile)
	assert.NoErr(t, err)
	assert.Eq(t, "go-dev", records[filepath.Clean(baseDir)][0].Profile)
}
```

- [ ] **Step 6: Run profileapp tests**

Run:

```bash
go test ./internal/app/profileapp -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/app/profileapp/service.go internal/app/profileapp/service_test.go
git commit -m "feat(profile): plan and apply profiles"
```

## Task 8: Add Profile CLI And Remove Install Collection Flag

**Files:**
- Create: `internal/cli/profile_cmd.go`
- Modify: `internal/cli/app.go`
- Modify: `internal/cli/manage_cmd.go`
- Modify: `internal/cli/app_test.go`

- [ ] **Step 1: Add failing CLI registration tests**

Modify `internal/cli/app_test.go`:

```go
func TestNewApp_RegistersProfileCommand(t *testing.T) {
	app := newTestApp()

	profile := findCommandByName(app, "profile")
	assert.NotNil(t, profile)
	assert.Eq(t, "Manage Skillc profiles", profile.Desc)
}

func TestInstallCommand_DoesNotAcceptCollectionFlag(t *testing.T) {
	baseDir := t.TempDir()
	config := cfg.DefaultConfig()
	assert.NoErr(t, configstore.NewYAMLStore().Save(filepath.Join(baseDir, "skillc.yaml"), config))

	output := runAppInDirWithStdout(t, baseDir, []string{"install", "--collection", "go"})

	assert.Contains(t, output, "unknown option")
}
```

- [ ] **Step 2: Run CLI tests to verify failure**

Run:

```bash
go test ./internal/cli -count=1
```

Expected: FAIL because profile command is missing and install still accepts `--collection`.

- [ ] **Step 3: Add profile CLI command**

Create `internal/cli/profile_cmd.go`:

```go
package cli

import (
	"fmt"
	"os"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/cliui/show"
	"github.com/gookit/cliui/show/table"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/skillc/internal/app/profileapp"
)

func newProfileService() *profileapp.Service {
	cwd := getWorkdir()
	return profileapp.NewService(defaultConfigFile(cwd), cwd)
}

func buildProfileCommand() *gcli.Command {
	cmd := &gcli.Command{Name: "profile", Desc: "Manage Skillc profiles"}
	cmd.Add(buildProfileListCommand())
	cmd.Add(buildProfileShowCommand())
	cmd.Add(buildProfileCreateCommand())
	cmd.Add(buildProfileDiffCommand())
	cmd.Add(buildProfileApplyCommand())
	return cmd
}

func buildProfileListCommand() *gcli.Command {
	return &gcli.Command{
		Name: "list",
		Desc: "List profiles",
		Aliases: []string{"ls"},
		Func: func(c *gcli.Command, _ []string) error {
			items, err := newProfileService().List()
			if err != nil {
				return err
			}
			tb := table.New("Profiles").SetHeads("Name", "Targets", "Description")
			for _, item := range items {
				tb.AddRow(item.Name, len(item.Targets), item.Description)
			}
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
		},
	}
}

func buildProfileShowCommand() *gcli.Command {
	return &gcli.Command{
		Name: "show",
		Desc: "Show profile details",
		Config: func(c *gcli.Command) {
			c.AddArg("name", "profile name", true)
		},
		Func: func(c *gcli.Command, _ []string) error {
			item, err := newProfileService().Show(c.Arg("name").String())
			if err != nil {
				return err
			}
			show.AList("Profile", item)
			return nil
		},
	}
}
```

Continue the file with `create`, `diff`, and `apply` builders:

```go
func buildProfileCreateCommand() *gcli.Command {
	var fromInstalled bool
	var fromCollection string
	var agentName string
	var scope string
	return &gcli.Command{
		Name: "create",
		Desc: "Create a profile",
		Config: func(c *gcli.Command) {
			c.AddArg("name", "profile name", true)
			c.BoolOpt(&fromInstalled, "from-installed", "", false, "create from installed skills")
			c.StrOpt(&fromCollection, "from-collection", "", "", "create from <source>/<collection>")
			c.StrOpt(&agentName, "agent", "a", "", "agent name")
			c.StrOpt(&scope, "scope", "s", "project", "scope")
		},
		Func: func(c *gcli.Command, _ []string) error {
			name := c.Arg("name").String()
			svc := newProfileService()
			switch {
			case fromCollection != "":
				_, err := svc.CreateFromCollection(name, fromCollection)
				if err != nil {
					return err
				}
			case fromInstalled:
				_, err := svc.CreateFromInstalled(name, profileapp.CreateFromInstalledReq{
					Agent: agentName,
					Scope: scope,
					WorkDir: getWorkdir(),
				})
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("use --from-installed or --from-collection")
			}
			ccolor.Successf("profile created: %s\n", name)
			return nil
		},
	}
}

func buildProfileDiffCommand() *gcli.Command {
	var agentName string
	var scope string
	return &gcli.Command{
		Name: "diff",
		Desc: "Preview profile apply plan",
		Config: func(c *gcli.Command) {
			c.AddArg("name", "profile name", true)
			c.StrOpt(&agentName, "agent", "a", "", "agent name")
			c.StrOpt(&scope, "scope", "s", "project", "scope")
		},
		Func: func(c *gcli.Command, _ []string) error {
			plan, err := newProfileService().PlanApply(c.Arg("name").String(), profileapp.ApplyReq{
				Agent: agentName,
				Scope: scope,
				WorkDir: getWorkdir(),
			})
			if err != nil {
				return err
			}
			return printProfilePlan(plan)
		},
	}
}

func buildProfileApplyCommand() *gcli.Command {
	var agentName string
	var scope string
	var dryRun bool
	var yes bool
	return &gcli.Command{
		Name: "apply",
		Desc: "Apply a profile",
		Config: func(c *gcli.Command) {
			c.AddArg("name", "profile name", true)
			c.StrOpt(&agentName, "agent", "a", "", "agent name")
			c.StrOpt(&scope, "scope", "s", "project", "scope")
			c.BoolOpt(&dryRun, "dry-run", "", false, "preview without installing")
			c.BoolOpt(&yes, "yes", "y", false, "skip confirmation")
		},
		Func: func(c *gcli.Command, _ []string) error {
			name := c.Arg("name").String()
			svc := newProfileService()
			req := profileapp.ApplyReq{Agent: agentName, Scope: scope, WorkDir: getWorkdir()}
			plan, err := svc.PlanApply(name, req)
			if err != nil {
				return err
			}
			if err := printProfilePlan(plan); err != nil {
				return err
			}
			if dryRun {
				return nil
			}
			if !yes {
				confirmed, err := confirmPrompt(os.Stdin, os.Stdout, "Apply profile?")
				if err != nil {
					return err
				}
				if !confirmed {
					ccolor.Warnln("profile apply cancelled")
					return nil
				}
			}
			result, err := svc.Apply(name, req)
			if err != nil {
				return err
			}
			ccolor.Successf("profile applied: %s installed=%d\n", name, len(result.Installed))
			return nil
		},
	}
}
```

Add `printProfilePlan`:

```go
func printProfilePlan(plan profile.ApplyPlan) error {
	tb := table.New("Profile Plan").SetHeads("Action", "Source", "Skill", "Reason")
	for _, item := range plan.Items {
		tb.AddRow(item.Action, item.Target.Source, item.Target.Skill, item.Reason)
	}
	_, err := fmt.Fprint(os.Stdout, tb.Render())
	return err
}
```

Import `github.com/inhere/skillc/internal/domain/profile`.

- [ ] **Step 4: Register profile command**

Modify `internal/cli/app.go`:

```go
app.Add(buildProfileCommand())
```

Place it after `buildSourceCommand()` and before `buildSearchCommand()`.

- [ ] **Step 5: Remove install collection flag**

Modify `internal/cli/manage_cmd.go`:

```go
type ManageOptions struct {
	Scope       string
	Agent       string
	Yes         bool
	UseCopy     bool
	InstallMode string
}
```

Remove:

```go
c.BoolOpt(&opts.Collection, "collection", "c", false, "treat targets as collection selectors")
```

Change:

```go
searchResult, err := newSearchService().ResolveInstallTargets(targets, opts.Collection)
```

to:

```go
searchResult, err := newSearchService().ResolveInstallTargets(targets, false)
```

- [ ] **Step 6: Add profile CLI behavior tests**

Add tests in `internal/cli/app_test.go`:

```go
func TestProfileCreateFromCollectionCommand(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Collection: "go"},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"profile", "create", "go-dev", "--from-collection", "gstack/go"})

	assert.Contains(t, output, "profile created: go-dev")
	loaded, err := configstore.NewYAMLStore().Load(configFile, baseDir)
	assert.NoErr(t, err)
	assert.Len(t, loaded.Profiles["go-dev"].Targets, 1)
}

func TestProfileApplyDryRunPrintsPlan(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.Profiles = map[string]profile.Profile{
		"go-dev": {Targets: []profile.Target{{Source: "gstack", Skill: "go-pro"}}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack"},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"profile", "apply", "go-dev", "--dry-run"})

	assert.Contains(t, output, "Profile Plan")
	assert.Contains(t, output, "install")
	assert.Contains(t, output, "go-pro")
}
```

- [ ] **Step 7: Run CLI tests**

Run:

```bash
go test ./internal/cli -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/cli/profile_cmd.go internal/cli/app.go internal/cli/manage_cmd.go internal/cli/app_test.go
git commit -m "feat(cli): add profile commands"
```

## Task 9: Run Full Regression And Update Docs

**Files:**
- Modify: `docs/design/skillc-v0-enhance-design.md`
- Modify: `docs/TODO.md`

- [ ] **Step 1: Run full test suite**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 2: Update design document implementation link**

Ensure `docs/design/skillc-v0-enhance-design.md` contains:

```markdown
一期开发计划：`docs/superpowers/plans/2026-06-13-skillc-v0-phase1-profile.md`
```

- [ ] **Step 3: Update TODO plan link**

Ensure `docs/TODO.md` contains:

```markdown
一期计划：`docs/superpowers/plans/2026-06-13-skillc-v0-phase1-profile.md`
```

Do not mark the feature checkbox complete until implementation is finished and verified.

- [ ] **Step 4: Run docs grep self-check**

Run:

```bash
rg -n -- "collection \\(coll\\)|install --collection|Collection string|include_collection" docs/design/skillc-v0-enhance-design.md docs/superpowers/plans/2026-06-13-skillc-v0-phase1-profile.md
```

Expected: matches only in sections describing current-state compatibility or explicitly non-recommended behavior.

- [ ] **Step 5: Commit**

```bash
git add docs/design/skillc-v0-enhance-design.md docs/TODO.md docs/superpowers/plans/2026-06-13-skillc-v0-phase1-profile.md
git commit -m "docs: add skillc v0 phase 1 profile plan"
```

## Self-Review

Spec coverage:

- Profile model and config persistence: Tasks 1-2.
- Profile lock attribution: Task 3.
- Collection moved under source: Task 4.
- Profile create/list/show/from-installed/from-collection: Tasks 5-6.
- Profile dry-run/apply: Task 7.
- Profile CLI and install collection flag removal: Task 8.
- Documentation links and regression: Task 9.

Out of scope for this phase:

- Web management.
- Registry.
- Status command.
- Update check/outdated.
- Install map/version drift.
- Profile export/import.
- Project manifest.
- Dynamic collection references in profile.

What already exists and must be reused:

- Config persistence: `configstore.YAMLStore`.
- Install facts: `lockstore.Store` and existing JSON lock structure.
- Skill/source/collection lookup: `repoindex.Store` and `searchapp` target resolution patterns.
- Current installation state: `listapp.Service`.
- Actual writes to agent skill directories: `installapp.Service`.
- CLI confirmation/output helpers: `confirmPrompt`, `table.New`, existing `getWorkdir/defaultConfigFile`.

Failure modes checked:

- Invalid profile names are rejected before config mutation.
- Missing profile returns `profile not found`.
- Missing source collection returns a user-visible error.
- Missing skill in index becomes an `error` plan item.
- Already installed target becomes `skip`.
- Duplicate profile targets are normalized and sorted.
- Failed install is reported through install result; profileapp does not bypass installapp lock handling.

Placeholder scan:

- No unresolved placeholder markers.
- No open-ended implementation placeholders.
- Each task includes exact files, expected tests, commands, and commit step.

Type consistency:

- `profile.Profile`, `profile.Target`, `profile.ApplyPlan`, and `profile.ApplyPlanItem` are defined in Task 1 and reused consistently.
- `Config.Profiles` is introduced in Task 2 before profile service tasks.
- `lock.Record.Profile` and `installapp.InstallReq.Profile` are introduced in Task 3 before apply execution uses them.
