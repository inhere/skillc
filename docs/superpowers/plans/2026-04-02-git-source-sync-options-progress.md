# Git Source Sync SyncOptions And Progress Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the positional `proxyURL` argument in `gitx.Sync` with a `SyncOptions` struct and make `skillc source sync` show live `git clone` progress on `stderr` only when running in an interactive terminal.

**Architecture:** Keep sync policy in `internal/app/sourceapp`: it reads `config.ProxyURL`, decides whether the process is interactive, and constructs `gitx.SyncOptions`. Keep command execution details in `internal/infra/gitx`: `git clone` gets proxy env vars plus optional progress output and `--progress`, while local commands like `rev-parse` stay silent and unproxied.

**Tech Stack:** Go, `os/exec`, `io`, `os`, `golang.org/x/term`, Go table-style unit tests with `gookit/goutil/testutil/assert`, markdown docs in `mvp-arch.md` and `mvp-plan.md`

---

## File Structure

### Documents to consult
- `docs/superpowers/specs/2026-04-01-git-source-sync-proxy-design.md` — approved behavior for `SyncOptions`, TTY-only progress, and stderr output
- `AGENTS.md` — app/infra layering, doc sync requirements, and `go test ./...` completion gate
- `mvp-arch.md` — architecture notes for source sync behavior
- `mvp-plan.md` — project task ledger that must be updated to reflect the new work

### Files to modify

**Git infrastructure**
- Modify: `internal/infra/gitx/client.go` — define `SyncOptions`, update `Sync`, attach proxy env only to clone, attach progress writer only to clone, add `--progress`
- Modify: `internal/infra/gitx/client_test.go` — cover options plumbing, clone/rev-parse separation, and progress writer behavior

**Application layer**
- Modify: `internal/app/sourceapp/service.go` — switch `gitRunner` to `SyncOptions`, add interactive detection/writer injection, build options for Git sync only
- Modify: `internal/app/sourceapp/service_test.go` — assert `ProxyURL`, `Progress`, and non-TTY behavior through the runner stub

**Project docs**
- Modify: `mvp-arch.md` — document `SyncOptions`, TTY-only clone progress, and stderr output boundary
- Modify: `mvp-plan.md` — add/update the task note for `SyncOptions` + progress behavior and mark the phase state accurately

---

### Task 1: Add `gitx.SyncOptions` and clone-only progress support

**Files:**
- Modify: `internal/infra/gitx/client.go`
- Modify: `internal/infra/gitx/client_test.go`

- [ ] **Step 1: Write the failing tests**

Update `internal/infra/gitx/client_test.go` to define the target behavior first:

```go
package gitx

import (
    "bytes"
    "path/filepath"
    "testing"

    "github.com/gookit/goutil/testutil/assert"
)

func TestCloneCommandUsesProxyAndProgressOptions(t *testing.T) {
    client := New("git")
    buf := &bytes.Buffer{}

    clone := client.cloneCommand("https://example.com/repo.git", "/tmp/repo", "main", SyncOptions{
        ProxyURL: "http://localhost:7890",
        Progress: buf,
    })

    assert.Contains(t, clone.Args, "--progress")
    assert.Contains(t, clone.Env, "HTTP_PROXY=http://localhost:7890")
    assert.Contains(t, clone.Env, "HTTPS_PROXY=http://localhost:7890")
    assert.Same(t, buf, clone.Stdout)
    assert.Same(t, buf, clone.Stderr)
}

func TestCloneCommandWithoutProgressLeavesWritersNil(t *testing.T) {
    client := New("git")

    clone := client.cloneCommand("https://example.com/repo.git", "/tmp/repo", "", SyncOptions{})

    assert.Contains(t, clone.Args, "--progress")
    assert.Nil(t, clone.Stdout)
    assert.Nil(t, clone.Stderr)
}

func TestRevParseCommandIgnoresProxyAndProgress(t *testing.T) {
    client := New("git")
    buf := &bytes.Buffer{}

    revParse := client.revParseHeadCommand("/tmp/repo")
    _ = buf

    assert.Eq(t, 0, len(revParse.Env))
    assert.Nil(t, revParse.Stdout)
    assert.Nil(t, revParse.Stderr)
}

func TestClient_SyncWithoutGitBinaryReturnsActionableError(t *testing.T) {
    client := New("__missing_git__")
    _, err := client.Sync("https://example.com/repo.git", filepath.Join(t.TempDir(), "repo"), "main", SyncOptions{
        ProxyURL: "http://localhost:7890",
    })
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "git executable not found")
}
```

Keep the existing env helper coverage, but update signatures and assertions to use `SyncOptions`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/infra/gitx -run 'TestCloneCommandUsesProxyAndProgressOptions|TestCloneCommandWithoutProgressLeavesWritersNil|TestRevParseCommandIgnoresProxyAndProgress|TestClient_SyncWithoutGitBinaryReturnsActionableError'`
Expected: FAIL because `SyncOptions` does not exist yet, `cloneCommand` still accepts `proxyURL string`, and `Sync` still uses the old signature.

- [ ] **Step 3: Write the minimal implementation**

Update `internal/infra/gitx/client.go` to this shape:

```go
package gitx

import (
    "fmt"
    "io"
    "os"
    "os/exec"
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

    cmd := c.cloneCommand(url, dir, ref, opts)
    if opts.Progress != nil {
        if err := cmd.Run(); err != nil {
            return "", fmt.Errorf("git clone failed: %w", err)
        }
    } else {
        if out, err := cmd.CombinedOutput(); err != nil {
            return "", fmt.Errorf("git clone failed: %s", string(out))
        }
    }

    resolved, err := c.revParseHead(dir)
    if err != nil {
        return "", err
    }
    return resolved, nil
}

func (c *Client) cloneCommand(url, dir, ref string, opts SyncOptions) *exec.Cmd {
    args := []string{"clone", "--progress", url, dir}
    if ref != "" {
        args = []string{"clone", "--progress", "--branch", ref, url, dir}
    }

    cmd := exec.Command(c.bin, args...)
    cmd.Env = buildGitEnv(os.Environ(), opts.ProxyURL)
    if opts.Progress != nil {
        cmd.Stdout = opts.Progress
        cmd.Stderr = opts.Progress
    }
    return cmd
}

func (c *Client) revParseHeadCommand(dir string) *exec.Cmd {
    return exec.Command(c.bin, "-C", dir, "rev-parse", "HEAD")
}
```

Keep `buildGitEnv` and `trimOutput` unchanged except for any import fixes.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/infra/gitx -run 'TestCloneCommandUsesProxyAndProgressOptions|TestCloneCommandWithoutProgressLeavesWritersNil|TestRevParseCommandIgnoresProxyAndProgress|TestClient_SyncWithoutGitBinaryReturnsActionableError'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/infra/gitx/client.go internal/infra/gitx/client_test.go
git commit -m "feat(gitx): add sync options for clone progress"
```

### Task 2: Build `SyncOptions` in `sourceapp` using TTY detection

**Files:**
- Modify: `internal/app/sourceapp/service.go`
- Modify: `internal/app/sourceapp/service_test.go`

- [ ] **Step 1: Write the failing service tests**

Update the runner stub in `internal/app/sourceapp/service_test.go`:

```go
type gitRunnerStub struct {
    syncFn func(url, dir, ref string, opts gitx.SyncOptions) (string, error)
}

func (s gitRunnerStub) Sync(url, dir, ref string, opts gitx.SyncOptions) (string, error) {
    return s.syncFn(url, dir, ref, opts)
}
```

Add tests that pin down the desired `sourceapp` behavior:

```go
func TestService_SyncGitBuildsSyncOptionsWithProxyAndProgressOnTTY(t *testing.T) {
    baseDir := t.TempDir()
    configFile := filepath.Join(baseDir, "skillc.yaml")
    cfgService := configapp.NewService(configFile, baseDir)

    _, err := cfgService.Init()
    assert.NoErr(t, err)
    assert.NoErr(t, cfgService.Set("proxy_url", "http://localhost:7890"))

    service := NewService(configFile, baseDir)
    service.isInteractive = func() bool { return true }
    progressBuf := &bytes.Buffer{}
    service.progressWriter = progressBuf

    var calledOpts gitx.SyncOptions
    service.git = gitRunnerStub{syncFn: func(url, dir, ref string, opts gitx.SyncOptions) (string, error) {
        calledOpts = opts
        return "deadbeefcafebabe", os.MkdirAll(dir, 0o755)
    }}

    src, err := service.AddGit("https://example.com/repo.git", "main")
    assert.NoErr(t, err)

    err = service.Sync(src.ID)
    assert.NoErr(t, err)
    assert.Eq(t, "http://localhost:7890", calledOpts.ProxyURL)
    assert.Same(t, progressBuf, calledOpts.Progress)
}

func TestService_SyncGitBuildsSyncOptionsWithoutProgressWhenNotTTY(t *testing.T) {
    baseDir := t.TempDir()
    configFile := filepath.Join(baseDir, "skillc.yaml")
    service := NewService(configFile, baseDir)
    service.isInteractive = func() bool { return false }

    var calledOpts gitx.SyncOptions
    service.git = gitRunnerStub{syncFn: func(url, dir, ref string, opts gitx.SyncOptions) (string, error) {
        calledOpts = opts
        return "deadbeefcafebabe", os.MkdirAll(dir, 0o755)
    }}

    src, err := service.AddGit("https://example.com/repo.git", "main")
    assert.NoErr(t, err)

    err = service.Sync(src.ID)
    assert.NoErr(t, err)
    assert.Eq(t, "", calledOpts.ProxyURL)
    assert.Nil(t, calledOpts.Progress)
}
```

Keep the local-source sync test intact so it continues proving that the local branch never uses the Git runner.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/sourceapp -run 'TestService_SyncGitBuildsSyncOptionsWithProxyAndProgressOnTTY|TestService_SyncGitBuildsSyncOptionsWithoutProgressWhenNotTTY|TestService_AddGitAndSyncStatus|TestService_SyncGit'`
Expected: FAIL because `gitRunner` still expects `proxyURL string`, `Service` has no `isInteractive` or `progressWriter`, and `s.git.Sync` still passes a string.

- [ ] **Step 3: Write the minimal implementation**

Update `internal/app/sourceapp/service.go` imports and fields:

```go
import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "time"

    "golang.org/x/term"
)

type gitRunner interface {
    Sync(url, dir, ref string, opts gitx.SyncOptions) (string, error)
}

type gitRunnerFunc func(url, dir, ref string, opts gitx.SyncOptions) (string, error)

func (f gitRunnerFunc) Sync(url, dir, ref string, opts gitx.SyncOptions) (string, error) {
    return f(url, dir, ref, opts)
}

type Service struct {
    configFile     string
    baseDir        string
    store          *configstore.YAMLStore
    git            gitRunner
    scanner        *repoindex.Scanner
    indexStore     *repoindex.Store
    now            func() time.Time
    isInteractive  func() bool
    progressWriter io.Writer
}
```

Initialize the new fields in `NewService`:

```go
func NewService(configFile string, baseDir string) *Service {
    return &Service{
        configFile:     configFile,
        baseDir:        baseDir,
        store:          configstore.NewYAMLStore(),
        git:            gitx.New("git"),
        scanner:        repoindex.NewScanner(),
        indexStore:     repoindex.NewStore(),
        now:            time.Now,
        isInteractive:  func() bool { return term.IsTerminal(int(os.Stderr.Fd())) },
        progressWriter: os.Stderr,
    }
}
```

Add a helper and use it in the Git branch:

```go
func (s *Service) gitSyncOptions(data cfg.Config) gitx.SyncOptions {
    opts := gitx.SyncOptions{ProxyURL: data.ProxyURL}
    if s.isInteractive != nil && s.isInteractive() {
        opts.Progress = s.progressWriter
    }
    return opts
}
```

Replace the Git sync call:

```go
resolvedRef, err := s.git.Sync(src.URL, targetDir, src.Ref, s.gitSyncOptions(data))
```

Do not change the local-source branch.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/sourceapp -run 'TestService_SyncGitBuildsSyncOptionsWithProxyAndProgressOnTTY|TestService_SyncGitBuildsSyncOptionsWithoutProgressWhenNotTTY|TestService_AddGitAndSyncStatus|TestService_SyncGit'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/sourceapp/service.go internal/app/sourceapp/service_test.go
git commit -m "feat(sourceapp): build git sync options from tty state"
```

### Task 3: Update architecture and task ledger docs

**Files:**
- Modify: `mvp-arch.md`
- Modify: `mvp-plan.md`

- [ ] **Step 1: Write the failing documentation assertions**

Before editing, decide the exact lines to add. Update the existing proxy note in `mvp-arch.md` section `9.2 集成测试重点` so it states the new output behavior too:

```md
- 对于 git source：若配置了 `proxy_url`，仅在 `skillc` 发起的网络型 Git 命令（当前为 `git clone`）上注入代理；`gitx.Sync` 通过 `SyncOptions` 接收代理和输出控制；交互终端下 `git clone --progress` 实时输出到 `stderr`，非交互场景不显示进度；不会写入任何 git config，本地命令如 `rev-parse` 不使用代理也不显示进度
```

Replace the old Task 26 block in `mvp-plan.md` with the expanded note:

```md
### Task 26: Add SyncOptions and TTY-only progress for git source sync

**Files:**
- Modify: `internal/app/sourceapp/service.go`
- Modify: `internal/app/sourceapp/service_test.go`
- Modify: `internal/infra/gitx/client.go`
- Modify: `internal/infra/gitx/client_test.go`
- Modify: `mvp-arch.md`
- Modify: `mvp-plan.md`

**Design note (2026-04-02):** `gitx.Sync` now moves from a positional `proxyURL` argument to `SyncOptions`, so Git sync can carry `ProxyURL`, `Progress`, `Quiet`, and `Verbose` without growing the signature again. `source sync` should show live `git clone --progress` output on `stderr` only when running in an interactive terminal.

**Verification note (2026-04-02):** Git source sync now builds `gitx.SyncOptions` in `sourceapp`, forwards config `proxy_url`, attaches progress output only on TTY, keeps `rev-parse` unproxied and silent, and still passes `go test ./...`.
```

- [ ] **Step 2: Update the docs**

Apply the edits above exactly in `mvp-arch.md` and `mvp-plan.md`.

- [ ] **Step 3: Verify the docs read cleanly**

Read these sections back and confirm:
- `mvp-arch.md` mentions `SyncOptions`, TTY-only progress, and `stderr`
- `mvp-plan.md` task title and notes match the approved spec date and behavior
- there are no old references claiming only `proxyURL string`

Expected: both files describe the same behavior without contradiction.

- [ ] **Step 4: Commit**

```bash
git add mvp-arch.md mvp-plan.md
git commit -m "docs: describe git sync options and tty progress"
```

### Task 4: Run the full regression suite and update the task ledger status

**Files:**
- Modify: `mvp-plan.md`

- [ ] **Step 1: Run focused regression tests**

Run: `go test ./internal/infra/gitx ./internal/app/sourceapp ./internal/cli`
Expected: PASS

- [ ] **Step 2: Run the full suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 3: Mark the plan ledger steps complete**

Update the Task 26 checklist in `mvp-plan.md` so it reflects reality after the tests:

```md
- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run `go test ./...`**
- [ ] **Step 6: Commit**
```

If you created the three commits above, change the last line to:

```md
- [x] **Step 6: Commit**
```

- [ ] **Step 4: Commit**

```bash
git add mvp-plan.md
git commit -m "test: verify git sync options regression coverage"
```

---

## Self-Review

### Spec coverage
- `SyncOptions` replacing positional `proxyURL` is covered in Task 1 and Task 2.
- TTY-only progress and `stderr` output are covered in Task 1 command wiring and Task 2 service option construction.
- Keeping proxy/progress off local commands like `rev-parse` is covered in Task 1 tests and implementation.
- `Quiet` / `Verbose` being present but not fully used is covered by the `SyncOptions` struct in Task 1.
- `mvp-arch.md` / `mvp-plan.md` synchronization is covered in Task 3 and Task 4.

### Placeholder scan
- No `TODO`, `TBD`, or “similar to” shortcuts remain.
- Every code-changing step includes concrete code and exact commands.
- The plan references only concrete files already present in the repo.

### Type consistency
- The plan consistently uses `gitx.SyncOptions`.
- The `gitRunner` interface is consistently `Sync(url, dir, ref string, opts gitx.SyncOptions)`.
- `Service` consistently uses `isInteractive func() bool` and `progressWriter io.Writer`.

Plan complete and saved to `docs/superpowers/plans/2026-04-02-git-source-sync-options-progress.md`. Two execution options:

1. Subagent-Driven (recommended) - I dispatch a fresh subagent per task, review between tasks, fast iteration

2. Inline Execution - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
