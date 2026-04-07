# Lockfile Project-Key Format Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change the install lockfile to group installed skills by project path / `__global__`, store `agents []string` per skill item, and update install/uninstall/restore/update/list flows to use the new model without old-format compatibility.

**Architecture:** Keep CLI behavior thin and move the new grouping/aggregation rules into the lock domain, lockstore, and application services. Treat the lockfile as `map[key][]Record`, where the key is either an absolute project path or `__global__`; runtime install paths are recalculated from `scope + agent + workdir` instead of being stored in the lockfile.

**Tech Stack:** Go, standard library JSON I/O, existing install/update/list services, `gookit/goutil/testutil/assert`, existing `go test` suite

---

## File Structure

### Files to modify
- `internal/domain/lock/model.go` — replace single-agent/single-path record fields with grouped record + lock file type
- `internal/infra/lockstore/json_store.go` — save/load the new grouped JSON structure
- `internal/infra/lockstore/json_store_test.go` — round-trip tests for grouped lock data
- `internal/app/installapp/service.go` — compute lock keys, aggregate `agents`, recalculate paths at runtime, and restore grouped records
- `internal/app/installapp/service_test.go` — lock aggregation, uninstall pruning, restore behavior, global grouping
- `internal/app/updateapp/service.go` — flatten grouped lock data into update jobs and expand one record across multiple agents
- `internal/app/updateapp/service_test.go` — update against grouped records and per-agent reinstall expansion
- `internal/app/listapp/service.go` — read grouped records and project them into list items without `InstalledPath` in the lockfile
- `internal/app/listapp/service_test.go` — list behavior for grouped lock entries and missing/install status
- `internal/cli/manage_cmd.go` — print restored agents from `Agents []string` and keep install output working from runtime paths
- `tests/e2e/install_restore_test.go` — verify install/list/uninstall/restore with grouped lockfile JSON
- `arch.md` — update lock design section to reflect project-key grouping and `agents []string`
- `plan.md` — mark the relevant task/checklist items for this lockfile redesign

### Existing files to consult while implementing
- `docs/superpowers/specs/2026-04-06-lockfile-project-key-design.md`
- `internal/domain/agent/model.go`
- `internal/domain/agent/resolver.go`
- `internal/app/updateapp/service.go`
- `internal/app/installapp/service.go`

---

### Task 1: Redesign the lock domain model and JSON store

**Files:**
- Modify: `internal/domain/lock/model.go`
- Modify: `internal/infra/lockstore/json_store.go`
- Test: `internal/infra/lockstore/json_store_test.go`

- [ ] **Step 1: Write the failing lockstore test for grouped data**

```go
func TestStore_SaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skillc-install.lock")
	store := NewStore()
	now := time.Unix(1710000000, 0).UTC()
	want := lockpkg.File{
		lockpkg.GlobalKey: {
			{
				SkillID:             "hello-skill",
				QualifiedName:       "marketplaces/hello-skill",
				SourceQualifiedName: "workflow-repo/marketplaces/hello-skill",
				Version:             "1.0.0",
				SourceID:            "local-demo",
				SourceType:          "local",
				InstallEntry:        "commands",
				Agents:              []string{"claude-code", "codex"},
				Checksum:            "abc123",
				InstalledAt:         now,
				UpdatedAt:           now,
				Pinned:              true,
			},
		},
		"/workspace/demo": {
			{
				SkillID:             "project-skill",
				QualifiedName:       "marketplaces/project-skill",
				SourceQualifiedName: "workflow-repo/marketplaces/project-skill",
				Version:             "2.0.0",
				SourceID:            "project-demo",
				SourceType:          "git",
				InstallEntry:        ".",
				Agents:              []string{"claude-code"},
				InstalledAt:         now,
				UpdatedAt:           now,
			},
		},
	}

	assert.NoErr(t, store.Save(path, want))
	got, err := store.Load(path)
	assert.NoErr(t, err)
	assert.Eq(t, want, got)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/infra/lockstore -run TestStore_SaveAndLoadRoundTrip -v`
Expected: FAIL because `Save`/`Load` still use `[]lock.Record` and `Record` still requires `Agent` / `InstalledPath`.

- [ ] **Step 3: Write the minimal lock model and store implementation**

```go
package lock

import "time"

const GlobalKey = "__global__"

type File map[string][]Record

type Record struct {
	SkillID             string    `json:"skill_id"`
	QualifiedName       string    `json:"qualified_name"`
	SourceQualifiedName string    `json:"source_qualified_name"`
	Version             string    `json:"version"`
	SourceID            string    `json:"source_id"`
	SourceType          string    `json:"source_type"`
	InstallEntry        string    `json:"install_entry"`
	Agents              []string  `json:"agents"`
	Checksum            string    `json:"checksum"`
	InstalledAt         time.Time `json:"installed_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	Pinned              bool      `json:"pinned"`
}
```

```go
package lockstore

func (s *Store) Save(path string, items lockpkg.File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *Store) Load(path string) (lockpkg.File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var items lockpkg.File
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = lockpkg.File{}
	}
	return items, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/infra/lockstore -run TestStore_SaveAndLoadRoundTrip -v`
Expected: PASS.

- [ ] **Step 5: Run package tests**

Run: `go test ./internal/domain/lock ./internal/infra/lockstore`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/lock/model.go internal/infra/lockstore/json_store.go internal/infra/lockstore/json_store_test.go
git commit -m "refactor(lock): store records by project key"
```

### Task 2: Adapt install writes to grouped records and aggregated agents

**Files:**
- Modify: `internal/app/installapp/service.go`
- Test: `internal/app/installapp/service_test.go`

- [ ] **Step 1: Write the failing install aggregation tests**

```go
func TestService_InstallStoresProjectScopedRecordWithAgents(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := filepath.Join(baseDir, "source")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "commands", "hello.txt"), []byte("hello"), 0o644))

	service := NewService(lockFile)
	item := skill.Skill{
		ID:                  "hello-skill",
		QualifiedName:       "marketplaces/hello-skill",
		SourceQualifiedName: "repo-a/marketplaces/hello-skill",
		Version:             "1.0.0",
		SourceID:            "local-demo",
		SourceType:          sourcepkg.TypeLocal,
		InstallEntry:        "commands",
		Path:                sourceDir,
	}

	_, err := service.Install(item, "claude-code", agent.ScopeProject, filepath.Join(baseDir, ".claude", "skills"), baseDir)
	assert.NoErr(t, err)

	locks, err := service.store.Load(lockFile)
	assert.NoErr(t, err)
	records := locks[baseDir]
	assert.Len(t, records, 1)
	assert.Eq(t, []string{"claude-code"}, records[0].Agents)
}

func TestService_InstallAggregatesAgentsIntoExistingRecord(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	service := NewService(lockFile)
	service.store.Save(lockFile, lockpkg.File{
		baseDir: {
			{
				SkillID:             "hello-skill",
				QualifiedName:       "marketplaces/hello-skill",
				SourceQualifiedName: "repo-a/marketplaces/hello-skill",
				SourceID:            "local-demo",
				SourceType:          "local",
				InstallEntry:        "commands",
				Agents:              []string{"claude-code"},
			},
		},
	})

	// install same skill for codex and verify single record has both agents
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/installapp -run 'TestService_InstallStoresProjectScopedRecordWithAgents|TestService_InstallAggregatesAgentsIntoExistingRecord' -v`
Expected: FAIL because `Install` still stores flat records with `Agent`, `Scope`, and `InstalledPath`.

- [ ] **Step 3: Write the minimal grouped install implementation**

```go
func lockKey(scope agent.Scope, workDir string) (string, error) {
	if scope == agent.ScopeUser {
		return lockpkg.GlobalKey, nil
	}
	return filepath.Abs(workDir)
}

func (s *Service) Install(item skill.Skill, agentName string, scope agent.Scope, targetRoot string, workDir string) (lockpkg.Record, string, error) {
	locks, err := s.loadFile()
	if err != nil {
		return lockpkg.Record{}, "", err
	}
	key, err := lockKey(scope, workDir)
	if err != nil {
		return lockpkg.Record{}, "", err
	}
	targetPath := installTargetPath(locks[key], item, targetRoot)
	if err := s.installer.Install(filepath.Join(item.Path, item.InstallEntry), targetPath); err != nil {
		return lockpkg.Record{}, "", err
	}
	now := s.now()
	next := lockpkg.Record{
		SkillID:             item.ID,
		QualifiedName:       item.QualifiedName,
		SourceQualifiedName: item.SourceQualifiedName,
		Version:             item.Version,
		SourceID:            item.SourceID,
		SourceType:          string(item.SourceType),
		InstallEntry:        item.InstallEntry,
		Agents:              []string{agentName},
		InstalledAt:         now,
		UpdatedAt:           now,
	}
	locks[key] = upsertGroupedRecord(locks[key], next, agentName)
	if err := s.store.Save(s.lockFile, locks); err != nil {
		return lockpkg.Record{}, "", err
	}
	return findRecord(locks[key], next), targetPath, nil
}
```

```go
func upsertGroupedRecord(records []lockpkg.Record, next lockpkg.Record, agentName string) []lockpkg.Record {
	for i, record := range records {
		if sameGroupedRecord(record, next) {
			next.InstalledAt = record.InstalledAt
			next.Agents = appendUnique(record.Agents, agentName)
			records[i] = next
			return records
		}
	}
	next.Agents = appendUnique(nil, agentName)
	return append(records, next)
}
```

- [ ] **Step 4: Update install result plumbing to keep CLI output working**

```go
type InstalledRecord struct {
	Record        lockpkg.Record
	InstalledPath string
}

type CommandResult struct {
	Installed     []InstalledRecord
	Restored      []InstalledRecord
	ResolveFailed []searchapp.TargetError
	InstallFailed []InstallItemError
}
```

```go
result.Installed = append(result.Installed, InstalledRecord{
	Record:        record,
	InstalledPath: targetPath,
})
```

- [ ] **Step 5: Run package tests to verify install aggregation passes**

Run: `go test ./internal/app/installapp -v`
Expected: PASS, including duplicate-source install dir fallback tests after adapting them to grouped records.

- [ ] **Step 6: Commit**

```bash
git add internal/app/installapp/service.go internal/app/installapp/service_test.go
git commit -m "refactor(install): group lock entries by scope key"
```

### Task 3: Update uninstall, restore, and list to compute paths at runtime

**Files:**
- Modify: `internal/app/installapp/service.go`
- Modify: `internal/app/listapp/service.go`
- Test: `internal/app/installapp/service_test.go`
- Test: `internal/app/listapp/service_test.go`
- Test: `tests/e2e/install_restore_test.go`

- [ ] **Step 1: Write the failing restore/list tests for grouped records**

```go
func TestService_RunRestoresGroupedRecordsForEachAgent(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := filepath.Join(baseDir, "cache", "hello-skill")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "commands", "hello.txt"), []byte("restored"), 0o644))
	assert.NoErr(t, NewService(lockFile).store.Save(lockFile, lockpkg.File{
		baseDir: {
			{
				SkillID:             "hello-skill",
				QualifiedName:       "marketplaces/hello-skill",
				SourceQualifiedName: "repo-a/marketplaces/hello-skill",
				SourceID:            "local-demo",
				SourceType:          "local",
				InstallEntry:        "commands",
				Agents:              []string{"claude-code", "codex"},
			},
		},
	}))

	result, err := NewService(lockFile).Run(cfg.Config{
		Sources: []sourcepkg.Source{{ID: "local-demo", Path: sourceDir}},
		AgentTools: map[string]cfg.AgentToolConfig{
			"claude-code": {ProjectDir: filepath.Join(baseDir, ".claude")},
			"codex":       {ProjectDir: filepath.Join(baseDir, ".codex")},
		},
	}, InstallReq{WorkDir: baseDir}, nil)
	assert.NoErr(t, err)
	assert.Len(t, result.Restored, 2)
}

func TestService_ListExpandsGroupedRecordPerAgent(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		baseDir: {
			{SkillID: "hello-skill", QualifiedName: "marketplaces/hello-skill", Agents: []string{"claude-code"}},
		},
		lockpkg.GlobalKey: {
			{SkillID: "hello-skill", QualifiedName: "marketplaces/hello-skill", Agents: []string{"codex"}},
		},
	}))

	items, err := NewService(lockFile).List(baseDir, map[string]cfg.AgentToolConfig{
		"claude-code": {ProjectDir: filepath.Join(baseDir, ".claude")},
		"codex":       {UserDir: filepath.Join(baseDir, ".codex-user")},
	}, "", "")
	assert.NoErr(t, err)
	assert.Len(t, items, 2)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/installapp ./internal/app/listapp ./tests/e2e -run 'TestService_RunRestoresGroupedRecordsForEachAgent|TestService_ListExpandsGroupedRecordPerAgent|TestInstallListAndRestoreFlow' -v`
Expected: FAIL because restore and list still require flat records with `InstalledPath`, `Agent`, and `Scope`.

- [ ] **Step 3: Write the minimal restore/uninstall implementation**

```go
func (s *Service) Restore(config cfg.Config, sourcePaths map[string]string, workDir string) ([]InstalledRecord, error) {
	locks, err := s.loadFile()
	if err != nil {
		return nil, err
	}
	var restored []InstalledRecord
	for key, records := range locks {
		scope, baseDir := scopeFromKey(key, workDir)
		for _, record := range records {
			sourcePath := sourcePaths[record.SourceID]
			installSourcePath := sourcePath
			if record.InstallEntry != "" {
				installSourcePath = filepath.Join(sourcePath, record.InstallEntry)
			}
			for _, agentName := range record.Agents {
				targetRoot, err := agent.ResolveInstallPath(config, baseDir, agentName, scope)
				if err != nil {
					return nil, err
				}
				targetPath := installTargetPath(records, skill.Skill{ID: record.SkillID, SourceQualifiedName: record.SourceQualifiedName, SourceID: record.SourceID}, targetRoot)
				if err := s.installer.Install(installSourcePath, targetPath); err != nil {
					return nil, err
				}
				restored = append(restored, InstalledRecord{Record: withSingleAgent(record, agentName), InstalledPath: targetPath})
			}
		}
	}
	return restored, nil
}
```

```go
func (s *Service) Uninstall(target string, agentName string, scope agent.Scope, workDir string, config cfg.Config) error {
	locks, err := s.loadFile()
	if err != nil {
		return err
	}
	key, err := lockKey(scope, workDir)
	if err != nil {
		return err
	}
	records := locks[key]
	targetRoot, err := agent.ResolveInstallPath(config, workDir, agentName, scope)
	if err != nil {
		return err
	}
	kept := make([]lockpkg.Record, 0, len(records))
	for _, record := range records {
		if !matchesRecordTarget(record, target) {
			kept = append(kept, record)
			continue
		}
		targetPath := installTargetPath(records, skill.Skill{ID: record.SkillID, SourceQualifiedName: record.SourceQualifiedName, SourceID: record.SourceID}, targetRoot)
		if err := s.installer.Remove(targetPath); err != nil {
			return err
		}
		record.Agents = removeAgent(record.Agents, agentName)
		if len(record.Agents) > 0 {
			kept = append(kept, record)
		}
	}
	locks[key] = kept
	return s.store.Save(s.lockFile, locks)
}
```

- [ ] **Step 4: Write the minimal list projection implementation**

```go
type Item struct {
	SkillID             string
	QualifiedName       string
	SourceQualifiedName string
	Agent               string
	Scope               string
	Version             string
	SourceID            string
	SourceType          string
	InstalledPath       string
	Checksum            string
	UpdatedAt           string
	Status              string
}

func (s *Service) List(workDir string, tools map[string]cfg.AgentToolConfig, agentName string, scope string) ([]Item, error) {
	locks, err := s.store.Load(s.lockFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Item{}, nil
		}
		return nil, err
	}
	var items []Item
	for key, records := range locks {
		recordScope, baseDir := scopeFromKey(key, workDir)
		for _, record := range records {
			for _, installedAgent := range record.Agents {
				if agentName != "" && installedAgent != agentName {
					continue
				}
				if scope != "" && string(recordScope) != scope {
					continue
				}
				path, err := resolveRecordedPath(tools, baseDir, installedAgent, recordScope, record)
				if err != nil {
					return nil, err
				}
				items = append(items, toItem(record, installedAgent, string(recordScope), path))
			}
		}
	}
	return items, nil
}
```

- [ ] **Step 5: Update the E2E test to assert JSON object keys and aggregated agents**

```go
raw, err := os.ReadFile(lockFile)
assert.NoErr(t, err)
assert.Contains(t, string(raw), baseDir)
assert.Contains(t, string(raw), `"agents":["claude-code"]`)
assert.NotContains(t, string(raw), `"installed_path"`)
```

- [ ] **Step 6: Run tests to verify install/list/restore behavior passes**

Run: `go test ./internal/app/installapp ./internal/app/listapp ./tests/e2e -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/app/installapp/service.go internal/app/installapp/service_test.go internal/app/listapp/service.go internal/app/listapp/service_test.go tests/e2e/install_restore_test.go
git commit -m "refactor(restore): derive install paths from grouped lockfile"
```

### Task 4: Expand update flow from grouped lock records into per-agent reinstall work

**Files:**
- Modify: `internal/app/updateapp/service.go`
- Test: `internal/app/updateapp/service_test.go`

- [ ] **Step 1: Write the failing update tests for grouped records**

```go
func TestService_RunUsesGroupedLockRecordsAndRecomputesTargetPaths(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{ProjectDir: filepath.Join(baseDir, "project-claude")}
	config.AgentTools["codex"] = cfg.AgentToolConfig{ProjectDir: filepath.Join(baseDir, "project-codex")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		baseDir: {
			{
				SkillID:             "hello-skill",
				QualifiedName:       "marketplaces/hello-skill",
				SourceQualifiedName: "repo-a/marketplaces/hello-skill",
				SourceID:            "source-a",
				InstallEntry:        "commands",
				Agents:              []string{"claude-code", "codex"},
			},
		},
	}))

	calls := make([]string, 0)
	service := NewService(configFile, baseDir)
	service.newInstaller = func(path string) reinstallService {
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, targetPath string) (lockpkg.Record, error) {
			calls = append(calls, agentName+"@"+targetPath)
			return lockpkg.Record{SkillID: item.ID, Agents: []string{agentName}, SourceID: item.SourceID}, nil
		}}
	}

	result, err := service.Run(Req{Scope: "project", WorkDir: baseDir})
	assert.NoErr(t, err)
	assert.Len(t, result.Updated, 2)
	assert.Len(t, calls, 2)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/updateapp -run 'TestService_RunUsesGroupedLockRecordsAndRecomputesTargetPaths|TestService_RunAggregatesSyncAndReinstallFailures' -v`
Expected: FAIL because update still consumes flat records with `Agent`, `Scope`, and `InstalledPath`.

- [ ] **Step 3: Write the minimal grouped update selection logic**

```go
type InstalledTarget struct {
	Key       string
	Scope     agent.Scope
	WorkDir   string
	Agent     string
	Record    lockpkg.Record
	TargetPath string
}

func flattenRecords(config cfg.Config, workDir string, locks lockpkg.File) ([]InstalledTarget, error) {
	var targets []InstalledTarget
	for key, records := range locks {
		scope, baseDir := scopeFromKey(key, workDir)
		for _, record := range records {
			for _, agentName := range record.Agents {
				targetRoot, err := agent.ResolveInstallPath(config, baseDir, agentName, scope)
				if err != nil {
					return nil, err
				}
				targets = append(targets, InstalledTarget{
					Key:        key,
					Scope:      scope,
					WorkDir:    baseDir,
					Agent:      agentName,
					Record:     record,
					TargetPath: installTargetPath(records, skill.Skill{ID: record.SkillID, SourceQualifiedName: record.SourceQualifiedName, SourceID: record.SourceID}, targetRoot),
				})
			}
		}
	}
	return targets, nil
}
```

```go
for _, candidate := range result.Candidates {
	record, err := worker.ReinstallAtPath(candidate.Latest, candidate.Target.Agent, candidate.Target.Scope, candidate.Target.TargetPath)
	if err != nil {
		...
		continue
	}
	result.Updated = append(result.Updated, record)
}
```

- [ ] **Step 4: Keep candidate matching keyed by grouped identity**

```go
func sameCandidateIdentity(record lockpkg.Record, item skill.Skill) bool {
	if record.SkillID != item.ID {
		return false
	}
	if record.SourceID != "" || item.SourceID != "" {
		return record.SourceID != "" && record.SourceID == item.SourceID
	}
	if record.SourceQualifiedName != "" || item.SourceQualifiedName != "" {
		return record.SourceQualifiedName != "" && record.SourceQualifiedName == item.SourceQualifiedName
	}
	return record.QualifiedName != "" && record.QualifiedName == item.QualifiedName
}
```

- [ ] **Step 5: Run package tests**

Run: `go test ./internal/app/updateapp -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/app/updateapp/service.go internal/app/updateapp/service_test.go
git commit -m "refactor(update): expand grouped lock entries per agent"
```

### Task 5: Update CLI output and project docs

**Files:**
- Modify: `internal/cli/manage_cmd.go`
- Modify: `arch.md`
- Modify: `plan.md`
- Test: `internal/cli/app_test.go`

- [ ] **Step 1: Write the failing CLI test for restore output from grouped records**

```go
func TestInstallCommandRestorePrintsEachRestoredAgent(t *testing.T) {
	// arrange grouped restore result and assert output contains one line per restored agent
	assert.Contains(t, output, "hello-skill claude-code project")
	assert.Contains(t, output, "hello-skill codex project")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli -run TestInstallCommandRestorePrintsEachRestoredAgent -v`
Expected: FAIL because CLI still assumes `record.Agent` and `record.Scope` directly from flat lock records.

- [ ] **Step 3: Update CLI output to read the new install/restore result shape**

```go
for _, installed := range result.Installed {
	if err := WriteLine(os.Stdout, fmt.Sprintf("installed %s %s", installed.Record.SkillID, installed.InstalledPath)); err != nil {
		return err
	}
}
for _, restored := range result.Restored {
	scopeLabel := "project"
	if restored.Scope == agent.ScopeUser {
		scopeLabel = "global"
	}
	if err := WriteLine(os.Stdout, fmt.Sprintf("%s %s %s", restored.Record.SkillID, restored.Record.Agents[0], scopeLabel)); err != nil {
		return err
	}
}
```

- [ ] **Step 4: Update architecture and task tracking docs**

Add to `arch.md` lock section:

```md
- lock file 顶层改为按 scope key 分组：project 使用绝对项目路径，global 使用 `__global__`
- 单条 skill record 使用 `agents []string` 聚合多个 agent 安装事实
- `InstalledPath` 不再落盘；restore/update/list 在运行时重新解析目标路径
```

Update the relevant task in `plan.md` from pending wording to completed wording for this redesign, for example:

```md
### Task 16: Redesign lockfile grouping for project keys ✅ Completed
- [x] 改为 project path / `__global__` 分组
- [x] skill record 聚合 `agents []string`
- [x] install / restore / update / list 适配新模型
- [x] `go test ./...`
```

- [ ] **Step 5: Run CLI tests and targeted doc sanity check**

Run: `go test ./internal/cli -v`
Expected: PASS.

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/manage_cmd.go internal/cli/app_test.go arch.md plan.md
git commit -m "docs(cli): align output and docs with grouped lockfile"
```

### Task 6: Run full verification and close out

**Files:**
- Verify only: repository-wide changes from Tasks 1-5

- [ ] **Step 1: Run the full test suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 2: Inspect the lockfile shape in the E2E artifact mentally against the spec**

Checklist:

```text
- top-level JSON is an object, not an array
- project installs live under absolute project path keys
- global installs live under __global__
- each skill record stores agents[]
- installed_path is absent from the lockfile
- restore still uses SourceID + InstallEntry
```

- [ ] **Step 3: Review git diff for unintended churn**

Run: `git diff -- internal/domain/lock/model.go internal/infra/lockstore/json_store.go internal/app/installapp/service.go internal/app/updateapp/service.go internal/app/listapp/service.go internal/cli/manage_cmd.go arch.md plan.md tests/e2e/install_restore_test.go`
Expected: only the planned lockfile redesign and doc updates appear.

- [ ] **Step 4: Commit final verification touch-ups if needed**

```bash
git add internal/domain/lock/model.go internal/infra/lockstore/json_store.go internal/app/installapp/service.go internal/app/updateapp/service.go internal/app/listapp/service.go internal/cli/manage_cmd.go arch.md plan.md tests/e2e/install_restore_test.go
git commit -m "test(lock): verify grouped lockfile workflow"
```
