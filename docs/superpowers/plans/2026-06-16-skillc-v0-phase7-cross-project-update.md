# Skillc v0 Phase 7 Cross-Project Update Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 增加明确的 project registry、project CLI 和跨项目 update plan/run，让用户可以安全查看并一键更新已登记项目中的 skills。

**Architecture:** P7 先把“哪些项目允许被管理”落到本机配置中的 `projects` 列表，再由新的 `projectupdateapp` 只遍历这些已登记项目。CLI 和 Web 都坚持 plan-first：`--check` 或 plan endpoint 只展示跨项目候选，执行入口必须经过 `--yes` 或 `confirm:true`。同时修正 `updateapp` 的 project scope 选择边界，让普通 `skillc update` 只更新当前项目，跨项目更新只能通过 registry allowlist 进入。

**Tech Stack:** Go, `gookit/gcli`, existing `configstore`/`lockstore`/`statusapp`/`updateapp`/`webapp`, embedded static HTML/CSS/JS, Go unit tests with `github.com/gookit/goutil/testutil/assert`, HTTP handler tests with `httptest`, final verification via `go test ./...`.

---

| 修订时间 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-16 | v0.1 | Codex | 基于 Phase 6 完成状态输出 P7 跨项目更新实施计划 |

相关文档：

- 设计文档：`docs/design/skillc-v0-enhance-design.md`
- 参考分析：`docs/design/skillc-reference-projects-analysis.md`
- Phase 6 计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase6-web-current-project-management.md`
- 任务入口：`docs/TODO.md`

## Phase 7 Scope

本期做：

- 新增 project registry：
  - 在 config 中保存本机已登记项目：`id`、`name`、`path`、`description`。
  - 支持 `skillc project list/add/remove/import-lock`。
  - `project import-lock` 从已有 lock 的 project scope keys 导入仍存在的项目路径，便于迁移现有安装记录。
- 修正 project scope update 边界：
  - 普通 `skillc update` 只更新当前项目。
  - 跨项目更新必须显式走 `--all-projects`，且只作用于 registered projects。
- 新增跨项目 update app service：
  - 生成跨项目 update plan。
  - 按项目执行 update，并保留 per-project result / error。
  - 执行时只更新 plan 中的 `outdated` / `missing` 候选，不重复更新 already installed 项。
- 新增 CLI：
  - `skillc update --all-projects --check`
  - `skillc update --all-projects [--projects <id,id>] [--target <skill>] --yes`
  - 没有 `--yes` 时必须打印 plan 并要求用户确认。
- 新增 Web：
  - `GET /api/projects`
  - `POST /api/update/all/plan`
  - `POST /api/update/all/run`
  - Projects 页面展示 registered projects、跨项目 update plan 和执行结果。
  - run endpoint 必须要求 `confirm:true`，并写入 history action `update.all_projects`。
- 更新 README、README.zh-CN、docs/TODO.md、设计文档和本计划验证记录。

本期不做：

- 不做远程 Registry 发现能力和 `skillc registry ...`。
- 不做 source ID 命名清理、`source info <id>` 或自定义 source `--id/--name`。
- 不做 Git commit / local checksum 的精确 drift 判断。
- 不做 `skillc.profile.yaml` 项目 manifest、profile export/import 增强。
- 不做远程 Web、登录、token、多用户权限、安全审计。
- 不做后台 job 队列或定时更新。
- 不做跨项目 uninstall。

## User-Facing Behavior

### Project Registry

用户显式登记可被管理的项目：

```bash
skillc project add . --id skillc --name "Skillc"
skillc project list
skillc project remove skillc
skillc project import-lock
```

`project list` 输出：

```text
Project List
ID      Name    Path
skillc  Skillc  D:\work\aidev\lite-tools\skillc
```

`project import-lock` 只导入 lock 中仍存在的 project scope path，不导入 `__global__`。

### Current Project Update Boundary

普通更新继续保持当前项目语义：

```bash
skillc update --check
skillc update --yes
```

若 lock 文件中有多个项目记录，普通 `skillc update` 只处理当前工作目录对应的 project key，不再隐式遍历所有项目。

### Cross-Project Update CLI

查看所有已登记项目的 update plan：

```bash
skillc update --all-projects --check
```

输出按 project 分组：

```text
Cross-Project Update Check
Project  Path                Skill     Agent      Current  Latest  Status
skillc   ...\skillc          go-pro    universal  1.0.0    2.0.0  outdated
demo     ...\demo            review    universal           1.0.0  missing
```

执行全部登记项目的候选更新：

```bash
skillc update --all-projects --yes
```

限定项目：

```bash
skillc update --all-projects --projects skillc,demo --target go-pro --yes
```

没有 `--yes` 时必须打印 plan 并提示：

```text
Run cross-project update for 2 project(s) and 3 candidate(s)? [y/N]
```

### Web Cross-Project Update

Projects 页面增加 registered projects 表格，并提供：

- `Plan all-projects update`
- project checkbox selection
- optional target input
- `Run all-projects update`

Web 请求示例：

```http
POST /api/update/all/plan?agent=universal&scope=project
Content-Type: application/json

{"project_ids":["skillc","demo"],"target":"go-pro"}
```

执行请求必须确认：

```json
{
  "confirm": true,
  "project_ids": ["skillc", "demo"],
  "target": "go-pro"
}
```

## Data Model

新增 domain package：

```go
package project

type Project struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name,omitempty" json:"name,omitempty"`
	Path        string `yaml:"path" json:"path"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}
```

Config 增加：

```go
Projects []project.Project `yaml:"projects,omitempty"`
```

YAML 示例：

```yaml
projects:
  - id: skillc
    name: Skillc
    path: D:\work\aidev\lite-tools\skillc
  - id: lite-tools
    name: Lite Tools
    path: D:\work\aidev\lite-tools
```

跨项目执行约束：

- Project registry 中保存清理后的绝对项目路径，避免从不同目录运行 CLI/Web 时相对路径含义变化。
- `projectupdateapp` 读取 registry 时使用当前 `configFile/baseDir`；对每个项目执行 `statusapp` / `updateapp` 时，`WorkDir` 和 app service runtime base 都使用该项目路径，让 project scope 的 `.agents`、`.codex` 等相对 `project_dir` 按目标项目解析。
- P7 不引入数据库或后台队列；项目 allowlist 仍是 config YAML 中的本机数据。

## File Structure

新增文件：

- `internal/domain/project/model.go`
  - 定义 `Project`、ID/path 规范化和基础校验。
- `internal/domain/project/model_test.go`
  - 覆盖 project ID 规范化、路径必填、名称 fallback。
- `internal/app/projectapp/service.go`
  - 提供 project registry 的 list/add/remove/import-lock。
- `internal/app/projectapp/service_test.go`
  - 覆盖 add/list/remove/import-lock、重复 ID/path、缺失路径跳过。
- `internal/app/projectupdateapp/service.go`
  - 提供跨项目 update plan/run。
- `internal/app/projectupdateapp/service_test.go`
  - 覆盖 registered project plan、selection、run per project、partial failure。
- `internal/cli/project_cmd.go`
  - 新增 `skillc project ...` 命令。

修改文件：

- `internal/domain/config/model.go`
  - 增加 `Projects []project.Project`。
- `internal/domain/config/defaults.go`
  - 默认 `Projects` 为空列表。
- `internal/infra/configstore/yaml_store.go`
  - 读写 `projects`，并对 project path 做 expand/compact。
- `internal/infra/configstore/yaml_store_test.go`
  - 覆盖 projects YAML round-trip 和 portable relative path。
- `internal/app/updateapp/service.go`
  - 增加 `ProjectPaths []string` allowlist，修正普通 project update 只处理当前项目。
- `internal/app/updateapp/service_test.go`
  - 覆盖当前项目默认边界和 explicit all-project paths。
- `internal/cli/app.go`
  - 注册 `project` 命令。
- `internal/cli/manage_cmd.go`
  - `update` 增加 `--all-projects`、`--projects`、`--yes`，并接入 `projectupdateapp`。
- `internal/cli/app_test.go`
  - 覆盖 project CLI 和 `update --all-projects` 输出/确认。
- `internal/app/webapp/manager.go`
  - 增加 Projects 查询和跨项目 update helper。
- `internal/app/webapp/manager_actions.go`
  - 增加 cross-project update action result JSON model。
- `internal/app/webapp/manager_server.go`
  - 增加 `/api/projects`、`/api/update/all/plan`、`/api/update/all/run`。
- `internal/app/webapp/manager_server_test.go`
  - 覆盖新 endpoints、confirm guard、history action。
- `internal/app/webapp/manager_static.go`
  - Projects 视图增加 registered projects 和 all-project update controls。
- `README.md`
  - 更新 project registry / all-project update 说明。
- `README.zh-CN.md`
  - 更新中文说明。
- `docs/TODO.md`
  - 增加 P7 计划链接和状态。
- `docs/design/skillc-v0-enhance-design.md`
  - 增加 P7 计划链接、修订记录和阶段说明。
- `docs/superpowers/plans/2026-06-16-skillc-v0-phase7-cross-project-update.md`
  - 实施中持续更新 checkbox 和验证记录。

## API Model

新增 Web routes：

```text
GET  /api/projects
POST /api/update/all/plan
POST /api/update/all/run
```

新增 request / response 模型建议：

```go
package webapp

type updateAllProjectsReq struct {
	Confirm    bool     `json:"confirm,omitempty"`
	Target     string   `json:"target,omitempty"`
	ProjectIDs []string `json:"project_ids,omitempty"`
}

type updateAllProjectsActionResult struct {
	Error   string                `json:"error,omitempty"`
	Plan    projectupdateapp.Plan `json:"plan"`
	Results []projectUpdateResult `json:"results,omitempty"`
}

type projectUpdateResult struct {
	ProjectID     string                  `json:"project_id"`
	Path          string                  `json:"path"`
	Updated       []actionRuntimeRecord   `json:"updated,omitempty"`
	Skipped       []actionErrorItem       `json:"skipped,omitempty"`
	Failed        []actionErrorItem       `json:"failed,omitempty"`
	SyncFailed    []actionSourceErrorItem `json:"sync_failed,omitempty"`
	CleanupFailed []actionErrorItem       `json:"cleanup_failed,omitempty"`
	Error         string                  `json:"error,omitempty"`
}
```

## Task 1: Add Project Domain and Config Persistence

**Files:**
- Create: `internal/domain/project/model.go`
- Create: `internal/domain/project/model_test.go`
- Modify: `internal/domain/config/model.go`
- Modify: `internal/domain/config/defaults.go`
- Modify: `internal/infra/configstore/yaml_store.go`
- Modify: `internal/infra/configstore/yaml_store_test.go`

- [x] **Step 1: Write failing project domain tests**

Create `internal/domain/project/model_test.go`:

```go
package project

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestNewProjectNormalizesIDAndName(t *testing.T) {
	got, err := New("", "", filepath.Join("work", "My App"))

	assert.NoErr(t, err)
	assert.Eq(t, "my-app", got.ID)
	assert.Eq(t, "My App", got.Name)
	assert.Contains(t, got.Path, filepath.Join("work", "My App"))
}

func TestNewProjectUsesExplicitIDAndName(t *testing.T) {
	got, err := New("demo_api", "Demo API", filepath.Join("work", "demo"))

	assert.NoErr(t, err)
	assert.Eq(t, "demo_api", got.ID)
	assert.Eq(t, "Demo API", got.Name)
}

func TestNewProjectRejectsEmptyPath(t *testing.T) {
	_, err := New("demo", "", "")

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "project path is required")
}
```

- [x] **Step 2: Run domain tests to verify they fail**

Run:

```bash
go test ./internal/domain/project -v
```

Expected: FAIL because package `internal/domain/project` does not exist.

- [x] **Step 3: Implement project domain model**

Create `internal/domain/project/model.go`:

```go
package project

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

type Project struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name,omitempty" json:"name,omitempty"`
	Path        string `yaml:"path" json:"path"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

func New(id string, name string, path string) (Project, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Project{}, fmt.Errorf("project path is required")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Project{}, err
	}
	cleanPath := filepath.Clean(absPath)
	baseName := filepath.Base(cleanPath)
	if name == "" {
		name = baseName
	}
	if id == "" {
		id = NormalizeID(baseName)
	} else {
		id = NormalizeID(id)
	}
	if id == "" {
		return Project{}, fmt.Errorf("project id is required")
	}
	return Project{ID: id, Name: name, Path: cleanPath}, nil
}

func NormalizeID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case r == '_' || r == '-':
			b.WriteRune(r)
			lastDash = r == '-'
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-_")
}
```

- [x] **Step 4: Add config model fields and failing YAML tests**

Modify `internal/domain/config/model.go`:

```go
import (
	"slices"

	"github.com/inhere/skillc/internal/domain/profile"
	"github.com/inhere/skillc/internal/domain/project"
	domainsource "github.com/inhere/skillc/internal/domain/source"
)
```

Add to `Config`:

```go
Projects []project.Project `yaml:"projects,omitempty"`
```

Modify `internal/domain/config/defaults.go`:

```go
import (
	"github.com/inhere/skillc/internal/domain/project"
	domainsource "github.com/inhere/skillc/internal/domain/source"
)

// ...

Projects: []project.Project{},
```

Add tests to `internal/infra/configstore/yaml_store_test.go`:

```go
func TestYAMLStore_LoadSaveProjects(t *testing.T) {
	baseDir := t.TempDir()
	path := filepath.Join(baseDir, "skillc.yaml")
	projectPath := filepath.Join(baseDir, "demo")
	assert.NoErr(t, os.MkdirAll(projectPath, 0o755))

	data := cfg.DefaultConfig()
	data.Projects = []project.Project{{ID: "demo", Name: "Demo", Path: projectPath}}

	store := NewYAMLStore()
	assert.NoErr(t, store.Save(path, data, baseDir))

	got, err := store.Load(path, baseDir)
	assert.NoErr(t, err)
	assert.Len(t, got.Projects, 1)
	assert.Eq(t, "demo", got.Projects[0].ID)
	assert.Eq(t, projectPath, got.Projects[0].Path)
}

func TestYAMLStore_SaveDefaultConfigOmitsEmptyProjects(t *testing.T) {
	baseDir := t.TempDir()
	path := filepath.Join(baseDir, "skillc.yaml")

	store := NewYAMLStore()
	assert.NoErr(t, store.Save(path, cfg.DefaultConfig(), baseDir))

	content, err := os.ReadFile(path)
	assert.NoErr(t, err)
	assert.NotContains(t, string(content), "projects:")
}
```

Add `github.com/inhere/skillc/internal/domain/project` to the test imports.

- [x] **Step 5: Run config tests to verify they fail**

Run:

```bash
go test ./internal/infra/configstore -run 'TestYAMLStore_(LoadSaveProjects|SaveDefaultConfigOmitsEmptyProjects)' -v
```

Expected: FAIL because `YAMLStore` does not read/write `projects`.

- [x] **Step 6: Implement YAML projects persistence**

Modify `internal/infra/configstore/yaml_store.go` imports:

```go
import (
	"os"
	"path/filepath"

	gkconfig "github.com/gookit/config/v2"
	gkyaml "github.com/gookit/config/v2/yaml"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/profile"
	"github.com/inhere/skillc/internal/domain/project"
	domainsource "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/fsx"
)
```

Add raw record:

```go
type projectRecord struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name,omitempty"`
	Path        string `yaml:"path"`
	Description string `yaml:"description,omitempty"`
}
```

Add to `rawConfig`:

```go
Projects []projectRecord `yaml:"projects,omitempty"`
```

Add to `Save` output:

```go
if len(persisted.Projects) > 0 {
	out["projects"] = toProjectRecords(persisted.Projects)
}
```

Add to `cloneConfig`:

```go
if data.Projects != nil {
	clone.Projects = append([]project.Project(nil), data.Projects...)
}
```

Add to `compactRuntimePaths`:

```go
for i, item := range data.Projects {
	existingRaw := ""
	if hasExisting {
		for _, existingProject := range existing.Projects {
			if existingProject.ID == item.ID {
				existingRaw = existingProject.Path
				break
			}
		}
	}
	item.Path, err = compactPath(item.Path, existingRaw, "", baseDir, hasExisting)
	if err != nil {
		return cfg.Config{}, err
	}
	data.Projects[i] = item
}
```

Add to `expandRuntimePaths`:

```go
for i, item := range data.Projects {
	if item.Path == "" {
		continue
	}
	item.Path, err = fsx.ExpandPath(item.Path, baseDir)
	if err != nil {
		return cfg.Config{}, err
	}
	item.Path = filepath.Clean(item.Path)
	data.Projects[i] = item
}
```

Add helpers:

```go
func toProjectRecords(projects []project.Project) []projectRecord {
	if len(projects) == 0 {
		return nil
	}
	records := make([]projectRecord, 0, len(projects))
	for _, item := range projects {
		records = append(records, projectRecord{
			ID:          item.ID,
			Name:        item.Name,
			Path:        item.Path,
			Description: item.Description,
		})
	}
	return records
}

func fromProjectRecords(records []projectRecord) []project.Project {
	if len(records) == 0 {
		return []project.Project{}
	}
	items := make([]project.Project, 0, len(records))
	for _, record := range records {
		items = append(items, project.Project{
			ID:          record.ID,
			Name:        record.Name,
			Path:        record.Path,
			Description: record.Description,
		})
	}
	return items
}
```

Add to `fromRawConfig`:

```go
Projects: fromProjectRecords(raw.Projects),
```

- [x] **Step 7: Run domain and config tests**

Run:

```bash
go test ./internal/domain/project ./internal/infra/configstore
```

Expected: PASS.

- [x] **Step 8: Commit project config model**

Run:

```bash
git add internal/domain/project internal/domain/config/model.go internal/domain/config/defaults.go internal/infra/configstore/yaml_store.go internal/infra/configstore/yaml_store_test.go
git commit -m "feat(skillc): add project registry config"
```

## Task 2: Add Project Registry App Service

**Files:**
- Create: `internal/app/projectapp/service.go`
- Create: `internal/app/projectapp/service_test.go`

- [x] **Step 1: Write failing projectapp tests**

Create `internal/app/projectapp/service_test.go`:

```go
package projectapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/project"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
)

func TestService_AddListRemoveProject(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	projectDir := filepath.Join(baseDir, "demo")
	assert.NoErr(t, os.MkdirAll(projectDir, 0o755))
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, cfg.DefaultConfig(), baseDir))

	service := NewService(configFile, baseDir)
	added, err := service.Add(AddReq{ID: "demo", Name: "Demo", Path: projectDir})
	assert.NoErr(t, err)
	assert.Eq(t, "demo", added.ID)

	items, err := service.List()
	assert.NoErr(t, err)
	assert.Len(t, items, 1)
	assert.Eq(t, projectDir, items[0].Path)

	assert.NoErr(t, service.Remove("demo"))
	items, err = service.List()
	assert.NoErr(t, err)
	assert.Len(t, items, 0)
}

func TestService_AddRejectsDuplicatePath(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	projectDir := filepath.Join(baseDir, "demo")
	assert.NoErr(t, os.MkdirAll(projectDir, 0o755))
	data := cfg.DefaultConfig()
	data.Projects = []project.Project{{ID: "demo", Path: projectDir}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, data, baseDir))

	_, err := NewService(configFile, baseDir).Add(AddReq{ID: "copy", Path: projectDir})

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "project path already registered")
}

func TestService_AddRejectsDuplicateExplicitID(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	firstDir := filepath.Join(baseDir, "first")
	secondDir := filepath.Join(baseDir, "second")
	assert.NoErr(t, os.MkdirAll(firstDir, 0o755))
	assert.NoErr(t, os.MkdirAll(secondDir, 0o755))
	data := cfg.DefaultConfig()
	data.Projects = []project.Project{{ID: "demo", Path: firstDir}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, data, baseDir))

	_, err := NewService(configFile, baseDir).Add(AddReq{ID: "demo", Path: secondDir})

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "project id already registered")
}

func TestService_ImportFromLockRegistersExistingProjectKeys(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	projectDir := filepath.Join(baseDir, "project-a")
	missingDir := filepath.Join(baseDir, "missing-project")
	assert.NoErr(t, os.MkdirAll(projectDir, 0o755))
	data := cfg.DefaultConfig()
	data.LockFile = lockFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, data, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectDir:                 {{SkillID: "go-pro", Agents: []string{"universal"}}},
		missingDir:                 {{SkillID: "review", Agents: []string{"universal"}}},
		lockpkg.GlobalKey:          {{SkillID: "global", Agents: []string{"universal"}}},
	}))

	result, err := NewService(configFile, baseDir).ImportFromLock()

	assert.NoErr(t, err)
	assert.Len(t, result.Added, 1)
	assert.Eq(t, "project-a", result.Added[0].ID)
	assert.Len(t, result.Skipped, 2)
}
```

- [x] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/app/projectapp -v
```

Expected: FAIL because package `internal/app/projectapp` does not exist.

- [x] **Step 3: Implement projectapp service**

Create `internal/app/projectapp/service.go`:

```go
package projectapp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/inhere/skillc/internal/app/apputil"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/agent"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/project"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
)

type AddReq struct {
	ID          string
	Name        string
	Path        string
	Description string
}

type ImportResult struct {
	Added   []project.Project `json:"added"`
	Skipped []ImportSkip      `json:"skipped,omitempty"`
}

type ImportSkip struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type Service struct {
	configFile string
	baseDir    string
	store      *configstore.YAMLStore
	lockStore  *lockstore.Store
}

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile: configFile,
		baseDir:    baseDir,
		store:      configstore.NewYAMLStore(),
		lockStore:  lockstore.NewStore(),
	}
}

func (s *Service) List() ([]project.Project, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	items := append([]project.Project(nil), data.Projects...)
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (s *Service) Add(req AddReq) (project.Project, error) {
	data, err := s.load()
	if err != nil {
		return project.Project{}, err
	}
	item, err := project.New(req.ID, req.Name, req.Path)
	if err != nil {
		return project.Project{}, err
	}
	item.Description = req.Description
	if err := ensureProjectDir(item.Path); err != nil {
		return project.Project{}, err
	}
	if req.ID != "" && containsID(data.Projects, item.ID) {
		return project.Project{}, fmt.Errorf("project id already registered: %s", item.ID)
	}
	if req.ID == "" {
		item.ID = uniqueID(item.ID, data.Projects)
	}
	for _, current := range data.Projects {
		if filepath.Clean(current.Path) == filepath.Clean(item.Path) {
			return project.Project{}, fmt.Errorf("project path already registered: %s", item.Path)
		}
	}
	data.Projects = append(data.Projects, item)
	if err := s.save(data); err != nil {
		return project.Project{}, err
	}
	return item, nil
}

func (s *Service) Remove(id string) error {
	data, err := s.load()
	if err != nil {
		return err
	}
	out := data.Projects[:0]
	removed := false
	for _, item := range data.Projects {
		if item.ID == id {
			removed = true
			continue
		}
		out = append(out, item)
	}
	if !removed {
		return fmt.Errorf("project not found: %s", id)
	}
	data.Projects = out
	return s.save(data)
}

func (s *Service) ImportFromLock() (ImportResult, error) {
	data, err := s.load()
	if err != nil {
		return ImportResult{}, err
	}
	records, err := s.lockStore.Load(data.LockFile)
	if os.IsNotExist(err) {
		return ImportResult{}, nil
	}
	if err != nil {
		return ImportResult{}, err
	}
	result := ImportResult{}
	keys := make([]string, 0, len(records))
	for key := range records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key == lockpkg.GlobalKey || apputil.ScopeFromKey(key) != agent.ScopeProject {
			result.Skipped = append(result.Skipped, ImportSkip{Path: key, Reason: "not a project scope key"})
			continue
		}
		if _, err := os.Stat(key); err != nil {
			result.Skipped = append(result.Skipped, ImportSkip{Path: key, Reason: "project path does not exist"})
			continue
		}
		if containsPath(data.Projects, key) {
			result.Skipped = append(result.Skipped, ImportSkip{Path: key, Reason: "project already registered"})
			continue
		}
		item, err := project.New("", "", key)
		if err != nil {
			return ImportResult{}, err
		}
		item.ID = uniqueID(item.ID, data.Projects)
		data.Projects = append(data.Projects, item)
		result.Added = append(result.Added, item)
	}
	if len(result.Added) > 0 {
		if err := s.save(data); err != nil {
			return ImportResult{}, err
		}
	}
	return result, nil
}

func (s *Service) load() (cfg.Config, error) {
	return s.store.Load(s.configFile, s.baseDir)
}

func (s *Service) save(data cfg.Config) error {
	return s.store.Save(s.configFile, data, s.baseDir)
}

func ensureProjectDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("project path is not a directory: %s", path)
	}
	return nil
}

func uniqueID(id string, projects []project.Project) string {
	if !containsID(projects, id) {
		return id
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", id, i)
		if !containsID(projects, candidate) {
			return candidate
		}
	}
}

func containsID(projects []project.Project, id string) bool {
	for _, item := range projects {
		if item.ID == id {
			return true
		}
	}
	return false
}

func containsPath(projects []project.Project, path string) bool {
	path = filepath.Clean(path)
	for _, item := range projects {
		if filepath.Clean(item.Path) == path {
			return true
		}
	}
	return false
}
```

- [x] **Step 4: Run projectapp tests**

Run:

```bash
go test ./internal/app/projectapp -v
```

Expected: PASS.

- [x] **Step 5: Commit projectapp service**

Run:

```bash
git add internal/app/projectapp
git commit -m "feat(skillc): add project registry service"
```

## Task 3: Add Project CLI Commands

**Files:**
- Create: `internal/cli/project_cmd.go`
- Modify: `internal/cli/app.go`
- Modify: `internal/cli/app_test.go`

- [x] **Step 1: Write failing CLI tests**

Add tests to `internal/cli/app_test.go`:

```go
func TestProjectCommand_AddListRemove(t *testing.T) {
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "demo")
	assert.NoErr(t, os.MkdirAll(projectDir, 0o755))
	configFile := filepath.Join(baseDir, "skillc.yaml")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, cfg.DefaultConfig(), baseDir))

	addOutput := runAppInDirWithStdout(t, baseDir, []string{"project", "add", projectDir, "--id", "demo", "--name", "Demo"})
	assert.Contains(t, addOutput, "project added: demo")

	listOutput := runAppInDirWithStdout(t, baseDir, []string{"project", "list"})
	assert.Contains(t, listOutput, "demo")
	assert.Contains(t, listOutput, "Demo")

	removeOutput := runAppInDirWithStdout(t, baseDir, []string{"project", "remove", "demo"})
	assert.Contains(t, removeOutput, "project removed: demo")
}

func TestProjectCommand_ImportLock(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	projectDir := filepath.Join(baseDir, "project-a")
	assert.NoErr(t, os.MkdirAll(projectDir, 0o755))
	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectDir: {{SkillID: "go-pro", Agents: []string{"universal"}}},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"project", "import-lock"})

	assert.Contains(t, output, "imported project-a")
}
```

Add `lockpkg` and `lockstore` imports if missing.

- [x] **Step 2: Run CLI tests to verify they fail**

Run:

```bash
go test ./internal/cli -run 'TestProjectCommand' -v
```

Expected: FAIL because `project` command is not registered.

- [x] **Step 3: Implement project CLI**

Create `internal/cli/project_cmd.go`:

```go
package cli

import (
	"fmt"
	"os"

	"github.com/gookit/cliui/show/table"
	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/skillc/internal/app/projectapp"
)

func buildProjectCommand() *gcli.Command {
	cmd := &gcli.Command{
		Name: "project",
		Desc: "Manage registered projects",
	}
	cmd.Add(buildProjectListCommand())
	cmd.Add(buildProjectAddCommand())
	cmd.Add(buildProjectRemoveCommand())
	cmd.Add(buildProjectImportLockCommand())
	return cmd
}

func buildProjectListCommand() *gcli.Command {
	return &gcli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Desc:    "List registered projects",
		Func: func(c *gcli.Command, args []string) error {
			items, err := newProjectService().List()
			if err != nil {
				return err
			}
			tb := table.New("Project List").SetHeads("ID", "Name", "Path")
			for _, item := range items {
				tb.AddRow(item.ID, item.Name, item.Path)
			}
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
		},
	}
}

func buildProjectAddCommand() *gcli.Command {
	var id string
	var name string
	var description string
	return &gcli.Command{
		Name: "add",
		Desc: "Register a project path",
		Config: func(c *gcli.Command) {
			c.AddArg("path", "project path", true)
			c.StrOpt(&id, "id", "", "", "project id")
			c.StrOpt(&name, "name", "", "", "project name")
			c.StrOpt(&description, "description", "", "", "project description")
		},
		Func: func(c *gcli.Command, args []string) error {
			item, err := newProjectService().Add(projectapp.AddReq{
				ID:          id,
				Name:        name,
				Path:        c.Arg("path").String(),
				Description: description,
			})
			if err != nil {
				return err
			}
			ccolor.Infof("project added: %s %s\n", item.ID, item.Path)
			return nil
		},
	}
}

func buildProjectRemoveCommand() *gcli.Command {
	return &gcli.Command{
		Name:    "remove",
		Aliases: []string{"rm"},
		Desc:    "Remove a registered project",
		Config: func(c *gcli.Command) {
			c.AddArg("id", "project id", true)
		},
		Func: func(c *gcli.Command, args []string) error {
			id := c.Arg("id").String()
			if err := newProjectService().Remove(id); err != nil {
				return err
			}
			ccolor.Infof("project removed: %s\n", id)
			return nil
		},
	}
}

func buildProjectImportLockCommand() *gcli.Command {
	return &gcli.Command{
		Name: "import-lock",
		Desc: "Register projects found in the lock file",
		Func: func(c *gcli.Command, args []string) error {
			result, err := newProjectService().ImportFromLock()
			if err != nil {
				return err
			}
			for _, item := range result.Added {
				ccolor.Infof("imported %s %s\n", item.ID, item.Path)
			}
			for _, item := range result.Skipped {
				ccolor.Warnf("skipped %s %s\n", item.Path, item.Reason)
			}
			return nil
		},
	}
}

func newProjectService() *projectapp.Service {
	cwd := getWorkdir()
	return projectapp.NewService(defaultConfigFile(cwd), cwd)
}
```

Modify `internal/cli/app.go`:

```go
app.Add(buildProjectCommand())
```

Place it after `buildProfileCommand()` or before `buildWebCommand()` so concept commands stay grouped.

- [x] **Step 4: Run CLI project tests**

Run:

```bash
go test ./internal/cli -run 'TestProjectCommand' -v
```

Expected: PASS.

- [x] **Step 5: Commit project CLI**

Run:

```bash
git add internal/cli/project_cmd.go internal/cli/app.go internal/cli/app_test.go
git commit -m "feat(skillc): add project cli"
```

## Task 4: Fix UpdateApp Project Scope Selection Boundary

**Files:**
- Modify: `internal/app/updateapp/service.go`
- Modify: `internal/app/updateapp/service_test.go`

- [x] **Step 1: Write failing update boundary tests**

Add tests to `internal/app/updateapp/service_test.go`:

```go
func TestService_RunProjectScopeDefaultsToCurrentWorkDirOnly(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	projectA := filepath.Join(baseDir, "project-a")
	projectB := filepath.Join(baseDir, "project-b")
	assert.NoErr(t, os.MkdirAll(projectA, 0o755))
	assert.NoErr(t, os.MkdirAll(projectB, 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectA: {{SkillID: "go-pro", SourceID: "source-a", Agents: []string{"universal"}}},
		projectB: {{SkillID: "review", SourceID: "source-a", Agents: []string{"universal"}}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "source-a", Version: "2.0.0", Path: createSkillSource(t, baseDir, filepath.Join("source", "go-pro"), "payload.txt", "go")},
		{ID: "review", SourceID: "source-a", Version: "2.0.0", Path: createSkillSource(t, baseDir, filepath.Join("source", "review"), "payload.txt", "review")},
	}))

	service := NewService(configFile, baseDir)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error { return nil }}
	updated := make([]string, 0)
	service.newInstaller = func(path string, _ cfg.Config) reinstallService {
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error) {
			updated = append(updated, scopeKey+"|"+item.ID)
			return installapp.RuntimeRecord{Record: lockpkg.Record{SkillID: item.ID, Version: item.Version}, Agent: agentName, Scope: string(scope), InstalledPath: targetPath}, nil
		}}
	}

	result, err := service.Run(Req{Scope: "project", WorkDir: projectA})

	assert.NoErr(t, err)
	assert.Len(t, result.Updated, 1)
	assert.Eq(t, []string{projectA + "|go-pro"}, updated)
}

func TestService_RunProjectScopeAllUsesExplicitProjectPaths(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	projectA := filepath.Join(baseDir, "project-a")
	projectB := filepath.Join(baseDir, "project-b")
	assert.NoErr(t, os.MkdirAll(projectA, 0o755))
	assert.NoErr(t, os.MkdirAll(projectB, 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectA: {{SkillID: "go-pro", SourceID: "source-a", Agents: []string{"universal"}}},
		projectB: {{SkillID: "review", SourceID: "source-a", Agents: []string{"universal"}}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "source-a", Version: "2.0.0", Path: createSkillSource(t, baseDir, filepath.Join("source", "go-pro-2"), "payload.txt", "go")},
		{ID: "review", SourceID: "source-a", Version: "2.0.0", Path: createSkillSource(t, baseDir, filepath.Join("source", "review-2"), "payload.txt", "review")},
	}))

	service := NewService(configFile, baseDir)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error { return nil }}
	updated := make([]string, 0)
	service.newInstaller = func(path string, _ cfg.Config) reinstallService {
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error) {
			updated = append(updated, scopeKey+"|"+item.ID)
			return installapp.RuntimeRecord{Record: lockpkg.Record{SkillID: item.ID, Version: item.Version}, Agent: agentName, Scope: string(scope), InstalledPath: targetPath}, nil
		}}
	}

	result, err := service.Run(Req{Scope: "project", WorkDir: baseDir, All: true, ProjectPaths: []string{projectB}})

	assert.NoErr(t, err)
	assert.Len(t, result.Updated, 1)
	assert.Eq(t, []string{projectB + "|review"}, updated)
}
```

- [x] **Step 2: Run tests to verify current bug**

Run:

```bash
go test ./internal/app/updateapp -run 'TestService_RunProjectScope(Default|All)' -v
```

Expected: FAIL. The first test updates both project records before the boundary fix, and `ProjectPaths` is not yet defined.

- [x] **Step 3: Add project path allowlist to UpdateReq**

Modify `internal/app/updateapp/service.go`:

```go
type UpdateReq struct {
	Target       string
	Agent        string
	Scope        string
	All          bool
	WorkDir      string
	ProjectPaths []string
}
```

Change `collectSelected` call:

```go
return selectRecords(config, req.WorkDir, records, req.Target, req.Agent, string(scope), req.All, req.ProjectPaths)
```

Replace `selectRecords` with:

```go
func selectRecords(config cfg.Config, workDir string, records lockpkg.File, target string, agentName string, scope string, all bool, projectPaths []string) ([]InstalledItem, []SkippedItem, error) {
	selected := make([]InstalledItem, 0)
	skipped := make([]SkippedItem, 0)
	allowedProjectKeys := allowedProjectScopeKeys(workDir, projectPaths, all)
	for _, scopeKey := range sortedScopeKeys(records) {
		recordScope := scopeFromKey(scopeKey)
		if scope != "" && string(recordScope) != scope {
			continue
		}
		if recordScope == agent.ScopeProject && !allowedProjectKeys[filepath.Clean(scopeKey)] {
			continue
		}
		for _, record := range records[scopeKey] {
			filteredAgents := filterAgents(record.Agents, agentName)
			if len(filteredAgents) == 0 {
				continue
			}
			if target != "" && !matchesRecordTarget(record, target) {
				continue
			}
			if record.Pinned {
				skipped = append(skipped, SkippedItem{SkillID: record.SkillID, Reason: "pinned"})
				continue
			}
			for _, currentAgent := range filteredAgents {
				installedPath, err := resolveInstalledPath(config, workDir, scopeKey, currentAgent, recordScope, record)
				if err != nil {
					return nil, nil, err
				}
				selected = append(selected, InstalledItem{
					Record:        record,
					Agent:         currentAgent,
					Scope:         string(recordScope),
					ScopeKey:      scopeKey,
					InstalledPath: installedPath,
					FromLock:      true,
				})
			}
		}
	}
	if target != "" && len(selected) == 0 && len(skipped) == 0 {
		return nil, nil, fmt.Errorf("skill not found: %s", target)
	}
	return selected, skipped, nil
}

func allowedProjectScopeKeys(workDir string, projectPaths []string, all bool) map[string]bool {
	out := make(map[string]bool)
	if len(projectPaths) > 0 {
		for _, path := range projectPaths {
			out[filepath.Clean(path)] = true
		}
		return out
	}
	if all {
		return nil
	}
	out[filepath.Clean(workDir)] = true
	return out
}
```

Because `nil` map means all project keys, adjust the call site check:

```go
if recordScope == agent.ScopeProject && allowedProjectKeys != nil && !allowedProjectKeys[filepath.Clean(scopeKey)] {
	continue
}
```

Update existing updateapp tests that intentionally expect multi-project behavior by passing `All: true` or explicit `ProjectPaths`. Do not change tests that should represent current-project behavior.

- [x] **Step 4: Run updateapp tests**

Run:

```bash
go test ./internal/app/updateapp -v
```

Expected: PASS.

- [x] **Step 5: Commit update boundary fix**

Run:

```bash
git add internal/app/updateapp/service.go internal/app/updateapp/service_test.go
git commit -m "fix(skillc): scope project updates to current project"
```

## Task 5: Add Cross-Project Update App Service

**Files:**
- Create: `internal/app/projectupdateapp/service.go`
- Create: `internal/app/projectupdateapp/service_test.go`

- [x] **Step 1: Write failing projectupdateapp tests**

Create `internal/app/projectupdateapp/service_test.go`:

```go
package projectupdateapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/project"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

func TestService_PlanUsesRegisteredProjectsOnly(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeProjectUpdateFixture(t, baseDir)

	plan, err := NewService(configFile, baseDir).Plan(Req{Agent: "universal", Scope: "project", Sync: false})

	assert.NoErr(t, err)
	assert.Eq(t, "universal", plan.Agent)
	assert.Eq(t, "project", plan.Scope)
	assert.Len(t, plan.Projects, 2)
	assert.Eq(t, config.Projects[0].ID, plan.Projects[0].ProjectID)
	assert.Len(t, plan.Projects[0].Items, 1)
	assert.Eq(t, "go-pro", plan.Projects[0].Items[0].SkillID)
	assert.Eq(t, "outdated", plan.Projects[0].Items[0].Status)
}

func TestService_PlanFiltersSelectedProjectsAndTarget(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeProjectUpdateFixture(t, baseDir)

	plan, err := NewService(configFile, baseDir).Plan(Req{Agent: "universal", Scope: "project", ProjectIDs: []string{"demo"}, Target: "review", Sync: false})

	assert.NoErr(t, err)
	assert.Len(t, plan.Projects, 1)
	assert.Eq(t, "demo", plan.Projects[0].ProjectID)
	assert.Len(t, plan.Projects[0].Items, 1)
	assert.Eq(t, "review", plan.Projects[0].Items[0].SkillID)
}

func TestService_RunRequiresConfirmation(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeProjectUpdateFixture(t, baseDir)

	_, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", Sync: false})

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "confirmation required")
}

func writeProjectUpdateFixture(t *testing.T, baseDir string) (string, cfg.Config) {
	t.Helper()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	projectA := filepath.Join(baseDir, "skillc")
	projectB := filepath.Join(baseDir, "demo")
	assert.NoErr(t, os.MkdirAll(filepath.Join(projectA, ".agents", "skills", "go-pro"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(projectB, ".agents", "skills", "review"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.Projects = []project.Project{
		{ID: "skillc", Name: "Skillc", Path: projectA},
		{ID: "demo", Name: "Demo", Path: projectB},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectA: {{SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}}},
		projectB: {{SkillID: "review", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Version: "2.0.0", SourceType: sourcepkg.TypeLocal, Path: filepath.Join(baseDir, "source", "go-pro")},
		{ID: "review", SourceID: "gstack", Version: "2.0.0", SourceType: sourcepkg.TypeLocal, Path: filepath.Join(baseDir, "source", "review")},
	}))
	return configFile, config
}
```

- [x] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/app/projectupdateapp -v
```

Expected: FAIL because package `projectupdateapp` does not exist.

- [x] **Step 3: Implement projectupdateapp plan/run**

Create `internal/app/projectupdateapp/service.go`:

```go
package projectupdateapp

import (
	"fmt"
	"os"
	"strings"

	"github.com/inhere/skillc/internal/app/configapp"
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/statusapp"
	"github.com/inhere/skillc/internal/app/updateapp"
	"github.com/inhere/skillc/internal/domain/project"
)

type Req struct {
	Agent      string
	Scope      string
	Target     string
	ProjectIDs []string
	Sync       bool
	Confirm    bool
}

type Plan struct {
	Agent          string        `json:"agent"`
	Scope          string        `json:"scope"`
	Target         string        `json:"target,omitempty"`
	Projects       []ProjectPlan `json:"projects"`
	CandidateCount int           `json:"candidate_count"`
}

type ProjectPlan struct {
	ProjectID string           `json:"project_id"`
	Name      string           `json:"name,omitempty"`
	Path      string           `json:"path"`
	Items     []statusapp.Item `json:"items"`
	Summary   statusapp.Summary `json:"summary"`
	Error     string           `json:"error,omitempty"`
}

type Result struct {
	Plan    Plan            `json:"plan"`
	Results []ProjectResult `json:"results"`
}

type ProjectResult struct {
	ProjectID     string                       `json:"project_id"`
	Path          string                       `json:"path"`
	Updated       []installapp.RuntimeRecord   `json:"updated,omitempty"`
	Skipped       []updateapp.SkippedItem      `json:"skipped,omitempty"`
	Failed        []updateapp.FailedItem       `json:"failed,omitempty"`
	SyncFailed    []updateapp.SourceSyncError  `json:"sync_failed,omitempty"`
	CleanupFailed []updateapp.FailedItem       `json:"cleanup_failed,omitempty"`
	Error         string                       `json:"error,omitempty"`
}

type Service struct {
	configFile    string
	baseDir       string
	configService *configapp.Service
}

func NewService(configFile string, baseDir string) *Service {
	return &Service{
		configFile:    configFile,
		baseDir:       baseDir,
		configService: configapp.NewService(configFile, baseDir),
	}
}

func (s *Service) Plan(req Req) (Plan, error) {
	config, err := s.configService.Show()
	if err != nil {
		return Plan{}, err
	}
	projects, err := selectProjects(config.Projects, req.ProjectIDs)
	if err != nil {
		return Plan{}, err
	}
	agentName := defaultString(req.Agent, "universal")
	scope := defaultString(req.Scope, "project")
	plan := Plan{Agent: agentName, Scope: scope, Target: req.Target}
	for _, item := range projects {
		projectPlan := ProjectPlan{ProjectID: item.ID, Name: item.Name, Path: item.Path}
		if _, err := os.Stat(item.Path); err != nil {
			projectPlan.Error = err.Error()
			plan.Projects = append(plan.Projects, projectPlan)
			continue
		}
		statusResult, err := statusapp.NewService(s.configFile, item.Path).Run(statusapp.Req{
			Agent:   agentName,
			Scope:   scope,
			WorkDir: item.Path,
			Sync:    req.Sync,
		})
		if err != nil {
			projectPlan.Error = err.Error()
			plan.Projects = append(plan.Projects, projectPlan)
			continue
		}
		projectPlan.Summary = statusResult.Summary
		for _, statusItem := range statusResult.Items {
			if req.Target != "" && !matchesStatusTarget(statusItem, req.Target) {
				continue
			}
			if statusItem.Status != statusapp.StatusOutdated && statusItem.Status != statusapp.StatusMissing {
				continue
			}
			projectPlan.Items = append(projectPlan.Items, statusItem)
		}
		plan.CandidateCount += len(projectPlan.Items)
		plan.Projects = append(plan.Projects, projectPlan)
	}
	return plan, nil
}

func (s *Service) Run(req Req) (Result, error) {
	if !req.Confirm {
		return Result{}, fmt.Errorf("confirmation required")
	}
	plan, err := s.Plan(req)
	if err != nil {
		return Result{}, err
	}
	result := Result{Plan: plan}
	for _, projectPlan := range plan.Projects {
		projectResult := ProjectResult{ProjectID: projectPlan.ProjectID, Path: projectPlan.Path}
		if projectPlan.Error != "" {
			projectResult.Error = projectPlan.Error
			result.Results = append(result.Results, projectResult)
			continue
		}
		for _, item := range projectPlan.Items {
			updateResult, err := updateapp.NewService(s.configFile, projectPlan.Path).Run(updateapp.Req{
				Target:       updateTarget(item),
				Agent:        plan.Agent,
				Scope:        plan.Scope,
				WorkDir:      projectPlan.Path,
				ProjectPaths: []string{projectPlan.Path},
			})
			projectResult.Updated = append(projectResult.Updated, updateResult.Updated...)
			projectResult.Skipped = append(projectResult.Skipped, updateResult.Skipped...)
			projectResult.Failed = append(projectResult.Failed, updateResult.Failed...)
			projectResult.SyncFailed = append(projectResult.SyncFailed, updateResult.SyncFailed...)
			projectResult.CleanupFailed = append(projectResult.CleanupFailed, updateResult.CleanupFailed...)
			if err != nil && projectResult.Error == "" {
				projectResult.Error = err.Error()
			}
		}
		result.Results = append(result.Results, projectResult)
	}
	return result, nil
}

func selectProjects(projects []project.Project, ids []string) ([]project.Project, error) {
	if len(ids) == 0 {
		return append([]project.Project(nil), projects...), nil
	}
	byID := make(map[string]project.Project, len(projects))
	for _, item := range projects {
		byID[item.ID] = item
	}
	out := make([]project.Project, 0, len(ids))
	for _, id := range ids {
		item, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("project not found: %s", id)
		}
		out = append(out, item)
	}
	return out, nil
}

func matchesStatusTarget(item statusapp.Item, target string) bool {
	return item.SkillID == target || item.QualifiedName == target || item.SourceQualifiedName == target
}

func updateTarget(item statusapp.Item) string {
	if item.SourceQualifiedName != "" {
		return item.SourceQualifiedName
	}
	if item.QualifiedName != "" {
		return item.QualifiedName
	}
	return item.SkillID
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
```

- [x] **Step 4: Add focused run test for confirmed execution**

Append to `internal/app/projectupdateapp/service_test.go`:

```go
func TestService_RunExecutesConfirmedCandidateUpdates(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeProjectUpdateFixture(t, baseDir)

	result, err := NewService(configFile, baseDir).Run(Req{
		Agent:      "universal",
		Scope:      "project",
		ProjectIDs: []string{"skillc"},
		Confirm:    true,
		Sync:       false,
	})

	assert.NoErr(t, err)
	assert.Len(t, result.Results, 1)
	assert.Eq(t, "skillc", result.Results[0].ProjectID)
	assert.Len(t, result.Results[0].Updated, 1)
}
```

For this run test, update `writeProjectUpdateFixture` to create source payload directories matching the index paths:

```go
assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, "source", "go-pro"), 0o755))
assert.NoErr(t, os.WriteFile(filepath.Join(baseDir, "source", "go-pro", "SKILL.md"), []byte("# Go Pro\n"), 0o644))
assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, "source", "review"), 0o755))
assert.NoErr(t, os.WriteFile(filepath.Join(baseDir, "source", "review", "SKILL.md"), []byte("# Review\n"), 0o644))
```

- [x] **Step 5: Run projectupdateapp tests**

Run:

```bash
go test ./internal/app/projectupdateapp -v
```

Expected: PASS.

- [x] **Step 6: Commit cross-project update app service**

Run:

```bash
git add internal/app/projectupdateapp
git commit -m "feat(skillc): add cross-project update service"
```

## Task 6: Add CLI `update --all-projects`

**Files:**
- Modify: `internal/cli/manage_cmd.go`
- Modify: `internal/cli/app_test.go`

- [x] **Step 1: Write failing CLI tests**

Add helper interface and tests to `internal/cli/app_test.go` after existing update command tests:

```go
type projectUpdateRunnerStub struct {
	planFn func(projectupdateapp.Req) (projectupdateapp.Plan, error)
	runFn  func(projectupdateapp.Req) (projectupdateapp.Result, error)
}

func (s projectUpdateRunnerStub) Plan(req projectupdateapp.Req) (projectupdateapp.Plan, error) {
	return s.planFn(req)
}

func (s projectUpdateRunnerStub) Run(req projectupdateapp.Req) (projectupdateapp.Result, error) {
	return s.runFn(req)
}

func TestUpdateCommand_AllProjectsCheckPrintsPlan(t *testing.T) {
	baseDir := t.TempDir()
	assert.NoErr(t, configstore.NewYAMLStore().Save(filepath.Join(baseDir, "skillc.yaml"), cfg.DefaultConfig(), baseDir))
	prevFactory := newProjectUpdateService
	newProjectUpdateService = func(configFile string, baseDir string) projectUpdateRunner {
		return projectUpdateRunnerStub{
			planFn: func(req projectupdateapp.Req) (projectupdateapp.Plan, error) {
				assert.Eq(t, "go-pro", req.Target)
				return projectupdateapp.Plan{
					Agent: "universal", Scope: "project", Target: "go-pro", CandidateCount: 1,
					Projects: []projectupdateapp.ProjectPlan{{
						ProjectID: "skillc",
						Path:      baseDir,
						Items:     []statusapp.Item{{SkillID: "go-pro", Agent: "universal", Status: statusapp.StatusOutdated, CurrentVersion: "1.0.0", LatestVersion: "2.0.0"}},
					}},
				}, nil
			},
		}
	}
	defer func() { newProjectUpdateService = prevFactory }()

	output := runAppInDirWithStdout(t, baseDir, []string{"update", "--all-projects", "--check", "--target", "go-pro"})

	assert.Contains(t, output, "Cross-Project Update Check")
	assert.Contains(t, output, "skillc")
	assert.Contains(t, output, "go-pro")
	assert.Contains(t, output, "2.0.0")
}

func TestUpdateCommand_AllProjectsRequiresConfirmation(t *testing.T) {
	baseDir := t.TempDir()
	assert.NoErr(t, configstore.NewYAMLStore().Save(filepath.Join(baseDir, "skillc.yaml"), cfg.DefaultConfig(), baseDir))
	runCalled := false
	prevFactory := newProjectUpdateService
	newProjectUpdateService = func(configFile string, baseDir string) projectUpdateRunner {
		return projectUpdateRunnerStub{
			planFn: func(req projectupdateapp.Req) (projectupdateapp.Plan, error) {
				return projectupdateapp.Plan{Agent: "universal", Scope: "project", CandidateCount: 1}, nil
			},
			runFn: func(req projectupdateapp.Req) (projectupdateapp.Result, error) {
				runCalled = true
				return projectupdateapp.Result{}, nil
			},
		}
	}
	defer func() { newProjectUpdateService = prevFactory }()

	output := runAppInDirWithInput(t, baseDir, []string{"update", "--all-projects"}, "n\n")

	assert.Contains(t, output, "cross-project update cancelled")
	assert.Eq(t, false, runCalled)
}

func TestUpdateCommand_AllProjectsYesRuns(t *testing.T) {
	baseDir := t.TempDir()
	assert.NoErr(t, configstore.NewYAMLStore().Save(filepath.Join(baseDir, "skillc.yaml"), cfg.DefaultConfig(), baseDir))
	runCalled := false
	prevFactory := newProjectUpdateService
	newProjectUpdateService = func(configFile string, baseDir string) projectUpdateRunner {
		return projectUpdateRunnerStub{
			planFn: func(req projectupdateapp.Req) (projectupdateapp.Plan, error) {
				return projectupdateapp.Plan{Agent: "universal", Scope: "project", CandidateCount: 1}, nil
			},
			runFn: func(req projectupdateapp.Req) (projectupdateapp.Result, error) {
				runCalled = true
				assert.Eq(t, true, req.Confirm)
				return projectupdateapp.Result{
					Plan: projectupdateapp.Plan{Agent: "universal", Scope: "project", CandidateCount: 1},
					Results: []projectupdateapp.ProjectResult{{
						ProjectID: "skillc",
						Path:      baseDir,
						Updated:   []installapp.RuntimeRecord{{Record: lockpkg.Record{SkillID: "go-pro", Version: "2.0.0"}, Agent: "universal", Scope: "project"}},
					}},
				}, nil
			},
		}
	}
	defer func() { newProjectUpdateService = prevFactory }()

	output := runAppInDirWithStdout(t, baseDir, []string{"update", "--all-projects", "--yes"})

	assert.Eq(t, true, runCalled)
	assert.Contains(t, output, "updated skillc go-pro")
}
```

Add imports: `projectupdateapp` and `statusapp` if missing.

- [x] **Step 2: Run CLI tests to verify they fail**

Run:

```bash
go test ./internal/cli -run 'TestUpdateCommand_AllProjects' -v
```

Expected: FAIL because `newProjectUpdateService`, `projectUpdateRunner`, and flags are missing.

- [x] **Step 3: Add all-projects runner wiring and flags**

Modify `internal/cli/manage_cmd.go` imports:

```go
	"github.com/inhere/skillc/internal/app/projectupdateapp"
```

Add after `newUpdateService`:

```go
type projectUpdateRunner interface {
	Plan(projectupdateapp.Req) (projectupdateapp.Plan, error)
	Run(projectupdateapp.Req) (projectupdateapp.Result, error)
}

var newProjectUpdateService = func(configFile string, baseDir string) projectUpdateRunner {
	return projectupdateapp.NewService(configFile, baseDir)
}
```

Modify `buildUpdateCommand` local vars:

```go
var allProjects bool
var projectsRaw string
```

Add flags:

```go
c.BoolOpt(&opts.Yes, "yes", "y", false, "skip confirmation prompt for cross-project update")
c.BoolOpt(&allProjects, "all-projects", "", false, "update registered projects")
c.StrOpt(&projectsRaw, "projects", "", "", "comma-separated project ids for --all-projects")
```

At the beginning of the command `Func`, after `validateUpdateMode`, add:

```go
if allProjects {
	req := projectupdateapp.Req{
		Agent:      opts.Agent,
		Scope:      opts.Scope,
		Target:     target,
		ProjectIDs: splitProjectIDs(projectsRaw),
		Sync:       true,
	}
	service := newProjectUpdateService(defaultConfigFile(cwd), cwd)
	plan, err := service.Plan(req)
	if err != nil {
		return err
	}
	if checkOnly {
		return printCrossProjectUpdatePlan(plan)
	}
	if err := printCrossProjectUpdatePlan(plan); err != nil {
		return err
	}
	if plan.CandidateCount == 0 {
		ccolor.Successln("no cross-project update candidates")
		return nil
	}
	if !opts.Yes {
		confirmed, err := confirmPrompt(os.Stdin, os.Stdout, fmt.Sprintf("Run cross-project update for %d project(s) and %d candidate(s)?", len(plan.Projects), plan.CandidateCount))
		if err != nil {
			return err
		}
		if !confirmed {
			ccolor.Warnln("cross-project update cancelled")
			return nil
		}
	}
	req.Confirm = true
	result, err := service.Run(req)
	if err != nil {
		return err
	}
	return printCrossProjectUpdateResult(result)
}
```

Update `validateUpdateMode` signature:

```go
func validateUpdateMode(checkOnly bool, interactive bool, allProjects bool) error {
	if checkOnly && interactive {
		return fmt.Errorf("--check and --interactive are mutually exclusive")
	}
	if allProjects && interactive {
		return fmt.Errorf("--all-projects and --interactive are mutually exclusive")
	}
	return nil
}
```

Change call site:

```go
if err := validateUpdateMode(checkOnly, interactive, allProjects); err != nil {
	return err
}
```

Add helpers:

```go
func splitProjectIDs(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func printCrossProjectUpdatePlan(plan projectupdateapp.Plan) error {
	tb := table.New("Cross-Project Update Check").SetHeads("Project", "Path", "Skill", "Agent", "Current", "Latest", "Status")
	for _, projectPlan := range plan.Projects {
		if projectPlan.Error != "" {
			tb.AddRow(projectPlan.ProjectID, projectPlan.Path, "", "", "", "", "error: "+projectPlan.Error)
			continue
		}
		for _, item := range projectPlan.Items {
			tb.AddRow(projectPlan.ProjectID, projectPlan.Path, item.SkillID, item.Agent, item.CurrentVersion, item.LatestVersion, item.Status)
		}
	}
	if _, err := fmt.Fprint(os.Stdout, tb.Render()); err != nil {
		return err
	}
	ccolor.Infof("cross-project candidates: %d\n", plan.CandidateCount)
	return nil
}

func printCrossProjectUpdateResult(result projectupdateapp.Result) error {
	for _, projectResult := range result.Results {
		if projectResult.Error != "" {
			ccolor.Errorf("project update failed %s %s\n", projectResult.ProjectID, projectResult.Error)
		}
		for _, record := range projectResult.Updated {
			ccolor.Infof("updated %s %s %s\n", projectResult.ProjectID, record.SkillID, record.Version)
		}
		for _, failed := range projectResult.Failed {
			ccolor.Errorf("update failed %s %s %s\n", projectResult.ProjectID, failed.SkillID, failed.Reason)
		}
	}
	return nil
}
```

- [x] **Step 4: Run CLI all-project tests**

Run:

```bash
go test ./internal/cli -run 'TestUpdateCommand_AllProjects' -v
```

Expected: PASS.

- [x] **Step 5: Run focused CLI update tests**

Run:

```bash
go test ./internal/cli -run 'TestUpdateCommand|TestProjectCommand' -v
```

Expected: PASS.

- [x] **Step 6: Commit all-project CLI**

Run:

```bash
git add internal/cli/manage_cmd.go internal/cli/app_test.go
git commit -m "feat(skillc): add all-projects update cli"
```

## Task 7: Add Web Projects and Cross-Project Update Endpoints

**Files:**
- Modify: `internal/app/webapp/manager.go`
- Modify: `internal/app/webapp/manager_actions.go`
- Modify: `internal/app/webapp/manager_server.go`
- Modify: `internal/app/webapp/manager_server_test.go`
- Modify: `internal/app/webapp/manager_static.go`

- [x] **Step 1: Write failing Web API tests**

Add tests to `internal/app/webapp/manager_server_test.go`:

```go
func TestManagerServerProjectsEndpoint(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeWebManagerFixture(t, baseDir)
	config.Projects = []project.Project{{ID: "skillc", Name: "Skillc", Path: baseDir}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequest(server, http.MethodGet, "/api/projects")

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"id":"skillc"`)
	assert.Contains(t, rec.Body.String(), `"name":"Skillc"`)
}

func TestManagerServerUpdateAllPlanEndpoint(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeWebManagerFixture(t, baseDir)
	config.Projects = []project.Project{{ID: "skillc", Name: "Skillc", Path: baseDir}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequestWithBody(server, http.MethodPost, "/api/update/all/plan?agent=universal&scope=project", strings.NewReader(`{"project_ids":["skillc"],"target":"go-pro"}`))

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"candidate_count":1`)
	assert.Contains(t, rec.Body.String(), `"project_id":"skillc"`)
}

func TestManagerServerUpdateAllRunRequiresConfirmation(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeWebManagerFixture(t, baseDir)
	config.Projects = []project.Project{{ID: "skillc", Name: "Skillc", Path: baseDir}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequestWithBody(server, http.MethodPost, "/api/update/all/run", strings.NewReader(`{"project_ids":["skillc"]}`))

	assert.Eq(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), `"confirmation required"`)
}
```

Add `project` import.

- [x] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/app/webapp -run 'TestManagerServer(Projects|UpdateAll)' -v
```

Expected: FAIL because routes and manager methods do not exist.

- [x] **Step 3: Add manager methods and action models**

Modify `internal/app/webapp/manager.go` imports:

```go
	"github.com/inhere/skillc/internal/app/projectapp"
	"github.com/inhere/skillc/internal/app/projectupdateapp"
	projectpkg "github.com/inhere/skillc/internal/domain/project"
```

Add methods:

```go
func (m *Manager) Projects() ([]projectpkg.Project, error) {
	return projectapp.NewService(m.configFile, m.baseDir).List()
}

func (m *Manager) PlanAllProjectsUpdate(req WebUpdateAllReq) (projectupdateapp.Plan, error) {
	return projectupdateapp.NewService(m.configFile, m.baseDir).Plan(projectupdateapp.Req{
		Agent:      req.Agent,
		Scope:      req.Scope,
		Target:     req.Target,
		ProjectIDs: req.ProjectIDs,
		Sync:       true,
	})
}

func (m *Manager) RunAllProjectsUpdate(req WebUpdateAllReq) (updateAllProjectsActionResult, error) {
	result, err := projectupdateapp.NewService(m.configFile, m.baseDir).Run(projectupdateapp.Req{
		Agent:      req.Agent,
		Scope:      req.Scope,
		Target:     req.Target,
		ProjectIDs: req.ProjectIDs,
		Sync:       true,
		Confirm:    true,
	})
	out := toUpdateAllProjectsActionResult(result)
	if err != nil {
		if out.Plan.Agent != "" || len(out.Results) > 0 {
			out.Error = err.Error()
			return out, nil
		}
		return out, err
	}
	return out, nil
}
```

Modify `internal/app/webapp/manager_actions.go`:

```go
type WebUpdateAllReq struct {
	ManagerReq
	Target     string
	ProjectIDs []string
}

type updateAllProjectsReq struct {
	Confirm    bool     `json:"confirm,omitempty"`
	Target     string   `json:"target,omitempty"`
	ProjectIDs []string `json:"project_ids,omitempty"`
}

type updateAllProjectsActionResult struct {
	Error   string                `json:"error,omitempty"`
	Plan    projectupdateapp.Plan `json:"plan"`
	Results []projectUpdateResult `json:"results,omitempty"`
}

type projectUpdateResult struct {
	ProjectID string                `json:"project_id"`
	Path      string                `json:"path"`
	Updated   []actionRuntimeRecord `json:"updated,omitempty"`
	Skipped   []actionErrorItem     `json:"skipped,omitempty"`
	Failed    []actionErrorItem     `json:"failed,omitempty"`
	Error     string                `json:"error,omitempty"`
}

func toUpdateAllProjectsActionResult(result projectupdateapp.Result) updateAllProjectsActionResult {
	out := updateAllProjectsActionResult{Plan: result.Plan}
	for _, item := range result.Results {
		converted := projectUpdateResult{ProjectID: item.ProjectID, Path: item.Path, Error: item.Error}
		converted.Updated = append(converted.Updated, runtimeRecords(item.Updated)...)
		converted.Skipped = append(converted.Skipped, skippedErrors(item.Skipped)...)
		converted.Failed = append(converted.Failed, failedErrors(item.Failed)...)
		converted.SyncFailed = append(converted.SyncFailed, sourceSyncErrors(item.SyncFailed)...)
		converted.CleanupFailed = append(converted.CleanupFailed, failedErrors(item.CleanupFailed)...)
		out.Results = append(out.Results, converted)
	}
	return out
}
```

Add `projectupdateapp` import to `manager_actions.go`.

- [x] **Step 4: Add routes and confirm guard**

Modify `ManagerServer.Handler()`:

```go
mux.HandleFunc("/api/projects", s.handleProjects)
mux.HandleFunc("/api/update/all/plan", s.handleUpdateAllPlan)
mux.HandleFunc("/api/update/all/run", s.handleUpdateAllRun)
```

Add handlers:

```go
func (s *ManagerServer) handleProjects(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	result, err := s.manager.Projects()
	writeResult(w, result, err)
}

func (s *ManagerServer) handleUpdateAllPlan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	body, ok := readJSONReq[updateAllProjectsReq](w, r)
	if !ok {
		return
	}
	req := managerReqFromQuery(r)
	result, err := s.manager.PlanAllProjectsUpdate(WebUpdateAllReq{ManagerReq: req, Target: body.Target, ProjectIDs: body.ProjectIDs})
	writeResult(w, result, err)
}

func (s *ManagerServer) handleUpdateAllRun(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	body, ok := requireConfirmedUpdateAllReq(w, r)
	if !ok {
		return
	}
	req := managerReqFromQuery(r)
	result, err := s.manager.RunAllProjectsUpdate(WebUpdateAllReq{ManagerReq: req, Target: body.Target, ProjectIDs: body.ProjectIDs})
	s.recordHistory(r, "update.all_projects", WebUpdateAllReq{ManagerReq: req, Target: body.Target, ProjectIDs: body.ProjectIDs}, result, err)
	writeResult(w, result, err)
}

func requireConfirmedUpdateAllReq(w http.ResponseWriter, r *http.Request) (updateAllProjectsReq, bool) {
	req, ok := readJSONReq[updateAllProjectsReq](w, r)
	if !ok {
		return updateAllProjectsReq{}, false
	}
	if !req.Confirm {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "confirmation required"})
		return updateAllProjectsReq{}, false
	}
	return req, true
}
```

Update `resultErrorMessage`:

```go
case updateAllProjectsActionResult:
	return item.Error
```

- [x] **Step 5: Run Web API tests**

Run:

```bash
go test ./internal/app/webapp -run 'TestManagerServer(Projects|UpdateAll)' -v
```

Expected: PASS.

- [x] **Step 6: Add static UI tests and minimal UI**

Append to `manager_server_test.go`:

```go
func TestManagerServerStaticPageContainsAllProjectsUpdateControls(t *testing.T) {
	baseDir := t.TempDir()
	configFile, _ := writeWebManagerFixture(t, baseDir)
	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequest(server, http.MethodGet, "/")
	body := rec.Body.String()

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, body, "/api/projects")
	assert.Contains(t, body, "/api/update/all/plan")
	assert.Contains(t, body, "/api/update/all/run")
	assert.Contains(t, body, `id="run-update-all-btn"`)
}
```

Modify `manager_static.go`:

- Add `/api/projects` to `loadAll()`.
- Add `projects: []` and `updateAllPlan: null` to state.
- In Projects view, add a registered projects table with checkboxes:

```html
<div class="section-head"><h3>Registered Projects</h3><span class="hint">Explicit update allowlist</span></div>
<div id="registered-projects-table"></div>
<div class="toolbar">
  <input id="update-all-target-input" placeholder="optional skill target" />
  <button id="plan-update-all-btn">Plan all-projects update</button>
  <button id="run-update-all-btn" class="danger" disabled>Run all-projects update</button>
</div>
<pre id="update-all-plan-output" class="plan-output"></pre>
```

- Render checkboxes:

```js
function renderRegisteredProjects() {
  var rows = state.projects.map(function (item) {
    return '<tr><td><input type="checkbox" class="project-select" value="' + esc(item.id) + '" checked></td><td>' +
      esc(item.id) + '</td><td>' + esc(item.name || '') + '</td><td class="wrap mono">' + esc(item.path || '') + '</td></tr>';
  });
  byId('registered-projects-table').innerHTML = table(['', 'ID', 'Name', 'Path'], rows, 'No registered projects.');
}
```

- Add handlers:

```js
function selectedProjectIDs() {
  return Array.prototype.slice.call(document.querySelectorAll('.project-select:checked')).map(function (el) {
    return el.value;
  });
}

function planUpdateAll() {
  var payload = { project_ids: selectedProjectIDs(), target: byId('update-all-target-input').value.trim() };
  return postJSON('/api/update/all/plan' + managerQuery(), payload).then(function (plan) {
    state.updateAllPlan = plan;
    byId('update-all-plan-output').textContent = JSON.stringify(plan, null, 2);
    byId('run-update-all-btn').disabled = !plan.candidate_count;
  }).catch(showError);
}

function runUpdateAll() {
  if (!state.updateAllPlan) return;
  if (!window.confirm('Run update for selected registered projects?')) return;
  var payload = { confirm: true, project_ids: selectedProjectIDs(), target: byId('update-all-target-input').value.trim() };
  return postJSON('/api/update/all/run' + managerQuery(), payload).then(function (result) {
    byId('update-all-plan-output').textContent = JSON.stringify(result, null, 2);
    return loadAll();
  }).catch(showError);
}
```

- Wire buttons:

```js
byId('plan-update-all-btn').addEventListener('click', planUpdateAll);
byId('run-update-all-btn').addEventListener('click', runUpdateAll);
```

- Call `renderRegisteredProjects()` from `renderAll()`.

- [x] **Step 7: Run Web static tests**

Run:

```bash
go test ./internal/app/webapp -run 'TestManagerServerStaticPageContainsAllProjectsUpdateControls|TestManagerServerIndexPageContainsAppShell' -v
```

Expected: PASS.

- [x] **Step 8: Commit Web cross-project update**

Run:

```bash
git add internal/app/webapp/manager.go internal/app/webapp/manager_actions.go internal/app/webapp/manager_server.go internal/app/webapp/manager_server_test.go internal/app/webapp/manager_static.go
git commit -m "feat(skillc): add web cross-project updates"
```

## Task 8: Documentation and Final Verification

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/TODO.md`
- Modify: `docs/design/skillc-v0-enhance-design.md`
- Modify: `docs/superpowers/plans/2026-06-16-skillc-v0-phase7-cross-project-update.md`

- [ ] **Step 1: Update README files**

Add to `README.zh-CN.md`:

```markdown
### 项目登记与跨项目更新

`skillc project` 用于登记允许被 Web 和 `update --all-projects` 管理的本机项目。跨项目更新只作用于这些 registered projects，不会直接扫描 lock 中的未知项目。推荐先运行 `skillc project add . --id <id>` 或 `skillc project import-lock`，再使用 `skillc update --all-projects --check` 查看计划，确认后用 `skillc update --all-projects --yes` 执行。
```

Add to `README.md`:

```markdown
### Project Registry and Cross-Project Updates

`skillc project` registers local projects that are allowed to be managed by Web and `update --all-projects`. Cross-project updates only operate on registered projects; they do not blindly scan unknown lock entries. Use `skillc project add . --id <id>` or `skillc project import-lock`, inspect with `skillc update --all-projects --check`, then execute with `skillc update --all-projects --yes`.
```

- [ ] **Step 2: Update design and task docs**

Update `docs/design/skillc-v0-enhance-design.md`:

- Add revision row `2026-06-16 | v0.16 | Codex | 增加 Phase 7 跨项目 project registry / update --all-projects 实施计划链接和范围`.
- Add related doc link:

```markdown
- `docs/superpowers/plans/2026-06-16-skillc-v0-phase7-cross-project-update.md`
```

- Add Phase 7 section after Phase 6 status:

```markdown
七期开发计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase7-cross-project-update.md`

七期目标：新增 project registry / project selection / per-project confirmation，并接入 `update --all-projects` 与 Web 跨项目 update plan/run。P7 只处理已登记项目，不直接扫描未知 lock key；Registry 发现、source ID 清理、checksum/Git commit 精确 drift、project manifest 和远程 Web 权限继续后置。
```

Update `docs/TODO.md`:

- Under “支持一键批量更新所有下游项目的 skills 版本”, add:

```markdown
  - 七期计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase7-cross-project-update.md`
```

- Under Web section, add:

```markdown
  - 七期计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase7-cross-project-update.md`
  - 七期目标：补齐 registered projects、跨项目 update plan/run 和 per-project 确认边界。
```

- Under skillc 增强重构, add:

```markdown
七期计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase7-cross-project-update.md`
七期目标：新增 project registry、`project` CLI、`update --all-projects` 和 Web 跨项目更新闭环。
```

- [ ] **Step 3: Run focused tests**

Run:

```bash
go test ./internal/domain/project ./internal/infra/configstore ./internal/app/projectapp ./internal/app/updateapp ./internal/app/projectupdateapp ./internal/cli ./internal/app/webapp
```

Expected: PASS.

- [ ] **Step 4: Run full verification**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 5: Update this plan checkbox statuses and verification record**

Add a verification section at the end of this file:

```markdown
## Verification

- `go test ./internal/domain/project ./internal/infra/configstore ./internal/app/projectapp ./internal/app/updateapp ./internal/app/projectupdateapp ./internal/cli ./internal/app/webapp`
- `go test ./...`
```

Check off completed steps as each task lands.

- [ ] **Step 6: Commit documentation and verification**

Run:

```bash
git add README.md README.zh-CN.md docs/TODO.md docs/design/skillc-v0-enhance-design.md docs/superpowers/plans/2026-06-16-skillc-v0-phase7-cross-project-update.md
git commit -m "docs(skillc): document phase 7 cross-project updates"
```

## Self-Review Checklist

- [ ] Project registry is persisted in config and portable through YAML load/save.
- [ ] `skillc project list/add/remove/import-lock` covers the local registry workflow.
- [ ] Ordinary project-scope `skillc update` only updates the current project.
- [ ] `update --all-projects` only uses registered projects.
- [ ] Cross-project update has a plan-only path and an explicit execution confirmation path.
- [ ] Web run endpoint requires `confirm:true` and records `update.all_projects` history.
- [ ] P7 does not implement remote Registry, source ID cleanup, precise checksum/Git drift, project manifest, or remote Web permissions.
- [ ] `go test ./...` passes before claiming Phase 7 complete.

## Remaining After Phase 7

- Registry discovery model and `skillc registry ...` commands.
- Source UX cleanup: custom `--id/--name`, no forced `local-` / `git-` prefix for new source IDs, `source info <id>`.
- Precise drift detection using Git commit / resolved ref and local checksum.
- Project manifest / `skillc.profile.yaml` export/import flow.
- Remote Web access, multi-user permissions, and security audit model.
