# Skillc v0 Phase 10 Web Registry and Archive Download Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 补齐 registry skill archive `download_url` 下载安装能力，并在 `skillc web` 中提供 Registry 搜索、同步、安装和 add-source 的当前项目管理入口。

**Architecture:** 先在 `registryapp` materializer 中把 archive 下载/解压封装成独立可测单元，再复用 P9 已有 registry skill cache、`installapp.RunResolved` 和 lock provenance。Web 层新增 Registry query/action adapter，继续遵循现有 plan-first + `confirm:true` + history 记录模式，前端沿用 `manager_static.go` 的单页管理后台结构。

**Tech Stack:** Go, `net/http`, `archive/zip`, `archive/tar`, `compress/gzip`, existing `registryapp` / `installapp` / `webapp`, embedded static HTML/JS, Go unit tests with `github.com/gookit/goutil/x/assert`, final verification via `go test ./...`.

---

| 修订时间 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-16 | v0.1 | Codex | 输出 P10 Web Registry 与 archive download 实施计划 |

相关文档：

- P10 设计：`docs/superpowers/specs/2026-06-16-skillc-v0-phase10-web-registry-archive-design.md`
- 总设计：`docs/design/skillc-v0-enhance-design.md`
- Phase 9 计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase9-registry-skill-search-install.md`
- PRD：`docs/prd.md`
- MVP 架构：`docs/mvp-arch.md`

## Scope

P10 做：

- zip / tar.gz / tgz archive `download_url` materialization。
- archive SHA-256 checksum 校验，支持 `sha256:<hex>` 和 raw hex。
- archive path traversal 防护。
- local registry catalog 相对 `download_url` 路径解析。
- Web Registry 页面：list registries、search skills、search sources、sync registry/all、install registry skill、add source from registry source result。
- Web plan/run API 和 history。
- README / 设计文档更新。

P10 不做：

- skills.sh / SkillsMP / SkillsLLM 真实 provider adapter。
- Web registry add/remove。
- 跨项目 registry install。
- registry auth、签名、评分、审核。
- archive cache GC。

## File Structure

新增文件：

- `internal/app/registryapp/archive_materializer.go`
  - 下载/读取 archive、checksum 校验、安全解压 zip/tar.gz，并导出 archive checksum 缺失 warning 文案供 Web plan 复用。
- `internal/app/registryapp/archive_materializer_test.go`
  - 覆盖 zip、tar.gz、checksum、path traversal。
- `internal/app/webapp/manager_registry_actions.go`
  - Web Registry plan/run action 类型和 `Manager` 方法。
- `internal/app/webapp/manager_registry_actions_test.go`
  - 覆盖 plan/run 当前项目 registry install、confirm 行为通过 server 测试补充。

修改文件：

- `internal/app/registryapp/materializer.go`
  - 接入 archive materializer；允许 download-only entry 映射成 `skill.Skill`。
- `internal/app/registryapp/materializer_test.go`
  - 补充 download-only provenance 测试。
- `internal/app/registryapp/service.go`
  - local catalog 相对 `download_url` 解析；remote catalog 相对 `download_url` 拒绝。
- `internal/app/registryapp/service_test.go`
  - 补充 `download_url` normalization 测试。
- `internal/app/webapp/manager.go`
  - 新增 `Registries`、`RegistrySkills`、`RegistrySources` 查询方法。
- `internal/app/webapp/manager_server.go`
  - 新增 `/api/registries`、`/api/registry/*` routes 和 handlers。
- `internal/app/webapp/manager_server_test.go`
  - 覆盖 API route、confirm gate、history。
- `internal/app/webapp/manager_static.go`
  - 新增 Registry nav/view、查询表格、plan/run action panel。
- `docs/design/skillc-v0-enhance-design.md`
  - 追加 Phase 10 链接和状态。
- `README.md`
  - 补充 registry archive download 和 Web Registry 概览。
- `README.zh-CN.md`
  - 补充中文说明。
- `docs/superpowers/plans/2026-06-16-skillc-v0-phase10-web-registry-archive.md`
  - 实施过程中更新 checkbox。

## Task 1: Archive Download Materializer Foundation

**Files:**
- Create: `internal/app/registryapp/archive_materializer.go`
- Create: `internal/app/registryapp/archive_materializer_test.go`
- Modify: `internal/app/registryapp/materializer.go`
- Modify: `internal/app/registryapp/materializer_test.go`

- [x] **Step 1: Write failing zip archive materialization test**

Add `TestArchiveMaterializer_ExtractsZipDownload` to `internal/app/registryapp/archive_materializer_test.go`.

Test shape:

```go
func TestArchiveMaterializer_ExtractsZipDownload(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "go-pro.zip")
	writeZipArchive(t, archivePath, map[string]string{
		"skills/go-pro/SKILL.md": "# Go Pro",
	})
	targetDir := filepath.Join(baseDir, "cache")

	warning, err := extractArchiveDownload(archiveDownloadReq{
		URL:       archivePath,
		TargetDir: targetDir,
	})

	assert.NoErr(t, err)
	assert.Eq(t, ArchiveChecksumMissingWarning, warning)
	assert.FileExists(t, filepath.Join(targetDir, "skills", "go-pro", "SKILL.md"))
}
```

- [x] **Step 2: Run focused test and verify failure**

Run:

```bash
go test ./internal/app/registryapp -run TestArchiveMaterializer_ExtractsZipDownload -count=1
```

Expected: FAIL because `extractArchiveDownload` and `archiveDownloadReq` do not exist.

- [x] **Step 3: Implement minimal archive request and zip extraction**

Create `internal/app/registryapp/archive_materializer.go` with:

```go
package registryapp

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const ArchiveChecksumMissingWarning = "checksum is missing; archive integrity is not verified"

type archiveDownloadReq struct {
	URL       string
	Checksum  string
	TargetDir string
	Client    *http.Client
}

func extractArchiveDownload(req archiveDownloadReq) (string, error) {
	data, err := readArchiveBytes(req)
	if err != nil {
		return "", err
	}
	if err := os.RemoveAll(req.TargetDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(req.TargetDir, 0o755); err != nil {
		return "", err
	}
	if strings.HasSuffix(strings.ToLower(req.URL), ".zip") {
		if err := extractZipBytes(data, req.TargetDir); err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("unsupported registry archive format: %s", req.URL)
	}
	if strings.TrimSpace(req.Checksum) == "" {
		return ArchiveChecksumMissingWarning, nil
	}
	return "", nil
}

func readArchiveBytes(req archiveDownloadReq) ([]byte, error) {
	if strings.HasPrefix(strings.ToLower(req.URL), "http://") || strings.HasPrefix(strings.ToLower(req.URL), "https://") {
		client := req.Client
		if client == nil {
			client = http.DefaultClient
		}
		resp, err := client.Get(req.URL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("registry archive http status: %d", resp.StatusCode)
		}
		return io.ReadAll(resp.Body)
	}
	return os.ReadFile(req.URL)
}

func extractZipBytes(data []byte, targetDir string) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, file := range reader.File {
		targetPath, err := safeArchiveTarget(targetDir, file.Name)
		if err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		err = writeFileFromReader(targetPath, src)
		closeErr := src.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func safeArchiveTarget(root string, name string) (string, error) {
	cleanName := filepath.Clean(filepath.FromSlash(name))
	if cleanName == "." || filepath.IsAbs(cleanName) || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) || cleanName == ".." {
		return "", fmt.Errorf("unsafe archive path: %s", name)
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(filepath.Join(rootAbs, cleanName))
	if err != nil {
		return "", err
	}
	if targetAbs != rootAbs && !strings.HasPrefix(targetAbs, rootAbs+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe archive path: %s", name)
	}
	return targetAbs, nil
}

func writeFileFromReader(path string, src io.Reader) error {
	dst, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}
```

- [x] **Step 4: Add test helper for zip archives**

In `archive_materializer_test.go`, add helper:

```go
func writeZipArchive(t *testing.T, path string, files map[string]string) {
	t.Helper()
	out, err := os.Create(path)
	assert.Require(t, assert.NoErr(t, err))
	defer out.Close()
	zw := zip.NewWriter(out)
	for name, body := range files {
		w, err := zw.Create(name)
		assert.Require(t, assert.NoErr(t, err))
		_, err = w.Write([]byte(body))
		assert.Require(t, assert.NoErr(t, err))
	}
	assert.Require(t, assert.NoErr(t, zw.Close()))
}
```

- [x] **Step 5: Run focused test and verify pass**

Run:

```bash
go test ./internal/app/registryapp -run TestArchiveMaterializer_ExtractsZipDownload -count=1
```

Expected: PASS.

- [x] **Step 6: Commit archive zip foundation**

```bash
git add internal/app/registryapp/archive_materializer.go internal/app/registryapp/archive_materializer_test.go docs/superpowers/plans/2026-06-16-skillc-v0-phase10-web-registry-archive.md
git commit -m "feat(skillc): extract registry zip archives"
```

## Task 2: Checksum, Tar.gz, and Safe Extraction Coverage

**Files:**
- Modify: `internal/app/registryapp/archive_materializer.go`
- Modify: `internal/app/registryapp/archive_materializer_test.go`

- [x] **Step 1: Write failing checksum tests**

Add tests:

```go
func TestArchiveMaterializer_VerifiesSha256Checksum(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "go-pro.zip")
	writeZipArchive(t, archivePath, map[string]string{"SKILL.md": "# Go Pro"})
	checksum := "sha256:" + sha256FileHex(t, archivePath)

	warning, err := extractArchiveDownload(archiveDownloadReq{
		URL: archivePath, Checksum: checksum, TargetDir: filepath.Join(baseDir, "cache"),
	})

	assert.NoErr(t, err)
	assert.Eq(t, "", warning)
}

func TestArchiveMaterializer_RejectsChecksumMismatch(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "go-pro.zip")
	writeZipArchive(t, archivePath, map[string]string{"SKILL.md": "# Go Pro"})

	_, err := extractArchiveDownload(archiveDownloadReq{
		URL: archivePath, Checksum: strings.Repeat("0", 64), TargetDir: filepath.Join(baseDir, "cache"),
	})

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}
```

- [x] **Step 2: Write failing tar.gz extraction test**

Add:

```go
func TestArchiveMaterializer_ExtractsTarGzDownload(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "go-pro.tar.gz")
	writeTarGzArchive(t, archivePath, map[string]string{"skills/go-pro/SKILL.md": "# Go Pro"})
	targetDir := filepath.Join(baseDir, "cache")

	_, err := extractArchiveDownload(archiveDownloadReq{URL: archivePath, TargetDir: targetDir})

	assert.NoErr(t, err)
	assert.FileExists(t, filepath.Join(targetDir, "skills", "go-pro", "SKILL.md"))
}
```

- [x] **Step 3: Write failing path traversal tests**

Add:

```go
func TestArchiveMaterializer_RejectsZipPathTraversal(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "evil.zip")
	writeZipArchive(t, archivePath, map[string]string{"../outside.txt": "bad"})

	_, err := extractArchiveDownload(archiveDownloadReq{URL: archivePath, TargetDir: filepath.Join(baseDir, "cache")})

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "unsafe archive path")
	_, statErr := os.Stat(filepath.Join(baseDir, "outside.txt"))
	assert.True(t, os.IsNotExist(statErr))
}

func TestArchiveMaterializer_RejectsTarPathTraversal(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "evil.tgz")
	writeTarGzArchive(t, archivePath, map[string]string{"../outside.txt": "bad"})

	_, err := extractArchiveDownload(archiveDownloadReq{URL: archivePath, TargetDir: filepath.Join(baseDir, "cache")})

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "unsafe archive path")
	_, statErr := os.Stat(filepath.Join(baseDir, "outside.txt"))
	assert.True(t, os.IsNotExist(statErr))
}
```

- [x] **Step 4: Run focused tests and verify failure**

Run:

```bash
go test ./internal/app/registryapp -run 'TestArchiveMaterializer_(Verifies|Rejects|ExtractsTar)' -count=1
```

Expected: FAIL for checksum/tar.gz unsupported behavior.

- [x] **Step 5: Implement checksum and tar.gz support**

Update `archive_materializer.go` imports with `archive/tar`, `compress/gzip`, `crypto/sha256`, `encoding/hex`.

Add checksum helpers:

```go
func verifyArchiveChecksum(data []byte, checksum string) error {
	checksum = strings.TrimSpace(strings.ToLower(checksum))
	if checksum == "" {
		return nil
	}
	checksum = strings.TrimPrefix(checksum, "sha256:")
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	if checksum != actual {
		return fmt.Errorf("registry archive checksum mismatch: expected %s, got %s", checksum, actual)
	}
	return nil
}
```

Call it after `readArchiveBytes(req)` and before deleting `TargetDir`.

Add tar.gz branch:

```go
lowerURL := strings.ToLower(req.URL)
switch {
case strings.HasSuffix(lowerURL, ".zip"):
	err = extractZipBytes(data, req.TargetDir)
case strings.HasSuffix(lowerURL, ".tar.gz"), strings.HasSuffix(lowerURL, ".tgz"):
	err = extractTarGzBytes(data, req.TargetDir)
default:
	err = fmt.Errorf("unsupported registry archive format: %s", req.URL)
}
if err != nil {
	return "", err
}
```

Add:

```go
func extractTarGzBytes(data []byte, targetDir string) error {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		targetPath, err := safeArchiveTarget(targetDir, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			if err := writeFileFromReader(targetPath, tr); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported archive entry type for %s", header.Name)
		}
	}
}
```

- [x] **Step 6: Add tar.gz and checksum test helpers**

Add helpers:

```go
func writeTarGzArchive(t *testing.T, path string, files map[string]string) {
	t.Helper()
	out, err := os.Create(path)
	assert.Require(t, assert.NoErr(t, err))
	defer out.Close()
	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		data := []byte(body)
		err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg})
		assert.Require(t, assert.NoErr(t, err))
		_, err = tw.Write(data)
		assert.Require(t, assert.NoErr(t, err))
	}
	assert.Require(t, assert.NoErr(t, tw.Close()))
	assert.Require(t, assert.NoErr(t, gz.Close()))
}

func sha256FileHex(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	assert.Require(t, assert.NoErr(t, err))
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
```

- [x] **Step 7: Run focused tests and verify pass**

Run:

```bash
go test ./internal/app/registryapp -run 'TestArchiveMaterializer_' -count=1
```

Expected: PASS.

- [x] **Step 8: Commit archive hardening**

```bash
git add internal/app/registryapp/archive_materializer.go internal/app/registryapp/archive_materializer_test.go docs/superpowers/plans/2026-06-16-skillc-v0-phase10-web-registry-archive.md
git commit -m "feat(skillc): verify registry archive downloads"
```

## Task 3: Connect Archive Materializer to Registry Skill Install

**Files:**
- Modify: `internal/app/registryapp/materializer.go`
- Modify: `internal/app/registryapp/materializer_test.go`
- Modify: `internal/app/registryapp/service.go`
- Modify: `internal/app/registryapp/service_test.go`

- [x] **Step 1: Write failing download-only materializer test**

Add to `materializer_test.go`:

```go
func TestMaterializer_MaterializeDownloadURLArchive(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "go-pro.zip")
	writeZipArchive(t, archivePath, map[string]string{"skills/go-pro/SKILL.md": "# Go Pro"})
	targetRoot := filepath.Join(baseDir, "cache", "skills", "team", "go-pro", "1.0.0")

	got, err := newMaterializer(nil).Materialize(registry.SkillEntry{
		ID: "go-pro", Name: "Go Pro", Version: "1.0.0", RegistryID: "team",
		DownloadURL: archivePath, InstallEntry: "skills/go-pro",
	}, targetRoot, gitx.SyncOptions{})

	assert.NoErr(t, err)
	assert.Eq(t, targetRoot, got.Path)
	assert.Eq(t, archivePath, got.DownloadURL)
	assert.FileExists(t, filepath.Join(targetRoot, "skills", "go-pro", "SKILL.md"))
}
```

- [x] **Step 2: Run focused test and verify failure**

Run:

```bash
go test ./internal/app/registryapp -run TestMaterializer_MaterializeDownloadURLArchive -count=1
```

Expected: FAIL because materializer requires `source_url`.

- [x] **Step 3: Update materializer source/download branching**

Modify `Materialize`:

```go
func (m *materializer) Materialize(entry registry.SkillEntry, targetDir string, opts gitx.SyncOptions) (skill.Skill, error) {
	resolvedRef := ""
	if strings.TrimSpace(entry.SourceURL) != "" {
		if sourceapp.IsGitURL(entry.SourceURL) {
			ref, err := m.git.Sync(entry.SourceURL, targetDir, entry.SourceRef, opts)
			if err != nil {
				return skill.Skill{}, err
			}
			resolvedRef = ref
		} else {
			if err := copyLocalSourceSnapshot(entry.SourceURL, targetDir); err != nil {
				return skill.Skill{}, err
			}
		}
	} else if strings.TrimSpace(entry.DownloadURL) != "" {
		if _, err := extractArchiveDownload(archiveDownloadReq{
			URL: entry.DownloadURL, Checksum: entry.Checksum, TargetDir: targetDir,
		}); err != nil {
			return skill.Skill{}, err
		}
	} else {
		return skill.Skill{}, fmt.Errorf("registry skill source_url or download_url is required for install: %s/%s", entry.RegistryID, entry.ID)
	}
	item, err := m.skillFromEntry(entry, targetDir)
	if err != nil {
		return skill.Skill{}, err
	}
	item.SourceResolvedRef = resolvedRef
	return item, nil
}
```

Extract local copy into:

```go
func copyLocalSourceSnapshot(sourceURL string, targetDir string) error {
	info, err := os.Stat(sourceURL)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("registry skill source_url must be git URL or local directory: %s", sourceURL)
	}
	if err := os.RemoveAll(targetDir); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return err
	}
	return os.CopyFS(targetDir, os.DirFS(sourceURL))
}
```

Update `skillFromEntry` to remove the `SourceURL` required check and replace it with source-or-download check.

- [x] **Step 4: Run materializer tests**

Run:

```bash
go test ./internal/app/registryapp -run 'TestMaterializer_' -count=1
```

Expected: PASS.

- [x] **Step 5: Write failing catalog download_url normalization tests**

Add to `service_test.go`:

```go
func TestRegistryService_NormalizeLocalSkillRelativeDownloadURL(t *testing.T) {
	baseDir := t.TempDir()
	catalogDir := filepath.Join(baseDir, "registry")
	assert.NoErr(t, os.MkdirAll(catalogDir, 0o755))
	item := registry.Registry{ID: "team", Type: registry.TypeLocal, Path: filepath.Join(catalogDir, "registry.json")}

	catalog, err := normalizeCatalog(registry.Catalog{Skills: []registry.SkillEntry{{
		ID: "go-pro", DownloadURL: "archives/go-pro.zip",
	}}}, item, catalogDir, false)

	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(catalogDir, "archives", "go-pro.zip"), catalog.Skills[0].DownloadURL)
}

func TestRegistryService_RejectsRemoteRelativeDownloadURL(t *testing.T) {
	item := registry.Registry{ID: "team", Type: registry.TypeHTTP, URL: "https://example.com/registry.json"}

	_, err := normalizeCatalog(registry.Catalog{Skills: []registry.SkillEntry{{
		ID: "go-pro", DownloadURL: "archives/go-pro.zip",
	}}}, item, "", true)

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "download_url must be http URL")
}
```

- [x] **Step 6: Implement download_url normalization**

In `normalizeSkillEntries`, before `NormalizeSkillEntry`, add:

```go
entry.DownloadURL = strings.TrimSpace(entry.DownloadURL)
if entry.DownloadURL != "" && !registry.IsHTTPURL(entry.DownloadURL) {
	if remote {
		return nil, fmt.Errorf("registry skill download_url must be http URL for remote catalog: %s", entry.ID)
	}
	if !filepath.IsAbs(entry.DownloadURL) {
		entry.DownloadURL = filepath.Join(catalogDir, entry.DownloadURL)
	}
	entry.DownloadURL = filepath.Clean(entry.DownloadURL)
}
```

- [x] **Step 7: Run registryapp tests**

Run:

```bash
go test ./internal/app/registryapp -count=1
```

Expected: PASS.

- [x] **Step 8: Commit materializer integration**

```bash
git add internal/app/registryapp/materializer.go internal/app/registryapp/materializer_test.go internal/app/registryapp/service.go internal/app/registryapp/service_test.go docs/superpowers/plans/2026-06-16-skillc-v0-phase10-web-registry-archive.md
git commit -m "feat(skillc): install registry skills from archives"
```

## Task 4: Web Registry Query APIs

**Files:**
- Modify: `internal/app/webapp/manager.go`
- Modify: `internal/app/webapp/manager_server.go`
- Modify: `internal/app/webapp/manager_server_test.go`

- [x] **Step 1: Write failing server tests for registry query routes**

Add tests in `manager_server_test.go`:

```go
func TestManagerServer_RegistryQueryRoutes(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeWebManagerFixture(t, baseDir)
	writeWebRegistryFixture(t, configFile, baseDir, config)

	server := NewManagerServer(configFile, baseDir)

	rec := performManagerRequest(server, http.MethodGet, "/api/registries")
	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"id":"team"`)

	rec = performManagerRequest(server, http.MethodGet, "/api/registry/skills?keyword=go")
	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"id":"go-pro"`)

	rec = performManagerRequest(server, http.MethodGet, "/api/registry/sources?keyword=gstack")
	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"id":"gstack"`)
}

func writeWebRegistryFixture(t *testing.T, configFile string, baseDir string, config cfg.Config) {
	t.Helper()
	config.RegistryCacheDir = filepath.Join(baseDir, "cache", "registry")
	config.Registries = []registry.Registry{{
		ID: "team", Name: "Team", Type: registry.TypeLocal, Path: filepath.Join(baseDir, "registry.json"),
	}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, registrystore.NewStore().SaveFile(filepath.Join(config.RegistryCacheDir, "registry-index.json"), registrystore.File{
		Skills: []registry.SkillEntry{{
			ID: "go-pro", Name: "Go Pro", Version: "1.0.0", RegistryID: "team",
			SourceURL: "https://example.com/skills.git", InstallEntry: "skills/go-pro",
		}},
		Sources: []registry.Entry{{
			ID: "gstack", Name: "GStack", Type: "git", URL: "https://example.com/gstack.git", RegistryID: "team",
		}},
	}))
}
```

Add imports for `cfg`, `registry`, and `registrystore` if they are not already present in `manager_server_test.go`.

- [x] **Step 2: Run focused test and verify failure**

Run:

```bash
go test ./internal/app/webapp -run TestManagerServer_RegistryQueryRoutes -count=1
```

Expected: FAIL or 404 for new routes.

- [x] **Step 3: Add Manager query methods**

In `manager.go`, import `registryapp`, `strings`, and `github.com/inhere/skillc/internal/domain/registry`, then add:

```go
func (m *Manager) Registries() ([]registry.Registry, error) {
	return registryapp.NewService(m.configFile, m.baseDir).List()
}

func (m *Manager) RegistrySkills(keyword string, registryID string) ([]registry.SkillEntry, error) {
	return registryapp.NewService(m.configFile, m.baseDir).SearchSkills(registryapp.SearchReq{
		Keyword: keyword, RegistryID: registryID,
	})
}

func (m *Manager) RegistrySources(keyword string, registryID string) ([]registry.Entry, error) {
	items, err := registryapp.NewService(m.configFile, m.baseDir).Search(keyword)
	if err != nil {
		return nil, err
	}
	return filterRegistrySourcesByID(items, registryID), nil
}
```

Add `filterRegistrySourcesByID` locally or reuse equivalent logic if already available in webapp.

Use this local helper if there is no existing equivalent:

```go
func filterRegistrySourcesByID(items []registry.Entry, registryID string) []registry.Entry {
	registryID = strings.ToLower(strings.TrimSpace(registryID))
	if registryID == "" {
		return items
	}
	out := make([]registry.Entry, 0, len(items))
	for _, item := range items {
		if strings.ToLower(item.RegistryID) == registryID {
			out = append(out, item)
		}
	}
	return out
}
```

- [x] **Step 4: Add HTTP routes and handlers**

In `Handler()` add:

```go
mux.HandleFunc("/api/registries", s.handleRegistries)
mux.HandleFunc("/api/registry/skills", s.handleRegistrySkills)
mux.HandleFunc("/api/registry/sources", s.handleRegistrySources)
```

Add handlers:

```go
func (s *ManagerServer) handleRegistries(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	result, err := s.manager.Registries()
	writeResult(w, result, err)
}

func (s *ManagerServer) handleRegistrySkills(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	result, err := s.manager.RegistrySkills(r.URL.Query().Get("keyword"), r.URL.Query().Get("registry"))
	writeResult(w, result, err)
}

func (s *ManagerServer) handleRegistrySources(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	result, err := s.manager.RegistrySources(r.URL.Query().Get("keyword"), r.URL.Query().Get("registry"))
	writeResult(w, result, err)
}
```

- [x] **Step 5: Run webapp query route tests**

Run:

```bash
go test ./internal/app/webapp -run TestManagerServer_RegistryQueryRoutes -count=1
```

Expected: PASS.

- [x] **Step 6: Commit Web query APIs**

```bash
git add internal/app/webapp/manager.go internal/app/webapp/manager_server.go internal/app/webapp/manager_server_test.go docs/superpowers/plans/2026-06-16-skillc-v0-phase10-web-registry-archive.md
git commit -m "feat(skillc): expose registry queries in web manager"
```

## Task 5: Web Registry Plan and Run Actions

**Files:**
- Create: `internal/app/webapp/manager_registry_actions.go`
- Create: `internal/app/webapp/manager_registry_actions_test.go`
- Modify: `internal/app/webapp/manager_server.go`
- Modify: `internal/app/webapp/manager_server_test.go`

- [x] **Step 1: Write failing manager install plan test**

Create `manager_registry_actions_test.go`:

```go
func TestManager_PlanRegistryInstall(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	writeWebTestConfig(t, configFile, baseDir)
	writeWebTestRegistryCache(t, baseDir)

	got, err := NewManager(configFile, baseDir).PlanRegistryInstall(WebRegistryInstallReq{
		Target: "team/go-pro",
		ManagerReq: ManagerReq{Agent: "codex", Scope: "project", WorkDir: baseDir},
	})

	assert.NoErr(t, err)
	assert.Eq(t, "team/go-pro", got.Target)
	assert.Eq(t, "go-pro", got.SkillID)
	assert.Eq(t, "team", got.RegistryID)
	assert.Eq(t, "codex", got.Agent)
	assert.Eq(t, "project", got.Scope)
}
```

- [x] **Step 2: Run focused test and verify failure**

Run:

```bash
go test ./internal/app/webapp -run TestManager_PlanRegistryInstall -count=1
```

Expected: FAIL because types/method do not exist.

- [x] **Step 3: Implement registry action types and plan methods**

Create `manager_registry_actions.go` with:

```go
package webapp

import (
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/registryapp"
	"github.com/inhere/skillc/internal/domain/skill"
)

type WebRegistryInstallReq struct {
	ManagerReq
	Target string `json:"target"`
}

type registryInstallPlan struct {
	Target       string   `json:"target"`
	RegistryID   string   `json:"registry_id"`
	SkillID      string   `json:"skill_id"`
	Name         string   `json:"name,omitempty"`
	Version      string   `json:"version,omitempty"`
	Agent        string   `json:"agent"`
	Scope        string   `json:"scope"`
	InstallEntry string   `json:"install_entry"`
	SourceURL    string   `json:"source_url,omitempty"`
	DownloadURL  string   `json:"download_url,omitempty"`
	Checksum     string   `json:"checksum,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

type registryInstallActionResult struct {
	Error     string                `json:"error,omitempty"`
	Plan      registryInstallPlan   `json:"plan"`
	Installed []actionRuntimeRecord `json:"installed"`
	Failed    []actionErrorItem     `json:"failed,omitempty"`
}

func (m *Manager) PlanRegistryInstall(req WebRegistryInstallReq) (registryInstallPlan, error) {
	item, err := registryapp.NewService(m.configFile, m.baseDir).InfoSkill(req.Target)
	if err != nil {
		return registryInstallPlan{}, err
	}
	plan := registryInstallPlan{
		Target: req.Target, RegistryID: item.RegistryID, SkillID: item.ID, Name: item.Name,
		Version: item.Version, Agent: req.Agent, Scope: req.Scope, InstallEntry: item.InstallEntry,
		SourceURL: item.SourceURL, DownloadURL: item.DownloadURL, Checksum: item.Checksum,
	}
	if item.DownloadURL != "" && item.Checksum == "" {
		plan.Warnings = append(plan.Warnings, registryapp.ArchiveChecksumMissingWarning)
	}
	return plan, nil
}
```

- [x] **Step 4: Implement RunRegistryInstall**

Add:

```go
func (m *Manager) RunRegistryInstall(req WebRegistryInstallReq) (registryInstallActionResult, error) {
	plan, err := m.PlanRegistryInstall(req)
	if err != nil {
		return registryInstallActionResult{}, err
	}
	config, err := m.config()
	if err != nil {
		return registryInstallActionResult{Plan: plan}, err
	}
	item, err := registryapp.NewService(m.configFile, m.baseDir).MaterializeSkill(req.Target)
	if err != nil {
		return registryInstallActionResult{Plan: plan, Error: err.Error()}, nil
	}
	result, err := installapp.NewService(config.LockFile).RunResolved(config, installapp.InstallReq{
		SkillID: item.SourceQualifiedName, Agent: req.Agent, Scope: req.Scope, WorkDir: req.WorkDir,
	}, []skill.Skill{item}, nil)
	out := registryInstallActionResult{
		Plan: plan, Installed: runtimeRecords(result.Installed), Failed: installErrors(result.InstallFailed),
	}
	if err != nil {
		out.Error = err.Error()
		return out, nil
	}
	return out, nil
}
```

- [x] **Step 5: Add sync and add-source action plans**

Add types and methods:

```go
type WebRegistrySyncReq struct {
	RegistryID string `json:"registry_id,omitempty"`
}

type registrySyncPlan struct {
	RegistryID string   `json:"registry_id,omitempty"`
	Items      []string `json:"items"`
}

type WebRegistryAddSourceReq struct {
	EntryID string `json:"entry_id"`
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Sync    bool   `json:"sync,omitempty"`
}

type registryAddSourcePlan struct {
	EntryID  string `json:"entry_id"`
	SourceID string `json:"source_id"`
	Name     string `json:"name,omitempty"`
	Type     string `json:"type"`
	Location string `json:"location"`
	Sync     bool   `json:"sync"`
}
```

Implement:

```go
func (m *Manager) PlanRegistrySync(req WebRegistrySyncReq) (registrySyncPlan, error)
func (m *Manager) RunRegistrySync(req WebRegistrySyncReq) (registrySyncPlan, error)
func (m *Manager) PlanRegistryAddSource(req WebRegistryAddSourceReq) (registryAddSourcePlan, error)
func (m *Manager) RunRegistryAddSource(req WebRegistryAddSourceReq) (registryAddSourcePlan, error)
```

Plan methods should only read `registryapp.List` / `Info`; run methods call `Sync` / `SyncAll` / `AddSource`.

- [x] **Step 6: Add server route tests for confirm gate**

In `manager_server_test.go`, add tests that POST run endpoints without confirm:

```go
assertJSONStatus(t, server.Handler(), http.MethodPost, "/api/registry/install/run", strings.NewReader(`{"target":"team/go-pro"}`), http.StatusBadRequest)
assertJSONStatus(t, server.Handler(), http.MethodPost, "/api/registry/sync/run", strings.NewReader(`{"registry_id":"team"}`), http.StatusBadRequest)
assertJSONStatus(t, server.Handler(), http.MethodPost, "/api/registry/add-source/run", strings.NewReader(`{"entry_id":"team/gstack"}`), http.StatusBadRequest)
```

- [x] **Step 7: Add server handlers**

Add routes:

```go
mux.HandleFunc("/api/registry/sync/plan", s.handleRegistrySyncPlan)
mux.HandleFunc("/api/registry/sync/run", s.handleRegistrySyncRun)
mux.HandleFunc("/api/registry/install/plan", s.handleRegistryInstallPlan)
mux.HandleFunc("/api/registry/install/run", s.handleRegistryInstallRun)
mux.HandleFunc("/api/registry/add-source/plan", s.handleRegistryAddSourcePlan)
mux.HandleFunc("/api/registry/add-source/run", s.handleRegistryAddSourceRun)
```

Each plan handler reads JSON body and calls manager plan. Each run handler uses `requireConfirm`, then calls manager run and `s.recordHistory` with action names:

- `registry.sync`
- `registry.install`
- `registry.add_source`

- [x] **Step 8: Run webapp tests**

Run:

```bash
go test ./internal/app/webapp -count=1
```

Expected: PASS.

- [x] **Step 9: Commit Web registry actions**

```bash
git add internal/app/webapp/manager_registry_actions.go internal/app/webapp/manager_registry_actions_test.go internal/app/webapp/manager_server.go internal/app/webapp/manager_server_test.go docs/superpowers/plans/2026-06-16-skillc-v0-phase10-web-registry-archive.md
git commit -m "feat(skillc): add web registry actions"
```

## Task 6: Web Registry Static UI

**Files:**
- Modify: `internal/app/webapp/manager_static.go`
- Modify: `internal/app/webapp/manager_server_test.go`

- [x] **Step 1: Add static UI smoke test expectation**

In an existing index/static test, assert the HTML contains:

```go
assert.Contains(t, body, `data-view="registry"`)
assert.Contains(t, body, `/api/registry/skills`)
assert.Contains(t, body, `registry-install`)
```

- [x] **Step 2: Run focused static test and verify failure**

Run:

```bash
go test ./internal/app/webapp -run TestManagerServer_Index -count=1
```

Expected: FAIL because Registry UI is not present.

- [x] **Step 3: Add Registry nav and view skeleton**

In `manager_static.go`, add sidebar button:

```html
<button data-view="registry">Registry</button>
```

Add view section:

```html
<section id="view-registry" class="view">
  <div class="section-head"><h3>Registry</h3><span class="hint" id="registry-count">0 result(s)</span></div>
  <div class="toolbar">
    <input id="registry-keyword" placeholder="Search registry skills or sources">
    <select id="registry-filter"></select>
    <select id="registry-kind">
      <option value="skill">Skills</option>
      <option value="source">Sources</option>
    </select>
    <button id="registry-search">Search</button>
    <button id="registry-sync-all">Sync All</button>
  </div>
  <div id="registry-table"></div>
</section>
```

Keep styling consistent with existing toolbar/table/button classes.

- [x] **Step 4: Extend frontend state and loading**

Add to state:

```js
registries: [],
registrySkills: [],
registrySources: [],
registryKind: 'skill'
```

In `loadAll`, fetch `/api/registries`, `/api/registry/skills`, `/api/registry/sources`, then assign into state.

- [x] **Step 5: Render registry filter and result tables**

Add `renderRegistry()`:

```js
function renderRegistry() {
  var filter = byId('registry-filter');
  filter.innerHTML = '<option value="">All registries</option>' + state.registries.map(function (r) {
    return '<option value="' + escapeHtml(r.id) + '">' + escapeHtml(r.id) + '</option>';
  }).join('');
  var kind = byId('registry-kind').value || 'skill';
  var rows = kind === 'source' ? registrySourceRows() : registrySkillRows();
  byId('registry-count').textContent = rows.length + ' result(s)';
  byId('registry-table').innerHTML = kind === 'source'
    ? table(['Registry', 'ID', 'Name', 'Type', 'Location', 'Actions'], rows)
    : table(['Registry', 'ID', 'Name', 'Version', 'Agents', 'Source', 'Actions'], rows);
}
```

Rows should include buttons:

- Skill: `Install`
- Source: `Add Source`
- Registry sync: per registry button can be added in table or registry list later; P10 must include `Sync All`.

- [x] **Step 6: Add search and action handlers**

Add:

```js
function searchRegistry() {
  var keyword = encodeURIComponent(byId('registry-keyword').value || '');
  var registry = encodeURIComponent(byId('registry-filter').value || '');
  var qs = '?keyword=' + keyword + '&registry=' + registry;
  Promise.all([api('/api/registry/skills' + qs), api('/api/registry/sources' + qs)]).then(function (all) {
    state.registrySkills = all[0] || [];
    state.registrySources = all[1] || [];
    renderRegistry();
  }).catch(showError);
}
```

Add plan functions:

```js
function planRegistryInstall(target) {
  postJSON('/api/registry/install/plan', withManagerReq({ target: target }))
    .then(function (plan) { showPlan('registry-install', plan, 'Install registry skill?'); })
    .catch(showError);
}

function planRegistrySync(registryID) {
  postJSON('/api/registry/sync/plan', { registry_id: registryID || '' })
    .then(function (plan) { showPlan('registry-sync', plan, 'Sync registry?'); })
    .catch(showError);
}

function planRegistryAddSource(entryID) {
  postJSON('/api/registry/add-source/plan', { entry_id: entryID, sync: true })
    .then(function (plan) { showPlan('registry-add-source', plan, 'Add source from registry?'); })
    .catch(showError);
}
```

Wire confirm route map:

```js
'registry-install': '/api/registry/install/run',
'registry-sync': '/api/registry/sync/run',
'registry-add-source': '/api/registry/add-source/run'
```

- [x] **Step 7: Run webapp tests**

Run:

```bash
go test ./internal/app/webapp -count=1
```

Expected: PASS.

- [x] **Step 8: Manual smoke test with local server**

Run:

```bash
go run . web --host 127.0.0.1 --port 8765
```

Open `http://127.0.0.1:8765`, verify:

- Registry nav item appears.
- Search does not break Dashboard/Sources/Profiles views.
- Plan panel appears for install/sync/add-source.

Stop server after verification.

- [ ] **Step 9: Commit static UI**

```bash
git add internal/app/webapp/manager_static.go internal/app/webapp/manager_server_test.go docs/superpowers/plans/2026-06-16-skillc-v0-phase10-web-registry-archive.md
git commit -m "feat(skillc): add registry web page"
```

## Task 7: Documentation and Final Verification

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/design/skillc-v0-enhance-design.md`
- Modify: `docs/superpowers/plans/2026-06-16-skillc-v0-phase10-web-registry-archive.md`

- [ ] **Step 1: Update README registry examples**

Add a short example showing archive skill entry:

```json
{
  "skills": [
    {
      "id": "go-pro",
      "name": "Go Pro",
      "version": "1.0.0",
      "download_url": "https://example.com/skills/go-pro.zip",
      "checksum": "sha256:<archive-sha256>",
      "install_entry": "skills/go-pro"
    }
  ]
}
```

Mention Web Registry:

```text
Run `skillc web` and open the Registry view to search synced registry skills, preview install plans, and install a registry skill into the current project.
```

- [ ] **Step 2: Update Chinese README**

Add equivalent Chinese explanation for:

- `download_url` archive 支持 zip/tar.gz/tgz。
- checksum 校验 archive 原始字节。
- `skillc web` 的 Registry 页面支持搜索、同步、安装、add-source。

- [ ] **Step 3: Update design doc revision table**

Add revision row:

```markdown
| 2026-06-16 | v0.22 | Codex | 增加 Phase 10 Web Registry 页面与 archive download 实施计划链接 |
```

Add Phase 10 section near Phase 9:

```markdown
十期开发计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase10-web-registry-archive.md`

十期目标：补齐 registry skill archive `download_url` zip/tar.gz 下载安装能力，并在 Web 中新增 Registry 页面，支持当前项目范围内 registry skill 搜索、同步、安装和 source result add-source。
```

- [ ] **Step 4: Run focused package tests**

Run:

```bash
go test ./internal/domain/registry ./internal/app/registryapp ./internal/app/webapp ./internal/cli -count=1
```

Expected: PASS.

- [ ] **Step 5: Run full test suite**

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 6: Update all completed checkboxes in this plan**

Mark every completed step as `[x]`. Do not mark a step complete unless its command passed or the documented artifact was updated.

- [ ] **Step 7: Commit docs and plan completion**

```bash
git add README.md README.zh-CN.md docs/design/skillc-v0-enhance-design.md docs/superpowers/plans/2026-06-16-skillc-v0-phase10-web-registry-archive.md
git commit -m "docs(skillc): document web registry archive workflow"
```

- [ ] **Step 8: Final branch status**

Run:

```bash
git status --short --branch
```

Expected: clean branch with local commits ready to merge/push according to the current workflow.

## Self-Review

- Spec coverage：计划覆盖 archive download、安全解压、checksum、catalog normalization、Web query API、Web plan/run API、static UI、docs 和最终测试。
- Placeholder scan：没有保留未完成占位词；后续真实实现中如测试 helper 名称与现有文件不同，应按现有测试风格调整，但行为要求已明确。
- Type consistency：计划中的 `WebRegistryInstallReq`、`registryInstallPlan`、`registrySyncPlan`、`registryAddSourcePlan` 在 Task 5 定义，并在 server/static UI 中复用；archive checksum warning 通过 `registryapp.ArchiveChecksumMissingWarning` 导出，避免 Web 包引用未导出标识符。
- Scope check：P10 是单一闭环，底层 archive materialization 是 Web registry install 的必要依赖；真实 provider adapter 和跨项目 install 已明确排除。
