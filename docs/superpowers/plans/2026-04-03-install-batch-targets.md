# Install Batch Targets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add batch install target parsing to `skillc install`, including comma-separated targets, explicit collection mode, `skill.ID` prefix expansion, confirmation skipping with `-y`, and partial-success execution.

**Architecture:** Keep command-line parsing and user interaction in `internal/cli/manage_cmd.go`, move batch target resolution into `internal/app/searchapp/service.go`, and extend `internal/app/installapp/service.go` to aggregate install failures instead of aborting on the first one. Preserve existing restore behavior when no `skill-id` is provided, and update `arch.md` and `plan.md` so the documented install semantics match the implementation.

**Tech Stack:** Go, gookit/gcli, standard library, existing `repoindex` search resolution, Go testing package

---

## File Structure

### Documents to consult
- `docs/superpowers/specs/2026-04-03-install-batch-targets-design.md` — approved behavior for `-c`, `-y`, mixed targets, prefix matching, and partial success
- `arch.md` — update CLI/app/install responsibilities and install semantics
- `plan.md` — add a completed follow-up task for batch install targets and confirmation semantics

### Planned files

**CLI**
- Modify: `internal/cli/manage_cmd.go` — add `-c` / `-y`, split raw targets, call batch resolver, handle confirmation, print resolve/install summaries
- Modify: `internal/cli/app_test.go` — command-level regression coverage for batch install behavior

**Search / target resolution**
- Modify: `internal/app/searchapp/service.go` — add batch install target resolver entrypoint and result types
- Modify: `internal/app/searchapp/service_test.go` — cover comma targets, collection-only mode, prefix expansion, partial resolution, and dedup
- Reuse: `internal/infra/repoindex/search.go` — keep existing exact target semantics; batch resolver builds on current `Resolve` behavior plus `skill.ID` prefix matching

**Install execution**
- Modify: `internal/app/installapp/service.go` — extend request/result types and add partial-success batch install aggregation
- Modify: `internal/app/installapp/service_test.go` — cover continuing past failed installs and lock-file writes for successful items only

**Docs**
- Modify: `arch.md` — record explicit collection install mode, prefix matching by `skill.ID`, and partial-success install semantics
- Modify: `plan.md` — add this follow-up task and mark it complete when done

---

### Task 1: Add batch install target resolution in searchapp

**Files:**
- Modify: `internal/app/searchapp/service.go`
- Modify: `internal/app/searchapp/service_test.go`

- [ ] **Step 1: Write the failing tests**

Add tests for mixed target parsing, explicit collection mode, prefix expansion, partial failures, and dedup.

```go
func TestService_ResolveInstallTargets_MixedTargetsAndFailures(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", Collection: "marketplaces"},
		{ID: "hello-tools", QualifiedName: "marketplaces/hello-tools", SourceQualifiedName: "repo-a/marketplaces/hello-tools", Collection: "marketplaces"},
		{ID: "world-skill", QualifiedName: "marketplaces/world-skill", SourceQualifiedName: "repo-a/marketplaces/world-skill", Collection: "marketplaces"},
	}))

	service := NewService(indexPath)
	result, err := service.ResolveInstallTargets([]string{"hello-skill", "hello-*", "missing"}, false)
	assert.NoErr(t, err)
	assert.Len(t, result.Resolved, 2)
	assert.Eq(t, "hello-skill", result.Resolved[0].ID)
	assert.Eq(t, "hello-tools", result.Resolved[1].ID)
	assert.Len(t, result.Failed, 1)
	assert.Eq(t, "missing", result.Failed[0].Target)
}

func TestService_ResolveInstallTargets_CollectionModeRequiresCollectionResolution(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", Collection: "marketplaces"},
		{ID: "world-skill", QualifiedName: "marketplaces/world-skill", SourceQualifiedName: "repo-a/marketplaces/world-skill", Collection: "marketplaces"},
	}))

	service := NewService(indexPath)
	result, err := service.ResolveInstallTargets([]string{"repo-a/marketplaces", "missing"}, true)
	assert.NoErr(t, err)
	assert.Len(t, result.Resolved, 2)
	assert.Len(t, result.Failed, 1)
	assert.Eq(t, "missing", result.Failed[0].Target)
}

func TestService_ResolveInstallTargets_DoesNotAutoExpandCollectionWithoutFlag(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", Collection: "marketplaces"},
	}))

	service := NewService(indexPath)
	result, err := service.ResolveInstallTargets([]string{"marketplaces"}, false)
	assert.NoErr(t, err)
	assert.Len(t, result.Resolved, 0)
	assert.Len(t, result.Failed, 1)
	assert.Contains(t, result.Failed[0].Reason, "skill not found")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
go test ./internal/app/searchapp -run ResolveInstallTargets -count=1
```

Expected: FAIL because `ResolveInstallTargets` and its result types do not exist yet.

- [ ] **Step 3: Write minimal implementation**

Extend `internal/app/searchapp/service.go` with install-target result types plus a resolver that delegates exact skill resolution to `Resolve`, expands `prefix-*` against `skill.ID`, forces collection resolution when `collectionMode=true`, and deduplicates by `SourceQualifiedName` fallbacking to `QualifiedName`/`ID`.

```go
type InstallTargetResolveResult struct {
	Resolved []skill.Skill
	Failed   []TargetError
}

type TargetError struct {
	Target string
	Reason string
}

func (s *Service) ResolveInstallTargets(targets []string, collectionMode bool) (InstallTargetResolveResult, error) {
	items, err := s.store.Load(s.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return InstallTargetResolveResult{}, nil
		}
		return InstallTargetResolveResult{}, err
	}

	result := InstallTargetResolveResult{
		Resolved: make([]skill.Skill, 0),
		Failed:   make([]TargetError, 0),
	}
	seen := make(map[string]struct{})

	for _, target := range targets {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}

		var matches []skill.Skill
		if collectionMode {
			matches, err = resolveCollectionInstallTargets(items, target)
		} else if strings.HasSuffix(target, "*") {
			matches = resolvePrefixInstallTargets(items, strings.TrimSuffix(target, "*"))
			if len(matches) == 0 {
				err = fmt.Errorf("skill not found: %s", target)
			} else {
				err = nil
			}
		} else {
			matches, err = ResolveSkills(items, target)
			if err == nil && len(matches) != 1 {
				err = fmt.Errorf("target resolves multiple skills: %s", target)
			}
		}

		if err != nil {
			result.Failed = append(result.Failed, TargetError{Target: target, Reason: err.Error()})
			continue
		}
		for _, item := range matches {
			key := dedupeInstallTargetKey(item)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result.Resolved = append(result.Resolved, item)
		}
	}
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/app/searchapp -run ResolveInstallTargets -count=1
```

Expected: PASS with the new target resolution tests green.

- [ ] **Step 5: Commit**

```bash
git add internal/app/searchapp/service.go internal/app/searchapp/service_test.go
git commit -m "feat(install): resolve batch install targets"
```

---

### Task 2: Add partial-success batch install execution in installapp

**Files:**
- Modify: `internal/app/installapp/service.go`
- Modify: `internal/app/installapp/service_test.go`

- [ ] **Step 1: Write the failing tests**

Add tests proving batch installation continues after one item fails and writes locks only for successful items.

```go
func TestService_InstallMultiContinuesAfterInstallFailure(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	targetRoot := filepath.Join(baseDir, ".claude", "skills")
	goodSource := filepath.Join(baseDir, "good")
	badSource := filepath.Join(baseDir, "bad")
	assert.NoErr(t, os.MkdirAll(filepath.Join(goodSource, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(goodSource, "commands", "ok.txt"), []byte("ok"), 0o644))

	service := NewService(lockFile)
	result, err := service.InstallMulti([]skill.Skill{
		{ID: "good-skill", QualifiedName: "demo/good-skill", SourceQualifiedName: "repo-a/demo/good-skill", SourceID: "src-a", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: goodSource},
		{ID: "bad-skill", QualifiedName: "demo/bad-skill", SourceQualifiedName: "repo-a/demo/bad-skill", SourceID: "src-a", SourceType: sourcepkg.TypeLocal, InstallEntry: "missing", Path: badSource},
	}, "claude-code", agent.ScopeProject, targetRoot)
	assert.NoErr(t, err)
	assert.Len(t, result.Installed, 1)
	assert.Len(t, result.Failed, 1)
	assert.Eq(t, "bad-skill", result.Failed[0].SkillID)

	locks, loadErr := service.store.Load(lockFile)
	assert.NoErr(t, loadErr)
	assert.Len(t, locks, 1)
	assert.Eq(t, "good-skill", locks[0].SkillID)
}

func TestService_RunInstallsResolvedTargetsAndReturnsSkippedFailures(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := filepath.Join(baseDir, "source")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "commands", "hello.txt"), []byte("hello"), 0o644))

	service := NewService(lockFile)
	result, err := service.RunResolved(cfg.Config{
		AgentTools: map[string]cfg.AgentToolConfig{
			"claude-code": {ProjectDir: filepath.Join(baseDir, ".claude")},
		},
	}, InstallReq{Agent: "claude-code", Scope: "project", WorkDir: baseDir}, []skill.Skill{{
		ID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: sourceDir,
	}}, []TargetError{{Target: "missing", Reason: "skill not found: missing"}})
	assert.NoErr(t, err)
	assert.Len(t, result.Installed, 1)
	assert.Len(t, result.ResolveFailed, 1)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
go test ./internal/app/installapp -run 'InstallMultiContinuesAfterInstallFailure|RunInstallsResolvedTargetsAndReturnsSkippedFailures' -count=1
```

Expected: FAIL because `InstallMulti` still returns `[]lockpkg.Record`, aborts on first failure, and `RunResolved` / result fields do not exist.

- [ ] **Step 3: Write minimal implementation**

Refactor `internal/app/installapp/service.go` so batch install reports both installed and failed items, while restore behavior remains unchanged.

```go
type InstallItemError struct {
	SkillID string
	Reason  string
}

type BatchInstallResult struct {
	Installed []lockpkg.Record
	Failed    []InstallItemError
}

type CommandResult struct {
	Installed     []lockpkg.Record
	Restored      []lockpkg.Record
	ResolveFailed []searchapp.TargetError
	InstallFailed []InstallItemError
}

func (s *Service) InstallMulti(items []skill.Skill, agentName string, scope agent.Scope, targetRoot string) (BatchInstallResult, error) {
	result := BatchInstallResult{
		Installed: make([]lockpkg.Record, 0, len(items)),
		Failed:    make([]InstallItemError, 0),
	}
	for _, item := range items {
		record, err := s.Install(item, agentName, scope, targetRoot)
		if err != nil {
			result.Failed = append(result.Failed, InstallItemError{SkillID: item.ID, Reason: err.Error()})
			continue
		}
		result.Installed = append(result.Installed, record)
	}
	return result, nil
}

func (s *Service) RunResolved(config cfg.Config, req InstallReq, items []skill.Skill, resolveFailed []searchapp.TargetError) (CommandResult, error) {
	scope, err := parseScope(req.Scope)
	if err != nil {
		return CommandResult{}, err
	}
	targetRoot, err := agent.ResolveInstallPath(config, req.WorkDir, req.Agent, scope)
	if err != nil {
		return CommandResult{}, err
	}
	batch, err := s.InstallMulti(items, req.Agent, scope, targetRoot)
	if err != nil {
		return CommandResult{}, err
	}
	if len(batch.Installed) == 0 && len(resolveFailed) > 0 && len(batch.Failed) == 0 {
		return CommandResult{ResolveFailed: resolveFailed}, nil
	}
	return CommandResult{
		Installed:     batch.Installed,
		ResolveFailed: resolveFailed,
		InstallFailed: batch.Failed,
	}, nil
}
```

Keep `Run()` restore path unchanged; for non-restore installs it may continue to call the single-target lookup path until CLI is migrated in Task 3.

- [ ] **Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/app/installapp -run 'InstallMultiContinuesAfterInstallFailure|RunInstallsResolvedTargetsAndReturnsSkippedFailures' -count=1
```

Expected: PASS with partial-success semantics and lock-file behavior covered.

- [ ] **Step 5: Commit**

```bash
git add internal/app/installapp/service.go internal/app/installapp/service_test.go
git commit -m "feat(install): allow partial batch install success"
```

---

### Task 3: Wire batch install flags, confirmation, and output in the CLI

**Files:**
- Modify: `internal/cli/manage_cmd.go`
- Modify: `internal/cli/app_test.go`

- [ ] **Step 1: Write the failing tests**

Add CLI tests for comma-separated targets, `-c`, `-y`, skipped resolve failures, and install failures continuing past a bad item. Use `-y` in tests to avoid interactive stdin.

```go
func TestInstallCommand_InstallsCommaSeparatedTargetsWithoutPromptWhenYes(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexPath := filepath.Join(baseDir, "cache", "index.json")
	firstSource := filepath.Join(baseDir, "source", "hello-skill")
	secondSource := filepath.Join(baseDir, "source", "hello-tools")
	assert.NoErr(t, os.MkdirAll(filepath.Join(firstSource, "commands"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(secondSource, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(firstSource, "commands", "hello.txt"), []byte("hello"), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(secondSource, "commands", "tools.txt"), []byte("tools"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = filepath.Join(baseDir, "skillc-install.lock")
	config.IndexFile = indexPath
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{ProjectDir: filepath.Join(baseDir, "project-claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexPath, []skill.Skill{
		{ID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: firstSource},
		{ID: "hello-tools", QualifiedName: "marketplaces/hello-tools", SourceQualifiedName: "repo-a/marketplaces/hello-tools", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: secondSource},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"install", "--agent", "claude-code", "--yes", "hello-skill,hello-*"})
	assert.Contains(t, output, "installed hello-skill")
	assert.Contains(t, output, "installed hello-tools")
}

func TestInstallCommand_CollectionFlagExpandsCollectionTargets(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexPath := filepath.Join(baseDir, "cache", "index.json")
	config := cfg.DefaultConfig()
	config.LockFile = filepath.Join(baseDir, "skillc-install.lock")
	config.IndexFile = indexPath
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{ProjectDir: filepath.Join(baseDir, "project-claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexPath, []skill.Skill{
		{ID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", Collection: "marketplaces", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: filepath.Join(baseDir, "hello-skill")},
		{ID: "world-skill", QualifiedName: "marketplaces/world-skill", SourceQualifiedName: "repo-a/marketplaces/world-skill", Collection: "marketplaces", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: filepath.Join(baseDir, "world-skill")},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"install", "--agent", "claude-code", "--collection", "--yes", "repo-a/marketplaces"})
	assert.Contains(t, output, "installed hello-skill")
	assert.Contains(t, output, "installed world-skill")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
go test ./internal/cli -run 'InstallCommand_InstallsCommaSeparatedTargetsWithoutPromptWhenYes|InstallCommand_CollectionFlagExpandsCollectionTargets' -count=1
```

Expected: FAIL because the CLI does not yet define `--collection` / `--yes`, does not split targets, and does not print batch summaries.

- [ ] **Step 3: Write minimal implementation**

Update `internal/cli/manage_cmd.go` so install parses flags, keeps restore when `skill-id` is empty, resolves batch targets through `searchapp`, prints a one-shot confirmation unless `--yes`, and then executes via `installapp.RunResolved`.

```go
type InstallOptions struct {
	ManageOptions
	Collection bool
	Yes        bool
}

func (io *InstallOptions) bindCommand(c *gcli.Command) {
	io.ManageOptions.bindCommand(c)
	c.BoolOpt(&io.Collection, "collection", "c", false, "treat each target as a collection")
	c.BoolOpt(&io.Yes, "yes", "y", false, "skip confirmation")
}

func splitInstallTargets(raw string) []string {
	parts := strings.Split(raw, ",")
	targets := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			targets = append(targets, part)
		}
	}
	return targets
}

func buildInstallCommand() *gcli.Command {
	var opts InstallOptions
	return &gcli.Command{
		Name:    "install",
		Desc:    "Install skills",
		Aliases: []string{"ins"},
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
			c.AddArg("skill-id", "skill id. if empty, restore from lock file")
		},
		Func: func(c *gcli.Command, _ []string) error {
			config, cwd, err := loadConfig()
			if err != nil {
				return err
			}
			req := installapp.InstallReq{SkillID: c.Arg("skill-id").String(), Agent: opts.Agent, Scope: opts.Scope, WorkDir: cwd}
			if req.SkillID == "" {
				result, runErr := installapp.NewService(config.LockFile).Run(config, req, nil)
				if runErr != nil {
					return runErr
				}
				return printInstallCommandResult(result)
			}

			targets := splitInstallTargets(req.SkillID)
			resolver := newSearchService()
			resolved, err := resolver.ResolveInstallTargets(targets, opts.Collection)
			if err != nil {
				return err
			}
			if len(resolved.Resolved) == 0 {
				for _, failed := range resolved.Failed {
					_ = WriteLine(os.Stdout, fmt.Sprintf("skipped %s: %s", failed.Target, failed.Reason))
				}
				return fmt.Errorf("no installable skills found")
			}
			if !opts.Yes {
				return fmt.Errorf("interactive confirmation not implemented yet in this task")
			}
			result, err := installapp.NewService(config.LockFile).RunResolved(config, req, resolved.Resolved, resolved.Failed)
			if err != nil {
				return err
			}
			return printInstallCommandResult(result)
		},
	}
}
```

Then replace the temporary confirmation error with a real confirmation helper before closing the task.

```go
func confirmInstall(targetCount int, resolvedCount int, failedCount int, collectionMode bool) (bool, error) {
	mode := "skill"
	if collectionMode {
		mode = "collection"
	}
	if _, err := fmt.Fprintf(os.Stdout, "about to install %d skills from %d targets (%d failed, mode=%s). continue? [y/N]: ", resolvedCount, targetCount, failedCount, mode); err != nil {
		return false, err
	}
	var answer string
	if _, err := fmt.Fscanln(os.Stdin, &answer); err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/cli -run InstallCommand -count=1
```

Expected: PASS, including the new `--yes` batch-install scenarios and the existing single-skill restore/install regressions.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/manage_cmd.go internal/cli/app_test.go
git commit -m "feat(cli): support batch install targets"
```

---

### Task 4: Update architecture and project plan docs for new install semantics

**Files:**
- Modify: `arch.md`
- Modify: `plan.md`

- [ ] **Step 1: Write the failing doc regression check**

Before editing, inspect the existing install sections and add a new follow-up task entry in `plan.md` for this feature. The change should explicitly cover `-c`, `-y`, comma-separated targets, `prefix-*`, and partial-success install execution.

Planned `arch.md` replacement content:

```md
### 2.5 人机双模式

默认输出要对人类用户友好，同时通过 `--yes`、`--dry-run`、`--json` 支持 CI/CD 与脚本化场景。

其中 `skillc install` 在批量目标场景下默认进行一次统一确认；传入 `-y/--yes` 时跳过确认并直接执行。
```

Planned install-module update:

```md
### 4.6 install

职责：
- 生成安装计划
- 做冲突检测
- 执行复制安装
- 支持 skill 级与 collection 级安装/卸载
- 支持 install 批量目标：逗号分隔多个目标、显式 `-c/--collection` collection 安装、以及仅按 `skill.ID` 的 `prefix-*` 前缀展开
- 支持 source 限定的歧义消解
- 支持部分成功：单个目标解析失败或单个 skill 安装失败时记录并跳过，不中断整个批次
- 支持卸载与批量恢复
- 驱动写入 lock file
```

- [ ] **Step 2: Apply the documentation updates**

Add a completed follow-up task to `plan.md` using the existing style near Tasks 25-27.

```md
### Task 28: Add batch install target parsing and partial-success execution ✅ Completed

**Files:**
- Modify: `internal/app/searchapp/service.go`
- Modify: `internal/app/searchapp/service_test.go`
- Modify: `internal/app/installapp/service.go`
- Modify: `internal/app/installapp/service_test.go`
- Modify: `internal/cli/manage_cmd.go`
- Modify: `internal/cli/app_test.go`
- Modify: `arch.md`
- Modify: `plan.md`

**Design note (2026-04-03):** `skillc install` now accepts comma-separated targets, explicit collection mode via `-c/--collection`, and `prefix-*` expansion only against `skill.ID`. Without `-c`, collection-like targets are not auto-expanded. `-y/--yes` skips confirmation, and both target-resolution failures and per-skill install failures are reported and skipped so the batch can partially succeed.

**Verification note (2026-04-03):** CLI, searchapp, and installapp tests verify mixed target parsing, collection-only resolution, `skill.ID` prefix expansion, skipped failed targets, continued execution after single-item install failure, and successful lock writes for installed items only.
```

- [ ] **Step 3: Run focused verification**

Run:
```bash
go test ./internal/cli ./internal/app/searchapp ./internal/app/installapp -count=1
```

Expected: PASS after code and docs are aligned.

- [ ] **Step 4: Run full regression suite**

Run:
```bash
go test ./... 
```

Expected: PASS per project instruction for MVP-path changes.

- [ ] **Step 5: Commit**

```bash
git add arch.md plan.md
git commit -m "docs: record batch install target semantics"
```

---

## Self-Review Checklist

### Spec coverage
- `-c/--collection` explicit collection mode: covered in Task 1 tests/implementation and Task 3 CLI wiring
- `-y/--yes` skip confirmation: covered in Task 3
- comma-separated multi-target input: covered in Task 3 splitting and tests
- `prefix-*` matches only `skill.ID`: covered in Task 1
- no implicit collection expansion without `-c`: covered in Task 1
- partial success for resolve failures and install failures: covered in Tasks 1-3
- docs sync in `arch.md` and `plan.md`: covered in Task 4

### Placeholder scan
- No `TODO` / `TBD`
- Every task names exact files
- Every code-changing step includes concrete code snippets
- Every verification step includes exact commands and expected outcomes

### Type consistency
- Search resolver result types use `TargetError`
- Install execution result types use `InstallItemError` and `BatchInstallResult`
- CLI handoff uses `ResolveInstallTargets(...)` then `RunResolved(...)`
- `InstallReq` remains the CLI input carrier; restore still uses empty `SkillID`
