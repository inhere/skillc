# Skillc MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Skillc CLI MVP that can manage local/Git sources, index skills from `SKILL.md`, search them, install/uninstall them for multiple agents, and restore installations from a lock file.

**Architecture:** Use the approved layered design from `arch.md`: `cmd/` for CLI wiring, `internal/app/` for use-case orchestration, `internal/domain/` for stable models and rules, and `internal/infra/` for filesystem, Git, config, lock, hashing, and indexing implementations. Deliver in vertical slices so each milestone leaves the repo in a runnable, testable state.

**Tech Stack:** Go, standard library, YAML config parsing, JSON lock file, Git CLI integration, filesystem-based cache, Go testing package

---

## File Structure

### Documents to consult
- `prd.md` — product requirements and acceptance criteria
- `prd-review.md` — clarified decisions and resolved blockers
- `arch.md` — approved architecture and module boundaries

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
- Modify: `arch.md`
- Modify: `plan.md`

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
- Modify: `arch.md`
- Modify: `plan.md`

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
- Modify: `arch.md`
- Modify: `plan.md`

### Task 26: Add proxy support for git source sync

**Files:**
- Modify: `internal/app/sourceapp/service.go`
- Modify: `internal/app/sourceapp/service_test.go`
- Modify: `internal/infra/gitx/client.go`
- Modify: `internal/infra/gitx/client_test.go`
- Modify: `arch.md`
- Modify: `plan.md`

**Design note (2026-04-01):** When `source` type is `git` and config `proxy_url` is set, `source sync` must apply that proxy only to skillc-started network Git commands such as `git clone`. It must not write any Git config, and local Git commands such as `rev-parse` must continue to run without proxy injection.

**Verification note (2026-04-01):** Git source sync now forwards config `proxy_url` into the Git client, injects proxy env vars only for `git clone`, keeps local `rev-parse` unproxied, and passes `go test ./...`.

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run `go test ./...`**
- [ ] **Step 6: Commit**
