# Skillc v0 Phase 9 Registry Skill Search and Install Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修正 Registry 回到 PRD 定位：从第三方或团队 Registry 搜索 Skill 级结果，并支持不经 `add-source` 直接安装单个 registry skill。

**Architecture:** 保留 P8 的 JSON source catalog 作为 Registry 的一个子能力，同时扩展 Registry catalog 支持 `skills`。Registry skill 安装时先 materialize 到本地 registry skill cache，再复用现有 install service 写入目标 agent 目录和 lock；lock 使用 `source_type=registry` 并记录 registry provenance。Generic JSON skill catalog 是 P9 第一条可执行路径，skills.sh / skillsmp / skillsllm 等站点作为后续 provider adapter 接入。

**Tech Stack:** Go, `gookit/gcli`, existing `registryapp` / `installapp` / `statusapp` / `updateapp`, `gitx` for Git source materialization, local directory snapshot copy for test/team catalogs, JSON registry cache, Go unit tests with `github.com/gookit/goutil/x/assert`, final verification via `go test ./...`.

---

| 修订时间 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-16 | v0.1 | Codex | 输出 P9 Registry skill search/install 修正实施计划 |

相关文档：

- 设计文档：`docs/design/skillc-v0-enhance-design.md`
- 原 PRD：`docs/prd.md`
- MVP 架构：`docs/mvp-arch.md`
- Phase 8 计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase8-registry-source-drift.md`
- 任务入口：`docs/TODO.md`

## Scope

P9 做：

- 将 Registry catalog 从 source-only 扩展为 `skills + sources`。
- `registry search` 默认搜索 Skill 级结果；保留 `--kind source` 查看 P8 source catalog。
- `registry info <registry>/<skill>` 显示 Skill 结果详情；source 结果仍可查看。
- `registry install <registry>/<skill>` 直接 materialize registry skill 并安装，不要求先 `registry add-source`。
- lock 写入 `source_type=registry`，并记录 registry id、entry id、provider URL、source URL/ref、download URL 等 provenance。
- registry-installed skills 能被 `list` / `uninstall` 继续处理。
- `install` 无参数 restore 能恢复 registry-installed skills。
- `status` / `update --check` 不再把 registry-installed skills 误判为 orphan；只基于 registry cache 比较 version/checksum，不做 clone/copy。
- `update` 能对 registry-installed skill 执行重新 materialize + reinstall 的最小闭环。

P9 不做：

- 不实现真实 skills.sh / skillsmp / skillsllm 站点 API adapter；P9 先落地 generic JSON catalog，后续再按站点补 provider adapter。
- 不做 Web Registry 页面。
- 不做签名、评分、账号、token、权限和远程信任模型。
- 不做 profile/bundle registry 安装。
- 不做 archive download/extract 的完整格式矩阵；P9 可执行 materializer 支持 `source_url + source_ref + install_entry`。其中 `source_url` 可以是 Git URL 或本地目录：Git URL 通过 `gitx.Sync` 同步到 registry skill cache，本地目录复制成 registry skill cache snapshot。`download_url` 先记录到模型和 lock，缺少 `source_url` 时给出清晰错误。

## User-Facing Behavior

Generic JSON catalog:

```json
{
  "skills": [
    {
      "id": "go-pro",
      "name": "Go Pro",
      "description": "Go development helper",
      "version": "1.2.0",
      "supported_agents": ["codex", "claude-code"],
      "source_url": "https://github.com/acme/skills.git",
      "source_ref": "main",
      "install_entry": "skills/go-pro",
      "checksum": "sha256:abc123",
      "tags": ["go", "dev"]
    }
  ],
  "sources": [
    {
      "id": "gstack",
      "name": "GStack Skills",
      "type": "git",
      "url": "https://github.com/acme/gstack-skills.git",
      "ref": "main"
    }
  ]
}
```

CLI:

```bash
skillc registry add https://example.com/skillc-registry.json --id team --name "Team Registry"
skillc registry sync team
skillc registry search go
skillc registry info team/go-pro
skillc registry install team/go-pro --agent codex --scope project
skillc registry search gstack --kind source
skillc registry add-source team/gstack --sync
```

Expected semantics:

- `registry search` returns skill rows by default.
- `registry search --kind source` returns P8 source rows.
- `registry install` installs a single Skill result directly.
- `registry add-source` remains optional and only converts source results into long-lived source config.
- A target with duplicate skill ID across registries must require `registry-id/skill-id`.

## Data Model

Extend `internal/domain/source/model.go`:

```go
const (
	TypeLocal    Type = "local"
	TypeGit      Type = "git"
	TypeRegistry Type = "registry"
)
```

Extend `internal/domain/registry/model.go`:

```go
type Catalog struct {
	Skills  []SkillEntry `json:"skills,omitempty"`
	Sources []Entry      `json:"sources,omitempty"`
}

type SkillEntry struct {
	ID              string   `json:"id"`
	Name            string   `json:"name,omitempty"`
	Description     string   `json:"description,omitempty"`
	Version         string   `json:"version,omitempty"`
	SupportedAgents []string `json:"supported_agents,omitempty"`
	SourceURL       string   `json:"source_url,omitempty"`
	SourceRef       string   `json:"source_ref,omitempty"`
	DownloadURL     string   `json:"download_url,omitempty"`
	InstallEntry    string   `json:"install_entry,omitempty"`
	Checksum         string   `json:"checksum,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	Homepage         string   `json:"homepage,omitempty"`
	RegistryID       string   `json:"registry_id,omitempty"`
	RegistryURL      string   `json:"registry_url,omitempty"`
}
```

Extend `internal/domain/skill/model.go` and `internal/domain/lock/model.go` with optional registry provenance:

```go
RegistryEntryID string `json:"registry_entry_id,omitempty"`
RegistryURL     string `json:"registry_url,omitempty"`
DownloadURL     string `json:"download_url,omitempty"`
SourceURL       string `json:"source_url,omitempty"`
SourceRef       string `json:"source_ref,omitempty"`
```

For installed registry skills:

- `Skill.SourceID = entry.RegistryID`
- `Skill.SourceType = source.TypeRegistry`
- `Skill.Path = <registry skill cache repo/snapshot path>`
- `Skill.InstallEntry = entry.InstallEntry`
- `LockRecord.SourceID = registry id`
- `LockRecord.SourceType = "registry"`
- `LockRecord.RegistryEntryID = entry id`
- `RegistryURL` 由 sync/service 根据 `config.Registries` 的 provider `URL` 或 `Path` 填充；catalog 文件中不要求发布者手写。
- `SourceURL` 是 Skill 内容来源，和 `RegistryURL` 分开记录：前者用于 materialize，后者用于说明这条搜索结果来自哪个 registry provider。

## File Structure

新增文件：

- `internal/app/registryapp/materializer.go`
  - 把 `registry.SkillEntry` materialize 成可交给 `installapp` 的 `skill.Skill`。
- `internal/app/registryapp/materializer_test.go`
  - 覆盖 source URL materialization、缺少 source URL 的错误、provenance 映射。
- `internal/app/registryapp/locked_resolver.go`
  - 从 lock record 解析 registry-installed skill，用于 restore/status/update。
- `internal/app/registryapp/locked_resolver_test.go`
  - 覆盖 registry lock record 重新解析。

修改文件：

- `internal/domain/source/model.go`
- `internal/domain/source/model_test.go`
- `internal/domain/registry/model.go`
- `internal/domain/registry/model_test.go`
- `internal/infra/registrystore/json_store.go`
- `internal/infra/registrystore/json_store_test.go`
- `internal/app/registryapp/service.go`
- `internal/app/registryapp/service_test.go`
- `internal/cli/registry_cmd.go`
- `internal/cli/app_test.go`
- `internal/domain/skill/model.go`
- `internal/domain/lock/model.go`
- `internal/app/installapp/service.go`
- `internal/app/installapp/service_test.go`
- `internal/app/statusapp/service.go`
- `internal/app/statusapp/service_test.go`
- `internal/app/updateapp/service.go`
- `internal/app/updateapp/service_test.go`
- `README.md`
- `README.zh-CN.md`
- `docs/TODO.md`
- `docs/design/skillc-v0-enhance-design.md`
- `docs/superpowers/plans/2026-06-16-skillc-v0-phase9-registry-skill-search-install.md`

## Task 1: Registry Domain Model Correction

**Files:**
- Modify: `internal/domain/source/model.go`
- Modify: `internal/domain/source/model_test.go`
- Modify: `internal/domain/registry/model.go`
- Modify: `internal/domain/registry/model_test.go`

- [x] **Step 1: Write failing source type test**

Add to `internal/domain/source/model_test.go`:

```go
func TestSourceTypeRegistryConstant(t *testing.T) {
	assert.Eq(t, Type("registry"), TypeRegistry)
}
```

- [x] **Step 2: Write failing registry skill entry tests**

Add to `internal/domain/registry/model_test.go`:

```go
func TestSkillEntryValidateRequiresInstallSource(t *testing.T) {
	err := SkillEntry{ID: "go-pro", Name: "Go Pro"}.Validate()

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "source_url or download_url is required")
}

func TestNormalizeSkillEntryDefaultsInstallEntryAndName(t *testing.T) {
	entry, err := NormalizeSkillEntry(SkillEntry{
		ID:        "Go Pro",
		SourceURL: "https://github.com/acme/skills.git",
	}, "skills-sh")

	assert.NoErr(t, err)
	assert.Eq(t, "go-pro", entry.ID)
	assert.Eq(t, "go-pro", entry.Name)
	assert.Eq(t, ".", entry.InstallEntry)
	assert.Eq(t, "skills-sh", entry.RegistryID)
}
```

- [x] **Step 3: Run tests to verify they fail**

Run:

```bash
go test ./internal/domain/source ./internal/domain/registry -run 'Test(SourceTypeRegistryConstant|SkillEntry|NormalizeSkillEntry)' -v
```

Expected: FAIL because `TypeRegistry`, `SkillEntry`, and `NormalizeSkillEntry` do not exist.

- [x] **Step 4: Implement registry domain model**

Modify `internal/domain/source/model.go`:

```go
const (
	TypeLocal    Type = "local"
	TypeGit      Type = "git"
	TypeRegistry Type = "registry"
)
```

Modify `internal/domain/registry/model.go`:

```go
type Catalog struct {
	Skills  []SkillEntry `json:"skills,omitempty"`
	Sources []Entry      `json:"sources,omitempty"`
}

type SkillEntry struct {
	ID              string   `json:"id"`
	Name            string   `json:"name,omitempty"`
	Description     string   `json:"description,omitempty"`
	Version         string   `json:"version,omitempty"`
	SupportedAgents []string `json:"supported_agents,omitempty"`
	SourceURL       string   `json:"source_url,omitempty"`
	SourceRef       string   `json:"source_ref,omitempty"`
	DownloadURL     string   `json:"download_url,omitempty"`
	InstallEntry    string   `json:"install_entry,omitempty"`
	Checksum         string   `json:"checksum,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	Homepage         string   `json:"homepage,omitempty"`
	RegistryID       string   `json:"registry_id,omitempty"`
	RegistryURL      string   `json:"registry_url,omitempty"`
}

func (e SkillEntry) Validate() error {
	if NormalizeID(e.ID) == "" {
		return fmt.Errorf("registry skill id is required")
	}
	if strings.TrimSpace(e.SourceURL) == "" && strings.TrimSpace(e.DownloadURL) == "" {
		return fmt.Errorf("registry skill source_url or download_url is required")
	}
	return nil
}

func NormalizeSkillEntry(entry SkillEntry, registryID string) (SkillEntry, error) {
	entry.ID = NormalizeID(entry.ID)
	entry.Name = strings.TrimSpace(entry.Name)
	if entry.Name == "" {
		entry.Name = entry.ID
	}
	entry.SourceURL = strings.TrimSpace(entry.SourceURL)
	entry.SourceRef = strings.TrimSpace(entry.SourceRef)
	entry.DownloadURL = strings.TrimSpace(entry.DownloadURL)
	entry.InstallEntry = strings.TrimSpace(entry.InstallEntry)
	if entry.InstallEntry == "" {
		entry.InstallEntry = "."
	}
	entry.Checksum = strings.TrimSpace(entry.Checksum)
	entry.RegistryURL = strings.TrimSpace(entry.RegistryURL)
	entry.RegistryID = registryID
	if err := entry.Validate(); err != nil {
		return SkillEntry{}, err
	}
	return entry, nil
}
```

- [x] **Step 5: Run domain tests**

Run:

```bash
go test ./internal/domain/source ./internal/domain/registry
```

Expected: PASS.

- [x] **Step 6: Commit**

Run:

```bash
git add internal/domain/source/model.go internal/domain/source/model_test.go internal/domain/registry/model.go internal/domain/registry/model_test.go
git commit -m "feat(skillc): model registry skill entries"
```

## Task 2: Registry Cache Supports Skills and Sources

**Files:**
- Modify: `internal/infra/registrystore/json_store.go`
- Modify: `internal/infra/registrystore/json_store_test.go`

- [x] **Step 1: Write failing cache tests**

Add to `internal/infra/registrystore/json_store_test.go`:

```go
func TestStore_SaveAndLoadSkillsAndSources(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry-index.json")
	store := NewStore()
	file := File{
		Skills: []registry.SkillEntry{{ID: "go-pro", RegistryID: "team", SourceURL: "https://example.com/skills.git", InstallEntry: "skills/go-pro"}},
		Sources: []registry.Entry{{ID: "gstack", RegistryID: "team", Type: "git", URL: "https://example.com/gstack.git"}},
	}

	assert.NoErr(t, store.SaveFile(path, file))
	got, err := store.LoadFile(path)

	assert.NoErr(t, err)
	assert.Len(t, got.Skills, 1)
	assert.Eq(t, "go-pro", got.Skills[0].ID)
	assert.Len(t, got.Sources, 1)
	assert.Eq(t, "gstack", got.Sources[0].ID)
}

func TestStore_LoadLegacyEntriesAsSources(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry-index.json")
	assert.NoErr(t, os.WriteFile(path, []byte(`{"entries":[{"id":"gstack","registry_id":"team","type":"git","url":"https://example.com/gstack.git"}]}`), 0o644))

	got, err := NewStore().LoadFile(path)

	assert.NoErr(t, err)
	assert.Len(t, got.Sources, 1)
	assert.Eq(t, "gstack", got.Sources[0].ID)
}
```

- [x] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/infra/registrystore -run 'TestStore_(SaveAndLoadSkillsAndSources|LoadLegacyEntriesAsSources)' -v
```

Expected: FAIL because the store only persists `entries`.

- [x] **Step 3: Implement cache file format**

Modify `internal/infra/registrystore/json_store.go`:

```go
type File struct {
	Skills  []registry.SkillEntry `json:"skills,omitempty"`
	Sources []registry.Entry      `json:"sources,omitempty"`
	Entries []registry.Entry      `json:"entries,omitempty"`
}

func (s *Store) LoadFile(path string) (File, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return File{Skills: []registry.SkillEntry{}, Sources: []registry.Entry{}}, nil
	}
	if err != nil {
		return File{}, err
	}
	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return File{}, err
	}
	if file.Sources == nil && file.Entries != nil {
		file.Sources = file.Entries
	}
	if file.Skills == nil {
		file.Skills = []registry.SkillEntry{}
	}
	if file.Sources == nil {
		file.Sources = []registry.Entry{}
	}
	file.Entries = nil
	return file, nil
}

func (s *Store) SaveFile(path string, file File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file.Entries = nil
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
```

Keep existing compatibility methods:

```go
func (s *Store) Load(path string) ([]registry.Entry, error) {
	file, err := s.LoadFile(path)
	if err != nil {
		return nil, err
	}
	return file.Sources, nil
}

func (s *Store) Save(path string, entries []registry.Entry) error {
	return s.SaveFile(path, File{Sources: entries})
}
```

- [x] **Step 4: Run cache tests**

Run:

```bash
go test ./internal/infra/registrystore
```

Expected: PASS.

- [x] **Step 5: Commit**

Run:

```bash
git add internal/infra/registrystore/json_store.go internal/infra/registrystore/json_store_test.go
git commit -m "feat(skillc): cache registry skills and sources"
```

## Task 3: Registry Service Searches Skill Results

**Files:**
- Modify: `internal/app/registryapp/service.go`
- Modify: `internal/app/registryapp/service_test.go`

- [x] **Step 1: Write failing skill search tests**

Add to `internal/app/registryapp/service_test.go`:

```go
func TestService_SyncJSONRegistryCachesSkillEntries(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	catalogPath := filepath.Join(baseDir, "registry.json")
	assert.NoErr(t, os.WriteFile(catalogPath, []byte(`{"skills":[{"id":"go-pro","name":"Go Pro","description":"Go helper","version":"1.2.0","source_url":"https://example.com/skills.git","source_ref":"main","install_entry":"skills/go-pro","tags":["go"]}]}`), 0o644))

	service := NewService(configFile, baseDir)
	_, err := service.Add(AddReq{ID: "team", Name: "Team", Value: catalogPath})
	assert.NoErr(t, err)
	assert.NoErr(t, service.Sync("team"))

	results, err := service.SearchSkills(SearchReq{Keyword: "go"})

	assert.NoErr(t, err)
	assert.Len(t, results, 1)
	assert.Eq(t, "go-pro", results[0].ID)
	assert.Eq(t, "team", results[0].RegistryID)
	assert.Eq(t, catalogPath, results[0].RegistryURL)
}

func TestService_InfoSkillRequiresRegistryWhenAmbiguous(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	cacheDir := filepath.Join(baseDir, ".cache")
	config := cfg.DefaultConfig()
	config.RegistryCacheDir = cacheDir
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	service := NewService(configFile, baseDir)
	assert.NoErr(t, service.cache.SaveFile(filepath.Join(cacheDir, "registry-index.json"), registrystore.File{
		Skills: []registry.SkillEntry{
			{ID: "go-pro", RegistryID: "team-a", SourceURL: "https://example.com/a.git", InstallEntry: "."},
			{ID: "go-pro", RegistryID: "team-b", SourceURL: "https://example.com/b.git", InstallEntry: "."},
		},
	}))

	_, err := service.InfoSkill("go-pro")

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "ambiguous registry skill")
}
```

- [x] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/app/registryapp -run 'TestService_(SyncJSONRegistryCachesSkillEntries|InfoSkillRequiresRegistryWhenAmbiguous)' -v
```

Expected: FAIL because service only supports source entries.

- [x] **Step 3: Implement SearchReq and skill search/info**

Modify `internal/app/registryapp/service.go`:

```go
type SearchReq struct {
	Keyword    string
	RegistryID string
}

func (s *Service) loadCacheFile() (registrystore.File, error) {
	data, err := s.load()
	if err != nil {
		return registrystore.File{}, err
	}
	return s.cache.LoadFile(registryCachePath(data))
}

func (s *Service) SearchSkills(req SearchReq) ([]registry.SkillEntry, error) {
	file, err := s.loadCacheFile()
	if err != nil {
		return nil, err
	}
	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	registryID := strings.ToLower(strings.TrimSpace(req.RegistryID))
	var out []registry.SkillEntry
	for _, entry := range file.Skills {
		if registryID != "" && strings.ToLower(entry.RegistryID) != registryID {
			continue
		}
		if keyword == "" || skillEntryMatches(entry, keyword) {
			out = append(out, entry)
		}
	}
	sortSkillEntries(out)
	return out, nil
}

func (s *Service) InfoSkill(selector string) (registry.SkillEntry, error) {
	file, err := s.loadCacheFile()
	if err != nil {
		return registry.SkillEntry{}, err
	}
	matches := matchSkillEntries(file.Skills, selector)
	switch len(matches) {
	case 0:
		return registry.SkillEntry{}, fmt.Errorf("registry skill not found: %s", selector)
	case 1:
		return matches[0], nil
	default:
		return registry.SkillEntry{}, fmt.Errorf("ambiguous registry skill: %s", selector)
	}
}
```

Update `Sync` so it normalizes and caches both `catalog.Skills` and `catalog.Sources`. During normalization, populate `SkillEntry.RegistryURL` from the registry provider: use `item.URL` for HTTP registries and `item.Path` for local registries. For local registry catalogs, a non-Git relative `source_url` is resolved relative to the catalog file directory; for HTTP registry catalogs, a non-Git local path `source_url` is rejected because remote registries cannot safely point at the user's filesystem. Keep existing source `Search` / `Info` / `AddSource` behavior by reading `file.Sources`.

- [x] **Step 4: Run registry service tests**

Run:

```bash
go test ./internal/app/registryapp
```

Expected: PASS.

- [x] **Step 5: Commit**

Run:

```bash
git add internal/app/registryapp/service.go internal/app/registryapp/service_test.go
git commit -m "feat(skillc): search registry skill entries"
```

## Task 4: Materialize Registry Skills to Local Cache

**Files:**
- Create: `internal/app/registryapp/materializer.go`
- Create: `internal/app/registryapp/materializer_test.go`
- Modify: `internal/domain/skill/model.go`
- Modify: `internal/domain/lock/model.go`
- Modify: `internal/app/installapp/service.go`
- Modify: `internal/app/installapp/service_test.go`

- [x] **Step 1: Write failing materializer tests**

Create `internal/app/registryapp/materializer_test.go`:

```go
package registryapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/skillc/internal/domain/registry"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/gitx"
)

func TestMaterializer_ToSkillMapsRegistryProvenance(t *testing.T) {
	baseDir := t.TempDir()
	entry := registry.SkillEntry{
		ID: "go-pro", Name: "Go Pro", Version: "1.2.0", RegistryID: "team",
		SourceURL: "https://example.com/skills.git", SourceRef: "main", InstallEntry: "skills/go-pro",
		RegistryURL: "https://example.com/registry.json",
		Checksum: "abc123", SupportedAgents: []string{"codex"},
	}
	got, err := newMaterializer(nil).skillFromEntry(entry, filepath.Join(baseDir, "repo"))

	assert.NoErr(t, err)
	assert.Eq(t, "go-pro", got.ID)
	assert.Eq(t, "team", got.SourceID)
	assert.Eq(t, sourcepkg.TypeRegistry, got.SourceType)
	assert.Eq(t, "team/go-pro", got.SourceQualifiedName)
	assert.Eq(t, "skills/go-pro", got.InstallEntry)
	assert.Eq(t, "https://example.com/skills.git", got.SourceURL)
	assert.Eq(t, "main", got.SourceRef)
	assert.Eq(t, "go-pro", got.RegistryEntryID)
	assert.Eq(t, "https://example.com/registry.json", got.RegistryURL)
}

func TestMaterializer_MaterializeLocalSourceURLCopiesSnapshot(t *testing.T) {
	baseDir := t.TempDir()
	sourceRoot := filepath.Join(baseDir, "repo")
	targetRoot := filepath.Join(baseDir, "cache", "skills", "team", "go-pro", "1.0.0")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceRoot, "skills", "go-pro"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceRoot, "skills", "go-pro", "SKILL.md"), []byte("# Go Pro"), 0o644))

	got, err := newMaterializer(nil).Materialize(registry.SkillEntry{
		ID: "go-pro", Name: "Go Pro", Version: "1.0.0", RegistryID: "team",
		SourceURL: sourceRoot, InstallEntry: "skills/go-pro",
	}, targetRoot, gitx.SyncOptions{})

	assert.NoErr(t, err)
	assert.Eq(t, targetRoot, got.Path)
	assert.FileExists(t, filepath.Join(targetRoot, "skills", "go-pro", "SKILL.md"))
}
```

- [x] **Step 2: Write failing lock provenance test**

Add to `internal/app/installapp/service_test.go`:

```go
func TestService_InstallWritesRegistryProvenanceToLock(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	sourceDir := filepath.Join(baseDir, "cache", "go-pro")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Go Pro"), 0o644))

	_, err := NewService(lockFile).Install(skill.Skill{
		ID: "go-pro", Version: "1.2.0", SourceID: "team", SourceType: sourcepkg.TypeRegistry,
		RegistryEntryID: "go-pro", RegistryURL: "https://example.com/registry.json",
		SourceURL: "https://example.com/skills.git", SourceRef: "main",
		InstallEntry: ".", Path: sourceDir,
	}, "codex", agent.ScopeProject, baseDir, filepath.Join(baseDir, ".codex", "skills"))

	assert.NoErr(t, err)
	locks := mustLoadLockFile(t, NewService(lockFile), lockFile)
	assert.Eq(t, "registry", locks[baseDir][0].SourceType)
	assert.Eq(t, "go-pro", locks[baseDir][0].RegistryEntryID)
	assert.Eq(t, "https://example.com/skills.git", locks[baseDir][0].SourceURL)
}
```

- [x] **Step 3: Run tests to verify they fail**

Run:

```bash
go test ./internal/app/registryapp ./internal/app/installapp -run 'Test(Materializer_ToSkillMapsRegistryProvenance|Materializer_MaterializeLocalSourceURLCopiesSnapshot|Service_InstallWritesRegistryProvenanceToLock)' -v
```

Expected: FAIL because provenance fields, local snapshot materialization, and materializer do not exist.

- [x] **Step 4: Implement provenance fields and materializer**

Extend `skill.Skill` and `lock.Record` with:

```go
RegistryEntryID string `json:"registry_entry_id,omitempty"`
RegistryURL     string `json:"registry_url,omitempty"`
DownloadURL     string `json:"download_url,omitempty"`
SourceURL       string `json:"source_url,omitempty"`
SourceRef       string `json:"source_ref,omitempty"`
```

Update `installapp.installInto` and `ReinstallAtPath` to copy those fields from `skill.Skill` into `lock.Record`.

Create `internal/app/registryapp/materializer.go`:

```go
type gitSyncer interface {
	Sync(url, dir, ref string, opts gitx.SyncOptions) (string, error)
}

type materializer struct {
	git gitSyncer
}

func newMaterializer(git gitSyncer) *materializer {
	if git == nil {
		git = gitx.New("git")
	}
	return &materializer{git: git}
}

func (m *materializer) skillFromEntry(entry registry.SkillEntry, root string) (skill.Skill, error) {
	if strings.TrimSpace(entry.SourceURL) == "" {
		return skill.Skill{}, fmt.Errorf("registry skill source_url is required for install: %s/%s", entry.RegistryID, entry.ID)
	}
	return skill.Skill{
		ID:                  entry.ID,
		Name:                entry.Name,
		Description:         entry.Description,
		Version:             entry.Version,
		SupportedAgents:     append([]string(nil), entry.SupportedAgents...),
		SourceID:            entry.RegistryID,
		SourceName:          entry.RegistryID,
		SourceType:          source.TypeRegistry,
		QualifiedName:       entry.ID,
		SourceQualifiedName: entry.RegistryID + "/" + entry.ID,
		InstallEntry:        entry.InstallEntry,
		Path:                root,
		Checksum:            strings.TrimPrefix(entry.Checksum, "sha256:"),
		RegistryEntryID:     entry.ID,
		RegistryURL:         entry.RegistryURL,
		DownloadURL:         entry.DownloadURL,
		SourceURL:           entry.SourceURL,
		SourceRef:           entry.SourceRef,
	}, nil
}

func (m *materializer) Materialize(entry registry.SkillEntry, targetDir string, opts gitx.SyncOptions) (skill.Skill, error) {
	if strings.TrimSpace(entry.SourceURL) == "" {
		return skill.Skill{}, fmt.Errorf("registry skill source_url is required for install: %s/%s", entry.RegistryID, entry.ID)
	}
	resolvedRef := ""
	if sourceapp.IsGitURL(entry.SourceURL) {
		ref, err := m.git.Sync(entry.SourceURL, targetDir, entry.SourceRef, opts)
		if err != nil {
			return skill.Skill{}, err
		}
		resolvedRef = ref
	} else {
		info, err := os.Stat(entry.SourceURL)
		if err != nil {
			return skill.Skill{}, err
		}
		if !info.IsDir() {
			return skill.Skill{}, fmt.Errorf("registry skill source_url must be git URL or local directory: %s", entry.SourceURL)
		}
		if err := os.RemoveAll(targetDir); err != nil {
			return skill.Skill{}, err
		}
		if err := os.CopyFS(targetDir, os.DirFS(entry.SourceURL)); err != nil {
			return skill.Skill{}, err
		}
	}
	item, err := m.skillFromEntry(entry, targetDir)
	if err != nil {
		return skill.Skill{}, err
	}
	item.SourceResolvedRef = resolvedRef
	return item, nil
}
```

Add `Service.MaterializeSkill(selector string) (skill.Skill, error)`:

```go
func (s *Service) MaterializeSkill(selector string) (skill.Skill, error) {
	entry, err := s.InfoSkill(selector)
	if err != nil {
		return skill.Skill{}, err
	}
	data, err := s.load()
	if err != nil {
		return skill.Skill{}, err
	}
	targetDir := filepath.Join(data.RegistryCacheDir, "skills", entry.RegistryID, entry.ID, safeVersion(entry.Version))
	materializer := newMaterializer(nil)
	return materializer.Materialize(entry, targetDir, gitx.SyncOptions{ProxyURL: data.ProxyURL})
}

func safeVersion(version string) string {
	version = registry.NormalizeID(version)
	if version == "" {
		return "latest"
	}
	return version
}
```

- [x] **Step 5: Run materializer/install tests**

Run:

```bash
go test ./internal/app/registryapp ./internal/app/installapp
```

Expected: PASS.

- [x] **Step 6: Commit**

Run:

```bash
git add internal/app/registryapp/materializer.go internal/app/registryapp/materializer_test.go internal/domain/skill/model.go internal/domain/lock/model.go internal/app/installapp
git commit -m "feat(skillc): materialize registry skills"
```

## Task 5: Registry Install CLI

**Files:**
- Modify: `internal/cli/registry_cmd.go`
- Modify: `internal/cli/app_test.go`

- [ ] **Step 1: Write failing CLI install test**

Add to `internal/cli/app_test.go`:

```go
func TestRegistryCommandInstallInstallsSkillWithoutAddingSource(t *testing.T) {
	baseDir := t.TempDir()
	catalogPath := filepath.Join(baseDir, "registry.json")
	sourceRoot := filepath.Join(baseDir, "repo")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceRoot, "skills", "go-pro"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceRoot, "skills", "go-pro", "SKILL.md"), []byte("# Go Pro"), 0o644))
	assert.NoErr(t, os.WriteFile(catalogPath, []byte(fmt.Sprintf(`{"skills":[{"id":"go-pro","name":"Go Pro","version":"1.0.0","source_url":"%s","install_entry":"skills/go-pro","supported_agents":["universal"]}]}`, filepath.ToSlash(sourceRoot))), 0o644))

	runAppInDirWithStdout(t, baseDir, []string{"registry", "add", "--id", "team", "--name", "Team", catalogPath})
	runAppInDirWithStdout(t, baseDir, []string{"registry", "sync", "team"})
	output := runAppInDirWithStdout(t, baseDir, []string{"registry", "install", "team/go-pro", "--agent", "universal", "--scope", "project", "--yes"})

	assert.Contains(t, output, "Installed")
	assert.FileExists(t, filepath.Join(baseDir, ".agents", "skills", "go-pro", "SKILL.md"))
	config := loadTestConfig(t, filepath.Join(baseDir, "skillc.yaml"), baseDir)
	assert.Len(t, config.Sources, 0)
}
```

- [ ] **Step 2: Run CLI test to verify it fails**

Run:

```bash
go test ./internal/cli -run TestRegistryCommandInstallInstallsSkillWithoutAddingSource -v
```

Expected: FAIL because `registry install` does not exist.

- [ ] **Step 3: Implement `registry install` command**

Add `buildRegistryInstallCommand()`:

```go
func buildRegistryInstallCommand() *gcli.Command {
	var opts ManageOptions
	return &gcli.Command{
		Name: "install",
		Desc: "Install a skill from a registry result",
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
			opts.bindInstallModeFlags(c)
			c.BoolOpt(&opts.Yes, "yes", "y", false, "skip confirmation prompt")
			c.AddArg("skill", "registry skill target, e.g. team/go-pro", true)
		},
		Func: func(c *gcli.Command, _ []string) error {
			config, cwd, err := loadConfig()
			if err != nil {
				return err
			}
			item, err := newRegistryService().MaterializeSkill(c.Arg("skill").String())
			if err != nil {
				return err
			}
			if opts.UseCopy && opts.InstallMode != "" {
				return fmt.Errorf("--copy and --install-mode are mutually exclusive")
			}
			installMode := opts.resolveInstallMode(config)
			service := installapp.NewService(config.LockFile).WithInstallMode(installMode)
			ok, err := printInstallPlanAndConfirm([]skill.Skill{item}, opts)
			if err != nil || !ok {
				return err
			}
			result, err := service.RunResolved(config, installapp.InstallReq{
				SkillID: item.SourceQualifiedName,
				Agent: opts.Agent,
				Scope: opts.Scope,
				WorkDir: cwd,
			}, []skill.Skill{item}, nil)
			if err != nil {
				return err
			}
			printInstallResult(result)
			return nil
		},
	}
}
```

Register it in `buildRegistryCommand()`.

- [ ] **Step 4: Run CLI tests**

Run:

```bash
go test ./internal/cli
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/cli/registry_cmd.go internal/cli/app_test.go
git commit -m "feat(skillc): install registry skills"
```

## Task 6: Restore, Status, Update Check, and Update for Registry Records

**Files:**
- Create: `internal/app/registryapp/locked_resolver.go`
- Create: `internal/app/registryapp/locked_resolver_test.go`
- Modify: `internal/app/installapp/service.go`
- Modify: `internal/app/installapp/service_test.go`
- Modify: `internal/app/statusapp/service.go`
- Modify: `internal/app/statusapp/service_test.go`
- Modify: `internal/app/updateapp/service.go`
- Modify: `internal/app/updateapp/service_test.go`
- Modify: `internal/app/projectupdateapp/service.go`
- Modify: `internal/app/projectupdateapp/service_test.go`
- Modify: `internal/cli/manage_cmd.go`

- [ ] **Step 1: Write failing restore test**

Add to `internal/app/installapp/service_test.go`:

```go
func TestService_RestoreUsesResolverForRegistryRecords(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	sourceDir := filepath.Join(baseDir, "cache", "go-pro")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Go Pro"), 0o644))
	service := NewService(lockFile)
	config := testConfig(baseDir)
	config.LockFile = lockFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: ".agents"}
	assert.NoErr(t, service.store.Save(lockFile, lockpkg.File{
		baseDir: {{SkillID: "go-pro", SourceID: "team", SourceType: "registry", RegistryEntryID: "go-pro", InstallEntry: ".", Agents: []string{"universal"}}},
	}))
	service = service.WithRestoreResolver(func(record lockpkg.Record) (skill.Skill, bool, error) {
		return skill.Skill{ID: record.SkillID, SourceID: record.SourceID, SourceType: sourcepkg.TypeRegistry, InstallEntry: ".", Path: sourceDir}, true, nil
	})

	restored, err := service.WithRuntime(config, baseDir).Restore(map[string]string{})

	assert.NoErr(t, err)
	assert.Len(t, restored, 1)
	assert.FileExists(t, filepath.Join(baseDir, ".agents", "skills", "go-pro", "SKILL.md"))
}
```

- [ ] **Step 2: Write failing status/update tests**

Add to `internal/app/statusapp/service_test.go`:

```go
func TestService_RunMarksRegistrySkillInstalledFromRegistryCache(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeStatusDriftFixture(t, baseDir)
	cacheDir := filepath.Join(baseDir, "registry-cache")
	config.RegistryCacheDir = cacheDir
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))
	assert.NoErr(t, lockstore.NewStore().Save(config.LockFile, lockpkg.File{
		baseDir: {{
			SkillID: "go-pro", SourceID: "team", SourceType: "registry", RegistryEntryID: "go-pro",
			Version: "1.0.0", Checksum: "abc123", Agents: []string{"universal"},
		}},
	}))
	assert.NoErr(t, registrystore.NewStore().SaveFile(filepath.Join(cacheDir, "registry-index.json"), registrystore.File{
		Skills: []registry.SkillEntry{{
			ID: "go-pro", RegistryID: "team", Version: "1.0.0", Checksum: "abc123",
			SourceURL: "https://example.com/skills.git", InstallEntry: "skills/go-pro",
		}},
	}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	assert.Len(t, result.Items, 1)
	assert.Eq(t, StatusInstalled, result.Items[0].Status)
	assert.Eq(t, "team", result.Items[0].SourceID)
}
```

Add to `internal/app/updateapp/service_test.go`:

```go
type registryResolverStub struct {
	resolveFn func(record lockpkg.Record) (skill.Skill, bool, error)
}

func (s registryResolverStub) Resolve(record lockpkg.Record) (skill.Skill, bool, error) {
	return s.resolveFn(record)
}

func (s registryResolverStub) Latest(record lockpkg.Record) (skill.Skill, bool, error) {
	return s.resolveFn(record)
}

func TestService_RunUpdatesRegistrySkillFromRegistryResolver(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	projectKey := filepath.Join(baseDir, "projects", "alpha")
	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(projectKey, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectKey: {{
			SkillID: "go-pro", SourceID: "team", SourceType: "registry", RegistryEntryID: "go-pro",
			Version: "1.0.0", InstallEntry: ".", Agents: []string{"universal"},
		}},
	}))

	service := NewService(configFile, baseDir)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error {
		t.Fatalf("registry records must not call source sync, got %s", id)
		return nil
	}}
	service.registryResolver = registryResolverStub{resolveFn: func(record lockpkg.Record) (skill.Skill, bool, error) {
		return skill.Skill{
			ID: "go-pro", SourceID: "team", SourceType: sourcepkg.TypeRegistry,
			RegistryEntryID: "go-pro", Version: "2.0.0", InstallEntry: ".",
			Path: filepath.Join(baseDir, "registry-cache", "go-pro"),
		}, true, nil
	}}
	var installed skill.Skill
	service.newInstaller = func(path string, _ cfg.Config) reinstallService {
		assert.Eq(t, lockFile, path)
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error) {
			installed = item
			return installapp.RuntimeRecord{
				Record: lockpkg.Record{SkillID: item.ID, SourceID: item.SourceID, SourceType: string(item.SourceType), Version: item.Version},
				Agent: agentName, Scope: string(scope), InstalledPath: targetPath,
			}, nil
		}}
	}

	result, err := service.Run(Req{Scope: "project", WorkDir: baseDir, ProjectPaths: []string{projectKey}})

	assert.NoErr(t, err)
	assert.Len(t, result.Updated, 1)
	assert.Eq(t, sourcepkg.TypeRegistry, installed.SourceType)
	assert.Eq(t, "2.0.0", installed.Version)
	assert.Eq(t, "registry", result.Updated[0].SourceType)
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run:

```bash
go test ./internal/app/installapp ./internal/app/statusapp ./internal/app/updateapp -run 'TestService_(RestoreUsesResolverForRegistryRecords|RunMarksRegistrySkillInstalledFromRegistryCache|RunUpdatesRegistrySkillFromRegistryResolver)' -v
```

Expected: FAIL because restore/status/update are source-index-only.

- [ ] **Step 4: Implement registry lock resolver**

Create `internal/app/registryapp/locked_resolver.go`:

```go
type RecordResolver interface {
	Resolve(record lock.Record) (skill.Skill, bool, error)
	Latest(record lock.Record) (skill.Skill, bool, error)
}

type LockedResolver struct {
	service *Service
}

func NewLockedResolver(configFile string, baseDir string) *LockedResolver {
	return &LockedResolver{service: NewService(configFile, baseDir)}
}

func (r *LockedResolver) Resolve(record lock.Record) (skill.Skill, bool, error) {
	if record.SourceType != string(source.TypeRegistry) {
		return skill.Skill{}, false, nil
	}
	target := record.SourceID + "/" + firstNonEmpty(record.RegistryEntryID, record.SkillID)
	item, err := r.service.MaterializeSkill(target)
	return item, true, err
}

func (r *LockedResolver) Latest(record lock.Record) (skill.Skill, bool, error) {
	if record.SourceType != string(source.TypeRegistry) {
		return skill.Skill{}, false, nil
	}
	target := record.SourceID + "/" + firstNonEmpty(record.RegistryEntryID, record.SkillID)
	entry, err := r.service.InfoSkill(target)
	if err != nil {
		return skill.Skill{}, true, err
	}
	item, err := newMaterializer(nil).skillFromEntry(entry, "")
	return item, true, err
}
```

Modify `installapp.Service`:

```go
type RestoreResolver func(record lockpkg.Record) (skill.Skill, bool, error)

func (s *Service) WithRestoreResolver(resolver RestoreResolver) *Service
```

In `Restore`, if `record.SourceType == "registry"` or source path is missing, call the resolver. If resolver returns `handled=false`, preserve existing error behavior.

Modify `statusapp` to accept a registry latest resolver and classify registry records from registry cache before declaring orphan. `status` and `update --check` must use `Latest`, not `Resolve`, so check/plan operations do not clone or copy skill content.

Modify `updateapp` to:

- Group records by source type.
- Keep existing source sync for local/git.
- For registry records, materialize latest skill via registry resolver.
- Reuse `ReinstallAtPath` with materialized `skill.Skill`.

Modify `projectupdateapp` so cross-project update plans use the updated `statusapp` registry latest resolver, and confirmed cross-project runs continue to call `updateapp` with registry resolver support. Modify `buildInstallCommand` restore path to call `installapp.NewService(...).WithRestoreResolver(registryapp.NewLockedResolver(...).Resolve)` before `Run`; otherwise `skillc install` with no target cannot restore registry-installed records.

- [ ] **Step 5: Run registry lifecycle tests**

Run:

```bash
go test ./internal/app/registryapp ./internal/app/installapp ./internal/app/statusapp ./internal/app/updateapp
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```bash
git add internal/app/registryapp/locked_resolver.go internal/app/registryapp/locked_resolver_test.go internal/app/installapp internal/app/statusapp internal/app/updateapp
git commit -m "feat(skillc): restore and update registry skills"
```

## Task 7: CLI Search/Info UX and Documentation

**Files:**
- Modify: `internal/cli/registry_cmd.go`
- Modify: `internal/cli/app_test.go`
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/TODO.md`
- Modify: `docs/design/skillc-v0-enhance-design.md`
- Modify: `docs/superpowers/plans/2026-06-16-skillc-v0-phase9-registry-skill-search-install.md`

- [ ] **Step 1: Write failing CLI UX tests**

Add to `internal/cli/app_test.go`:

```go
func TestRegistryCommandSearchDefaultsToSkills(t *testing.T) {
	baseDir := t.TempDir()
	catalogPath := filepath.Join(baseDir, "registry.json")
	assert.NoErr(t, os.WriteFile(catalogPath, []byte(`{
		"skills":[{"id":"go-pro","name":"Go Pro","version":"1.0.0","source_url":"https://example.com/skills.git","install_entry":"skills/go-pro","tags":["go"]}],
		"sources":[{"id":"gstack","name":"GStack Skills","type":"git","url":"https://example.com/gstack.git","tags":["go"]}]
	}`), 0o644))
	runAppInDirWithStdout(t, baseDir, []string{"registry", "add", "--id", "team", "--name", "Team", catalogPath})
	runAppInDirWithStdout(t, baseDir, []string{"registry", "sync", "team"})

	output := runAppInDirWithStdout(t, baseDir, []string{"registry", "search", "go"})

	assert.Contains(t, output, "go-pro")
	assert.Contains(t, output, "Go Pro")
	assert.NotContains(t, output, "gstack")
}

func TestRegistryCommandSearchKindSourceStillShowsSourceCatalog(t *testing.T) {
	baseDir := t.TempDir()
	catalogPath := filepath.Join(baseDir, "registry.json")
	assert.NoErr(t, os.WriteFile(catalogPath, []byte(`{
		"skills":[{"id":"go-pro","name":"Go Pro","version":"1.0.0","source_url":"https://example.com/skills.git","install_entry":"skills/go-pro","tags":["go"]}],
		"sources":[{"id":"gstack","name":"GStack Skills","type":"git","url":"https://example.com/gstack.git","tags":["go"]}]
	}`), 0o644))
	runAppInDirWithStdout(t, baseDir, []string{"registry", "add", "--id", "team", "--name", "Team", catalogPath})
	runAppInDirWithStdout(t, baseDir, []string{"registry", "sync", "team"})

	output := runAppInDirWithStdout(t, baseDir, []string{"registry", "search", "gstack", "--kind", "source"})

	assert.Contains(t, output, "gstack")
	assert.Contains(t, output, "GStack Skills")
	assert.Contains(t, output, "git")
	assert.NotContains(t, output, "go-pro")
}
```

- [ ] **Step 2: Run CLI UX tests to verify they fail**

Run:

```bash
go test ./internal/cli -run 'TestRegistryCommandSearch(Default|KindSource)' -v
```

Expected: FAIL because `registry search` currently only shows source entries and has no `--kind`.

- [ ] **Step 3: Update registry CLI UX**

Modify `registry search`:

```text
registry search <keyword> --kind skill|source
```

Behavior:

- default `--kind skill`
- skill table heads: `Registry`, `ID`, `Name`, `Version`, `Agents`, `Tags`, `Source`
- source table keeps P8 output

Modify `registry info`:

```text
registry info <target> --kind skill|source
```

Default kind is `skill`; if no skill match exists, try source match and print a hint:

```text
source result found; use `registry add-source <target>` to register it
```

- [ ] **Step 4: Update README and design docs**

Document:

```bash
skillc registry search go
skillc registry info team/go-pro
skillc registry install team/go-pro --agent codex --scope project
skillc registry search gstack --kind source
skillc registry add-source team/gstack --sync
```

Clarify:

- `registry install` is for single Skill results.
- `registry add-source` is optional and intended for long-lived source subscription.
- P9 implements generic JSON skill catalog; public site adapters are next.

- [ ] **Step 5: Run docs/CLI focused tests**

Run:

```bash
go test ./internal/domain/registry ./internal/infra/registrystore ./internal/app/registryapp ./internal/cli
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```bash
git add internal/cli/registry_cmd.go internal/cli/app_test.go README.md README.zh-CN.md docs/TODO.md docs/design/skillc-v0-enhance-design.md docs/superpowers/plans/2026-06-16-skillc-v0-phase9-registry-skill-search-install.md
git commit -m "docs(skillc): document registry skill install workflow"
```

## Task 8: Final Verification

**Files:**
- Modify: `docs/superpowers/plans/2026-06-16-skillc-v0-phase9-registry-skill-search-install.md`

- [ ] **Step 1: Run focused verification**

Run:

```bash
go test ./internal/domain/source ./internal/domain/registry ./internal/infra/registrystore ./internal/app/registryapp ./internal/app/installapp ./internal/app/statusapp ./internal/app/updateapp ./internal/cli
```

Expected: PASS.

- [ ] **Step 2: Run full verification**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 3: Update verification record**

Add a `## Verification` section to this plan with:

```markdown
## Verification

- `go test ./internal/domain/source ./internal/domain/registry ./internal/infra/registrystore ./internal/app/registryapp ./internal/app/installapp ./internal/app/statusapp ./internal/app/updateapp ./internal/cli`
- `go test ./...`
```

- [ ] **Step 4: Commit verification record**

Run:

```bash
git add docs/superpowers/plans/2026-06-16-skillc-v0-phase9-registry-skill-search-install.md
git commit -m "docs(skillc): complete phase 9 registry plan verification"
```

## Self-Review Checklist

- [ ] `registry search` defaults to Skill results.
- [ ] `registry search --kind source` preserves P8 source catalog behavior.
- [ ] Generic JSON catalog supports both `skills` and `sources`.
- [ ] `registry install <registry>/<skill>` installs without adding a source config entry.
- [ ] Registry-installed lock records use `source_type=registry`.
- [ ] Lock records preserve registry provenance fields.
- [ ] `install` restore handles registry records.
- [ ] `status` / `update --check` do not mark registry records as orphan when registry cache has the skill.
- [ ] `update` can reinstall registry skills from materialized cache.
- [ ] README, TODO, design doc, and this plan agree that public site adapters are not implemented in P9.
- [ ] `go test ./...` passes before claiming Phase 9 complete.

## Remaining After Phase 9

- skills.sh / skillsmp / skillsllm real adapter discovery and implementation.
- Archive `download_url` materialization for zip/tar.gz packages.
- Registry trust model: signature, checksum policy, allow/deny source policy.
- Web Registry search/install/add-source UI.
- Registry profile/bundle search and apply.
- Global available registries list and `registry enable <id>` UX.
