# Skillc MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Skillc CLI MVP that can manage local/Git sources, index skills from `SKILL.md`, search them, install/uninstall them for multiple agents, and restore installations from a lock file.

**Architecture:** Use the approved layered design from `mvp-arch.md`: `cmd/` for CLI wiring, `internal/app/` for use-case orchestration, `internal/domain/` for stable models and rules, and `internal/infra/` for filesystem, Git, config, lock, hashing, and indexing implementations. Deliver in vertical slices so each milestone leaves the repo in a runnable, testable state.

**Tech Stack:** Go, standard library, YAML config parsing, JSON lock file, Git CLI integration, filesystem-based cache, Go testing package

---

## File Structure

### Documents to consult
- `prd.md` — product requirements and acceptance criteria
- `prd-review.md` — clarified decisions and resolved blockers
- `mvp-arch.md` — approved architecture and module boundaries

### Planned files

**Bootstrap / CLI**
- Create: `go.mod` — module definition
- Create: `cmd/skillc/main.go` — binary entrypoint
- Create: `internal/cli/app.go` — root command setup and command registration
- Create: `internal/cli/output.go` — shared stdout/stderr/json formatting helpers
- Create: `internal/cli/errors.go` — user-facing error to exit-code mapping

**Config**
- Create: `internal/domain/config/model.go` — config structs
- Create: `internal/domain/config/defaults.go` — default path and default agent config rules
- Create: `internal/infra/configstore/yaml_store.go` — YAML load/save
- Create: `internal/app/configapp/service.go` — `init/show/get/set`
- Test: `internal/domain/config/defaults_test.go`
- Test: `internal/infra/configstore/yaml_store_test.go`
- Test: `internal/app/configapp/service_test.go`

**Shared infra**
- Create: `internal/infra/fsx/paths.go` — `~`, env var, relative path expansion
- Create: `internal/infra/fsx/atomic.go` — atomic write helpers
- Create: `internal/infra/filelock/lock.go` — advisory file lock wrapper
- Create: `internal/infra/hashx/sha256.go` — checksum helpers
- Test: `internal/infra/fsx/paths_test.go`
- Test: `internal/infra/fsx/atomic_test.go`
- Test: `internal/infra/filelock/lock_test.go`
- Test: `internal/infra/hashx/sha256_test.go`

**Source management**
- Create: `internal/domain/source/model.go` — source types and state
- Create: `internal/app/sourceapp/service.go` — add/list/remove/sync/status orchestration
- Create: `internal/infra/gitx/client.go` — git clone/fetch/rev-parse wrapper
- Create: `internal/infra/sourcestore/config_sources.go` — config-backed source persistence helpers
- Test: `internal/domain/source/model_test.go`
- Test: `internal/app/sourceapp/service_test.go`
- Test: `internal/infra/gitx/client_test.go`

**Skill indexing and search**
- Create: `internal/domain/skill/model.go` — normalized skill metadata
- Create: `internal/domain/skill/parser.go` — `SKILL.md` front matter parser
- Create: `internal/infra/repoindex/scanner.go` — scan source cache dirs into skills
- Create: `internal/infra/repoindex/store.go` — index cache persistence
- Create: `internal/infra/repoindex/search.go` — filter/search helpers
- Create: `internal/app/searchapp/service.go` — `search/show`
- Test: `internal/domain/skill/parser_test.go`
- Test: `internal/infra/repoindex/scanner_test.go`
- Test: `internal/infra/repoindex/search_test.go`
- Test: `internal/app/searchapp/service_test.go`

**Agent install path resolution**
- Create: `internal/domain/agent/model.go` — agent names and scope types
- Create: `internal/domain/agent/resolver.go` — global/project path resolution
- Test: `internal/domain/agent/resolver_test.go`

**Install / uninstall / restore / list**
- Create: `internal/domain/install/model.go` — install plan and conflict types
- Create: `internal/domain/install/planner.go` — install plan generation
- Create: `internal/domain/install/conflict.go` — conflict detection rules
- Create: `internal/domain/lock/model.go` — lock file record structs
- Create: `internal/infra/lockstore/json_store.go` — JSON lock read/write
- Create: `internal/infra/agentfs/installer.go` — copy/remove installed files
- Create: `internal/app/installapp/service.go` — install/uninstall/restore orchestration
- Create: `internal/app/listapp/service.go` — installed list and status projection
- Test: `internal/domain/install/planner_test.go`
- Test: `internal/domain/install/conflict_test.go`
- Test: `internal/infra/lockstore/json_store_test.go`
- Test: `internal/infra/agentfs/installer_test.go`
- Test: `internal/app/installapp/service_test.go`
- Test: `internal/app/listapp/service_test.go`

**Doctor**
- Create: `internal/app/doctorapp/service.go` — environment checks
- Test: `internal/app/doctorapp/service_test.go`

**End-to-end smoke tests**
- Create: `tests/e2e/source_search_test.go` — local/git source to search flow
- Create: `tests/e2e/install_restore_test.go` — install/uninstall/restore flow

---

### Task 1: Bootstrap the Go CLI skeleton ✅ Completed

**Files:**
- Create: `go.mod`
- Create: `cmd/skillc/main.go`
- Create: `internal/cli/app.go`
- Create: `internal/cli/output.go`
- Create: `internal/cli/errors.go`

- [x] **Step 1: Write the failing bootstrap test**
- [x] **Step 2: Run test to verify it fails**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run test to verify it passes**
- [x] **Step 5: Run package tests**
- [x] **Step 6: Commit**

### Task 2: Implement config defaults and path expansion ✅ Completed

**Files:**
- Create: `internal/domain/config/model.go`
- Create: `internal/domain/config/defaults.go`
- Create: `internal/infra/fsx/paths.go`
- Test: `internal/domain/config/defaults_test.go`
- Test: `internal/infra/fsx/paths_test.go`

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Commit**

### Task 3: Implement YAML config storage and config app service ✅ Completed

**Files:**
- Create: `internal/infra/configstore/yaml_store.go`
- Create: `internal/app/configapp/service.go`
- Test: `internal/infra/configstore/yaml_store_test.go`
- Test: `internal/app/configapp/service_test.go`
- Modify: `internal/cli/app.go`

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run CLI smoke tests**
- [x] **Step 6: Commit**

### Task 4: Add shared atomic write, file lock, and checksum infra ✅ Completed

**Files:**
- Create: `internal/infra/fsx/atomic.go`
- Create: `internal/infra/filelock/lock.go`
- Create: `internal/infra/hashx/sha256.go`
- Test: `internal/infra/fsx/atomic_test.go`
- Test: `internal/infra/filelock/lock_test.go`
- Test: `internal/infra/hashx/sha256_test.go`

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Commit**

### Task 5: Add source domain model and local source management ✅ Completed

**Files:**
- Create: `internal/domain/source/model.go`
- Create: `internal/infra/sourcestore/config_sources.go`
- Create: `internal/app/sourceapp/service.go`
- Test: `internal/domain/source/model_test.go`
- Test: `internal/app/sourceapp/service_test.go`
- Modify: `internal/cli/app.go`

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Commit**

### Task 7: Parse `SKILL.md` metadata and scan source indexes ✅ Completed

**Files:**
- Create: `internal/domain/skill/model.go`
- Create: `internal/domain/skill/parser.go`
- Create: `internal/infra/repoindex/scanner.go`
- Create: `internal/infra/repoindex/store.go`
- Test: `internal/domain/skill/parser_test.go`
- Test: `internal/infra/repoindex/scanner_test.go`

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Commit**

### Task 8: Implement search and show services ✅ Completed

**Files:**
- Create: `internal/infra/repoindex/search.go`
- Create: `internal/app/searchapp/service.go`
- Test: `internal/infra/repoindex/search_test.go`
- Test: `internal/app/searchapp/service_test.go`
- Modify: `internal/cli/app.go`
- Test: `internal/cli/app_test.go`

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run CLI and internal regression tests**
- [x] **Step 6: Commit**

### Task 9: Add doctor checks ✅ Completed

**Files:**
- Create: `internal/app/doctorapp/service.go`
- Test: `internal/app/doctorapp/service_test.go`

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run internal regression tests**
- [x] **Step 6: Commit**

### Task 10: Add agent install path resolver ✅ Completed

**Files:**
- Create: `internal/domain/agent/model.go`
- Create: `internal/domain/agent/resolver.go`
- Test: `internal/domain/agent/resolver_test.go`

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run internal regression tests**
- [x] **Step 6: Commit**

### Task 11: Add install plan and conflict rules ✅ Completed

**Files:**
- Create: `internal/domain/install/model.go`
- Create: `internal/domain/install/planner.go`
- Create: `internal/domain/install/conflict.go`
- Test: `internal/domain/install/planner_test.go`
- Test: `internal/domain/install/conflict_test.go`

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run internal regression tests**
- [x] **Step 6: Commit**

### Task 12: Add lock file model and JSON store ✅ Completed

**Files:**
- Create: `internal/domain/lock/model.go`
- Create: `internal/infra/lockstore/json_store.go`
- Test: `internal/infra/lockstore/json_store_test.go`

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run internal regression tests**
- [x] **Step 6: Commit**

### Task 13: Add agent filesystem installer ✅ Completed

**Files:**
- Create: `internal/infra/agentfs/installer.go`
- Test: `internal/infra/agentfs/installer_test.go`

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run internal regression tests**
- [x] **Step 6: Commit**

### Task 14: Add install app service ✅ Completed

**Files:**
- Create: `internal/app/installapp/service.go`
- Test: `internal/app/installapp/service_test.go`

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run internal regression tests**
- [x] **Step 6: Commit**

### Task 15: Add installed list service ✅ Completed

**Files:**
- Create: `internal/app/listapp/service.go`
- Test: `internal/app/listapp/service_test.go`

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run internal regression tests**
- [x] **Step 6: Commit**

### Task 16: Add uninstall and restore service flows ✅ Completed

**Files:**
- Modify: `internal/app/installapp/service.go`
- Test: `internal/app/installapp/service_test.go`
- Modify: `internal/domain/lock/model.go`

**Verification note (2026-03-30):** Follow-up fixes verified that installs append/update lock records instead of overwriting them, persist `install_entry`, and restore from the recorded install entry path.

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run internal regression tests**
- [x] **Step 6: Commit**

### Task 17: Wire install, list, doctor, and uninstall CLI commands ✅ Completed

**Files:**
- Modify: `internal/cli/app.go`
- Test: `internal/cli/app_test.go`

**Verification note (2026-03-31):** Command-level tests now cover indexed install, lock-based restore, installed list output, source sync rebuilding the shared index for `search`, uninstall removal, and doctor health output.

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run internal regression tests**
- [x] **Step 6: Commit**

### Task 18: Add install/list/restore e2e smoke test ✅ Completed

**Files:**
- Create: `tests/e2e/install_restore_test.go`

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run full regression tests**
- [x] **Step 6: Commit**

### Task 19: Add source/search e2e smoke test ✅ Completed

**Files:**
- Create: `tests/e2e/source_search_test.go`

- [x] **Step 1: Write the test scenario**
- [x] **Step 2: Run test to verify current behavior**
- [x] **Step 3: Keep minimal implementation unchanged because flow already passes**
- [x] **Step 4: Re-run test to verify it passes**
- [x] **Step 5: Run full regression tests**
- [x] **Step 6: Commit**

### Task 20: Persist git sync metadata ✅ Completed

**Files:**
- Modify: `internal/domain/source/model.go`
- Modify: `internal/domain/skill/parser.go`
- Modify: `internal/infra/gitx/client.go`
- Modify: `internal/infra/configstore/yaml_store.go`
- Modify: `internal/app/sourceapp/service.go`
- Test: `internal/domain/skill/parser_test.go`
- Test: `internal/infra/gitx/client_test.go`
- Test: `internal/infra/configstore/yaml_store_test.go`
- Test: `internal/app/sourceapp/service_test.go`

**Verification note (2026-03-31):** Git sync now resolves and persists the synced commit SHA, records `last_sync_at` as RFC3339, preserves empty ref as default-branch sync, and keeps repeated sync working with full `go test ./...` coverage.

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run full regression tests**
- [x] **Step 6: Commit**

### Task 21: Add `source add --sync` UX ✅ Completed

**Files:**
- Modify: `internal/cli/app.go`
- Test: `internal/cli/app_test.go`
- Modify: `mvp-arch.md`
- Modify: `mvp-plan.md`

**Verification note (2026-03-31):** `source add local|git` now accepts `--sync`; when omitted, the CLI prints `next: skillc source sync <id>` so users know the required next step.

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run full regression tests**
- [x] **Step 6: Commit**

### Task 22: Add collection-aware source scanning and qualified skill identity ✅ Completed

**Files:**
- Modify: `internal/domain/skill/model.go`
- Modify: `internal/domain/lock/model.go`
- Modify: `internal/infra/repoindex/scanner.go`
- Test: `internal/infra/repoindex/scanner_test.go`
- Test: `internal/infra/lockstore/json_store_test.go`

**Design note (2026-04-01):** Source scanning must now support collection semantics. Rules: if no `skills` dir exists and exactly one skill is found, treat it as a top-level skill with no collection; if no `skills` dir exists and multiple skills are found, use `source.Name` as the collection; if exactly one `skills` dir exists anywhere under the source, use `source.Name` as the collection; if multiple `skills` dirs exist, use each `skills` parent directory name as the collection. `Skill` and lock records must persist `QualifiedName` and source-qualified names, while restore semantics remain based on `SourceID + InstallEntry`.

**Verification note (2026-04-01):** Scanner now emits collection-aware `QualifiedName` / `SourceQualifiedName`, search/install/uninstall resolve `skill` / `collection/skill` / `source/collection/skill` targets correctly with ambiguity errors, and e2e coverage verifies top-level sources, collection sources, source-qualified disambiguation, and restore via `SourceID + InstallEntry` with full `go test ./...` coverage.

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run focused regression tests**
- [x] **Step 6: Commit**

### Task 23: Add collection-aware search, show, install, and uninstall target resolution ✅ Completed

**Files:**
- Modify: `internal/infra/repoindex/search.go`
- Modify: `internal/app/searchapp/service.go`
- Modify: `internal/app/installapp/service.go`
- Modify: `internal/domain/install/planner.go`
- Modify: `internal/app/listapp/service.go`
- Modify: `internal/cli/app.go`
- Test: `internal/app/searchapp/service_test.go`
- Test: `internal/app/installapp/service_test.go`
- Test: `internal/domain/install/planner_test.go`
- Test: `internal/app/listapp/service_test.go`
- Test: `internal/cli/app_test.go`

**Design note (2026-04-01):** User-facing targets must support `skill`, `collection/skill`, `source/collection/skill`, `collection`, and `source/collection`. Primary display key is `collection/skill`; top-level skills remain bare `skill`; source qualification is only required for ambiguity. Collection-level install/uninstall must expand to all skills in the resolved collection. Ambiguous targets must fail with actionable hints rather than picking a source implicitly.

**Verification note (2026-04-01):** CLI and service resolution now prefer `QualifiedName`, keep bare top-level skill names, expand collection install/uninstall targets across matching skills, and require source qualification when names are ambiguous.

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run focused regression tests**
- [x] **Step 6: Commit**

### Task 24: Add collection-aware restore and e2e regression coverage ✅ Completed

**Files:**
- Modify: `tests/e2e/source_search_test.go`
- Modify: `tests/e2e/install_restore_test.go`
- Modify: `mvp-arch.md`
- Modify: `mvp-plan.md`

**Design note (2026-04-01):** E2E coverage must prove top-level single-skill sources still work, collection sources can be searched and installed by collection-aware names, ambiguous names require source qualification, and restore continues to copy from the path derived by `SourceID + InstallEntry` even after qualified naming is introduced.

**Verification note (2026-04-01):** E2E coverage now verifies top-level single-skill sources, collection-aware search/install flows, source-qualified ambiguity handling, `source add local --sync`, and restore continuing to copy from the path derived by `SourceID + InstallEntry`.

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run `go test ./...`**
- [x] **Step 6: Commit**

### Task 25: Add collection browsing commands ✅ Completed

**Files:**
- Create: `internal/infra/repoindex/collection.go`
- Test: `internal/infra/repoindex/collection_test.go`
- Create: `internal/cli/collection_cmd.go`
- Modify: `internal/app/searchapp/service.go`
- Modify: `internal/app/searchapp/service_test.go`
- Modify: `internal/cli/app.go`
- Modify: `internal/cli/app_test.go`
- Modify: `mvp-arch.md`
- Modify: `mvp-plan.md`

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

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run `go test ./...`**
- [ ] **Step 6: Commit**

### Task 27: Reuse healthy git repo cache for incremental source sync

**Files:**
- Modify: `internal/app/sourceapp/service.go`
- Modify: `internal/app/sourceapp/service_test.go`
- Modify: `internal/infra/gitx/client.go`
- Modify: `internal/infra/gitx/client_test.go`
- Modify: `mvp-arch.md`
- Modify: `mvp-plan.md`

**Design note (2026-04-02):** Git source sync should prefer reusing an existing repo cache when the cache directory is still a healthy clone of the same `origin`. Reused caches sync incrementally via `git fetch --prune origin`, `git reset --hard <target>`, and `git clean -fd`; missing or damaged caches, or caches whose `origin` no longer matches, must fall back to removing the cache dir and cloning again.

**Verification note (2026-04-02):** Focused regression passes for `internal/infra/gitx` and `internal/app/sourceapp`. Full `go test ./...` was attempted and still fails only at the known unrelated baseline `internal/cli/app_test.go:TestSourceSyncCommand_PrintsSourceStatusAfterSync`.

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run focused regression tests**
### Task 28: Add batch install target CLI options and partial-success flow ✅ Completed

**Files:**
- Modify: `internal/app/searchapp/service.go`
- Modify: `internal/app/searchapp/service_test.go`
- Modify: `internal/app/installapp/service.go`
- Modify: `internal/app/installapp/service_test.go`
- Modify: `internal/cli/manage_cmd.go`
- Modify: `internal/cli/app_test.go`
- Modify: `mvp-arch.md`
- Modify: `mvp-plan.md`

**Design note (2026-04-03):** `skillc install` should accept comma-separated batch targets in a single `skill-id` argument, treat `prefix-*` as `skill.ID` prefix matching only, require explicit `--collection/-c` before expanding a collection selector, and support `--yes/-y` to skip confirmation. Batch execution must use partial-success semantics so resolve/install failures are reported and skipped without aborting other targets.

**Verification note (2026-04-03):** Focused CLI install tests pass for batch targets with partial-success output, explicit collection mode, prompt-vs-`--yes` behavior, and installapp/searchapp regressions. `go test ./...` was re-run and still fails only at the pre-existing unrelated baseline `internal/cli/app_test.go:336` (`TestSourceAddLocalCommand_WithSyncRebuildsIndexForSearch`).

### Task 29: Add installed skill update flow ✅ Completed

**Files:**
- Create: `internal/app/updateapp/service.go`
- Test: `internal/app/updateapp/service_test.go`
- Modify: `internal/app/installapp/service.go`
- Modify: `internal/app/installapp/service_test.go`
- Modify: `internal/cli/manage_cmd.go`
- Modify: `internal/cli/app_test.go`
- Modify: `mvp-arch.md`
- Modify: `mvp-plan.md`

**Design note (2026-04-06):** `skillc update` now uses a lock-first workflow. It selects installed skills from the grouped lock file when present, expands each grouped record into per-agent update work, falls back to scanning the installed agent directory when the lock is missing or empty, skips pinned or ambiguous entries, syncs each referenced source once, reloads the shared index, and recomputes target install paths at runtime from the lock key + agent + flat install-dir rule `skills/{skillID}` instead of persisting `InstalledPath` in the lock file. For upgrades from older versions, list / uninstall / update remain compatible with legacy source-scoped install directories and migrate them to the flat layout during update.

**Verification note (2026-04-07):** Added regressions in `internal/app/installapp/service_test.go`, `internal/app/listapp/service_test.go`, `internal/app/updateapp/service_test.go`, and `internal/cli/app_test.go` to lock the flat install directory semantics. Coverage now verifies install/list/update/restore all resolve installed paths to `skills/{skillID}`, same-source reinstall keeps working, different sources with the same `SkillID` are rejected before any overwrite occurs, and legacy source-scoped install directories are still recognized by list / uninstall / update and migrated to the flat layout. Focused regression `go test ./internal/app/installapp ./internal/app/listapp ./internal/app/updateapp ./internal/cli -count=1` and full regression `go test ./...` both pass.

**Lockfile redesign follow-up (2026-04-06):** CLI fixtures and docs now reflect the grouped lock layout: top-level keys are `__global__` or absolute project paths, each skill record persists `agents[]`, restore output is emitted per runtime agent/scope pair, and `InstalledPath` is resolved at runtime instead of stored in the lock file.

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run `go test ./...`**
- [ ] **Step 6: Commit**
