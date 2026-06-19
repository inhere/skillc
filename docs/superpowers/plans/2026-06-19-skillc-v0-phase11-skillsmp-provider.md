# Skillc v0 Phase 11 SkillsMP Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

| 修订时间 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-19 | v0.1 | Codex | 首版 SkillsMP Registry provider adapter 实施计划 |

**Goal:** 实现第一个真实 Registry provider：`SkillsMP` 远程搜索，并把结果映射到现有 registry skill install 链路。

**Architecture:** 只新增最小 `type: provider` + `provider: skillsmp` 能力，不做通用 provider 框架。SkillsMP 搜索结果在 `registryapp` 内转换为现有 `registry.SkillEntry`，合并写入 `registry-index.json`，后续 `info/install/Web` 继续复用现有 cache/materialize/install。

**Tech Stack:** Go stdlib `net/http`、`encoding/json`、`net/url`；现有 `gookit/gcli` CLI、`gookit/goutil/x/assert` 测试断言、`registrystore` cache。

---

相关文档：

- 设计：`docs/superpowers/specs/2026-06-19-skillc-v0-phase11-skillsmp-provider-design.md`
- 总设计：`docs/design/skillc-v0-enhance-design.md`
- Phase 10 计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase10-web-registry-archive.md`

## File Structure

- Modify `internal/domain/registry/model.go`
  - 增加 `TypeProvider`、`Provider` 字段、`NewWithProvider`。
  - 保留 `New` 现有行为，避免破坏 generic HTTP registry。
- Modify `internal/domain/registry/model_test.go`
  - 覆盖 provider registry 构造和非法 provider。
- Create `internal/app/registryapp/skillsmp.go`
  - 负责请求 SkillsMP API、解析响应、转换 GitHub tree URL。
- Create `internal/app/registryapp/skillsmp_test.go`
  - 用 `httptest.Server` 覆盖映射、坏 URL 跳过、空结果错误。
- Modify `internal/app/registryapp/service.go`
  - `SearchSkills` 在指定 provider registry 时走远程搜索并合并 cache。
  - `Sync` 对 provider registry 返回清晰错误。
  - 增加 cache merge helper。
- Modify `internal/app/registryapp/service_test.go`
  - 覆盖 provider search cache merge、空 keyword、provider sync error。
- Modify `internal/cli/registry_cmd.go`
  - `registry add` 增加 `--provider`。
- Modify `internal/cli/app_test.go`
  - 覆盖 `registry add --provider skillsmp` 和 provider search 输出。
- Modify `internal/app/webapp/manager_server_test.go`
  - 覆盖 `/api/registry/skills?registry=skillsmp&keyword=go`。
- Modify `README.md` and `README.zh-CN.md`
  - 记录 SkillsMP provider 使用方式。
- Modify `docs/design/skillc-v0-enhance-design.md`
  - 追加 Phase 11 实施计划链接。
- Modify `docs/superpowers/specs/2026-06-19-skillc-v0-phase11-skillsmp-provider-design.md`
  - 反向链接本实施计划。

## Task 1: Add Provider Registry Model

**Files:**
- Modify: `internal/domain/registry/model.go`
- Modify: `internal/domain/registry/model_test.go`

- [ ] **Step 1: Write failing registry model tests**

Add tests:

```go
func TestNewRegistryFromProviderURL(t *testing.T) {
	got, err := NewWithProvider("skillsmp", "SkillsMP", "https://skillsmp.com/", "skillsmp")

	assert.NoErr(t, err)
	assert.Eq(t, "skillsmp", got.ID)
	assert.Eq(t, "SkillsMP", got.Name)
	assert.Eq(t, TypeProvider, got.Type)
	assert.Eq(t, "skillsmp", got.Provider)
	assert.Eq(t, "https://skillsmp.com", got.URL)
}

func TestNewRegistryRejectsUnsupportedProvider(t *testing.T) {
	_, err := NewWithProvider("bad", "Bad", "https://example.com", "unknown")

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "unsupported registry provider")
}
```

- [ ] **Step 2: Run failing model tests**

Run:

```bash
go test ./internal/domain/registry -run 'TestNewRegistryFromProviderURL|TestNewRegistryRejectsUnsupportedProvider' -v
```

Expected: fail because `NewWithProvider` and `TypeProvider` do not exist.

- [ ] **Step 3: Implement minimal provider model**

Update `internal/domain/registry/model.go`:

```go
const (
	TypeLocal    Type = "local"
	TypeHTTP     Type = "http"
	TypeProvider Type = "provider"
)

type Registry struct {
	ID           string `yaml:"id" json:"id"`
	Name         string `yaml:"name,omitempty" json:"name,omitempty"`
	Type         Type   `yaml:"type" json:"type"`
	Provider     string `yaml:"provider,omitempty" json:"provider,omitempty"`
	Path         string `yaml:"path,omitempty" json:"path,omitempty"`
	URL          string `yaml:"url,omitempty" json:"url,omitempty"`
	LastSyncAt   string `yaml:"last_sync_at,omitempty" json:"last_sync_at,omitempty"`
	Status       string `yaml:"status,omitempty" json:"status,omitempty"`
	ErrorMessage string `yaml:"error_message,omitempty" json:"error_message,omitempty"`
}

func New(id string, name string, value string) (Registry, error) {
	return NewWithProvider(id, name, value, "")
}

func NewWithProvider(id string, name string, value string, provider string) (Registry, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Registry{}, fmt.Errorf("registry path or url is required")
	}
	if name == "" {
		name = strings.TrimSpace(id)
	}
	if name == "" {
		name = registryNameFromValue(value)
	}
	id = NormalizeID(firstNonEmpty(id, name))
	if id == "" {
		return Registry{}, fmt.Errorf("registry id is required")
	}

	provider = NormalizeID(provider)
	if provider != "" {
		if provider != "skillsmp" {
			return Registry{}, fmt.Errorf("unsupported registry provider: %s", provider)
		}
		if !IsHTTPURL(value) {
			return Registry{}, fmt.Errorf("provider registry url must be http URL")
		}
		return Registry{ID: id, Name: name, Type: TypeProvider, Provider: provider, URL: strings.TrimRight(value, "/")}, nil
	}

	if IsHTTPURL(value) {
		return Registry{ID: id, Name: name, Type: TypeHTTP, URL: value}, nil
	}

	absPath, err := filepath.Abs(value)
	if err != nil {
		return Registry{}, err
	}
	return Registry{ID: id, Name: name, Type: TypeLocal, Path: filepath.Clean(absPath)}, nil
}
```

- [ ] **Step 4: Run model tests**

Run:

```bash
go test ./internal/domain/registry -v
```

Expected: pass.

- [ ] **Step 5: Commit model slice**

```bash
git add internal/domain/registry/model.go internal/domain/registry/model_test.go
git commit -m "feat(skillc): add provider registry model"
```

## Task 2: Add SkillsMP Response Mapper

**Files:**
- Create: `internal/app/registryapp/skillsmp.go`
- Create: `internal/app/registryapp/skillsmp_test.go`

- [ ] **Step 1: Write failing SkillsMP mapper tests**

Create `internal/app/registryapp/skillsmp_test.go`:

```go
package registryapp

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/skillc/internal/domain/registry"
)

func TestSearchSkillsMPMapsGitHubTreeURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Eq(t, "/api/v1/skills/search", r.URL.Path)
		assert.Eq(t, "go", r.URL.Query().Get("q"))
		_, _ = w.Write([]byte(`{"success":true,"data":{"skills":[{"id":"owner-repo-skills-go-skill-md","name":"go","author":"Owner","description":"Go helper","githubUrl":"https://github.com/Owner/Repo/tree/main/skills/go","skillUrl":"https://skillsmp.com/creators/owner/repo/skills-go","stars":3,"updatedAt":"1781679284"}]}}`))
	}))
	defer server.Close()

	got, err := searchSkillsMP(&http.Client{Timeout: time.Second}, registry.Registry{ID: "skillsmp", Type: registry.TypeProvider, Provider: "skillsmp", URL: server.URL}, "go")

	assert.NoErr(t, err)
	assert.Len(t, got, 1)
	assert.Eq(t, "owner-repo-skills-go-skill-md", got[0].ID)
	assert.Eq(t, "go", got[0].Name)
	assert.Eq(t, "Go helper", got[0].Description)
	assert.Eq(t, "https://github.com/Owner/Repo.git", got[0].SourceURL)
	assert.Eq(t, "main", got[0].SourceRef)
	assert.Eq(t, "skills/go", got[0].InstallEntry)
	assert.Eq(t, "https://skillsmp.com/creators/owner/repo/skills-go", got[0].Homepage)
	assert.Eq(t, "skillsmp", got[0].RegistryID)
	assert.Contains(t, got[0].Tags, "skillsmp")
	assert.Contains(t, got[0].Tags, "author:Owner")
}

func TestSearchSkillsMPSkipsUninstallableResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":{"skills":[{"id":"bad","name":"bad","githubUrl":"https://github.com/Owner/Repo"}]}}`))
	}))
	defer server.Close()

	_, err := searchSkillsMP(&http.Client{Timeout: time.Second}, registry.Registry{ID: "skillsmp", Type: registry.TypeProvider, Provider: "skillsmp", URL: server.URL}, "go")

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "no installable skillsmp results")
}
```

- [ ] **Step 2: Run failing mapper tests**

Run:

```bash
go test ./internal/app/registryapp -run 'TestSearchSkillsMP' -v
```

Expected: fail because `searchSkillsMP` does not exist.

- [ ] **Step 3: Implement SkillsMP mapper**

Create `internal/app/registryapp/skillsmp.go` with stdlib only:

```go
package registryapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/inhere/skillc/internal/domain/registry"
)

type skillsMPResp struct {
	Success bool `json:"success"`
	Data    struct {
		Skills []skillsMPItem `json:"skills"`
	} `json:"data"`
	Message string `json:"message"`
}

type skillsMPItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Author      string `json:"author"`
	Description string `json:"description"`
	GitHubURL   string `json:"githubUrl"`
	SkillURL    string `json:"skillUrl"`
}

func searchSkillsMP(client *http.Client, item registry.Registry, keyword string) ([]registry.SkillEntry, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("skillsmp search keyword is required")
	}
	base := strings.TrimRight(item.URL, "/")
	reqURL := base + "/api/v1/skills/search?q=" + url.QueryEscape(keyword) + "&page=1&limit=50"
	resp, err := client.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("skillsmp search failed: HTTP %d", resp.StatusCode)
	}
	var payload skillsMPResp
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("skillsmp search response is invalid: %w", err)
	}
	if !payload.Success {
		if payload.Message != "" {
			return nil, fmt.Errorf("skillsmp search failed: %s", payload.Message)
		}
		return nil, fmt.Errorf("skillsmp search failed")
	}

	var out []registry.SkillEntry
	for _, row := range payload.Data.Skills {
		entry, ok := skillsMPEntry(row, item)
		if ok {
			out = append(out, entry)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no installable skillsmp results for %q", keyword)
	}
	return out, nil
}

func skillsMPEntry(row skillsMPItem, item registry.Registry) (registry.SkillEntry, bool) {
	sourceURL, ref, installEntry, ok := parseGitHubTree(row.GitHubURL)
	if !ok {
		return registry.SkillEntry{}, false
	}
	entry, err := registry.NormalizeSkillEntry(registry.SkillEntry{
		ID:           firstNonEmpty(row.ID, row.Name),
		Name:         row.Name,
		Description:  row.Description,
		SourceURL:    sourceURL,
		SourceRef:    ref,
		InstallEntry: installEntry,
		Homepage:     row.SkillURL,
		RegistryURL:  strings.TrimRight(item.URL, "/"),
		Tags:         skillsMPTags(row.Author),
	}, item.ID)
	return entry, err == nil
}

func parseGitHubTree(raw string) (sourceURL string, ref string, installEntry string, ok bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || strings.ToLower(u.Host) != "github.com" {
		return "", "", "", false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 5 || parts[2] != "tree" {
		return "", "", "", false
	}
	ref = parts[3]
	installEntry = path.Join(parts[4:]...)
	if ref == "" || installEntry == "." || installEntry == "" {
		return "", "", "", false
	}
	return fmt.Sprintf("https://github.com/%s/%s.git", parts[0], parts[1]), ref, installEntry, true
}

func skillsMPTags(author string) []string {
	tags := []string{"skillsmp"}
	author = strings.TrimSpace(author)
	if author != "" {
		tags = append(tags, "author:"+author)
	}
	return tags
}
```

- [ ] **Step 4: Run mapper tests**

Run:

```bash
go test ./internal/app/registryapp -run 'TestSearchSkillsMP|TestSearchSkillsMPSkipsUninstallableResults' -v
```

Expected: pass.

- [ ] **Step 5: Commit mapper slice**

```bash
git add internal/app/registryapp/skillsmp.go internal/app/registryapp/skillsmp_test.go
git commit -m "feat(skillc): map skillsmp search results"
```

## Task 3: Wire Provider Search Into Registry Service

**Files:**
- Modify: `internal/app/registryapp/service.go`
- Modify: `internal/app/registryapp/service_test.go`

- [ ] **Step 1: Write failing service tests**

Append to `internal/app/registryapp/service_test.go`:

```go
func TestService_SearchSkillsProviderCachesSkillsMPResults(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeRegistryAppConfig(t, baseDir)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":{"skills":[{"id":"owner-repo-skills-go-skill-md","name":"go","author":"Owner","description":"Go helper","githubUrl":"https://github.com/Owner/Repo/tree/main/skills/go","skillUrl":"https://skillsmp.com/creators/owner/repo/skills-go"}]}}`))
	}))
	defer server.Close()
	service := NewService(configFile, baseDir)
	_, err := service.Add(AddReq{ID: "skillsmp", Name: "SkillsMP", Value: server.URL, Provider: "skillsmp"})
	assert.NoErr(t, err)

	results, err := service.SearchSkills(SearchReq{Keyword: "go", RegistryID: "skillsmp"})

	assert.NoErr(t, err)
	assert.Len(t, results, 1)
	assert.Eq(t, "https://github.com/Owner/Repo.git", results[0].SourceURL)

	cached, err := service.InfoSkill("skillsmp/owner-repo-skills-go-skill-md")
	assert.NoErr(t, err)
	assert.Eq(t, "skills/go", cached.InstallEntry)
}

func TestService_SearchSkillsProviderRequiresKeyword(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeRegistryAppConfig(t, baseDir)
	service := NewService(configFile, baseDir)
	_, err := service.Add(AddReq{ID: "skillsmp", Value: "https://skillsmp.com", Provider: "skillsmp"})
	assert.NoErr(t, err)

	_, err = service.SearchSkills(SearchReq{RegistryID: "skillsmp"})

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "keyword is required")
}

func TestService_SyncProviderRegistryReturnsKeywordHint(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeRegistryAppConfig(t, baseDir)
	service := NewService(configFile, baseDir)
	_, err := service.Add(AddReq{ID: "skillsmp", Value: "https://skillsmp.com", Provider: "skillsmp"})
	assert.NoErr(t, err)

	err = service.Sync("skillsmp")

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "does not support sync without keyword")
}
```

- [ ] **Step 2: Run failing service tests**

Run:

```bash
go test ./internal/app/registryapp -run 'TestService_SearchSkillsProvider|TestService_SyncProviderRegistry' -v
```

Expected: fail because `AddReq.Provider` and provider service path are missing.

- [ ] **Step 3: Implement service wiring**

Make these minimal changes in `internal/app/registryapp/service.go`:

```go
type AddReq struct {
	ID       string
	Name     string
	Value    string
	Provider string
}
```

In `Add`:

```go
item, err := registry.NewWithProvider(req.ID, req.Name, req.Value, req.Provider)
```

In `fetchCatalog`:

```go
case registry.TypeProvider:
	return registry.Catalog{}, fmt.Errorf("provider registry does not support sync without keyword; use registry search <keyword> --registry %s", item.ID)
```

At the top of `SearchSkills`, before loading cache:

```go
if strings.TrimSpace(req.RegistryID) != "" {
	if item, ok, err := s.findRegistry(req.RegistryID); err != nil {
		return nil, err
	} else if ok && item.Type == registry.TypeProvider {
		return s.searchProviderSkills(item, req.Keyword)
	}
}
```

Add helpers:

```go
func (s *Service) findRegistry(id string) (registry.Registry, bool, error) {
	data, err := s.load()
	if err != nil {
		return registry.Registry{}, false, err
	}
	id = strings.ToLower(strings.TrimSpace(id))
	for _, item := range data.Registries {
		if strings.ToLower(item.ID) == id {
			return item, true, nil
		}
	}
	return registry.Registry{}, false, nil
}

func (s *Service) searchProviderSkills(item registry.Registry, keyword string) ([]registry.SkillEntry, error) {
	if strings.TrimSpace(keyword) == "" {
		return nil, fmt.Errorf("provider registry search keyword is required")
	}
	switch item.Provider {
	case "skillsmp":
		results, err := searchSkillsMP(s.client, item, keyword)
		if err != nil {
			return nil, err
		}
		if err := s.mergeCachedSkills(item.ID, results); err != nil {
			return nil, err
		}
		sortSkillEntries(results)
		return results, nil
	default:
		return nil, fmt.Errorf("unsupported registry provider: %s", item.Provider)
	}
}

func (s *Service) mergeCachedSkills(registryID string, results []registry.SkillEntry) error {
	data, err := s.load()
	if err != nil {
		return err
	}
	current, err := s.cache.LoadFile(registryCachePath(data))
	if err != nil {
		return err
	}
	byKey := make(map[string]registry.SkillEntry, len(current.Skills)+len(results))
	for _, entry := range current.Skills {
		byKey[entry.RegistryID+"/"+entry.ID] = entry
	}
	for _, entry := range results {
		byKey[registryID+"/"+entry.ID] = entry
	}
	merged := make([]registry.SkillEntry, 0, len(byKey))
	for _, entry := range byKey {
		merged = append(merged, entry)
	}
	sortSkillEntries(merged)
	return s.cache.SaveFile(registryCachePath(data), registrystore.File{Skills: merged, Sources: current.Sources})
}
```

- [ ] **Step 4: Run service tests**

Run:

```bash
go test ./internal/domain/registry ./internal/app/registryapp -v
```

Expected: pass.

- [ ] **Step 5: Commit service slice**

```bash
git add internal/app/registryapp/service.go internal/app/registryapp/service_test.go
git commit -m "feat(skillc): search skillsmp provider registries"
```

## Task 4: Add CLI Provider Flag

**Files:**
- Modify: `internal/cli/registry_cmd.go`
- Modify: `internal/cli/app_test.go`

- [ ] **Step 1: Write failing CLI tests**

Append near existing registry CLI tests in `internal/cli/app_test.go`:

```go
func TestRegistryCommandAddProviderSkillsMP(t *testing.T) {
	baseDir := t.TempDir()

	output := runAppInDirWithStdout(t, baseDir, []string{"registry", "add", "https://skillsmp.com", "--id", "skillsmp", "--name", "SkillsMP", "--provider", "skillsmp"})

	xassert.Contains(t, output, "registry added: skillsmp")
	config, err := configstore.NewYAMLStore().Load(filepath.Join(baseDir, "skillc.yaml"), baseDir)
	xassert.NoErr(t, err)
	xassert.Len(t, config.Registries, 1)
	xassert.Eq(t, registry.TypeProvider, config.Registries[0].Type)
	xassert.Eq(t, "skillsmp", config.Registries[0].Provider)
}

func TestRegistryCommandSearchProviderSkillsMP(t *testing.T) {
	baseDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":{"skills":[{"id":"owner-repo-skills-go-skill-md","name":"go","author":"Owner","description":"Go helper","githubUrl":"https://github.com/Owner/Repo/tree/main/skills/go","skillUrl":"https://skillsmp.com/creators/owner/repo/skills-go"}]}}`))
	}))
	defer server.Close()
	runAppInDirWithStdout(t, baseDir, []string{"registry", "add", server.URL, "--id", "skillsmp", "--provider", "skillsmp"})

	output := runAppInDirWithStdout(t, baseDir, []string{"registry", "search", "go", "--registry", "skillsmp"})

	xassert.Contains(t, output, "owner-repo-skills-go-skill-md")
	xassert.Contains(t, output, "https://github.com/Owner/Repo.git")
}
```

Add imports if missing:

```go
import (
	"net/http"
	"net/http/httptest"

	"github.com/inhere/skillc/internal/domain/registry"
)
```

- [ ] **Step 2: Run failing CLI tests**

Run:

```bash
go test ./internal/cli -run 'TestRegistryCommand(AddProviderSkillsMP|SearchProviderSkillsMP)' -v
```

Expected: fail because `--provider` is not registered.

- [ ] **Step 3: Implement CLI flag**

In `buildRegistryAddCommand`, add:

```go
var provider string
```

In `Config`:

```go
c.StrOpt(&provider, "provider", "", "", "registry provider adapter, e.g. skillsmp")
```

In `AddReq`:

```go
Provider: provider,
```

- [ ] **Step 4: Run CLI tests**

Run:

```bash
go test ./internal/cli -run 'TestRegistryCommand(AddProviderSkillsMP|SearchProviderSkillsMP|AddSyncSearchAndAddSource|SearchDefaultsToSkills)' -v
```

Expected: pass.

- [ ] **Step 5: Commit CLI slice**

```bash
git add internal/cli/registry_cmd.go internal/cli/app_test.go
git commit -m "feat(skillc): add skillsmp registry cli"
```

## Task 5: Cover Web Registry Provider Search

**Files:**
- Modify: `internal/app/webapp/manager_server_test.go`

- [ ] **Step 1: Write failing Web test**

Append near `TestManagerServerRegistryQueryRoutes`:

```go
func TestManagerServerRegistrySkillsProviderRoute(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeWebManagerFixture(t, baseDir)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":{"skills":[{"id":"owner-repo-skills-go-skill-md","name":"go","author":"Owner","description":"Go helper","githubUrl":"https://github.com/Owner/Repo/tree/main/skills/go","skillUrl":"https://skillsmp.com/creators/owner/repo/skills-go"}]}}`))
	}))
	defer server.Close()
	config.RegistryCacheDir = filepath.Join(baseDir, "cache", "registry")
	config.Registries = []registry.Registry{{
		ID: "skillsmp", Name: "SkillsMP", Type: registry.TypeProvider, Provider: "skillsmp", URL: server.URL,
	}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	manager := NewManagerServer(configFile, baseDir)

	rec := performManagerRequest(manager, http.MethodGet, "/api/registry/skills?keyword=go&registry=skillsmp")

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"id":"owner-repo-skills-go-skill-md"`)
	assert.Contains(t, rec.Body.String(), `"source_url":"https://github.com/Owner/Repo.git"`)
}
```

- [ ] **Step 2: Run Web test**

Run:

```bash
go test ./internal/app/webapp -run 'TestManagerServerRegistrySkillsProviderRoute' -v
```

Expected: pass once Task 3 is complete. If it fails because of imports, add existing package imports only.

- [ ] **Step 3: Commit Web test slice**

```bash
git add internal/app/webapp/manager_server_test.go
git commit -m "test(skillc): cover web skillsmp registry search"
```

## Task 6: Update Docs And Run Full Verification

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/design/skillc-v0-enhance-design.md`
- Modify: `docs/superpowers/specs/2026-06-19-skillc-v0-phase11-skillsmp-provider-design.md`
- Modify: `docs/superpowers/plans/2026-06-19-skillc-v0-phase11-skillsmp-provider.md`

- [ ] **Step 1: Update README Registry examples**

In `README.md`, extend the registry block with:

```markdown
skillc registry add https://skillsmp.com --id skillsmp --name SkillsMP --provider skillsmp
skillc registry search go --registry skillsmp
```

Add one short paragraph:

```markdown
SkillsMP is the first built-in provider adapter. Unlike generic JSON registries, it performs remote keyword search and caches returned Skill results locally so `registry info` and `registry install` can reuse the normal Registry install flow.
```

In `README.zh-CN.md`, add the Chinese equivalent:

```markdown
skillc registry add https://skillsmp.com --id skillsmp --name SkillsMP --provider skillsmp
skillc registry search go --registry skillsmp
```

```markdown
SkillsMP 是第一个内置 provider adapter。它不同于 generic JSON registry：搜索时会按关键词请求远程站点，并把返回的 Skill 结果缓存到本地，让 `registry info` 和 `registry install` 继续复用普通 Registry 安装链路。
```

- [ ] **Step 2: Update design/spec/plan status**

In `docs/design/skillc-v0-enhance-design.md`:

- Add Phase 11 implementation plan link.
- After implementation, update Phase 11 status to mention SkillsMP provider adapter is complete.

In `docs/superpowers/specs/2026-06-19-skillc-v0-phase11-skillsmp-provider-design.md`:

- Add related implementation plan link.

In this plan:

- Mark completed checkboxes as implementation progresses.

- [ ] **Step 3: Run focused tests**

Run:

```bash
go test ./internal/domain/registry ./internal/app/registryapp ./internal/cli ./internal/app/webapp
```

Expected: pass.

- [ ] **Step 4: Run full test suite**

Run:

```bash
go test ./...
```

Expected: pass.

- [ ] **Step 5: Commit docs and plan completion**

```bash
git add README.md README.zh-CN.md docs/design/skillc-v0-enhance-design.md docs/superpowers/specs/2026-06-19-skillc-v0-phase11-skillsmp-provider-design.md docs/superpowers/plans/2026-06-19-skillc-v0-phase11-skillsmp-provider.md
git commit -m "docs(skillc): document skillsmp registry provider"
```

## Self-Review

- Spec coverage: plan covers provider config, SkillsMP mapping, provider search cache, `sync` boundary, CLI, Web route, docs, and `go test ./...`.
- Intentional non-scope: no skills.sh adapter, no SkillsLLM adapter, no provider factory/interface, no auth/rate-limit config, no full provider sync.
- Risk to watch during implementation: `parseGitHubTree` only supports normal `tree/<ref>/<path>` URLs where `<ref>` is one path segment. If SkillsMP returns branch names containing `/`, install mapping will need a GitHub API or detail endpoint check later.
