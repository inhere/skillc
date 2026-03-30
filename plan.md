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

### Task 1: Bootstrap the Go CLI skeleton

**Files:**
- Create: `go.mod`
- Create: `cmd/skillc/main.go`
- Create: `internal/cli/app.go`
- Create: `internal/cli/output.go`
- Create: `internal/cli/errors.go`

- [ ] **Step 1: Write the failing bootstrap test**

Create `cmd/skillc/main_test.go` with a smoke test that builds the command app and verifies the root command name is `skillc`.

```go
func TestNewApp_HasRootCommandName(t *testing.T) {
    app := cli.NewApp()
    if app == nil {
        t.Fatal("expected app")
    }
    if got := app.Name(); got != "skillc" {
        t.Fatalf("got %q want %q", got, "skillc")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/skillc -run TestNewApp_HasRootCommandName -v`
Expected: FAIL because `cli.NewApp` and module files do not exist yet.

- [ ] **Step 3: Write minimal implementation**

Create the module and a minimal CLI app that exposes:
- root command name `skillc`
- `version` and `help` working through the root app
- shared output/error helpers for later commands

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/skillc -run TestNewApp_HasRootCommandName -v`
Expected: PASS

- [ ] **Step 5: Run package tests**

Run: `go test ./cmd/skillc ./internal/cli/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add go.mod cmd/skillc/main.go cmd/skillc/main_test.go internal/cli/app.go internal/cli/output.go internal/cli/errors.go
git commit -m "feat(cli): bootstrap skillc command app"
```

### Task 2: Implement config defaults and path expansion

**Files:**
- Create: `internal/domain/config/model.go`
- Create: `internal/domain/config/defaults.go`
- Create: `internal/infra/fsx/paths.go`
- Test: `internal/domain/config/defaults_test.go`
- Test: `internal/infra/fsx/paths_test.go`

- [ ] **Step 1: Write the failing tests**

Cover:
- default config path resolution
- default agent directories for `claude-code`, `opencode`, `codex`
- `~` expansion
- env var expansion
- relative path expansion against cwd

Example:

```go
func TestDefaultAgentDirname_Codex(t *testing.T) {
    cfg := config.DefaultConfig()
    got := cfg.AgentTools["codex"].Dirname
    if got != ".codex" {
        t.Fatalf("got %q want .codex", got)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/config ./internal/infra/fsx -v`
Expected: FAIL because implementations do not exist yet.

- [ ] **Step 3: Write minimal implementation**

Implement:
- config structs matching `prd.md`
- default path generation from `arch.md` decisions
- expansion helpers that normalize paths safely on Windows/macOS/Linux

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/config ./internal/infra/fsx -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/config/model.go internal/domain/config/defaults.go internal/domain/config/defaults_test.go internal/infra/fsx/paths.go internal/infra/fsx/paths_test.go
git commit -m "feat(config): add default config and path expansion"
```

### Task 3: Implement YAML config storage and config app service

**Files:**
- Create: `internal/infra/configstore/yaml_store.go`
- Create: `internal/app/configapp/service.go`
- Test: `internal/infra/configstore/yaml_store_test.go`
- Test: `internal/app/configapp/service_test.go`
- Modify: `internal/cli/app.go`

- [ ] **Step 1: Write the failing tests**

Cover:
- missing config file auto-initializes minimal config
- `Show` returns current config
- `Get("lock_file")` returns configured value
- `Set("proxy_url", "http://localhost:7890")` persists to YAML

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/infra/configstore ./internal/app/configapp -v`
Expected: FAIL because store/service are not implemented yet.

- [ ] **Step 3: Write minimal implementation**

Implement:
- YAML-backed load/save with atomic writes
- config service methods: `Init`, `Show`, `Get`, `Set`
- wire `config show|get|set|init` subcommands into CLI

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/infra/configstore ./internal/app/configapp -v`
Expected: PASS

- [ ] **Step 5: Run CLI smoke tests**

Run: `go test ./cmd/skillc ./internal/cli/... ./internal/app/configapp/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/infra/configstore/yaml_store.go internal/infra/configstore/yaml_store_test.go internal/app/configapp/service.go internal/app/configapp/service_test.go internal/cli/app.go
git commit -m "feat(config): add config init and read-write commands"
```

### Task 4: Add shared atomic write, file lock, and checksum infra

**Files:**
- Create: `internal/infra/fsx/atomic.go`
- Create: `internal/infra/filelock/lock.go`
- Create: `internal/infra/hashx/sha256.go`
- Test: `internal/infra/fsx/atomic_test.go`
- Test: `internal/infra/filelock/lock_test.go`
- Test: `internal/infra/hashx/sha256_test.go`

- [ ] **Step 1: Write the failing tests**

Cover:
- atomic file write replaces target contents in one step
- second lock acquisition fails or waits according to chosen API
- checksum for a directory/file is deterministic

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/infra/fsx ./internal/infra/filelock ./internal/infra/hashx -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Implement shared helpers that later config and lock store reuse. Keep the API small and focused.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/infra/fsx ./internal/infra/filelock ./internal/infra/hashx -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/infra/fsx/atomic.go internal/infra/fsx/atomic_test.go internal/infra/filelock/lock.go internal/infra/filelock/lock_test.go internal/infra/hashx/sha256.go internal/infra/hashx/sha256_test.go
git commit -m "feat(infra): add atomic write, file lock, and checksum helpers"
```

### Task 5: Add source domain model and local source management

**Files:**
- Create: `internal/domain/source/model.go`
- Create: `internal/infra/sourcestore/config_sources.go`
- Create: `internal/app/sourceapp/service.go`
- Test: `internal/domain/source/model_test.go`
- Test: `internal/app/sourceapp/service_test.go`
- Modify: `internal/cli/app.go`

- [ ] **Step 1: Write the failing tests**

Cover:
- add local source with `id` and `name`
- duplicate local source rejected
- list returns sources with ids
- remove deletes by source id

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/source ./internal/app/sourceapp -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Implement:
- `Source` model and source type enum
- config-backed local source CRUD
- CLI commands: `source add local`, `source list`, `source remove`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/source ./internal/app/sourceapp -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/source/model.go internal/domain/source/model_test.go internal/infra/sourcestore/config_sources.go internal/app/sourceapp/service.go internal/app/sourceapp/service_test.go internal/cli/app.go
git commit -m "feat(source): add local source management"
```

### Task 6: Add Git client and Git source sync flow

**Files:**
- Create: `internal/infra/gitx/client.go`
- Test: `internal/infra/gitx/client_test.go`
- Modify: `internal/app/sourceapp/service.go`
- Modify: `internal/cli/app.go`
- Test: `internal/app/sourceapp/service_test.go`

- [ ] **Step 1: Write the failing tests**

Cover:
- add git source with `ref`
- sync clones repository into repo cache
- missing Git returns actionable error
- failed source sync marks source status error without corrupting others

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/infra/gitx ./internal/app/sourceapp -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Implement a thin Git CLI wrapper with methods for clone/fetch/current revision, then wire:
- `source add git`
- `source sync`
- `source status`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/infra/gitx ./internal/app/sourceapp -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/infra/gitx/client.go internal/infra/gitx/client_test.go internal/app/sourceapp/service.go internal/app/sourceapp/service_test.go internal/cli/app.go
git commit -m "feat(source): add git source sync support"
```

### Task 7: Parse `SKILL.md` metadata and scan source indexes

**Files:**
- Create: `internal/domain/skill/model.go`
- Create: `internal/domain/skill/parser.go`
- Create: `internal/infra/repoindex/scanner.go`
- Create: `internal/infra/repoindex/store.go`
- Test: `internal/domain/skill/parser_test.go`
- Test: `internal/infra/repoindex/scanner_test.go`

- [ ] **Step 1: Write the failing tests**

Cover:
- valid `SKILL.md` front matter parses into normalized `Skill`
- missing `version` falls back correctly for local/git sources
- directories without `SKILL.md` are skipped or marked incomplete per PRD
- one repo with multiple skill subdirectories indexes multiple results

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/skill ./internal/infra/repoindex -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Implement the parser and scanner. Persist a minimal cached index so search does not rescan every time.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/skill ./internal/infra/repoindex -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/skill/model.go internal/domain/skill/parser.go internal/domain/skill/parser_test.go internal/infra/repoindex/scanner.go internal/infra/repoindex/store.go internal/infra/repoindex/scanner_test.go
git commit -m "feat(index): parse skill metadata and build source indexes"
```

### Task 8: Implement search and show services

**Files:**
- Create: `internal/infra/repoindex/search.go`
- Create: `internal/app/searchapp/service.go`
- Test: `internal/infra/repoindex/search_test.go`
- Test: `internal/app/searchapp/service_test.go`
- Modify: `internal/cli/app.go`

- [ ] **Step 1: Write the failing tests**

Cover:
- search by name and description
- filter by agent and source type
- support installed-state projection placeholder when lock exists later
- `show <skill-id>` returns detail for exactly one skill

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/infra/repoindex ./internal/app/searchapp -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Implement:
- search filters and fuzzy/substring matching
- CLI commands `search` and `show`
- JSON output mode for search results

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/infra/repoindex ./internal/app/searchapp -v`
Expected: PASS

- [ ] **Step 5: Run M1 regression tests**

Run: `go test ./cmd/skillc ./internal/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/infra/repoindex/search.go internal/infra/repoindex/search_test.go internal/app/searchapp/service.go internal/app/searchapp/service_test.go internal/cli/app.go
git commit -m "feat(search): add indexed skill search and show"
```

### Task 9: Implement basic doctor checks for M1

**Files:**
- Create: `internal/app/doctorapp/service.go`
- Test: `internal/app/doctorapp/service_test.go`
- Modify: `internal/cli/app.go`

- [ ] **Step 1: Write the failing tests**

Cover:
- config file parse errors reported
- Git availability check reports missing executable clearly
- cache/config/lock parent directories writeability checked

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/doctorapp -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Implement M1 doctor checks only. Do not add missing/orphan/install-integrity checks yet.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/doctorapp -v`
Expected: PASS

- [ ] **Step 5: Run M1 full test suite**

Run: `go test ./cmd/skillc ./internal/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/app/doctorapp/service.go internal/app/doctorapp/service_test.go internal/cli/app.go
git commit -m "feat(doctor): add baseline environment diagnostics"
```

### Task 10: Implement agent resolver for global/project installs

**Files:**
- Create: `internal/domain/agent/model.go`
- Create: `internal/domain/agent/resolver.go`
- Test: `internal/domain/agent/resolver_test.go`

- [ ] **Step 1: Write the failing tests**

Cover:
- default global dirs `.claude`, `.opencode`, `.codex`
- project scope resolves under cwd
- explicit config overrides are honored
- unknown agent returns clear error

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/agent -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Implement the single path-resolution entrypoint used by all install/list logic.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/agent -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/agent/model.go internal/domain/agent/resolver.go internal/domain/agent/resolver_test.go
git commit -m "feat(agent): add install path resolver"
```

### Task 11: Implement lock model and JSON lock store

**Files:**
- Create: `internal/domain/lock/model.go`
- Create: `internal/infra/lockstore/json_store.go`
- Test: `internal/infra/lockstore/json_store_test.go`

- [ ] **Step 1: Write the failing tests**

Cover:
- create empty lock file
- append/update/delete records atomically
- preserve `source_type`, `checksum`, `pinned`
- concurrent access uses file lock safely

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/infra/lockstore -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Implement a JSON lock store that reuses atomic write and file lock helpers.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/infra/lockstore -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/lock/model.go internal/infra/lockstore/json_store.go internal/infra/lockstore/json_store_test.go
git commit -m "feat(lock): add atomic install lock store"
```

### Task 12: Implement install planning and conflict detection

**Files:**
- Create: `internal/domain/install/model.go`
- Create: `internal/domain/install/planner.go`
- Create: `internal/domain/install/conflict.go`
- Test: `internal/domain/install/planner_test.go`
- Test: `internal/domain/install/conflict_test.go`

- [ ] **Step 1: Write the failing tests**

Cover:
- `install <skill-id>` defaults to all configured agents and global scope
- explicit `--agent` narrows targets
- existing managed install is identified
- existing unmanaged path conflict is identified
- `--yes` maps to overwrite default behavior

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/install -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Implement:
- install plan generation from skill + config + CLI options
- conflict classification only; no file copying yet

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/install -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/install/model.go internal/domain/install/planner.go internal/domain/install/conflict.go internal/domain/install/planner_test.go internal/domain/install/conflict_test.go
git commit -m "feat(install): add planning and conflict detection"
```

### Task 13: Implement file installer and uninstall primitives

**Files:**
- Create: `internal/infra/agentfs/installer.go`
- Test: `internal/infra/agentfs/installer_test.go`

- [ ] **Step 1: Write the failing tests**

Cover:
- copy directory install into target root
- copy single-file install entry
- uninstall removes managed target
- dry-run returns plan without filesystem mutation

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/infra/agentfs -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Implement copy/remove primitives only. Keep prompting and lock writes outside this package.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/infra/agentfs -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/infra/agentfs/installer.go internal/infra/agentfs/installer_test.go
git commit -m "feat(agentfs): add install and uninstall filesystem operations"
```

### Task 14: Implement install, uninstall, and restore services

**Files:**
- Create: `internal/app/installapp/service.go`
- Test: `internal/app/installapp/service_test.go`
- Modify: `internal/cli/app.go`

- [ ] **Step 1: Write the failing tests**

Cover:
- install from indexed local source
- install from indexed git source after sync
- uninstall removes file and lock record
- `install` with no skill id restores from lock file
- install conflict prompt branches: overwrite / skip / cancel

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/installapp -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Implement orchestration that ties together:
- source/index lookup
- agent resolver
- planner/conflict detector
- filesystem installer
- checksum
- lock store
- prompt abstraction for interactive decisions

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/installapp -v`
Expected: PASS

- [ ] **Step 5: Run M2 regression tests**

Run: `go test ./cmd/skillc ./internal/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/app/installapp/service.go internal/app/installapp/service_test.go internal/cli/app.go
git commit -m "feat(install): add install uninstall and restore workflows"
```

### Task 15: Implement installed list and status projection

**Files:**
- Create: `internal/app/listapp/service.go`
- Test: `internal/app/listapp/service_test.go`
- Modify: `internal/cli/app.go`

- [ ] **Step 1: Write the failing tests**

Cover:
- list installed by agent and scope
- mark `missing` when target path is gone
- mark `orphan` when source no longer exists
- JSON output includes version, source, path, checksum

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/listapp -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Implement installed list projection from lock records plus live filesystem/source checks.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/listapp -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/listapp/service.go internal/app/listapp/service_test.go internal/cli/app.go
git commit -m "feat(list): add installed skill status listing"
```

### Task 16: Add end-to-end smoke coverage for source/search/install/restore

**Files:**
- Create: `tests/e2e/source_search_test.go`
- Create: `tests/e2e/install_restore_test.go`

- [ ] **Step 1: Write the failing end-to-end tests**

Cover two flows using temp dirs and fixture repos:
1. local source -> index -> search -> show
2. install -> uninstall -> restore from lock

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./tests/e2e -v`
Expected: FAIL until all dependent behavior is wired.

- [ ] **Step 3: Write minimal fixture/setup code**

Add reusable helpers inside the test files only. Do not create shared production abstractions just for tests.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./tests/e2e -v`
Expected: PASS

- [ ] **Step 5: Run full test suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add tests/e2e/source_search_test.go tests/e2e/install_restore_test.go
git commit -m "test(e2e): cover source search install and restore flows"
```

### Task 17: Extend to M3 registry and update work

**Files:**
- Create: `internal/app/registryapp/service.go`
- Create: `internal/infra/registryx/client.go`
- Create: `internal/app/updateapp/service.go`
- Create: `internal/app/cacheapp/service.go`
- Create: `internal/app/validateapp/service.go`
- Test: `internal/app/registryapp/service_test.go`
- Test: `internal/infra/registryx/client_test.go`
- Test: `internal/app/updateapp/service_test.go`
- Test: `internal/app/cacheapp/service_test.go`
- Test: `internal/app/validateapp/service_test.go`
- Modify: `internal/cli/app.go`

- [ ] **Step 1: Write the failing tests**

Cover:
- registry add/list/remove/refresh
- JSON-index-first registry lookup with HTTP fallback
- outdated detection using source-specific version rules
- update skips `pinned` records
- cache clean/rebuild and validate command behavior

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/registryapp ./internal/infra/registryx ./internal/app/updateapp ./internal/app/cacheapp ./internal/app/validateapp -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

Implement M3 incrementally, keeping registry adapter and update logic isolated from M1/M2 core flows.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/registryapp ./internal/infra/registryx ./internal/app/updateapp ./internal/app/cacheapp ./internal/app/validateapp -v`
Expected: PASS

- [ ] **Step 5: Run full regression suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/app/registryapp/service.go internal/infra/registryx/client.go internal/app/updateapp/service.go internal/app/cacheapp/service.go internal/app/validateapp/service.go internal/cli/app.go
git commit -m "feat(registry): add registry search and skill update workflows"
```

---

## Milestone checkpoints

### M1 checkpoint
Required passing commands:
- `go test ./cmd/skillc ./internal/...`

Required manual smoke checks:
- `skillc init`
- `skillc source add local <path>`
- `skillc source add git <url> --ref main`
- `skillc source sync`
- `skillc search <keyword>`
- `skillc show <skill-id>`
- `skillc doctor`

### M2 checkpoint
Required passing commands:
- `go test ./...`

Required manual smoke checks:
- `skillc install <skill-id>`
- `skillc uninstall <skill-id>`
- `skillc list`
- `skillc install`  # restore from lock

### M3 checkpoint
Required passing commands:
- `go test ./...`

Required manual smoke checks:
- `skillc registry add <url>`
- `skillc search <keyword>` against registry-backed results
- `skillc outdated`
- `skillc update <skill-id>`
- `skillc cache rebuild`
- `skillc validate <path>`

---

## Notes for execution

- Keep authoring changes vertical: command wiring + app service + domain/infra slice + tests in one task.
- Do not start registry/update before M1/M2 core flows are green.
- Reuse temp dirs in tests; never read/write real user config or agent dirs during tests.
- Keep Git integration as a thin wrapper around the Git CLI to avoid unnecessary library coupling in MVP.
- Prefer explicit interfaces at app boundaries only; do not over-abstract every package.
- Before claiming completion at each milestone, run the listed regression commands and manual smoke checks.
