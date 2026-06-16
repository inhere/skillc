# Git Source Incremental Sync Design

## Background

Current `skillc source sync` behavior for Git sources always deletes the cached repository directory and re-runs `git clone`. This guarantees a clean cache, but wastes network and disk work on every sync, even when the cache directory is already a valid clone of the same remote.

The user has approved a new default behavior:

- Git source sync should prefer incremental synchronization
- Local cache state is disposable and may be force-aligned to remote state
- Local dirty changes and untracked files in the cache may be removed during sync
- If the cache directory is invalid or unsafe to reuse, sync should fall back to deleting it and re-cloning

## Goal

Change Git source sync from “always delete and clone” to “reuse the existing cache when safe, otherwise self-heal by re-cloning”, while preserving the current app-layer contract:

- `sourceapp.Service.Sync` still orchestrates source status updates and index rebuilds
- `gitx.Client.Sync` still returns the resolved HEAD commit
- local sources remain unchanged
- proxy and TTY progress behavior remain unchanged

## Non-Goals

This design does not include:

- preserving manual edits inside the cache repository
- introducing `git pull` semantics
- changing local source sync behavior
- changing CLI UX or adding new flags
- changing lock/install behavior
- changing source ref semantics
- persisting any Git config to the user’s global or repo config

## Options Considered

### Option A: `fetch + reset --hard + clean -fd` on reusable cache (Recommended)

Behavior:
- if cache directory does not exist, run `git clone`
- if cache directory exists and is reusable, run `git fetch --prune origin`, align to the requested target, then `git reset --hard` and `git clean -fd`
- if cache directory is unusable, delete it and re-clone

Pros:
- matches the approved “force-align cache to remote state” semantics
- much cheaper than cloning every time
- predictable final cache contents
- easy to verify with tests

Cons:
- requires more Git command branches inside `gitx.Client`
- needs explicit fallback rules for damaged caches

### Option B: `fetch + checkout + pull --ff-only`

Pros:
- resembles common developer workflow

Cons:
- weaker cleanup semantics
- harder to reason about for tags, commits, and detached states
- does not naturally enforce a clean cache directory

### Option C: keep full re-clone and only optimize clone behavior

Pros:
- simplest implementation

Cons:
- does not satisfy the incremental sync requirement
- still wastes network and disk work

## Recommendation

Adopt **Option A**.

The repository cache in `skillc` is a generated artifact, not a user workspace. That makes “force remote state into cache” the right model. `fetch + reset --hard + clean -fd` preserves the clean-cache guarantee while avoiding needless full re-clones in the common case.

## Design

### Layer Responsibilities

#### CLI layer

No behavioral change.

`skillc source sync <id>` continues to:
- parse arguments
- call `sourceapp.Service.Sync`
- print user-facing output

CLI does not decide whether sync uses clone or incremental update.

#### App layer: `internal/app/sourceapp/service.go`

No policy expansion beyond current orchestration responsibilities.

`sourceapp.Service.Sync` continues to:
- load config and source list
- find the requested source
- keep local source behavior unchanged
- call `s.git.Sync(src.URL, targetDir, src.Ref, opts)` for Git sources
- persist `Path`, `ResolvedRef`, `LastSyncAt`, `Status`, and `ErrorMessage`
- rebuild the index after a successful sync

The app layer should not know the internal Git recovery path. Whether the sync used clone, incremental fetch/reset, or fallback re-clone remains encapsulated in `gitx`.

#### Infra layer: `internal/infra/gitx/client.go`

`gitx.Client.Sync` becomes the single place that decides how to synchronize the on-disk repository.

It is responsible for:
- deciding whether the cache directory can be reused
- validating remote identity before reuse
- running clone or incremental commands
- injecting proxy env vars only into network Git commands
- forwarding progress output only for clone/fetch commands that should show it
- returning the final `HEAD` commit as `ResolvedRef`

### Sync Flow

#### Case 1: cache directory does not exist

Run clone flow:
1. `git clone [--branch <ref>] <url> <dir>`
2. `git rev-parse HEAD`

#### Case 2: cache directory exists and is reusable

Run incremental sync flow:
1. confirm `<dir>/.git` is usable or otherwise confirm the directory is a Git worktree
2. `git -C <dir> remote get-url origin`
3. verify returned URL matches the source URL
4. `git -C <dir> fetch --prune origin`
5. determine target revision:
   - if `ref != ""`, sync to the requested ref
   - if `ref == ""`, sync to the remote default-branch fetch result
6. `git -C <dir> reset --hard <target>`
7. `git -C <dir> clean -fd`
8. `git -C <dir> rev-parse HEAD`

#### Case 3: cache directory exists but is unsafe to reuse

Fallback flow:
1. remove `<dir>`
2. run clone flow
3. `git rev-parse HEAD`

### Reuse Validation Rules

A cache directory is considered **unsafe to reuse** when any of the following is true:

- the directory is not a Git repository
- `remote get-url origin` fails
- the `origin` URL does not match the source URL
- incremental sync preconditions fail in a way that indicates repository corruption or wrong repository identity

In these cases, `gitx.Client.Sync` should self-heal by deleting the cache directory and cloning again.

### Failure Handling

#### Recoverable failures

These should trigger fallback re-clone:
- invalid or missing `.git` metadata
- wrong `origin` URL
- repository shape indicates the cache is not the expected repo

#### Non-recoverable failures

These should surface as sync errors:
- clone fails
- fetch fails and fallback clone also fails
- reset/clean succeeds poorly enough that final `rev-parse HEAD` cannot be obtained

The user-visible result remains the existing app-layer behavior: source status becomes `error`, `ErrorMessage` is updated, and index rebuild is skipped.

### Ref Semantics

Ref behavior stays the same at the service boundary.

- `ref != ""`: sync the cache to that requested branch, tag, or commit-compatible target
- `ref == ""`: sync the cache to the remote default branch result

This design intentionally preserves the meaning of `Source.Ref`; it only changes how the cache is updated.

### Cleanup Semantics

After a successful incremental sync:
- tracked file changes are discarded via `reset --hard`
- untracked files and directories are removed via `clean -fd`

This preserves the historical “cache directory ends clean” behavior without always deleting the entire repository.

### Compatibility

Unchanged behavior:
- local source sync path
- `ResolvedRef` persistence
- `LastSyncAt` updates
- source `ready/error` state transitions
- rebuild-index-on-success behavior
- proxy injection policy already implemented for Git network commands
- TTY-only progress behavior already implemented for Git sync options

Changed behavior:
- Git source sync no longer deletes the cache directory before every sync
- stale files are removed by Git cleanup during the incremental path, not by removing the entire directory
- corrupted or mismatched caches are repaired by fallback clone instead of failing immediately when safe to self-heal

## Testing Strategy

### `internal/app/sourceapp/service_test.go`

Keep app-layer assertions focused on orchestration, not Git command details.

Required coverage:
- successful Git sync still marks the source `ready`
- `ResolvedRef` and `LastSyncAt` still update after success
- existing `SyncOptions` forwarding expectations remain valid
- local source sync behavior remains unchanged

### `internal/infra/gitx/client_test.go`

Add focused tests for sync path selection and cleanup behavior.

Required coverage:
1. cache missing -> clone path
2. cache present and valid -> incremental sync path
3. cache present but not a Git repo -> fallback clone path
4. cache present with mismatched `origin` -> fallback clone path
5. incremental sync removes stale untracked files
6. final resolved ref comes from the synchronized HEAD

Tests should assert behavior at the `gitx.Client` boundary rather than duplicating app-layer tests.

## Docs Impact

Implementation must update:
- `mvp-arch.md` — reflect “prefer incremental sync, fallback to re-clone” behavior for Git sources
- `mvp-plan.md` — record this optimization task and mark phase state accurately

This keeps architecture and task tracking aligned with the code change, per project instructions.

## Success Criteria

The design is successful when:
- repeated syncs against the same healthy Git source no longer re-clone every time
- cache contents after sync are still clean and deterministic
- broken or mismatched cache directories self-heal automatically by re-cloning
- app-layer sync state and indexing behavior remain unchanged from the user perspective except for improved efficiency
