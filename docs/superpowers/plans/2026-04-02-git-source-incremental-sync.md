# Git Source Incremental Sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change Git source sync to reuse healthy cached repositories with incremental Git commands, while preserving clean-cache semantics and falling back to re-clone when the cache is invalid.

**Architecture:** Keep sync orchestration in `internal/app/sourceapp/service.go` and move all repository reuse, validation, incremental fetch/reset/clean logic, and fallback re-clone behavior into `internal/infra/gitx/client.go`. Preserve the current `sourceapp -> gitx.Sync -> ResolvedRef` contract so status updates, timestamps, indexing, proxy behavior, and TTY progress remain stable from the user’s perspective.

**Tech Stack:** Go, `os/exec`, filesystem operations from the standard library, Git CLI, `github.com/gookit/goutil/testutil/assert`, existing `go test` suite, markdown docs in `arch.md` and `plan.md`

---

## File Structure

### Documents to consult
- `docs/superpowers/specs/2026-04-02-git-source-incremental-sync-design.md` — approved sync behavior, fallback rules, and testing scope
- `AGENTS.md` — app/infra layering, doc sync requirements, and `go test ./...` completion gate
- `arch.md` — source/cache architecture notes that must be updated after implementation
- `plan.md` — task ledger that must reflect this optimization after implementation

### Files to modify

**Git infrastructure**
- Modify: `internal/infra/gitx/client.go` — add reusable-repo detection, remote validation, fetch/reset/clean flow, fallback re-clone flow, and helper commands
- Modify: `internal/infra/gitx/client_test.go` — add sync-path tests for clone, incremental reuse, stale-file cleanup, bad repo fallback, and mismatched-origin fallback

**App layer**
- Modify: `internal/app/sourceapp/service.go` — stop deleting the cache directory before every Git sync and rely on `gitx.Client.Sync` to manage reuse/fallback internally
- Modify: `internal/app/sourceapp/service_test.go` — replace the old “service deletes cache dir” expectation with the new “service reuses the same cache path and still succeeds” expectation

**Project docs**
- Modify: `arch.md` — describe incremental Git source sync with fallback re-clone
- Modify: `plan.md` — add/update the task entry for this incremental sync optimization and mark its status accurately

---

### Task 1: Implement incremental sync path selection in `gitx`

**Files:**
- Modify: `internal/infra/gitx/client.go`
- Modify: `internal/infra/gitx/client_test.go`

- [ ] **Step 1: Write the failing tests**

Add these tests to `internal/infra/gitx/client_test.go` first:

```go
package gitx

import (
    "bytes"
    "os"
    "path/filepath"
    "testing"

    "github.com/gookit/goutil/testutil/assert"
)

func TestClient_SyncCloneWhenDirMissing(t *testing.T) {
    remoteDir := createBareRepo(t)
    workDir := filepath.Join(t.TempDir(), "repo")

    client := New("git")
    resolved, err := client.Sync(remoteDir, workDir, "main", SyncOptions{})
    assert.NoErr(t, err)
    assert.NotEmpty(t, resolved)
    _, statErr := os.Stat(filepath.Join(workDir, ".git"))
    assert.NoErr(t, statErr)
}

func TestClient_SyncReusesExistingRepoAndRemovesStaleFile(t *testing.T) {
    remoteDir := createBareRepo(t)
    workDir := filepath.Join(t.TempDir(), "repo")

    client := New("git")
    _, err := client.Sync(remoteDir, workDir, "main", SyncOptions{})
    assert.NoErr(t, err)

    assert.NoErr(t, os.WriteFile(filepath.Join(workDir, "stale.txt"), []byte("old"), 0o644))

    pushCommitToRemote(t, remoteDir, "main", "second commit")

    resolved, err := client.Sync(remoteDir, workDir, "main", SyncOptions{})
    assert.NoErr(t, err)
    assert.NotEmpty(t, resolved)
    _, statErr := os.Stat(filepath.Join(workDir, "stale.txt"))
    assert.True(t, os.IsNotExist(statErr))
}

func TestClient_SyncFallbacksToCloneWhenDirIsNotGitRepo(t *testing.T) {
    remoteDir := createBareRepo(t)
    workDir := filepath.Join(t.TempDir(), "repo")
    assert.NoErr(t, os.MkdirAll(workDir, 0o755))
    assert.NoErr(t, os.WriteFile(filepath.Join(workDir, "placeholder.txt"), []byte("not a repo"), 0o644))

    client := New("git")
    resolved, err := client.Sync(remoteDir, workDir, "main", SyncOptions{})
    assert.NoErr(t, err)
    assert.NotEmpty(t, resolved)
    _, statErr := os.Stat(filepath.Join(workDir, ".git"))
    assert.NoErr(t, statErr)
}

func TestClient_SyncFallbacksToCloneWhenOriginMismatches(t *testing.T) {
    firstRemote := createBareRepo(t)
    secondRemote := createBareRepo(t)
    workDir := filepath.Join(t.TempDir(), "repo")

    client := New("git")
    _, err := client.Sync(firstRemote, workDir, "main", SyncOptions{})
    assert.NoErr(t, err)

    resolved, err := client.Sync(secondRemote, workDir, "main", SyncOptions{})
    assert.NoErr(t, err)
    assert.NotEmpty(t, resolved)

    origin, err := client.remoteGetURL(workDir, "origin")
    assert.NoErr(t, err)
    assert.Eq(t, secondRemote, origin)
}

func TestCloneCommandUsesProxyAndProgressOptions(t *testing.T) {
    client := New("git")
    progress := &bytes.Buffer{}
    cmd := client.cloneCommand("https://example.com/repo.git", "/tmp/repo", "main", SyncOptions{
        ProxyURL: "http://localhost:7890",
        Progress: progress,
    })

    assert.Contains(t, cmd.Args, "--progress")
    assert.Contains(t, cmd.Env, "HTTP_PROXY=http://localhost:7890")
    assert.Contains(t, cmd.Env, "HTTPS_PROXY=http://localhost:7890")
    if cmd.Stdout != progress {
        t.Fatalf("expected stdout to use progress writer")
    }
    if cmd.Stderr != progress {
        t.Fatalf("expected stderr to use progress writer")
    }
}
```

Also add test helpers in the same file so the new tests can create real local Git remotes:

```go
func createBareRepo(t *testing.T) string {
    t.Helper()
    root := t.TempDir()
    remoteDir := filepath.Join(root, "remote.git")
    worktreeDir := filepath.Join(root, "seed")

    runGit(t, root, "init", "--bare", remoteDir)
    runGit(t, root, "init", "-b", "main", worktreeDir)
    runGit(t, worktreeDir, "config", "user.name", "skillc-test")
    runGit(t, worktreeDir, "config", "user.email", "skillc@example.com")
    assert.NoErr(t, os.WriteFile(filepath.Join(worktreeDir, "README.md"), []byte("first\n"), 0o644))
    runGit(t, worktreeDir, "add", "README.md")
    runGit(t, worktreeDir, "commit", "-m", "seed")
    runGit(t, worktreeDir, "remote", "add", "origin", remoteDir)
    runGit(t, worktreeDir, "push", "-u", "origin", "main")
    return remoteDir
}

func pushCommitToRemote(t *testing.T, remoteDir, branch, content string) {
    t.Helper()
    cloneDir := filepath.Join(t.TempDir(), "writer")
    runGit(t, t.TempDir(), "clone", remoteDir, cloneDir)
    runGit(t, cloneDir, "config", "user.name", "skillc-test")
    runGit(t, cloneDir, "config", "user.email", "skillc@example.com")
    assert.NoErr(t, os.WriteFile(filepath.Join(cloneDir, "README.md"), []byte(content+"\n"), 0o644))
    runGit(t, cloneDir, "add", "README.md")
    runGit(t, cloneDir, "commit", "-m", content)
    runGit(t, cloneDir, "push", "origin", branch)
}

func runGit(t *testing.T, dir string, args ...string) string {
    t.Helper()
    cmd := exec.Command("git", args...)
    cmd.Dir = dir
    out, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
    }
    return strings.TrimSpace(string(out))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/infra/gitx -run 'TestClient_SyncCloneWhenDirMissing|TestClient_SyncReusesExistingRepoAndRemovesStaleFile|TestClient_SyncFallbacksToCloneWhenDirIsNotGitRepo|TestClient_SyncFallbacksToCloneWhenOriginMismatches'
```

Expected: FAIL because `gitx.Client.Sync` still always clones, does not validate an existing repo, does not expose `remoteGetURL`, and cannot reuse the cache directory.

- [ ] **Step 3: Write the minimal implementation**

Update `internal/infra/gitx/client.go` to introduce the incremental path and fallback helpers. The implementation should take this shape:

```go
package gitx

import (
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
)

type SyncOptions struct {
    ProxyURL string
    Progress io.Writer
    Quiet    bool
    Verbose  bool
}

type Client struct {
    bin string
}

func (c *Client) Sync(url, dir, ref string, opts SyncOptions) (string, error) {
    if _, err := exec.LookPath(c.bin); err != nil {
        return "", fmt.Errorf("git executable not found: %w", err)
    }

    if ok, err := c.canReuseRepo(dir, url); err == nil && ok {
        if err := c.fetch(dir, opts); err == nil {
            target, err := c.resolveTarget(dir, ref)
            if err == nil {
                if err := c.resetHard(dir, target); err == nil {
                    if err := c.clean(dir); err == nil {
                        return c.revParseHead(dir)
                    }
                }
            }
        }
    }

    if err := os.RemoveAll(dir); err != nil {
        return "", err
    }
    if err := c.clone(url, dir, ref, opts); err != nil {
        return "", err
    }
    return c.revParseHead(dir)
}

func (c *Client) canReuseRepo(dir, url string) (bool, error) {
    info, err := os.Stat(dir)
    if err != nil {
        if os.IsNotExist(err) {
            return false, nil
        }
        return false, err
    }
    if !info.IsDir() {
        return false, fmt.Errorf("repo cache path is not a directory: %s", dir)
    }
    if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
        return false, err
    }
    origin, err := c.remoteGetURL(dir, "origin")
    if err != nil {
        return false, err
    }
    if origin != url {
        return false, fmt.Errorf("origin mismatch: %s", origin)
    }
    return true, nil
}

func (c *Client) clone(url, dir, ref string, opts SyncOptions) error {
    cmd := c.cloneCommand(url, dir, ref, opts)
    if opts.Progress != nil {
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("git clone failed: %w", err)
        }
        return nil
    }
    if out, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("git clone failed: %s", string(out))
    }
    return nil
}

func (c *Client) fetch(dir string, opts SyncOptions) error {
    cmd := c.fetchCommand(dir, opts)
    if opts.Progress != nil {
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("git fetch failed: %w", err)
        }
        return nil
    }
    if out, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("git fetch failed: %s", string(out))
    }
    return nil
}

func (c *Client) resolveTarget(dir, ref string) (string, error) {
    if ref == "" {
        return c.revParse(dir, "FETCH_HEAD")
    }
    return c.revParse(dir, "FETCH_HEAD")
}

func (c *Client) resetHard(dir, target string) error {
    out, err := exec.Command(c.bin, "-C", dir, "reset", "--hard", target).CombinedOutput()
    if err != nil {
        return fmt.Errorf("git reset failed: %s", string(out))
    }
    return nil
}

func (c *Client) clean(dir string) error {
    out, err := exec.Command(c.bin, "-C", dir, "clean", "-fd").CombinedOutput()
    if err != nil {
        return fmt.Errorf("git clean failed: %s", string(out))
    }
    return nil
}

func (c *Client) remoteGetURL(dir, name string) (string, error) {
    out, err := exec.Command(c.bin, "-C", dir, "remote", "get-url", name).CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("git remote get-url failed: %s", string(out))
    }
    return trimOutput(string(out)), nil
}

func (c *Client) fetchCommand(dir string, opts SyncOptions) *exec.Cmd {
    cmd := exec.Command(c.bin, "-C", dir, "fetch", "--prune", "origin")
    cmd.Env = buildGitEnv(os.Environ(), opts.ProxyURL)
    if opts.Progress != nil {
        cmd.Stdout = opts.Progress
        cmd.Stderr = opts.Progress
    }
    return cmd
}

func (c *Client) revParse(dir, value string) (string, error) {
    out, err := exec.Command(c.bin, "-C", dir, "rev-parse", value).CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("git rev-parse failed: %s", string(out))
    }
    return trimOutput(string(out)), nil
}
```

Keep the existing `cloneCommand`, `revParseHead`, `revParseHeadCommand`, `buildGitEnv`, and `trimOutput` helpers, but adjust them only as needed to support this flow.

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
go test ./internal/infra/gitx -run 'TestClient_SyncCloneWhenDirMissing|TestClient_SyncReusesExistingRepoAndRemovesStaleFile|TestClient_SyncFallbacksToCloneWhenDirIsNotGitRepo|TestClient_SyncFallbacksToCloneWhenOriginMismatches|TestCloneCommandUsesProxyAndProgressOptions|TestCloneCommandWithoutProgressLeavesWritersNil|TestRevParseCommandIgnoresProxyAndProgress|TestClient_SyncWithoutGitBinaryReturnsActionableError'
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/infra/gitx/client.go internal/infra/gitx/client_test.go
git commit -m "feat(gitx): add incremental source sync"
```

### Task 2: Remove app-layer cache deletion and update service expectations

**Files:**
- Modify: `internal/app/sourceapp/service.go`
- Modify: `internal/app/sourceapp/service_test.go`

- [ ] **Step 1: Write the failing service test**

Replace the old cache-deletion-focused test in `internal/app/sourceapp/service_test.go` with this behavior-focused test:

```go
func TestService_SyncGitSourceReusesExistingCachePath(t *testing.T) {
    baseDir := t.TempDir()
    configFile := filepath.Join(baseDir, "skillc.yaml")
    service := NewService(configFile, baseDir)

    callCount := 0
    service.git = gitRunnerStub{syncFn: func(url, dir, ref string, opts gitx.SyncOptions) (string, error) {
        callCount++
        assert.NoErr(t, os.MkdirAll(dir, 0o755))
        return "deadbeefcafebabe", nil
    }}

    src, err := service.AddGit("https://example.com/repo.git", "main")
    assert.NoErr(t, err)

    err = service.Sync(src.ID)
    assert.NoErr(t, err)

    list, err := service.List()
    assert.NoErr(t, err)
    firstPath := list[0].Path
    assert.NotEmpty(t, firstPath)
    assert.NoErr(t, os.WriteFile(filepath.Join(firstPath, "stale.txt"), []byte("old"), 0o644))

    err = service.Sync(src.ID)
    assert.NoErr(t, err)

    list, err = service.List()
    assert.NoErr(t, err)
    assert.Eq(t, 2, callCount)
    assert.Eq(t, firstPath, list[0].Path)
}
```

This test pins down the app-layer contract: same cache path, repeated sync works, and the service no longer owns directory cleanup semantics.

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/app/sourceapp -run TestService_SyncGitSourceReusesExistingCachePath
```

Expected: FAIL because `sourceapp.Service.Sync` still deletes `targetDir` before calling `s.git.Sync`.

- [ ] **Step 3: Write the minimal implementation**

In `internal/app/sourceapp/service.go`, delete the unconditional cache removal block so the Git path becomes:

```go
// Git 源同步
targetDir := filepath.Join(data.RepoCacheDir, src.ID)

ccolor.Infof("Syncing Git source %s to %s\n", src.ID, targetDir)
resolvedRef, err := s.git.Sync(src.URL, targetDir, src.Ref, s.gitSyncOptions(data))
if err != nil {
    data.Sources[i].Status = "error"
    data.Sources[i].ErrorMessage = err.Error()
    _ = s.store.Save(s.configFile, data)
    return err
}
```

Do not change the local-source branch, timestamp handling, or index rebuild logic.

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./internal/app/sourceapp -run 'TestService_SyncGitSourceReusesExistingCachePath|TestService_SyncGitBuildsSyncOptionsWithProxyAndProgressOnTTY|TestService_SyncGitBuildsSyncOptionsWithoutProgressWhenNotTTY|TestService_AddGitAndSyncStatus|TestService_SyncMissingGitSetsSourceError'
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/sourceapp/service.go internal/app/sourceapp/service_test.go
git commit -m "refactor(source): move git cache reuse into gitx"
```

### Task 3: Update architecture/task docs and run full regression

**Files:**
- Modify: `arch.md`
- Modify: `plan.md`

- [ ] **Step 1: Write the failing doc-oriented verification checklist**

Create the exact updates below.

In `arch.md`, update the source/cache behavior section so it states:

```md
- Git source sync 优先复用现有 repo cache
- 对健康缓存执行增量同步：`fetch --prune`、`reset --hard`、`clean -fd`
- 若缓存目录损坏、不是 Git 仓库或 `origin` 与 source URL 不匹配，则回退为删除目录后重新 clone
- local source 行为保持不变
```

In `plan.md`, add a new completed item under the current source-sync related work:

```md
- [x] 优化 Git source sync：缓存目录存在且可复用时执行增量同步；缓存损坏或 origin 不匹配时回退为重新 clone；保持 sync 后缓存目录干净
```

- [ ] **Step 2: Apply the doc changes**

Edit `arch.md` and `plan.md` so the repository docs match the implemented behavior exactly.

- [ ] **Step 3: Run targeted regression tests**

Run:

```bash
go test ./internal/infra/gitx ./internal/app/sourceapp
```

Expected: PASS

- [ ] **Step 4: Run full project regression**

Run:

```bash
go test ./...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add arch.md plan.md internal/infra/gitx/client.go internal/infra/gitx/client_test.go internal/app/sourceapp/service.go internal/app/sourceapp/service_test.go
git commit -m "feat(source): use incremental git sync for repo cache"
```

---

## Self-Review

### Spec coverage
- Incremental sync on healthy cache: covered by Task 1
- Fallback re-clone on invalid repo or wrong origin: covered by Task 1
- App-layer orchestration remains stable: covered by Task 2
- `arch.md` and `plan.md` sync requirements: covered by Task 3
- Regression proof including `go test ./...`: covered by Task 3

### Placeholder scan
- No `TODO`, `TBD`, or “implement later” placeholders remain
- Every code-changing step includes concrete code or exact content
- Every verification step includes exact commands and expected outcomes

### Type consistency
- `gitx.Sync(url, dir, ref string, opts SyncOptions)` remains the central contract across tasks
- `sourceapp.Service.Sync` continues to call `s.git.Sync(...)`
- `ResolvedRef`, `Status`, `LastSyncAt`, and `ErrorMessage` naming matches current code
