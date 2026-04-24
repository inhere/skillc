# Skill Update Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a first-version `skillc update` command that updates already-installed skills by syncing their sources and reinstalling them in place, using lock records first and installed-directory scanning as fallback.

**Architecture:** Keep CLI thin by adding `update` wiring in `internal/cli/*` and putting candidate collection, source syncing, reinstall orchestration, and result aggregation in a new `internal/app/updateapp` service. Reuse existing lock, search/index, source sync, and install behaviors where possible; only add a minimal install helper if the current install service cannot safely reinstall to an explicit existing path.

**Tech Stack:** Go, gcli, existing `configapp` / `sourceapp` / `installapp` / `lockstore` / `repoindex` packages, Go testing package

---

## File Structure

### Documents to consult
- `docs/superpowers/specs/2026-04-03-skill-update-design.md` — approved design for v1 update behavior
- `arch.md` — project architecture; must be updated when implementation changes documented capability boundaries
- `plan.md` — project task ledger; must mark the new update work accurately per repo instructions

### Planned files

**CLI wiring**
- Modify: `internal/cli/app.go` — register `buildUpdateCommand()`
- Modify: `internal/cli/manage_cmd.go` — add `update` command and output formatting
- Modify: `internal/cli/app_test.go` — CLI registration and end-to-end-ish command behavior tests

**Update orchestration**
- Create: `internal/app/updateapp/service.go` — collect candidates, sync sources, reinstall, aggregate results
- Create: `internal/app/updateapp/service_test.go` — focused orchestration tests for lock-first and scan fallback behavior

**Install reuse**
- Modify: `internal/app/installapp/service.go` — add a minimal helper for reinstalling a `skill.Skill` into an explicit path while keeping lock semantics intact
- Modify: `internal/app/installapp/service_test.go` — prove explicit-path reinstall updates the lock correctly

**Docs / tracking**
- Modify: `arch.md` — document `skillc update` as implemented with lock-first + scan-fallback behavior for v1
- Modify: `plan.md` — add/update the task note for `skillc update` and mark phase progress accurately

---

### Task 1: Register the `update` CLI command

**Files:**
- Modify: `internal/cli/app.go:16-30`
- Modify: `internal/cli/manage_cmd.go:73-298`
- Test: `internal/cli/app_test.go`

- [x] **Step 1: Write the failing CLI registration test**

Add this test near the existing command registration tests in `internal/cli/app_test.go`:

```go
func TestNewApp_RegistersUpdateCommand(t *testing.T) {
	app := newTestApp()

	update := findCommandByName(app, "update")
	assert.NotNil(t, update)
	assert.Eq(t, "Update installed skills", update.Desc)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
go test ./internal/cli -run TestNewApp_RegistersUpdateCommand
```

Expected: FAIL because `update` is not registered yet.

- [ ] **Step 3: Write the minimal CLI wiring**

In `internal/cli/app.go`, register the new command next to the other manage commands:

```go
func NewApp(version, gitHash, buildTime string) *gcli.App {
	app := gcli.NewApp()
	app.Name = "skillc"
	app.Desc = "Skill manager for multi-agent ecosystems"
	app.Version = fmt.Sprintf("%s (Git Hash: %s, Build Time: %s)", version, gitHash, buildTime)
	app.Add(buildConfigCommand())
	app.Add(buildSourceCommand())
	app.Add(buildCollectionCommand())
	app.Add(buildSearchCommand())
	app.Add(buildShowCommand())
	app.Add(buildInstallCommand())
	app.Add(buildUpdateCommand())
	app.Add(buildUninstallCommand())
	app.Add(buildListCommand())
	app.Add(buildDoctorCommand())
	return app
}
```

In `internal/cli/manage_cmd.go`, add a placeholder command that only parses the shared manage flags and returns nil for now:

```go
func buildUpdateCommand() *gcli.Command {
	var opts ManageOptions
	return &gcli.Command{
		Name: "update",
		Desc: "Update installed skills",
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
			c.BoolOpt(&opts.Yes, "yes", "y", false, "skip confirmation prompt")
		},
		Func: func(c *gcli.Command, _ []string) error {
			return nil
		},
	}
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run:

```bash
go test ./internal/cli -run TestNewApp_RegistersUpdateCommand
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/app.go internal/cli/manage_cmd.go internal/cli/app_test.go
git commit -m "feat(cli): register update command"
```

### Task 2: Add explicit-path reinstall support in `installapp`

**Files:**
- Modify: `internal/app/installapp/service.go`
- Modify: `internal/app/installapp/service_test.go`

- [ ] **Step 1: Write the failing reinstall test**

Add this test in `internal/app/installapp/service_test.go` after the existing install tests:

```go
func TestService_ReinstallAtPathUpdatesExistingLockRecord(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := filepath.Join(baseDir, "source", "hello-skill")
	commandsDir := filepath.Join(sourceDir, "commands")
	targetPath := filepath.Join(baseDir, ".claude", "skills", "hello-skill")
	assert.NoErr(t, os.MkdirAll(commandsDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(commandsDir, "hello.txt"), []byte("updated"), 0o644))
	assert.NoErr(t, NewService(lockFile).store.Save(lockFile, []lockpkg.Record{{
		SkillID:             "hello-skill",
		QualifiedName:       "marketplaces/hello-skill",
		SourceQualifiedName: "repo-a/marketplaces/hello-skill",
		Agent:               "claude-code",
		Scope:               "project",
		SourceID:            "local-demo",
		SourceType:          "local",
		InstallEntry:        "commands",
		InstalledPath:       targetPath,
		InstalledAt:         time.Date(2026, 4, 1, 1, 0, 0, 0, time.UTC),
		UpdatedAt:           time.Date(2026, 4, 1, 1, 0, 0, 0, time.UTC),
	}}))

	service := NewService(lockFile)
	service.now = func() time.Time { return time.Date(2026, 4, 4, 9, 0, 0, 0, time.UTC) }

	record, err := service.ReinstallAtPath(skill.Skill{
		ID:                  "hello-skill",
		QualifiedName:       "marketplaces/hello-skill",
		SourceQualifiedName: "repo-a/marketplaces/hello-skill",
		Version:             "1.1.0",
		SourceID:            "local-demo",
		SourceType:          sourcepkg.TypeLocal,
		InstallEntry:        "commands",
		Path:                sourceDir,
	}, "claude-code", agent.ScopeProject, targetPath)
	assert.NoErr(t, err)
	assert.Eq(t, targetPath, record.InstalledPath)
	assert.Eq(t, "1.1.0", record.Version)
	assert.Eq(t, time.Date(2026, 4, 1, 1, 0, 0, 0, time.UTC), record.InstalledAt)
	assert.Eq(t, time.Date(2026, 4, 4, 9, 0, 0, 0, time.UTC), record.UpdatedAt)

	data, err := os.ReadFile(filepath.Join(targetPath, "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "updated", string(data))

	locks, err := service.store.Load(lockFile)
	assert.NoErr(t, err)
	assert.Len(t, locks, 1)
	assert.Eq(t, "1.1.0", locks[0].Version)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
go test ./internal/app/installapp -run TestService_ReinstallAtPathUpdatesExistingLockRecord
```

Expected: FAIL because `ReinstallAtPath` does not exist.

- [ ] **Step 3: Write the minimal explicit-path reinstall helper**

Add this method in `internal/app/installapp/service.go` next to `Install`:

```go
func (s *Service) ReinstallAtPath(item skill.Skill, agentName string, scope agent.Scope, targetPath string) (lockpkg.Record, error) {
	records, err := s.loadRecords()
	if err != nil {
		return lockpkg.Record{}, err
	}
	if err := s.installer.Install(filepath.Join(item.Path, item.InstallEntry), targetPath); err != nil {
		return lockpkg.Record{}, err
	}

	now := s.now()
	record := lockpkg.Record{
		SkillID:             item.ID,
		QualifiedName:       item.QualifiedName,
		SourceQualifiedName: item.SourceQualifiedName,
		Agent:               agentName,
		Scope:               string(scope),
		Version:             item.Version,
		SourceID:            item.SourceID,
		SourceType:          string(item.SourceType),
		InstallEntry:        item.InstallEntry,
		InstalledPath:       targetPath,
		InstalledAt:         now,
		UpdatedAt:           now,
	}

	for _, current := range records {
		if sameInstallIdentity(current, record) {
			record.InstalledAt = current.InstalledAt
			break
		}
	}

	records = upsertRecord(records, record)
	if err := s.store.Save(s.lockFile, records); err != nil {
		return lockpkg.Record{}, err
	}
	return record, nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run:

```bash
go test ./internal/app/installapp -run TestService_ReinstallAtPathUpdatesExistingLockRecord
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/installapp/service.go internal/app/installapp/service_test.go
git commit -m "feat(install): support reinstalling to explicit paths"
```

### Task 3: Implement lock-first update orchestration

**Files:**
- Create: `internal/app/updateapp/service.go`
- Create: `internal/app/updateapp/service_test.go`

- [ ] **Step 1: Write the failing lock-driven update test**

Create `internal/app/updateapp/service_test.go` with this first test:

```go
package updateapp

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/app/installapp"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

func TestService_RunUpdatesInstalledSkillsFromLock(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	sourceRoot := filepath.Join(baseDir, "source", "hello-skill")
	commandsDir := filepath.Join(sourceRoot, "commands")
	installedPath := filepath.Join(baseDir, ".claude", "skills", "hello-skill")
	assert.NoErr(t, os.MkdirAll(commandsDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(commandsDir, "hello.txt"), []byte("updated-from-source"), 0o644))
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(installedPath, "hello.txt"), []byte("old"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.RepoCacheDir = filepath.Join(baseDir, "repos")
	config.Sources = []sourcepkg.Source{{ID: "local-demo", Type: sourcepkg.TypeLocal, Path: sourceRoot, Status: "ready"}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, []lockpkg.Record{{
		SkillID:             "hello-skill",
		QualifiedName:       "marketplaces/hello-skill",
		SourceQualifiedName: "repo-a/marketplaces/hello-skill",
		Agent:               "claude-code",
		Scope:               "project",
		Version:             "1.0.0",
		SourceID:            "local-demo",
		SourceType:          "local",
		InstallEntry:        "commands",
		InstalledPath:       installedPath,
		InstalledAt:         time.Date(2026, 4, 1, 1, 0, 0, 0, time.UTC),
		UpdatedAt:           time.Date(2026, 4, 1, 1, 0, 0, 0, time.UTC),
	}}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{{
		ID:                  "hello-skill",
		QualifiedName:       "marketplaces/hello-skill",
		SourceQualifiedName: "repo-a/marketplaces/hello-skill",
		Version:             "1.1.0",
		SourceID:            "local-demo",
		SourceType:          sourcepkg.TypeLocal,
		InstallEntry:        "commands",
		Path:                sourceRoot,
	}}))

	service := NewService(configFile, baseDir)
	service.now = func() time.Time { return time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC) }

	result, err := service.Run(Req{Agent: "claude-code", Scope: "project", WorkDir: baseDir})
	assert.NoErr(t, err)
	assert.Len(t, result.Updated, 1)
	assert.Len(t, result.Skipped, 0)
	assert.Len(t, result.Failed, 0)

	data, err := os.ReadFile(filepath.Join(installedPath, "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "updated-from-source", string(data))
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
go test ./internal/app/updateapp -run TestService_RunUpdatesInstalledSkillsFromLock
```

Expected: FAIL because the package and service do not exist yet.

- [ ] **Step 3: Write the minimal lock-driven update service**

Create `internal/app/updateapp/service.go` with this minimal structure:

```go
package updateapp

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/searchapp"
	"github.com/inhere/skillc/internal/app/sourceapp"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/agent"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
)

type Req struct {
	Agent   string
	Scope   string
	WorkDir string
}

type SkippedItem struct {
	SkillID string
	Reason  string
}

type FailedItem struct {
	SkillID string
	Reason  string
}

type Result struct {
	Updated []lockpkg.Record
	Skipped []SkippedItem
	Failed  []FailedItem
}

type Service struct {
	configFile string
	baseDir    string
	config     *configstore.YAMLStore
	locks      *lockstore.Store
	sourceSvc  *sourceapp.Service
	searchSvc  *searchapp.Service
	installer  *installapp.Service
	now        func() time.Time
}

func NewService(configFile, baseDir string) *Service {
	return &Service{
		configFile: configFile,
		baseDir:    baseDir,
		config:     configstore.NewYAMLStore(),
		locks:      lockstore.NewStore(),
		sourceSvc:  sourceapp.NewService(configFile, baseDir),
		searchSvc:  searchapp.NewService("") ,
		installer:  nil,
		now:        time.Now,
	}
}

func (s *Service) Run(req Req) (Result, error) {
	data, err := s.config.Load(s.configFile, s.baseDir)
	if err != nil {
		return Result{}, err
	}
	scope, err := parseScope(req.Scope)
	if err != nil {
		return Result{}, err
	}
	installer := installapp.NewService(data.LockFile)
	items, skipped, err := s.collectFromLock(data, req.Agent, string(scope))
	if err != nil {
		return Result{}, err
	}
	if len(items) == 0 {
		return Result{Skipped: skipped}, nil
	}
	failedBySource := map[string]error{}
	for _, sourceID := range uniqueSourceIDs(items) {
		if err := s.sourceSvc.Sync(sourceID); err != nil {
			failedBySource[sourceID] = err
		}
	}
	index := searchapp.NewService(data.IndexFile)
	result := Result{Skipped: skipped}
	for _, item := range items {
		if err := failedBySource[item.SourceID]; err != nil {
			result.Failed = append(result.Failed, FailedItem{SkillID: item.SkillID, Reason: fmt.Sprintf("source sync failed: %v", err)})
			continue
		}
		resolved, err := index.Resolve(item.SkillID)
		if err != nil {
			result.Failed = append(result.Failed, FailedItem{SkillID: item.SkillID, Reason: err.Error()})
			continue
		}
		record, err := installer.ReinstallAtPath(resolved[0], item.Agent, agent.Scope(item.Scope), item.InstalledPath)
		if err != nil {
			result.Failed = append(result.Failed, FailedItem{SkillID: item.SkillID, Reason: err.Error()})
			continue
		}
		result.Updated = append(result.Updated, record)
	}
	return result, nil
}
```

Also add the lock-candidate collection helpers in the same file:

```go
type candidate struct {
	SkillID       string
	Agent         string
	Scope         string
	SourceID      string
	InstallEntry  string
	InstalledPath string
}

func (s *Service) collectFromLock(data cfg.Config, agentName, scope string) ([]candidate, []SkippedItem, error) {
	records, err := s.locks.Load(data.LockFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	items := make([]candidate, 0, len(records))
	for _, record := range records {
		if agentName != "" && record.Agent != agentName {
			continue
		}
		if scope != "" && record.Scope != scope {
			continue
		}
		items = append(items, candidate{
			SkillID:       record.SkillID,
			Agent:         record.Agent,
			Scope:         record.Scope,
			SourceID:      record.SourceID,
			InstallEntry:  record.InstallEntry,
			InstalledPath: record.InstalledPath,
		})
	}
	return items, nil, nil
}

func uniqueSourceIDs(items []candidate) []string {
	seen := map[string]struct{}{}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item.SourceID]; ok {
			continue
		}
		seen[item.SourceID] = struct{}{}
		ids = append(ids, item.SourceID)
	}
	sort.Strings(ids)
	return ids
}
```

Finally, add this scope helper inside `updateapp/service.go` so the package remains self-contained:

```go
func parseScope(value string) (agent.Scope, error) {
	scope := agent.Scope(value)
	switch scope {
	case agent.ScopeUser, agent.ScopeProject:
		return scope, nil
	default:
		return "", fmt.Errorf("unsupported scope: %s", value)
	}
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run:

```bash
go test ./internal/app/updateapp -run TestService_RunUpdatesInstalledSkillsFromLock
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/updateapp/service.go internal/app/updateapp/service_test.go
git commit -m "feat(update): add lock-based update orchestration"
```

### Task 4: Add scan fallback and skip semantics

**Files:**
- Modify: `internal/app/updateapp/service.go`
- Modify: `internal/app/updateapp/service_test.go`

- [ ] **Step 1: Write the failing scan-fallback tests**

Add these tests to `internal/app/updateapp/service_test.go`:

```go
func TestService_RunFallsBackToInstalledDirScanWhenLockMissing(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	sourceRoot := filepath.Join(baseDir, "source", "hello-skill")
	commandsDir := filepath.Join(sourceRoot, "commands")
	installedPath := filepath.Join(baseDir, ".claude", "skills", "hello-skill")
	assert.NoErr(t, os.MkdirAll(commandsDir, 0o755))
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(commandsDir, "hello.txt"), []byte("fresh"), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(installedPath, "hello.txt"), []byte("old"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = filepath.Join(baseDir, "missing.lock")
	config.IndexFile = indexFile
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{ProjectDir: filepath.Join(baseDir, ".claude")}
	config.Sources = []sourcepkg.Source{{ID: "local-demo", Type: sourcepkg.TypeLocal, Path: sourceRoot, Status: "ready"}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{{
		ID:                  "hello-skill",
		QualifiedName:       "marketplaces/hello-skill",
		SourceQualifiedName: "repo-a/marketplaces/hello-skill",
		Version:             "1.2.0",
		SourceID:            "local-demo",
		SourceType:          sourcepkg.TypeLocal,
		InstallEntry:        "commands",
		Path:                sourceRoot,
	}}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "claude-code", Scope: "project", WorkDir: baseDir})
	assert.NoErr(t, err)
	assert.Len(t, result.Updated, 1)
	assert.Len(t, result.Skipped, 0)
}

func TestService_RunSkipsAmbiguousScanMatches(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	installedPath := filepath.Join(baseDir, ".claude", "skills", "hello-skill")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = filepath.Join(baseDir, "missing.lock")
	config.IndexFile = indexFile
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{ProjectDir: filepath.Join(baseDir, ".claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "hello-skill", SourceID: "src-a"},
		{ID: "hello-skill", SourceID: "src-b"},
	}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "claude-code", Scope: "project", WorkDir: baseDir})
	assert.NoErr(t, err)
	assert.Len(t, result.Updated, 0)
	assert.Len(t, result.Skipped, 1)
	assert.Eq(t, "hello-skill", result.Skipped[0].SkillID)
	assert.Contains(t, result.Skipped[0].Reason, "ambiguous")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
go test ./internal/app/updateapp -run 'TestService_RunFallsBackToInstalledDirScanWhenLockMissing|TestService_RunSkipsAmbiguousScanMatches'
```

Expected: FAIL because scan fallback is not implemented.

- [ ] **Step 3: Write the minimal scan fallback implementation**

In `internal/app/updateapp/service.go`, extend `Run` so it uses lock candidates first, then falls back when none are found:

```go
items, skipped, err := s.collectFromLock(data, req.Agent, string(scope))
if err != nil {
	return Result{}, err
}
if len(items) == 0 {
	items, skipped, err = s.collectFromInstalledDirs(data, req.WorkDir, req.Agent, scope)
	if err != nil {
		return Result{}, err
	}
}
if len(items) == 0 {
	return Result{Skipped: skipped}, nil
}
```

Add this fallback collector in the same file:

```go
func (s *Service) collectFromInstalledDirs(data cfg.Config, workDir, agentName string, scope agent.Scope) ([]candidate, []SkippedItem, error) {
	targetRoot, err := agent.ResolveInstallPath(data, workDir, agentName, scope)
	if err != nil {
		return nil, nil, err
	}
	entries, err := os.ReadDir(targetRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	index := searchapp.NewService(data.IndexFile)
	items := make([]candidate, 0, len(entries))
	skipped := make([]SkippedItem, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		matches, err := index.Resolve(entry.Name())
		if err != nil {
			skipped = append(skipped, SkippedItem{SkillID: entry.Name(), Reason: "index match not found"})
			continue
		}
		if len(matches) != 1 {
			skipped = append(skipped, SkippedItem{SkillID: entry.Name(), Reason: "ambiguous index match"})
			continue
		}
		items = append(items, candidate{
			SkillID:       matches[0].ID,
			Agent:         agentName,
			Scope:         string(scope),
			SourceID:      matches[0].SourceID,
			InstallEntry:  matches[0].InstallEntry,
			InstalledPath: filepath.Join(targetRoot, entry.Name()),
		})
	}
	return items, skipped, nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run:

```bash
go test ./internal/app/updateapp -run 'TestService_RunFallsBackToInstalledDirScanWhenLockMissing|TestService_RunSkipsAmbiguousScanMatches'
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/updateapp/service.go internal/app/updateapp/service_test.go
git commit -m "feat(update): add installed-dir scan fallback"
```

### Task 5: Wire `update` into the CLI and verify output behavior

**Files:**
- Modify: `internal/cli/manage_cmd.go`
- Modify: `internal/cli/app_test.go`

- [x] **Step 1: Write the failing CLI behavior tests**
- [x] **Step 2: Run the tests to verify they fail**
- [x] **Step 3: Write the minimal command implementation**


```go
func TestUpdateCommand_UpdatesInstalledSkillsFromLock(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexPath := filepath.Join(baseDir, "cache", "index.json")
	sourceDir := filepath.Join(baseDir, "source", "hello-skill")
	commandsDir := filepath.Join(sourceDir, "commands")
	installedPath := filepath.Join(baseDir, ".claude", "skills", "hello-skill")
	assert.NoErr(t, os.MkdirAll(commandsDir, 0o755))
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(commandsDir, "hello.txt"), []byte("updated-from-cli"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexPath
	config.RepoCacheDir = filepath.Join(baseDir, "repos")
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", ProjectDir: filepath.Join(baseDir, ".claude")}
	config.Sources = []sourcepkg.Source{{ID: "local-demo", Type: sourcepkg.TypeLocal, Path: sourceDir, Status: "ready"}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, []lockpkg.Record{{
		SkillID:       "hello-skill",
		Agent:         "claude-code",
		Scope:         "project",
		SourceID:      "local-demo",
		SourceType:    "local",
		InstallEntry:  "commands",
		InstalledPath: installedPath,
	}}))
	assert.NoErr(t, repoindex.NewStore().Save(indexPath, []skill.Skill{{
		ID:           "hello-skill",
		SourceID:     "local-demo",
		SourceType:   sourcepkg.TypeLocal,
		InstallEntry: "commands",
		Path:         sourceDir,
	}}))

	output := runAppInDirWithStdout(t, baseDir, []string{"update", "--agent", "claude-code"})
	assert.Contains(t, output, "updated hello-skill")
}

func TestUpdateCommand_PrintsSkippedReasonForAmbiguousScanMatch(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexPath := filepath.Join(baseDir, "cache", "index.json")
	installedPath := filepath.Join(baseDir, ".claude", "skills", "hello-skill")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = filepath.Join(baseDir, "missing.lock")
	config.IndexFile = indexPath
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", ProjectDir: filepath.Join(baseDir, ".claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexPath, []skill.Skill{{ID: "hello-skill", SourceID: "src-a"}, {ID: "hello-skill", SourceID: "src-b"}}))

	output := runAppInDirWithStdout(t, baseDir, []string{"update", "--agent", "claude-code"})
	assert.Contains(t, output, "skipped hello-skill ambiguous index match")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
go test ./internal/cli -run 'TestUpdateCommand_UpdatesInstalledSkillsFromLock|TestUpdateCommand_PrintsSkippedReasonForAmbiguousScanMatch'
```

Expected: FAIL because the placeholder CLI command does not call the update service.

- [x] **Step 3: Write the minimal command implementation**

First, add an update service constructor in `internal/cli/app.go`:

```go
func newUpdateService() *updateapp.Service {
	return updateapp.NewService(defaultConfigFile(workDir), workDir)
}
```

Then replace the placeholder `buildUpdateCommand()` in `internal/cli/manage_cmd.go` with:

```go
func buildUpdateCommand() *gcli.Command {
	var opts ManageOptions
	return &gcli.Command{
		Name: "update",
		Desc: "Update installed skills",
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
			c.BoolOpt(&opts.Yes, "yes", "y", false, "skip confirmation prompt")
		},
		Func: func(c *gcli.Command, _ []string) error {
			_, cwd, err := loadConfig()
			if err != nil {
				slog.Error(err)
				return err
			}
			result, err := newUpdateService().Run(updateapp.Req{
				Agent:   opts.Agent,
				Scope:   opts.Scope,
				WorkDir: cwd,
			})
			if err != nil {
				slog.Error(err)
				return err
			}
			for _, item := range result.Updated {
				ccolor.Successf("updated %s %s\n", item.SkillID, item.InstalledPath)
			}
			for _, item := range result.Skipped {
				ccolor.Warnf("skipped %s %s\n", item.SkillID, item.Reason)
			}
			for _, item := range result.Failed {
				ccolor.Errorf("failed %s %s\n", item.SkillID, item.Reason)
			}
			return nil
		},
	}
}
```

Also add the import in `internal/cli/manage_cmd.go`:

```go
import (
	// existing imports...
	"github.com/inhere/skillc/internal/app/updateapp"
)
```

And in `internal/cli/app.go`:

```go
import (
	// existing imports...
	"github.com/inhere/skillc/internal/app/updateapp"
)
```

- [ ] **Step 4: Run the tests to verify they pass**

Run:

```bash
go test ./internal/cli -run 'TestUpdateCommand_UpdatesInstalledSkillsFromLock|TestUpdateCommand_PrintsSkippedReasonForAmbiguousScanMatch'
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/app.go internal/cli/manage_cmd.go internal/cli/app_test.go
git commit -m "feat(cli): wire update command"
```

### Task 6: Update docs, ledger, and run the full regression suite

**Files:**
- Modify: `arch.md`
- Modify: `plan.md`
- Optionally modify: `docs/superpowers/specs/2026-04-03-skill-update-design.md` only if implementation diverged and the spec needs correction

- [x] **Step 1: Update `arch.md` to reflect the implemented update behavior**

In the section that lists or describes CLI/manage capabilities, add/update the `skillc update` behavior so it states that v1:

```md
- `skillc update`
  - 默认按已安装项更新
  - 优先使用 lock 记录作为更新输入
  - lock 缺失或为空时，按 `agent + scope` 扫描已安装目录并基于索引唯一匹配回退
  - 执行更新前自动同步相关来源
  - 当前版本不做版本比较，仅执行“同步来源 + 原位重装”
```

- [x] **Step 2: Update `plan.md` to mark the update work accurately**

Add a new task note or update the current ledger entry so it records that `skillc update` has been implemented with:

```md
- 新增 `skillc update`：lock 优先、缺失时扫描已安装目录、先同步来源再原位重装。
```

If `plan.md` uses checkbox/task-state style, mark the relevant phase complete exactly where the file currently tracks feature delivery.

- [x] **Step 3: Run the focused package tests**

Run:

```bash
go test ./internal/app/installapp ./internal/app/updateapp ./internal/cli
```

Expected: PASS.

- [x] **Step 4: Run the full regression suite**

Run:

```bash
go test ./...
```

Expected: PASS for the full repository.

- [ ] **Step 5: Commit**

```bash
git add arch.md plan.md internal/app/installapp/service.go internal/app/installapp/service_test.go internal/app/updateapp/service.go internal/app/updateapp/service_test.go internal/cli/app.go internal/cli/manage_cmd.go internal/cli/app_test.go
git commit -m "feat(update): add installed skill update flow"
```

---

## Self-Review

### Follow-up completed
- [x] `skillc update` 支持通过位置参数指定单个 skill，例如 `skillc update hello-skill`；保留 `--target/-t` 兼容用法，且显式 flag 优先。

### Spec coverage
- Lock-first update path: covered by Task 3.
- Installed-directory scan fallback: covered by Task 4.
- Thin CLI wiring and result output: covered by Tasks 1 and 5.
- Reuse install behavior without duplicating copy logic: covered by Task 2.
- `arch.md` / `plan.md` synchronization required by repo instructions: covered by Task 6.

### Placeholder scan
- No `TBD`, `TODO`, or “implement later” placeholders remain.
- Each code-changing step includes concrete code or concrete text to add.
- Each test step includes an exact `go test` command and expected outcome.

### Type consistency
- `updateapp.Req`, `Result`, `SkippedItem`, and `FailedItem` are used consistently across the plan.
- `installapp.Service.ReinstallAtPath` is the only new install-layer helper referenced later.
- CLI output strings use the same `updated` / `skipped` / `failed` wording throughout.
