# Skillc v0 Phase 8 Registry, Source UX, and Precise Drift Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 落地 Registry 最小发现闭环、source UX cleanup，以及基于 Git resolved ref / local checksum 的精确 drift 判断。

**Architecture:** P8 拆成三个可独立提交的 slice。先把 source 创建 API 改成支持自定义 `--id/--name` 且新生成 ID 不再强加 `local-` / `git-` 前缀；再新增本机 registry catalog 配置、同步、搜索和 add-source；最后补齐 skill/index/lock/status/Web 的 drift metadata 链路，让 update plan 能解释“版本相同但内容或 commit 已变化”的情况。

**Tech Stack:** Go, `gookit/gcli`, existing `configstore`/`sourceapp`/`installapp`/`statusapp`/`webapp`, JSON registry catalog cache, Go unit tests with `github.com/gookit/goutil/testutil/assert`, final verification via `go test ./...`.

---

| 修订时间 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-16 | v0.1 | Codex | 输出 P8 Registry / source UX / precise drift 实施计划 |
| 2026-06-16 | v0.2 | Codex | 复审后补强 URL 解析、Registry HTTP/歧义测试、目录级 checksum 和 status fixture |
| 2026-06-16 | v0.3 | Codex | 记录 Phase 8 实施完成、最终验证和 e2e checksum 断言调整 |

相关文档：

- 设计文档：`docs/design/skillc-v0-enhance-design.md`
- Phase 7 计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase7-cross-project-update.md`
- 任务入口：`docs/TODO.md`

## Scope and Change Size

这三个方向如果都按完整产品做，会偏大：

- Registry 完整版会涉及远程发现协议、版本签名、权限、安全审计、profile 推荐、Web 浏览和安装入口。
- Source UX cleanup 会触及 source domain、sourceapp、CLI、Web source add plan/run、README 和旧测试期望。
- 精确 drift 会贯穿 skill parser/index、lock 写入、list/status/update check、Web version-drift 和跨项目 update plan。

P8 采用 MVP 边界，可以同一期完成，但必须分批提交：

1. Source UX cleanup 是 P8 的前置基础，因为 Registry 的 add-source 需要复用自定义 source ID/name。
2. Registry 只做“发现 source”的本机闭环，不做远程账号、签名、自动安装 skill 或 Web Registry 页面。
3. Precise drift 只补 metadata 和 plan/status 判断，不改变 update 执行策略；真正执行仍复用现有 update/install app service。

预计改动量：

- 新增 3 个 domain/app/infra package 或文件组：`registry`、`registryapp`、`registrystore`。
- 修改 20 到 30 个文件，主要集中在 source/config/registry/drift/status/Web。
- 代码改动属于中大型阶段，不建议再塞 project manifest、remote Web 或 registry 安全模型。

## Phase 8 Scope

本期做：

- Source UX cleanup：
  - `skillc source add <path-or-git-url> [--id <id>] [--name <name>] [--ref <ref>] [--sync]`
  - 保留 `skillc source add local <path>` 和 `skillc source add git <url> [ref]`，并补 `--id/--name`。
  - 新生成 source ID 不再强加 `local-` / `git-` 前缀；显式 ID 会 normalize 并校验重复。
  - 旧配置中的 `local-*` / `git-*` ID 保持不迁移、不重写。
  - 新增 `skillc source info <id>`，支持精确 ID 或现有 partial match。
- Registry 最小发现闭环：
  - Config 增加 `registries`。
  - 支持 local JSON catalog 和 HTTP JSON catalog。
  - `skillc registry list/add/remove/sync/search/info/add-source`。
  - registry search 只返回 catalog entry，不直接安装 skill、不写 lock。
  - `registry add-source <entry-id>` 把 catalog entry 转成 source 配置，可选 `--sync`。
- Precise drift：
  - index skill 记录 `checksum` 和 `source_resolved_ref`。
  - local checksum 使用 install entry 目录级 checksum，不只 hash `SKILL.md`。
  - lock record 写入安装时的 `checksum` 和 `source_resolved_ref`。
  - `status` / `update --check` 在版本相同但 Git commit 或 local checksum 变化时标记 `outdated`，并给出原因。
  - Web install-map/version-drift 暴露 checksum/ref metadata，并在版本相同但 metadata 不同时展示 drift group。
- 文档：
  - 更新 README / README.zh-CN 命令说明。
  - 更新 `docs/TODO.md` 和设计文档 Phase 8 状态。
  - 在本计划记录最终验证命令。

本期不做：

- 不做 Registry 账号、token、签名校验、信任策略或远程权限模型。
- 不做 Registry 中 profile 推荐、collection 推荐或一键安装 profile。
- 不做 Web Registry 页面。
- 不自动迁移旧 source ID，也不批量重写 lock/profile 中已有 source 引用。
- 不改变 update 执行策略，不新增后台 job 或定时同步。
- 不做 project manifest / `skillc.profile.yaml`。

## User-Facing Behavior

### Source UX

推荐入口：

```bash
skillc source add ./skills --id gstack --name "GStack Skills" --sync
skillc source add https://github.com/acme/skills.git --id acme --name "Acme Skills" --ref main --sync
skillc source info gstack
```

兼容入口：

```bash
skillc source add local ./skills --id gstack --name "GStack Skills"
skillc source add git https://github.com/acme/skills.git main --id acme --name "Acme Skills"
```

新自动 ID 示例：

```text
./gstack/skills                     -> gstack-skills
https://github.com/acme/skills.git  -> acme-skills
```

旧配置中的 `local-gstack-skills`、`git-acme-skills` 不会被自动改名。

### Registry

Catalog JSON 示例：

```json
{
  "sources": [
    {
      "id": "gstack",
      "name": "GStack Skills",
      "description": "Agent workflow skills",
      "type": "git",
      "url": "https://github.com/acme/gstack-skills.git",
      "ref": "main",
      "tags": ["agent", "workflow"]
    },
    {
      "id": "local-lab",
      "name": "Local Lab",
      "description": "Local experimental skills",
      "type": "local",
      "path": "./fixtures/local-lab",
      "tags": ["local"]
    }
  ]
}
```

CLI:

```bash
skillc registry add ./registry.json --id local --name "Local Registry"
skillc registry add https://example.com/skillc-registry.json --id official --name "Official Registry"
skillc registry sync --all
skillc registry search go
skillc registry info gstack
skillc registry add-source gstack --sync
```

`registry add-source` 只写 source 配置并可选 sync，不安装任何 skill。

### Precise Drift

`skillc status` / `skillc update --check` 继续优先显示 version drift：

```text
outdated  go-pro  1.0.0 -> 2.0.0
```

若 version 相同但 metadata 不同：

```text
outdated  review  git ref abc12345 -> def67890
outdated  local-rules  checksum 8e15a3aa -> b31c9a22
```

Web Version Drift 在同一版本下也能显示不同 checksum/ref 的 drift group。

## Data Model

### Source Options

```go
package source

type SourceOptions struct {
	ID   string
	Name string
}
```

新构造函数：

```go
func NewLocalSourceWithOptions(path string, opts SourceOptions) (Source, error)
func NewGitSourceWithOptions(url string, ref string, opts SourceOptions) (Source, error)
```

兼容函数：

```go
func NewLocalSource(path string) (Source, error) {
	return NewLocalSourceWithOptions(path, SourceOptions{})
}

func NewGitSource(url string, ref string) (Source, error) {
	return NewGitSourceWithOptions(url, ref, SourceOptions{})
}
```

### Registry Config

```go
package registry

type Type string

const (
	TypeLocal Type = "local"
	TypeHTTP  Type = "http"
)

type Registry struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name,omitempty" json:"name,omitempty"`
	Type        Type   `yaml:"type" json:"type"`
	Path        string `yaml:"path,omitempty" json:"path,omitempty"`
	URL         string `yaml:"url,omitempty" json:"url,omitempty"`
	LastSyncAt  string `yaml:"last_sync_at,omitempty" json:"last_sync_at,omitempty"`
	Status      string `yaml:"status,omitempty" json:"status,omitempty"`
	ErrorMessage string `yaml:"error_message,omitempty" json:"error_message,omitempty"`
}

type Entry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type"`
	URL         string   `json:"url,omitempty"`
	Path        string   `json:"path,omitempty"`
	Ref         string   `json:"ref,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	RegistryID  string   `json:"registry_id,omitempty"`
}
```

Config 增加：

```go
Registries []registry.Registry `yaml:"registries,omitempty"`
```

Registry cache：

```json
{
  "entries": [
    {
      "id": "gstack",
      "registry_id": "official",
      "name": "GStack Skills",
      "type": "git",
      "url": "https://github.com/acme/gstack-skills.git",
      "ref": "main"
    }
  ]
}
```

### Drift Metadata

`skill.Skill` 增加：

```go
Checksum          string `json:"checksum,omitempty"`
SourceResolvedRef string `json:"source_resolved_ref,omitempty"`
```

`lock.Record` 增加：

```go
SourceResolvedRef string `json:"source_resolved_ref"`
```

已有 `Checksum string` 字段开始写入真实值。

Checksum 语义：

- `skill.ParseSkillMarkdown` 先写入 `SKILL.md` 内容 hash，作为没有目录上下文时的 fallback。
- `repoindex.Scanner` 在知道 skill dir 和 `install_entry` 后，覆盖为 install entry 目录级 checksum。
- 目录级 checksum 按相对文件路径排序，hash 相对路径和文件内容，跳过 `.git` 目录，保证同内容跨机器稳定。

`statusapp.Item` 增加：

```go
CurrentChecksum          string `json:"current_checksum,omitempty"`
LatestChecksum           string `json:"latest_checksum,omitempty"`
CurrentSourceResolvedRef string `json:"current_source_resolved_ref,omitempty"`
LatestSourceResolvedRef  string `json:"latest_source_resolved_ref,omitempty"`
```

## File Structure

新增文件：

- `internal/domain/registry/model.go`
  - Registry config model、Entry model、ID normalize、catalog validation。
- `internal/domain/registry/model_test.go`
  - 覆盖 registry ID normalize、local/http 判断、entry validation。
- `internal/infra/registrystore/json_store.go`
  - Registry cache JSON load/save。
- `internal/infra/registrystore/json_store_test.go`
  - 覆盖 cache round-trip 和空文件行为。
- `internal/app/registryapp/service.go`
  - Registry list/add/remove/sync/search/info/add-source。
- `internal/app/registryapp/service_test.go`
  - 覆盖 local catalog、HTTP catalog、search、add-source。
- `internal/cli/registry_cmd.go`
  - 新增 `skillc registry ...` CLI。

修改文件：

- `internal/domain/source/model.go`
  - 支持 `SourceOptions`，新 source ID 不加 type 前缀。
- `internal/domain/source/model_test.go`
  - 更新新 ID 规则和显式 ID/name 测试。
- `internal/domain/config/model.go`
  - 增加 `Registries`。
- `internal/domain/config/defaults.go`
  - 默认 `Registries` 为空。
- `internal/infra/configstore/yaml_store.go`
  - 读写 registries，并 expand/compact local registry path。
- `internal/infra/configstore/yaml_store_test.go`
  - 覆盖 registries YAML round-trip。
- `internal/app/sourceapp/service.go`
  - 新增 AddReq、Add、AddLocalWithOptions、AddGitWithOptions、Info。
- `internal/app/sourceapp/service_test.go`
  - 覆盖自定义 id/name、自动 ID、重复 ID/path/url、info。
- `internal/cli/source_cmd.go`
  - 调整 `source add` 入口，增加 `source info`。
- `internal/cli/app.go`
  - 注册 `registry` 命令。
- `internal/cli/app_test.go`
  - 覆盖 source UX 和 registry CLI。
- `internal/domain/skill/model.go`
  - 增加 checksum/ref metadata。
- `internal/domain/skill/parser.go`
  - 计算 checksum，携带 source resolved ref。
- `internal/domain/skill/parser_test.go`
  - 覆盖 metadata。
- `internal/infra/repoindex/scanner.go`
  - 保留每个 indexed skill 的 checksum/ref metadata。
- `internal/infra/repoindex/scanner_test.go`
  - 覆盖 indexed metadata。
- `internal/infra/hashx/dir.go`
  - 新增 install entry 目录级 checksum。
- `internal/infra/hashx/dir_test.go`
  - 覆盖多文件顺序稳定、内容变化和跳过 `.git`。
- `internal/domain/lock/model.go`
  - 增加 source resolved ref。
- `internal/app/installapp/service.go`
  - install/reinstall 写入 checksum/ref。
- `internal/app/installapp/service_test.go`
  - 覆盖 lock metadata。
- `internal/app/listapp/service.go`
  - list item 暴露 checksum/ref。
- `internal/app/statusapp/service.go`
  - drift 判断增加 ref/checksum 比较。
- `internal/app/statusapp/service_test.go`
  - 覆盖 version/ref/checksum 三类 outdated。
- `internal/app/webapp/project_index.go`
  - install-map/version-drift 带 metadata 并按 metadata drift 生成 group。
- `internal/app/webapp/project_index_test.go`
  - 覆盖同版本不同 ref/checksum 的 drift group。
- `internal/app/webapp/manager_server.go`
  - status JSON 暴露 drift metadata。
- `internal/app/webapp/manager_static.go`
  - Version Drift 增加 drift signal 展示。
- `README.md`
  - 更新 source/registry/drift 命令说明。
- `README.zh-CN.md`
  - 更新中文命令说明。
- `docs/TODO.md`
  - 增加 P8 计划和状态。
- `docs/design/skillc-v0-enhance-design.md`
  - 增加 P8 链接和范围说明。
- `docs/superpowers/plans/2026-06-16-skillc-v0-phase8-registry-source-drift.md`
  - 实施中持续更新 checkbox 和验证记录。

## Task 1: Source Domain and App Service UX Cleanup

**Files:**
- Modify: `internal/domain/source/model.go`
- Modify: `internal/domain/source/model_test.go`
- Modify: `internal/app/sourceapp/service.go`
- Modify: `internal/app/sourceapp/service_test.go`

- [x] **Step 1: Write failing source domain tests**

Add tests to `internal/domain/source/model_test.go`:

```go
func TestNewLocalSourceGeneratesIDWithoutTypePrefix(t *testing.T) {
	src, err := NewLocalSource(filepath.Join("work", "gstack", "skills"))

	assert.NoErr(t, err)
	assert.Eq(t, "gstack-skills", src.ID)
	assert.Eq(t, "gstack-skills", src.Name)
	assert.Eq(t, TypeLocal, src.Type)
}

func TestNewGitSourceGeneratesIDWithoutTypePrefix(t *testing.T) {
	src, err := NewGitSource("https://github.com/acme/skills.git", "main")

	assert.NoErr(t, err)
	assert.Eq(t, "acme-skills", src.ID)
	assert.Eq(t, "acme-skills", src.Name)
	assert.Eq(t, TypeGit, src.Type)
	assert.Eq(t, "main", src.Ref)
}

func TestNewSourceWithExplicitIDAndName(t *testing.T) {
	src, err := NewGitSourceWithOptions("https://github.com/acme/skills.git", "", SourceOptions{
		ID:   "Acme Skills",
		Name: "Acme Registry",
	})

	assert.NoErr(t, err)
	assert.Eq(t, "acme-skills", src.ID)
	assert.Eq(t, "Acme Registry", src.Name)
}
```

- [x] **Step 2: Run source domain tests to verify they fail**

Run:

```bash
go test ./internal/domain/source -run 'TestNew(Local|Git)SourceGeneratesIDWithoutTypePrefix|TestNewSourceWithExplicitIDAndName' -v
```

Expected: FAIL because source IDs still include `local-` / `git-` and `SourceOptions` does not exist.

- [x] **Step 3: Implement source options and new ID rule**

Modify `internal/domain/source/model.go`:

```go
type SourceOptions struct {
	ID   string
	Name string
}

func NewLocalSource(path string) (Source, error) {
	return NewLocalSourceWithOptions(path, SourceOptions{})
}

func NewLocalSourceWithOptions(path string, opts SourceOptions) (Source, error) {
	clean := fsutil.ToAbsPath(filepath.Clean(path))
	name := sourceNameFromPath(clean)
	if name == "" {
		return Source{}, fmt.Errorf("invalid source path: %s", path)
	}
	if opts.Name != "" {
		name = strings.TrimSpace(opts.Name)
	}
	id := NormalizeID(firstNonEmpty(opts.ID, name))
	if id == "" {
		return Source{}, fmt.Errorf("source id is required")
	}
	return Source{ID: id, Type: TypeLocal, Name: name, Path: clean}, nil
}

func NewGitSource(url, ref string) (Source, error) {
	return NewGitSourceWithOptions(url, ref, SourceOptions{})
}

func NewGitSourceWithOptions(url, ref string, opts SourceOptions) (Source, error) {
	trimmed := strings.TrimSpace(url)
	if trimmed == "" {
		return Source{}, fmt.Errorf("invalid git source url")
	}
	name := sourceNameFromGitURL(trimmed)
	if name == "" {
		return Source{}, fmt.Errorf("invalid git source url: %s", url)
	}
	if opts.Name != "" {
		name = strings.TrimSpace(opts.Name)
	}
	id := NormalizeID(firstNonEmpty(opts.ID, name))
	if id == "" {
		return Source{}, fmt.Errorf("source id is required")
	}
	return Source{ID: id, Type: TypeGit, Name: name, URL: trimmed, Ref: strings.TrimSpace(ref)}, nil
}
```

Also add helpers:

```go
func sourceNameFromPath(path string) string {
	name := filepath.Base(path)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return ""
	}
	if name == "skills" || name == "skill" {
		parent := filepath.Base(filepath.Dir(path))
		if parent != "." && parent != string(filepath.Separator) && parent != "" {
			name = parent + "-" + name
		}
	}
	return name
}

func sourceNameFromGitURL(url string) string {
	name := strings.TrimSuffix(path.Base(strings.TrimSpace(url)), ".git")
	if name == "." || name == "/" || name == "" {
		return ""
	}
	if name == "skills" || name == "skill" {
		parent := path.Base(path.Dir(url))
		if parent != "." && parent != "/" && parent != "" {
			name = parent + "-" + name
		}
	}
	return name
}

func NormalizeID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
```

`sourceNameFromPath` keeps the existing `skills` / `skill` parent-name behavior.
`sourceNameFromGitURL` must use the standard `path` package, not `filepath`; on Windows `filepath.Base("https://github.com/acme/skills.git")` treats `/` as ordinary text and generates invalid IDs.

- [x] **Step 4: Run source domain tests**

Run:

```bash
go test ./internal/domain/source
```

Expected: PASS after updating old ID expectations to the new no-prefix rule.

- [x] **Step 5: Write failing sourceapp tests**

Add tests to `internal/app/sourceapp/service_test.go`:

```go
func TestService_AddLocalWithCustomIDAndName(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	sourceRoot := filepath.Join(baseDir, "skills")
	assert.NoErr(t, os.MkdirAll(sourceRoot, 0o755))

	src, err := NewService(configFile, baseDir).Add(AddReq{
		Value: sourceRoot,
		ID:    "GStack",
		Name:  "GStack Skills",
	})

	assert.NoErr(t, err)
	assert.Eq(t, "gstack", src.ID)
	assert.Eq(t, "GStack Skills", src.Name)
	assert.Eq(t, source.TypeLocal, src.Type)
}

func TestService_AddGeneratesUniqueIDForDuplicateNames(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	first := filepath.Join(baseDir, "a", "skills")
	second := filepath.Join(baseDir, "b", "skills")
	assert.NoErr(t, os.MkdirAll(first, 0o755))
	assert.NoErr(t, os.MkdirAll(second, 0o755))
	service := NewService(configFile, baseDir)

	one, err := service.Add(AddReq{Value: first, Name: "Shared Skills"})
	assert.NoErr(t, err)
	two, err := service.Add(AddReq{Value: second, Name: "Shared Skills"})
	assert.NoErr(t, err)

	assert.Eq(t, "shared-skills", one.ID)
	assert.Eq(t, "shared-skills-2", two.ID)
}

func TestService_InfoFindsSourceByPartialID(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	sourceRoot := filepath.Join(baseDir, "skills")
	assert.NoErr(t, os.MkdirAll(sourceRoot, 0o755))
	service := NewService(configFile, baseDir)
	_, err := service.Add(AddReq{Value: sourceRoot, ID: "gstack"})
	assert.NoErr(t, err)

	src, err := service.Info("gst")

	assert.NoErr(t, err)
	assert.Eq(t, "gstack", src.ID)
}
```

- [x] **Step 6: Run sourceapp tests to verify they fail**

Run:

```bash
go test ./internal/app/sourceapp -run 'TestService_AddLocalWithCustomIDAndName|TestService_AddGeneratesUniqueIDForDuplicateNames|TestService_InfoFindsSourceByPartialID' -v
```

Expected: FAIL because `AddReq`, `Add`, and `Info` do not exist.

- [x] **Step 7: Implement sourceapp AddReq and Info**

Modify `internal/app/sourceapp/service.go`:

```go
type AddReq struct {
	Value string
	Type  domainsource.Type
	ID    string
	Name  string
	Ref   string
}

func (s *Service) Add(req AddReq) (domainsource.Source, error) {
	value := strings.TrimSpace(req.Value)
	if value == "" {
		return domainsource.Source{}, fmt.Errorf("source value is required")
	}
	if req.Type == domainsource.TypeGit || (req.Type == "" && isGitURL(value)) {
		return s.AddGitWithOptions(value, req.Ref, domainsource.SourceOptions{ID: req.ID, Name: req.Name})
	}
	return s.AddLocalWithOptions(value, domainsource.SourceOptions{ID: req.ID, Name: req.Name})
}

func (s *Service) AddLocalWithOptions(path string, opts domainsource.SourceOptions) (domainsource.Source, error)
func (s *Service) AddGitWithOptions(url string, ref string, opts domainsource.SourceOptions) (domainsource.Source, error)
func (s *Service) Info(partial string) (domainsource.Source, error)
```

Use exact duplicate checks:

- explicit duplicate ID returns `source already exists: <id>`.
- duplicate path/url returns `source already exists: <path-or-url>`.
- generated ID collision receives `-2`, `-3` suffix by checking existing IDs.

Keep existing `AddLocal` and `AddGit` as wrappers around the new option-aware methods.

- [x] **Step 8: Run sourceapp tests**

Run:

```bash
go test ./internal/app/sourceapp
```

Expected: PASS.

- [ ] **Step 9: Commit source domain/app cleanup**

Run:

```bash
git add internal/domain/source/model.go internal/domain/source/model_test.go internal/app/sourceapp/service.go internal/app/sourceapp/service_test.go
git commit -m "feat(skillc): improve source identity creation"
```

## Task 2: Source CLI Cleanup and `source info`

**Files:**
- Modify: `internal/cli/source_cmd.go`
- Modify: `internal/cli/app_test.go`
- Modify: `README.md`
- Modify: `README.zh-CN.md`

- [x] **Step 1: Write failing CLI tests**

Add tests to `internal/cli/app_test.go`:

```go
func TestSourceAddCommandAcceptsDirectPathWithCustomIDAndName(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "skills")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))

	output := runAppInDirWithStdout(t, baseDir, []string{"source", "add", "--id", "gstack", "--name", "GStack Skills", sourceDir})

	assert.Contains(t, output, "gstack added")
	assert.Contains(t, output, "GStack Skills")
	config, err := configstore.NewYAMLStore().Load(filepath.Join(baseDir, "skillc.yaml"), baseDir)
	assert.NoErr(t, err)
	assert.Len(t, config.Sources, 1)
	assert.Eq(t, "gstack", config.Sources[0].ID)
	assert.Eq(t, "GStack Skills", config.Sources[0].Name)
}

func TestSourceAddGitCommandAcceptsCustomIDAndName(t *testing.T) {
	baseDir := t.TempDir()

	output := runAppInDirWithStdout(t, baseDir, []string{"source", "add", "git", "--id", "acme", "--name", "Acme Skills", "https://example.com/skills.git", "main"})

	assert.Contains(t, output, "acme added")
	assert.Contains(t, output, "Acme Skills")
}

func TestSourceInfoCommandPrintsDetails(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "skills")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	runAppInDirWithStdout(t, baseDir, []string{"source", "add", "--id", "gstack", "--name", "GStack Skills", sourceDir})

	output := runAppInDirWithStdout(t, baseDir, []string{"source", "info", "gst"})

	assert.Contains(t, output, "Source Info")
	assert.Contains(t, output, "gstack")
	assert.Contains(t, output, "GStack Skills")
	assert.Contains(t, output, sourceDir)
}
```

- [x] **Step 2: Run CLI tests to verify they fail**

Run:

```bash
go test ./internal/cli -run 'TestSource(Add|Info)' -v
```

Expected: FAIL because direct `source add <value>` and `source info` are not implemented.

- [x] **Step 3: Implement source CLI**

Modify `internal/cli/source_cmd.go`:

- Replace inline `add` command with `buildSourceAddCommand()`.
- Keep `buildSourceAddLocalCommand()` and `buildSourceAddGitCommand()` as subcommands.
- Add shared flags `--id`, `--name`, `--sync`, `--ref`.
- Add `buildSourceInfoCommand()`.

Direct add behavior:

```go
skillc source add <path-or-git-url> [--id <id>] [--name <name>] [--ref <ref>] [--sync]
```

Output includes ID, name, type, path/url, and next sync command when `--sync` is false.

Info behavior:

```go
skillc source info <id>
```

Use `sourceapp.Info()` so partial matching works the same as `source sync`.

- [x] **Step 4: Run source CLI tests**

Run:

```bash
go test ./internal/cli -run 'TestSource(Add|Info)' -v
```

Expected: PASS.

- [x] **Step 5: Update README source command snippets**

Update source command sections:

```markdown
skillc source add <path-or-git-url> [--id <id>] [--name <name>] [--ref <ref>] [--sync]
skillc source add local <path> [--id <id>] [--name <name>] [--sync]
skillc source add git <url> [ref] [--id <id>] [--name <name>] [--sync]
skillc source info <id>
```

Mention: new generated IDs no longer receive `local-` / `git-` prefixes; existing IDs are left untouched.

- [x] **Step 6: Run focused CLI and sourceapp tests**

Run:

```bash
go test ./internal/domain/source ./internal/app/sourceapp ./internal/cli
```

Expected: PASS.

- [x] **Step 7: Commit source CLI cleanup**

Run:

```bash
git add internal/cli/source_cmd.go internal/cli/app_test.go README.md README.zh-CN.md
git commit -m "feat(skillc): add source info and custom source ids"
```

## Task 3: Registry Domain and Config Persistence

**Files:**
- Create: `internal/domain/registry/model.go`
- Create: `internal/domain/registry/model_test.go`
- Modify: `internal/domain/config/model.go`
- Modify: `internal/domain/config/defaults.go`
- Modify: `internal/infra/configstore/yaml_store.go`
- Modify: `internal/infra/configstore/yaml_store_test.go`

- [x] **Step 1: Write failing registry domain tests**

Create `internal/domain/registry/model_test.go`:

```go
package registry

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestNewRegistryFromLocalPath(t *testing.T) {
	got, err := New("Local Registry", "Local Registry", "./registry.json")

	assert.NoErr(t, err)
	assert.Eq(t, "local-registry", got.ID)
	assert.Eq(t, "Local Registry", got.Name)
	assert.Eq(t, TypeLocal, got.Type)
	assert.NotEmpty(t, got.Path)
}

func TestNewRegistryFromHTTPURL(t *testing.T) {
	got, err := New("official", "Official", "https://example.com/registry.json")

	assert.NoErr(t, err)
	assert.Eq(t, "official", got.ID)
	assert.Eq(t, TypeHTTP, got.Type)
	assert.Eq(t, "https://example.com/registry.json", got.URL)
}

func TestEntryValidateRequiresSourceLocation(t *testing.T) {
	err := Entry{ID: "broken", Type: "git"}.Validate()

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "registry entry git url is required")
}
```

- [x] **Step 2: Run registry domain tests to verify they fail**

Run:

```bash
go test ./internal/domain/registry -v
```

Expected: FAIL because package does not exist.

- [x] **Step 3: Implement registry domain model**

Create `internal/domain/registry/model.go` with:

```go
type Type string

const (
	TypeLocal Type = "local"
	TypeHTTP  Type = "http"
)

type Registry struct {
	ID           string `yaml:"id" json:"id"`
	Name         string `yaml:"name,omitempty" json:"name,omitempty"`
	Type         Type   `yaml:"type" json:"type"`
	Path         string `yaml:"path,omitempty" json:"path,omitempty"`
	URL          string `yaml:"url,omitempty" json:"url,omitempty"`
	LastSyncAt   string `yaml:"last_sync_at,omitempty" json:"last_sync_at,omitempty"`
	Status       string `yaml:"status,omitempty" json:"status,omitempty"`
	ErrorMessage string `yaml:"error_message,omitempty" json:"error_message,omitempty"`
}

type Entry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type"`
	URL         string   `json:"url,omitempty"`
	Path        string   `json:"path,omitempty"`
	Ref         string   `json:"ref,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	RegistryID  string   `json:"registry_id,omitempty"`
}

type Catalog struct {
	Sources []Entry `json:"sources"`
}
```

Implement:

```go
func New(id string, name string, value string) (Registry, error)
func (e Entry) Validate() error
func NormalizeID(value string) string
func IsHTTPURL(value string) bool
```

- [x] **Step 4: Write failing configstore registry tests**

Add to `internal/infra/configstore/yaml_store_test.go`:

```go
func TestYAMLStore_LoadSaveRegistries(t *testing.T) {
	baseDir := t.TempDir()
	path := filepath.Join(baseDir, "skillc.yaml")
	registryPath := filepath.Join(baseDir, "registry.json")
	assert.NoErr(t, os.WriteFile(registryPath, []byte(`{"sources":[]}`), 0o644))

	data := cfg.DefaultConfig()
	data.Registries = []registry.Registry{{ID: "local", Name: "Local", Type: registry.TypeLocal, Path: registryPath}}

	store := NewYAMLStore()
	assert.NoErr(t, store.Save(path, data, baseDir))

	got, err := store.Load(path, baseDir)
	assert.NoErr(t, err)
	assert.Len(t, got.Registries, 1)
	assert.Eq(t, "local", got.Registries[0].ID)
	assert.Eq(t, registryPath, got.Registries[0].Path)
}
```

- [x] **Step 5: Run configstore registry tests to verify they fail**

Run:

```bash
go test ./internal/infra/configstore -run TestYAMLStore_LoadSaveRegistries -v
```

Expected: FAIL because config does not persist registries.

- [x] **Step 6: Implement config registries persistence**

Modify:

- `internal/domain/config/model.go`: add `Registries []registry.Registry`.
- `internal/domain/config/defaults.go`: default empty registries.
- `internal/infra/configstore/yaml_store.go`:
  - add `registryRecord`.
  - add `Registries []registryRecord` to `rawConfig`.
  - add `toRegistryRecords` / `fromRegistryRecords`.
  - clone registries.
  - expand/compact local registry `Path`; do not expand HTTP `URL`.
  - omit empty registries on save.

- [x] **Step 7: Run registry domain and configstore tests**

Run:

```bash
go test ./internal/domain/registry ./internal/infra/configstore
```

Expected: PASS.

- [x] **Step 8: Commit registry config model**

Run:

```bash
git add internal/domain/registry internal/domain/config/model.go internal/domain/config/defaults.go internal/infra/configstore/yaml_store.go internal/infra/configstore/yaml_store_test.go
git commit -m "feat(skillc): add registry config model"
```

## Task 4: Registry Store and App Service

**Files:**
- Create: `internal/infra/registrystore/json_store.go`
- Create: `internal/infra/registrystore/json_store_test.go`
- Create: `internal/app/registryapp/service.go`
- Create: `internal/app/registryapp/service_test.go`

- [x] **Step 1: Write failing registry store tests**

Create `internal/infra/registrystore/json_store_test.go`:

```go
package registrystore

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/registry"
)

func TestStoreLoadSaveEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry-index.json")
	store := NewStore()
	entries := []registry.Entry{{ID: "gstack", Name: "GStack", Type: "git", URL: "https://example.com/skills.git", RegistryID: "official"}}

	assert.NoErr(t, store.Save(path, entries))
	got, err := store.Load(path)

	assert.NoErr(t, err)
	assert.Len(t, got, 1)
	assert.Eq(t, "gstack", got[0].ID)
	assert.Eq(t, "official", got[0].RegistryID)
}
```

- [x] **Step 2: Run registry store tests to verify they fail**

Run:

```bash
go test ./internal/infra/registrystore -v
```

Expected: FAIL because package does not exist.

- [x] **Step 3: Implement registry JSON cache store**

Create `internal/infra/registrystore/json_store.go`:

```go
type File struct {
	Entries []registry.Entry `json:"entries"`
}

type Store struct{}

func NewStore() *Store
func (s *Store) Load(path string) ([]registry.Entry, error)
func (s *Store) Save(path string, entries []registry.Entry) error
```

Behavior:

- Missing file returns empty slice.
- Save creates parent directory.
- JSON is indented for inspectability.

- [x] **Step 4: Write failing registryapp tests**

Create `internal/app/registryapp/service_test.go`:

```go
func TestService_SyncLocalRegistryAndSearch(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	catalogPath := filepath.Join(baseDir, "registry.json")
	assert.NoErr(t, os.WriteFile(catalogPath, []byte(`{"sources":[{"id":"gstack","name":"GStack Skills","description":"Go workflow","type":"git","url":"https://example.com/gstack.git","ref":"main","tags":["go"]}]}`), 0o644))
	service := NewService(configFile, baseDir)

	item, err := service.Add(AddReq{ID: "local", Name: "Local", Value: catalogPath})
	assert.NoErr(t, err)
	assert.Eq(t, "local", item.ID)
	assert.NoErr(t, service.Sync("local"))

	results, err := service.Search("go")
	assert.NoErr(t, err)
	assert.Len(t, results, 1)
	assert.Eq(t, "gstack", results[0].ID)
	assert.Eq(t, "local", results[0].RegistryID)
}

func TestService_SyncHTTPRegistryAndSearch(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"sources":[{"id":"remote","name":"Remote Skills","description":"Remote Go workflow","type":"git","url":"https://example.com/remote.git","tags":["go"]}]}`))
	}))
	defer server.Close()
	service := NewService(configFile, baseDir)

	_, err := service.Add(AddReq{ID: "official", Name: "Official", Value: server.URL})
	assert.NoErr(t, err)
	assert.NoErr(t, service.Sync("official"))

	results, err := service.Search("remote")
	assert.NoErr(t, err)
	assert.Len(t, results, 1)
	assert.Eq(t, "remote", results[0].ID)
	assert.Eq(t, "official", results[0].RegistryID)
}

func TestService_AddSourceFromRegistryEntry(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	catalogPath := filepath.Join(baseDir, "registry.json")
	assert.NoErr(t, os.WriteFile(catalogPath, []byte(`{"sources":[{"id":"gstack","name":"GStack Skills","type":"git","url":"https://example.com/gstack.git","ref":"main"}]}`), 0o644))
	service := NewService(configFile, baseDir)
	_, err := service.Add(AddReq{ID: "local", Value: catalogPath})
	assert.NoErr(t, err)
	assert.NoErr(t, service.Sync("local"))

	src, err := service.AddSource(AddSourceReq{EntryID: "gstack"})

	assert.NoErr(t, err)
	assert.Eq(t, "gstack", src.ID)
	assert.Eq(t, "GStack Skills", src.Name)
	assert.Eq(t, "https://example.com/gstack.git", src.URL)
}

func TestService_InfoRejectsAmbiguousEntryID(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	firstPath := filepath.Join(baseDir, "first.json")
	secondPath := filepath.Join(baseDir, "second.json")
	assert.NoErr(t, os.WriteFile(firstPath, []byte(`{"sources":[{"id":"shared","name":"First","type":"git","url":"https://example.com/first.git"}]}`), 0o644))
	assert.NoErr(t, os.WriteFile(secondPath, []byte(`{"sources":[{"id":"shared","name":"Second","type":"git","url":"https://example.com/second.git"}]}`), 0o644))
	service := NewService(configFile, baseDir)
	_, err := service.Add(AddReq{ID: "first", Value: firstPath})
	assert.NoErr(t, err)
	_, err = service.Add(AddReq{ID: "second", Value: secondPath})
	assert.NoErr(t, err)
	assert.NoErr(t, service.SyncAll())

	_, err = service.Info("shared")

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "ambiguous registry entry")
}
```

- [x] **Step 5: Run registryapp tests to verify they fail**

Run:

```bash
go test ./internal/app/registryapp -v
```

Expected: FAIL because package does not exist.

- [x] **Step 6: Implement registryapp service**

Create `internal/app/registryapp/service.go` with:

```go
type AddReq struct {
	ID    string
	Name  string
	Value string
}

type AddSourceReq struct {
	EntryID string
	ID      string
	Name    string
	Sync    bool
}

type Service struct {
	configFile string
	baseDir    string
	store      *configstore.YAMLStore
	cache      *registrystore.Store
	client     *http.Client
	now        func() time.Time
}

func NewService(configFile string, baseDir string) *Service
func (s *Service) List() ([]registry.Registry, error)
func (s *Service) Add(req AddReq) (registry.Registry, error)
func (s *Service) Remove(id string) error
func (s *Service) Sync(id string) error
func (s *Service) SyncAll() error
func (s *Service) Search(keyword string) ([]registry.Entry, error)
func (s *Service) Info(entryID string) (registry.Entry, error)
func (s *Service) AddSource(req AddSourceReq) (source.Source, error)
```

Important behavior:

- Registry config duplicate ID is rejected.
- Registry local path must exist.
- HTTP sync uses `GET`, requires status 2xx, and decodes `registry.Catalog`.
- Each synced entry gets `RegistryID`.
- Cache path is `filepath.Join(config.RegistryCacheDir, "registry-index.json")`.
- Search matches ID, name, description, tags, URL/path case-insensitively.
- Duplicate entry IDs across registries are preserved in cache, but `Info(entryID)` returns an error if ambiguous; the user can pass `registryID/entryID`.
- `AddSource` calls `sourceapp.NewService(s.configFile, s.baseDir).Add(sourceapp.AddReq{Value: entry.URL or entry.Path, Type: source.Type(entry.Type), ID: firstNonEmpty(req.ID, entry.ID), Name: firstNonEmpty(req.Name, entry.Name), Ref: entry.Ref})`.
- Local catalog entry paths are resolved relative to the catalog file directory during sync and cached as cleaned absolute paths.
- HTTP catalog entries with `type:"local"` are rejected unless `path` is absolute; remote registries should normally publish git entries.
- `Sync` updates registry `Status`, `ErrorMessage`, and `LastSyncAt` in config before saving cache entries.

- [x] **Step 7: Run registryapp tests**

Run:

```bash
go test ./internal/infra/registrystore ./internal/app/registryapp
```

Expected: PASS.

- [x] **Step 8: Commit registry app service**

Run:

```bash
git add internal/infra/registrystore internal/app/registryapp
git commit -m "feat(skillc): add registry discovery service"
```

## Task 5: Registry CLI

**Files:**
- Create: `internal/cli/registry_cmd.go`
- Modify: `internal/cli/app.go`
- Modify: `internal/cli/app_test.go`
- Modify: `README.md`
- Modify: `README.zh-CN.md`

- [x] **Step 1: Write failing registry CLI tests**

Add tests to `internal/cli/app_test.go`:

```go
func TestNewApp_RegistersRegistryCommand(t *testing.T) {
	app := newTestApp()

	cmd := findCommandByName(app, "registry")

	assert.NotNil(t, cmd)
	assert.Eq(t, "Manage Skillc registries", cmd.Desc)
}

func TestRegistryCommandAddSyncSearchAndAddSource(t *testing.T) {
	baseDir := t.TempDir()
	catalogPath := filepath.Join(baseDir, "registry.json")
	assert.NoErr(t, os.WriteFile(catalogPath, []byte(`{"sources":[{"id":"gstack","name":"GStack Skills","description":"Go workflow","type":"git","url":"https://example.com/gstack.git","ref":"main","tags":["go"]}]}`), 0o644))

	addOutput := runAppInDirWithStdout(t, baseDir, []string{"registry", "add", "--id", "local", "--name", "Local Registry", catalogPath})
	assert.Contains(t, addOutput, "registry added: local")

	syncOutput := runAppInDirWithStdout(t, baseDir, []string{"registry", "sync", "local"})
	assert.Contains(t, syncOutput, "registry synced: local")

	searchOutput := runAppInDirWithStdout(t, baseDir, []string{"registry", "search", "go"})
	assert.Contains(t, searchOutput, "gstack")
	assert.Contains(t, searchOutput, "GStack Skills")

	sourceOutput := runAppInDirWithStdout(t, baseDir, []string{"registry", "add-source", "gstack"})
	assert.Contains(t, sourceOutput, "source added: gstack")
}
```

- [x] **Step 2: Run registry CLI tests to verify they fail**

Run:

```bash
go test ./internal/cli -run 'Test(NewApp_RegistersRegistryCommand|RegistryCommand)' -v
```

Expected: FAIL because registry command is not registered.

- [x] **Step 3: Implement registry CLI**

Create `internal/cli/registry_cmd.go` with commands:

```text
registry list
registry add <path-or-url> --id <id> --name <name>
registry remove <id>
registry sync [id] --all
registry search <keyword>
registry info <entry-id>
registry add-source <entry-id> --id <id> --name <name> --sync
```

Register in `internal/cli/app.go`:

```go
app.Add(buildRegistryCommand())
```

Output:

- `registry list`: ID, Name, Type, Status, Path/URL.
- `registry search`: Registry, ID, Name, Type, Tags, URL/Path.
- `registry info`: key/value table with description/ref/tags.
- `registry add-source`: prints `source added: <id>`.

- [x] **Step 4: Run registry CLI tests**

Run:

```bash
go test ./internal/cli -run 'Test(NewApp_RegistersRegistryCommand|RegistryCommand)' -v
```

Expected: PASS.

- [x] **Step 5: Update README registry command docs**

Add command sections:

```markdown
### `registry` — Discover source catalogs

skillc registry add <path-or-url> --id official --name "Official Registry"
skillc registry sync --all
skillc registry search go
skillc registry info gstack
skillc registry add-source gstack --sync
```

Explain that Registry discovers source entries and never writes lock records directly.

- [x] **Step 6: Run registry focused tests**

Run:

```bash
go test ./internal/domain/registry ./internal/infra/registrystore ./internal/app/registryapp ./internal/cli
```

Expected: PASS.

- [x] **Step 7: Commit registry CLI**

Run:

```bash
git add internal/cli/registry_cmd.go internal/cli/app.go internal/cli/app_test.go README.md README.zh-CN.md
git commit -m "feat(skillc): add registry cli"
```

## Task 6: Precise Drift Metadata in Index, Lock, List, and Status

**Files:**
- Modify: `internal/domain/skill/model.go`
- Modify: `internal/domain/skill/parser.go`
- Modify: `internal/domain/skill/parser_test.go`
- Create: `internal/infra/hashx/dir.go`
- Create: `internal/infra/hashx/dir_test.go`
- Modify: `internal/infra/repoindex/scanner.go`
- Modify: `internal/infra/repoindex/scanner_test.go`
- Modify: `internal/domain/lock/model.go`
- Modify: `internal/app/installapp/service.go`
- Modify: `internal/app/installapp/service_test.go`
- Modify: `internal/app/listapp/service.go`
- Modify: `internal/app/statusapp/service.go`
- Modify: `internal/app/statusapp/service_test.go`

- [x] **Step 1: Write failing skill parser metadata tests**

Add to `internal/domain/skill/parser_test.go`:

```go
func TestParseSkillMarkdownCarriesChecksumAndSourceResolvedRef(t *testing.T) {
	content := "---\nid: go-pro\nname: Go Pro\nversion: 1.0.0\n---\n# Go Pro\n"
	src := source.Source{ID: "gstack", Type: source.TypeGit, Path: "/tmp/go-pro", ResolvedRef: "deadbeefcafebabe"}

	got, err := ParseSkillMarkdown(content, src)

	assert.NoErr(t, err)
	assert.NotEmpty(t, got.Checksum)
	assert.Eq(t, "deadbeefcafebabe", got.SourceResolvedRef)
}
```

- [x] **Step 2: Run parser test to verify it fails**

Run:

```bash
go test ./internal/domain/skill -run TestParseSkillMarkdownCarriesChecksumAndSourceResolvedRef -v
```

Expected: FAIL because `Skill.Checksum` and `Skill.SourceResolvedRef` do not exist.

- [x] **Step 3: Implement skill metadata**

Modify:

- `internal/domain/skill/model.go`: add `Checksum` and `SourceResolvedRef`.
- `internal/domain/skill/parser.go`: set `Checksum: hashx.SumString(content)` and `SourceResolvedRef: src.ResolvedRef`.
- `internal/infra/hashx/dir.go`: add deterministic directory checksum:

```go
func SumDir(root string) (string, error) {
	// Walk root recursively, skip .git directories, sort relative file paths,
	// and hash each relative path plus file content using sha256.
}
```

- `internal/infra/repoindex/scanner.go`: after parsing `SKILL.md`, compute `hashx.SumDir(filepath.Join(skillDir, parsed.InstallEntry))` and assign it to `parsed.Checksum`.
- `internal/infra/repoindex/scanner_test.go`: add a test that changes a non-`SKILL.md` file under the install entry and proves the indexed checksum changes.

- [x] **Step 4: Write failing install/status drift tests**

Add to `internal/app/installapp/service_test.go`:

```go
func TestService_InstallWritesDriftMetadataToLock(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	targetRoot := filepath.Join(baseDir, "target")
	sourceDir := filepath.Join(baseDir, "source", "go-pro")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("content"), 0o644))

	svc := NewService(lockFile)
	_, err := svc.Install(skill.Skill{
		ID: "go-pro", Version: "1.0.0", SourceID: "gstack", SourceType: sourcepkg.TypeGit,
		SourceResolvedRef: "deadbeefcafebabe", Checksum: "abc123", InstallEntry: ".", Path: sourceDir,
	}, "universal", agent.ScopeProject, baseDir, targetRoot)

	assert.NoErr(t, err)
	locks, err := lockstore.NewStore().Load(lockFile)
	assert.NoErr(t, err)
	assert.Eq(t, "abc123", locks[baseDir][0].Checksum)
	assert.Eq(t, "deadbeefcafebabe", locks[baseDir][0].SourceResolvedRef)
}
```

Add to `internal/app/statusapp/service_test.go`:

```go
func TestService_RunMarksOutdatedWhenGitResolvedRefChanges(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeStatusDriftFixture(t, baseDir)
	projectKey := baseDir
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))
	assert.NoErr(t, lockstore.NewStore().Save(config.LockFile, lockpkg.File{
		projectKey: {{
			SkillID: "go-pro", SourceID: "gstack", SourceType: "git", Version: "1.0.0",
			SourceResolvedRef: "oldcommit", Agents: []string{"universal"},
		}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(config.IndexFile, []skill.Skill{{
		ID: "go-pro", SourceID: "gstack", SourceType: sourcepkg.TypeGit, Version: "1.0.0",
		SourceResolvedRef: "newcommit", Path: filepath.Join(baseDir, "source", "go-pro"),
	}}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Eq(t, StatusOutdated, result.Items[0].Status)
	assert.Contains(t, result.Items[0].Reason, "git ref")
}

func TestService_RunMarksOutdatedWhenLocalChecksumChanges(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeStatusDriftFixture(t, baseDir)
	projectKey := baseDir
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "rules"), 0o755))
	assert.NoErr(t, lockstore.NewStore().Save(config.LockFile, lockpkg.File{
		projectKey: {{
			SkillID: "rules", SourceID: "local", SourceType: "local", Version: "1.0.0",
			Checksum: "oldsum", Agents: []string{"universal"},
		}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(config.IndexFile, []skill.Skill{{
		ID: "rules", SourceID: "local", SourceType: sourcepkg.TypeLocal, Version: "1.0.0",
		Checksum: "newsum", Path: filepath.Join(baseDir, "source", "rules"),
	}}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Eq(t, StatusOutdated, result.Items[0].Status)
	assert.Contains(t, result.Items[0].Reason, "checksum")
}

func writeStatusDriftFixture(t *testing.T, baseDir string) (string, cfg.Config) {
	t.Helper()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	return configFile, config
}
```

- [x] **Step 5: Run drift tests to verify they fail**

Run:

```bash
go test ./internal/app/installapp ./internal/app/statusapp -run 'TestService_(InstallWritesDriftMetadataToLock|RunMarksOutdatedWhenGitResolvedRefChanges|RunMarksOutdatedWhenLocalChecksumChanges)' -v
```

Expected: FAIL because lock/status metadata is not wired.

- [x] **Step 6: Implement install/list/status metadata wiring**

Modify:

- `internal/domain/lock/model.go`: add `SourceResolvedRef string`.
- `internal/app/installapp/service.go`:
  - `installInto` writes `Checksum` and `SourceResolvedRef`.
  - `ReinstallAtPath` writes `Checksum` and `SourceResolvedRef`.
- `internal/app/listapp/service.go`:
  - `Item` carries `SourceResolvedRef`.
  - list conversion copies `record.SourceResolvedRef`.
- `internal/app/listapp/service_test.go`:
  - add `TestService_ListCarriesDriftMetadata` that writes lock `Checksum` and `SourceResolvedRef`, creates the installed skill dir, and asserts both fields appear on `listapp.Item`.
- `internal/app/statusapp/service.go`:
  - `Item` carries current/latest checksum/ref.
  - `classifyListItem` sets metadata from current and latest.
  - version comparison stays first.
  - git ref comparison runs when both refs are non-empty and differ.
  - checksum comparison runs when both checksums are non-empty and differ.
  - reason strings are deterministic:
    - `git ref <old> -> <new>`
    - `checksum <old8> -> <new8>`

- [x] **Step 7: Run focused drift tests**

Run:

```bash
go test ./internal/domain/skill ./internal/infra/repoindex ./internal/app/installapp ./internal/app/listapp ./internal/app/statusapp
```

Expected: PASS.

- [x] **Step 8: Commit drift metadata core**

Run:

```bash
git add internal/domain/skill internal/infra/repoindex internal/domain/lock/model.go internal/app/installapp internal/app/listapp internal/app/statusapp
git commit -m "feat(skillc): track precise drift metadata"
```

## Task 7: Web and CLI Drift Presentation

**Files:**
- Modify: `internal/cli/manage_cmd.go`
- Modify: `internal/app/webapp/project_index.go`
- Modify: `internal/app/webapp/project_index_test.go`
- Modify: `internal/app/webapp/manager_server.go`
- Modify: `internal/app/webapp/manager_server_test.go`
- Modify: `internal/app/webapp/manager_static.go`

- [x] **Step 1: Write failing Web drift tests**

Add to `internal/app/webapp/project_index_test.go`:

```go
func TestBuildVersionDriftReportsSameVersionChecksumDrift(t *testing.T) {
	items := []ProjectInstall{
		{ProjectPath: "a", SkillID: "rules", SourceID: "local", Version: "1.0.0", Checksum: "oldsum"},
		{ProjectPath: "b", SkillID: "rules", SourceID: "local", Version: "1.0.0", Checksum: "newsum"},
	}
	index := []skill.Skill{{ID: "rules", SourceID: "local", Version: "1.0.0", Checksum: "newsum"}}

	groups := BuildVersionDrift(items, index)

	assert.Len(t, groups, 1)
	assert.Eq(t, "rules", groups[0].SkillID)
	assert.Contains(t, strings.Join(groups[0].DriftReasons, ","), "checksum")
}

func TestBuildVersionDriftReportsSameVersionGitRefDrift(t *testing.T) {
	items := []ProjectInstall{
		{ProjectPath: "a", SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", SourceResolvedRef: "oldcommit"},
		{ProjectPath: "b", SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", SourceResolvedRef: "newcommit"},
	}
	index := []skill.Skill{{ID: "go-pro", SourceID: "gstack", Version: "1.0.0", SourceResolvedRef: "newcommit"}}

	groups := BuildVersionDrift(items, index)

	assert.Len(t, groups, 1)
	assert.Contains(t, strings.Join(groups[0].DriftReasons, ","), "git ref")
}
```

- [x] **Step 2: Run Web drift tests to verify they fail**

Run:

```bash
go test ./internal/app/webapp -run 'TestBuildVersionDriftReportsSameVersion' -v
```

Expected: FAIL because ProjectInstall and VersionDriftGroup do not expose drift metadata.

- [x] **Step 3: Implement Web drift metadata**

Modify `internal/app/webapp/project_index.go`:

```go
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
	Checksum            string `json:"checksum,omitempty"`
	SourceResolvedRef   string `json:"source_resolved_ref,omitempty"`
}

type VersionDriftGroup struct {
	SkillID                 string          `json:"skill_id"`
	SourceQualifiedName     string          `json:"source_qualified_name,omitempty"`
	SourceID                string          `json:"source_id,omitempty"`
	LatestVersion           string          `json:"latest_version,omitempty"`
	LatestChecksum          string          `json:"latest_checksum,omitempty"`
	LatestSourceResolvedRef string          `json:"latest_source_resolved_ref,omitempty"`
	DriftReasons            []string        `json:"drift_reasons,omitempty"`
	Versions                []VersionBucket `json:"versions"`
}
```

Behavior:

- `BuildProjectInstallIndex` copies checksum/ref from lock records.
- `BuildVersionDrift` includes a group when:
  - versions differ, or
  - installed checksum/ref buckets differ even if latest metadata is unavailable, or
  - latest checksum differs from at least one installed checksum, or
  - latest source resolved ref differs from at least one installed ref.
- `DriftReasons` contains stable values: `version`, `checksum`, `git ref`.

- [x] **Step 4: Update manager status JSON and static UI**

Modify `internal/app/webapp/manager_server.go`:

- `managerStatusItem` includes checksum/ref fields from `statusapp.Item`.

Modify `internal/app/webapp/manager_static.go`:

- Status table can continue showing reason.
- Version Drift table adds a `Signals` column using `drift_reasons`.
- Keep layout compact; do not add a new Web page.

- [x] **Step 5: Update CLI update check output**

Modify `internal/cli/manage_cmd.go`:

- `printUpdateCheckResult` already prints `Reason`; no new column required.
- Ensure status reason from Task 6 appears unchanged.
- Add a CLI test if no existing test asserts reason output.

- [x] **Step 6: Run Web and CLI drift presentation tests**

Run:

```bash
go test ./internal/app/webapp ./internal/cli
```

Expected: PASS.

- [x] **Step 7: Commit Web/CLI drift presentation**

Run:

```bash
git add internal/app/webapp internal/cli/manage_cmd.go internal/cli/app_test.go
git commit -m "feat(skillc): show precise drift signals"
```

## Task 8: Documentation, Plan Progress, and Final Verification

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/TODO.md`
- Modify: `docs/design/skillc-v0-enhance-design.md`
- Modify: `docs/superpowers/plans/2026-06-16-skillc-v0-phase8-registry-source-drift.md`
- Modify: `internal/cli/app_test.go`
- Modify: `tests/e2e/source_search_test.go`

- [x] **Step 1: Update README files**

Document:

- `source add <path-or-git-url> --id --name --ref --sync`.
- `source info <id>`.
- `registry add/list/sync/search/info/add-source`.
- Precise drift behavior in `status` / `update --check`.

- [x] **Step 2: Update design and task docs**

Update `docs/design/skillc-v0-enhance-design.md`:

- Add revision row for Phase 8 implementation completion.
- Add P8 plan link if not already present.
- Record Phase 8 status and remaining deferred work.

Update `docs/TODO.md`:

- Mark source ID cleanup item complete.
- Add Registry discovery status.
- Add precise drift status.

- [x] **Step 3: Run focused verification**

Run:

```bash
go test ./internal/domain/source ./internal/app/sourceapp ./internal/domain/registry ./internal/infra/registrystore ./internal/app/registryapp ./internal/cli ./internal/domain/skill ./internal/infra/repoindex ./internal/app/installapp ./internal/app/listapp ./internal/app/statusapp ./internal/app/webapp
```

Expected: PASS.

- [x] **Step 4: Run full verification**

Run:

```bash
go test ./...
```

Expected: PASS.

- [x] **Step 5: Update this plan checkbox statuses and verification record**

## Verification

- `go test ./tests/e2e -run TestLocalSourceToSearchFlow -v`
- `go test ./internal/domain/source ./internal/app/sourceapp ./internal/domain/registry ./internal/infra/registrystore ./internal/app/registryapp ./internal/cli ./internal/domain/skill ./internal/infra/repoindex ./internal/app/installapp ./internal/app/listapp ./internal/app/statusapp ./internal/app/webapp`
- `go test ./...`

Verification note: full verification initially exposed an e2e assertion that compared the entire `skill.Skill` struct without accounting for the new `Checksum` metadata. The e2e now checks the stable user-facing fields and asserts checksum is present.

- [x] **Step 6: Commit documentation and verification**

Run:

```bash
git add README.md README.zh-CN.md docs/TODO.md docs/design/skillc-v0-enhance-design.md docs/superpowers/plans/2026-06-16-skillc-v0-phase8-registry-source-drift.md
git commit -m "docs(skillc): document phase 8 registry source drift"
```

## Self-Review Checklist

- [x] New source IDs do not receive forced `local-` / `git-` prefixes.
- [x] Existing source IDs remain unchanged and no migration rewrites config, lock, or profiles.
- [x] `source add` supports direct path/url plus legacy local/git subcommands.
- [x] `source info <id>` provides a single-source detail view.
- [x] Registry sync/search works from local JSON and HTTP JSON catalogs.
- [x] Registry search does not install skills or write lock records.
- [x] `registry add-source` writes source config through sourceapp and can optionally sync.
- [x] Install/reinstall writes checksum and source resolved ref into lock records.
- [x] Status/update check mark same-version checksum/ref changes as outdated with clear reason.
- [x] Web version drift shows same-version metadata drift.
- [x] README, TODO, design doc, and this plan cross-reference each other.
- [x] `go test ./...` passes before claiming Phase 8 complete.

## Remaining After Phase 8

- Registry trust model: signatures, checksums for catalog entries, source allow/deny policy.
- Registry profiles or recommended bundles.
- Web Registry browsing and add-source UI.
- Project manifest / `skillc.profile.yaml` export/import.
- Remote Web access, multi-user permissions, and security audit model.
- Optional migration tool for old source IDs if users explicitly request it.
