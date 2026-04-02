# Git Source Sync Proxy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `skillc source sync` use `proxy_url` for Git sources by applying the proxy only to network Git commands started by skillc, without touching any Git config.

**Architecture:** Keep proxy policy in `internal/app/sourceapp` by reading `config.ProxyURL` and passing it explicitly into the Git infrastructure layer. Update `internal/infra/gitx` so only network Git commands (currently `git clone`) get per-process proxy environment variables, while local commands like `rev-parse` stay unchanged.

**Tech Stack:** Go, `os/exec`, Go unit tests, existing `sourceapp` service, existing `gitx` client, markdown docs in `arch.md` and `plan.md`

---

## File Structure

### Documents to consult
- `docs/superpowers/specs/2026-04-01-git-source-sync-proxy-design.md` — approved behavior and scope
- `AGENTS.md` — CLI/app layering, doc sync, and `go test ./...` requirement
- `arch.md` — source sync flow and architectural rules
- `plan.md` — task ledger that must be updated as work progresses

### Files to create or modify

**Application layer**
- Modify: `internal/app/sourceapp/service.go` — pass `ProxyURL` from config into the Git runner only for Git sources
- Modify: `internal/app/sourceapp/service_test.go` — prove Git sync forwards proxy values and local sync remains unchanged

**Git infrastructure**
- Modify: `internal/infra/gitx/client.go` — accept `proxyURL`, inject proxy env vars for `git clone`, and keep `rev-parse` unproxied
- Modify: `internal/infra/gitx/client_test.go` — cover command construction / env behavior and keep the missing-binary test green

**Project docs**
- Modify: `arch.md` — document Git source sync proxy behavior
- Modify: `plan.md` — add this task and mark each TDD phase as it completes

---

### Task 1: Pass `ProxyURL` through `sourceapp` Git sync

**Files:**
- Modify: `internal/app/sourceapp/service.go`
- Modify: `internal/app/sourceapp/service_test.go`

- [ ] **Step 1: Write the failing service tests**

Update `internal/app/sourceapp/service_test.go` so the stub captures `proxyURL` too:

```go
type gitRunnerStub struct {
    syncFn func(url, dir, ref, proxyURL string) (string, error)
}

func (s gitRunnerStub) Sync(url, dir, ref, proxyURL string) (string, error) {
    return s.syncFn(url, dir, ref, proxyURL)
}
```

Add these tests:

```go
func TestService_SyncGitPassesConfiguredProxyURL(t *testing.T) {
    baseDir := t.TempDir()
    configFile := filepath.Join(baseDir, "skillc.yaml")
    cfgService := configapp.NewService(configFile, baseDir)

    _, err := cfgService.Init()
    assert.NoErr(t, err)
    assert.NoErr(t, cfgService.Set("proxy_url", "http://localhost:7890"))

    service := NewService(configFile, baseDir)
    calledProxy := ""
    service.git = gitRunnerStub{syncFn: func(url, dir, ref, proxyURL string) (string, error) {
        calledProxy = proxyURL
        return "deadbeefcafebabe", os.MkdirAll(dir, 0o755)
    }}

    src, err := service.AddGit("https://example.com/repo.git", "main")
    assert.NoErr(t, err)

    err = service.Sync(src.ID)
    assert.NoErr(t, err)
    assert.Eq(t, "http://localhost:7890", calledProxy)
}

func TestService_SyncGitPassesEmptyProxyWhenUnset(t *testing.T) {
    baseDir := t.TempDir()
    configFile := filepath.Join(baseDir, "skillc.yaml")
    service := NewService(configFile, baseDir)

    calledProxy := "unexpected"
    service.git = gitRunnerStub{syncFn: func(url, dir, ref, proxyURL string) (string, error) {
        calledProxy = proxyURL
        return "deadbeefcafebabe", os.MkdirAll(dir, 0o755)
    }}

    src, err := service.AddGit("https://example.com/repo.git", "main")
    assert.NoErr(t, err)

    err = service.Sync(src.ID)
    assert.NoErr(t, err)
    assert.Eq(t, "", calledProxy)
}
```

Also update existing Git stub usages from:

```go
service.git = gitRunnerStub{syncFn: func(url, dir, ref string) (string, error) {
```

to:

```go
service.git = gitRunnerStub{syncFn: func(url, dir, ref, proxyURL string) (string, error) {
```

For local sync, keep the existing `TestService_SyncLocalRebuildsIndex` unchanged so it still proves the local path avoids the Git runner.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/sourceapp -run 'TestService_SyncGitPassesConfiguredProxyURL|TestService_SyncGitPassesEmptyProxyWhenUnset|TestService_AddGit|TestService_SyncGit'`
Expected: FAIL because `gitRunner` / `gitRunnerStub` / `service.git` call sites still use the old `Sync(url, dir, ref)` signature.

- [ ] **Step 3: Write minimal `sourceapp` implementation**

Update `internal/app/sourceapp/service.go`:

```go
type gitRunner interface {
    Sync(url, dir, ref, proxyURL string) (string, error)
}

type gitRunnerFunc func(url, dir, ref, proxyURL string) (string, error)

func (f gitRunnerFunc) Sync(url, dir, ref, proxyURL string) (string, error) {
    return f(url, dir, ref, proxyURL)
}
```

Then change the Git sync call inside `Service.Sync` from:

```go
resolvedRef, err := s.git.Sync(src.URL, targetDir, src.Ref)
```

to:

```go
resolvedRef, err := s.git.Sync(src.URL, targetDir, src.Ref, data.ProxyURL)
```

Do not change the local-source branch.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/sourceapp -run 'TestService_SyncGitPassesConfiguredProxyURL|TestService_SyncGitPassesEmptyProxyWhenUnset|TestService_AddGit|TestService_SyncGit'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/sourceapp/service.go internal/app/sourceapp/service_test.go
git commit -m "feat(sourceapp): pass proxy url to git sync"
```

### Task 2: Inject proxy env vars only for network Git commands

**Files:**
- Modify: `internal/infra/gitx/client.go`
- Modify: `internal/infra/gitx/client_test.go`

- [ ] **Step 1: Write the failing Git client tests**

Replace the current narrow test in `internal/infra/gitx/client_test.go` with a testable command-construction seam.

Add these tests:

```go
func TestBuildGitEnvAddsProxyVariables(t *testing.T) {
    got := buildGitEnv([]string{"PATH=/usr/bin"}, "http://localhost:7890")

    assert.Contains(t, got, "HTTP_PROXY=http://localhost:7890")
    assert.Contains(t, got, "HTTPS_PROXY=http://localhost:7890")
    assert.Contains(t, got, "http_proxy=http://localhost:7890")
    assert.Contains(t, got, "https_proxy=http://localhost:7890")
}

func TestBuildGitEnvLeavesEnvUnchangedWhenProxyEmpty(t *testing.T) {
    base := []string{"PATH=/usr/bin"}
    got := buildGitEnv(base, "")

    assert.DeepEq(t, base, got)
}

func TestCloneCommandUsesProxyButRevParseDoesNot(t *testing.T) {
    client := New("git")

    clone := client.cloneCommand("https://example.com/repo.git", "/tmp/repo", "main", "http://localhost:7890")
    assert.Contains(t, clone.Env, "HTTP_PROXY=http://localhost:7890")
    assert.Contains(t, clone.Env, "HTTPS_PROXY=http://localhost:7890")

    revParse := client.revParseHeadCommand("/tmp/repo")
    assert.Eq(t, 0, len(revParse.Env))
}
```

Keep and update the missing-binary test to call the new signature:

```go
func TestClient_SyncWithoutGitBinaryReturnsActionableError(t *testing.T) {
    client := New("__missing_git__")
    _, err := client.Sync("https://example.com/repo.git", filepath.Join(t.TempDir(), "repo"), "main", "http://localhost:7890")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "git executable not found")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/infra/gitx -run 'TestBuildGitEnv|TestCloneCommandUsesProxyButRevParseDoesNot|TestClient_SyncWithoutGitBinaryReturnsActionableError'`
Expected: FAIL with undefined `buildGitEnv`, `cloneCommand`, `revParseHeadCommand`, and the old `Sync` signature.

- [ ] **Step 3: Write minimal Git client implementation**

Update `internal/infra/gitx/client.go` like this:

```go
func (c *Client) Sync(url, dir, ref, proxyURL string) (string, error) {
    if _, err := exec.LookPath(c.bin); err != nil {
        return "", fmt.Errorf("git executable not found: %w", err)
    }

    cmd := c.cloneCommand(url, dir, ref, proxyURL)
    if out, err := cmd.CombinedOutput(); err != nil {
        return "", fmt.Errorf("git clone failed: %s", string(out))
    }

    resolved, err := c.revParseHead(dir)
    if err != nil {
        return "", err
    }
    return resolved, nil
}

func (c *Client) cloneCommand(url, dir, ref, proxyURL string) *exec.Cmd {
    args := []string{"clone", url, dir}
    if ref != "" {
        args = []string{"clone", "--branch", ref, url, dir}
    }
    cmd := exec.Command(c.bin, args...)
    cmd.Env = buildGitEnv(os.Environ(), proxyURL)
    return cmd
}

func (c *Client) revParseHead(dir string) (string, error) {
    cmd := c.revParseHeadCommand(dir)
    out, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("git rev-parse failed: %s", string(out))
    }
    return trimOutput(string(out)), nil
}

func (c *Client) revParseHeadCommand(dir string) *exec.Cmd {
    return exec.Command(c.bin, "-C", dir, "rev-parse", "HEAD")
}

func buildGitEnv(base []string, proxyURL string) []string {
    if proxyURL == "" {
        return base
    }
    env := append([]string{}, base...)
    env = append(env,
        "HTTP_PROXY="+proxyURL,
        "HTTPS_PROXY="+proxyURL,
        "http_proxy="+proxyURL,
        "https_proxy="+proxyURL,
    )
    return env
}
```

Add `os` to the imports.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/infra/gitx -run 'TestBuildGitEnv|TestCloneCommandUsesProxyButRevParseDoesNot|TestClient_SyncWithoutGitBinaryReturnsActionableError'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/infra/gitx/client.go internal/infra/gitx/client_test.go
git commit -m "feat(gitx): proxy git clone during source sync"
```

### Task 3: Sync architecture and task ledger docs

**Files:**
- Modify: `arch.md`
- Modify: `plan.md`

- [ ] **Step 1: Write the failing doc expectations as assertions in your head and diff target text**

Add this design note to `plan.md` under a new task section:

```md
### Task 26: Add proxy support for git source sync

**Files:**
- Modify: `internal/app/sourceapp/service.go`
- Modify: `internal/app/sourceapp/service_test.go`
- Modify: `internal/infra/gitx/client.go`
- Modify: `internal/infra/gitx/client_test.go`
- Modify: `arch.md`
- Modify: `plan.md`

**Design note (2026-04-01):** When `source` type is `git` and config `proxy_url` is set, `source sync` must apply that proxy only to skillc-started network Git commands such as `git clone`. It must not write any Git config, and local Git commands such as `rev-parse` must continue to run without proxy injection.

- [ ] **Step 1: Write the failing tests**
- [ ] **Step 2: Run tests to verify they fail**
- [ ] **Step 3: Write minimal implementation**
- [ ] **Step 4: Run tests to verify they pass**
- [ ] **Step 5: Run `go test ./...`**
- [ ] **Step 6: Commit**
```

Add this bullet to the source-sync flow in `arch.md` near the `skillc source sync` / source workflow section:

```md
- 对于 git source：若配置了 `proxy_url`，仅在 `skillc` 发起的网络型 Git 命令（当前为 `git clone`）上注入代理；不会写入任何 git config，本地命令如 `rev-parse` 不使用代理。
```

At this step the docs are intentionally still unchecked in `plan.md`; they should only be marked complete after code and verification finish.

- [ ] **Step 2: Apply the doc updates**

Edit `arch.md` and `plan.md` with the exact text above.

- [ ] **Step 3: Run a focused diff review**

Run: `git diff -- arch.md plan.md`
Expected: only the new Git proxy behavior note and the new Task 26 ledger entry appear.

- [ ] **Step 4: Mark the plan task complete after verification**

After Task 4 succeeds, replace the unchecked checklist in `plan.md` with:

```md
**Verification note (2026-04-01):** Git source sync now forwards config `proxy_url` into the Git client, injects proxy env vars only for `git clone`, keeps local `rev-parse` unproxied, and passes `go test ./...`.

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run `go test ./...`**
- [x] **Step 6: Commit**
```

- [ ] **Step 5: Commit**

```bash
git add arch.md plan.md
git commit -m "docs: record git source sync proxy behavior"
```

### Task 4: Run regression verification and finish the ledger

**Files:**
- Modify: `plan.md`

- [ ] **Step 1: Run focused regression tests**

Run: `go test ./internal/app/sourceapp ./internal/infra/gitx`
Expected: PASS

- [ ] **Step 2: Run full regression tests**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 3: Update `plan.md` verification note and checkboxes**

Use this exact block from Task 3 Step 4:

```md
**Verification note (2026-04-01):** Git source sync now forwards config `proxy_url` into the Git client, injects proxy env vars only for `git clone`, keeps local `rev-parse` unproxied, and passes `go test ./...`.

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run `go test ./...`**
- [x] **Step 6: Commit**
```

- [ ] **Step 4: Commit the verification update if needed**

```bash
git add plan.md
git commit -m "test: verify git source sync proxy support"
```

---

## Self-Review

- **Spec coverage:** The plan covers explicit proxy forwarding in `sourceapp`, proxy injection only for network Git commands in `gitx`, unchanged local Git behavior, doc sync in `arch.md`, and task tracking in `plan.md`.
- **Placeholder scan:** No `TBD`, `TODO`, or "write tests later" placeholders remain.
- **Type consistency:** All tasks use the same `Sync(url, dir, ref, proxyURL string)` signature and the same verification note text.
