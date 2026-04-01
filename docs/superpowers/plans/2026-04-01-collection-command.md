# Collection Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `skillc collection list` and `skillc collection skills <collection>` so users can browse collections derived from indexed skill metadata.

**Architecture:** Keep CLI responsibilities in `internal/cli`, expose collection read APIs from `internal/app/searchapp`, and put collection aggregation/filtering in a dedicated `internal/infra/repoindex/collection.go`. Reuse the existing index file loading path and return stable, sorted results for predictable CLI output and tests.

**Tech Stack:** Go, gookit/gcli, gookit table output, Go testing package, existing repoindex JSON store

---

## File Structure

### Documents to consult
- `docs/superpowers/specs/2026-04-01-collection-command-design.md` — approved behavior and scope
- `arch.md` — layered boundaries (`cli` → `app` → `infra`)
- `plan.md` — current MVP task ledger that should be updated when task status/design changes

### Files to create or modify

**Collection aggregation**
- Create: `internal/infra/repoindex/collection.go` — collection summary model, grouping, sorting, and collection skill lookup
- Create: `internal/infra/repoindex/collection_test.go` — collection aggregation and lookup tests

**Application service**
- Modify: `internal/app/searchapp/service.go` — add `ListCollections()` and `ListCollectionSkills(collection string)`
- Modify: `internal/app/searchapp/service_test.go` — service behavior for collection listing and missing-index handling

**CLI**
- Create: `internal/cli/collection_cmd.go` — `collection list` and `collection skills <collection>` command wiring and table output
- Modify: `internal/cli/app.go` — register the new `collection` command
- Modify: `internal/cli/app_test.go` — command registration and CLI output tests

**Project docs**
- Modify: `arch.md` — add collection browsing to repo index / CLI capability descriptions
- Modify: `plan.md` — add and then mark complete the collection-command task as work proceeds

---

### Task 1: Add repoindex collection aggregation primitives

**Files:**
- Create: `internal/infra/repoindex/collection.go`
- Test: `internal/infra/repoindex/collection_test.go`

- [ ] **Step 1: Write the failing aggregation tests**

```go
package repoindex

import (
    "testing"

    "github.com/gookit/goutil/testutil/assert"
    "github.com/inhere/skillc/internal/domain/skill"
)

func TestListCollections_GroupsByCollectionAndCountsUniqueSources(t *testing.T) {
    items := []skill.Skill{
        {ID: "alpha-one", Name: "Alpha One", Collection: "alpha", SourceID: "src-a", SourceName: "repo-a"},
        {ID: "alpha-two", Name: "Alpha Two", Collection: "alpha", SourceID: "src-a", SourceName: "repo-a"},
        {ID: "alpha-three", Name: "Alpha Three", Collection: "alpha", SourceID: "src-b", SourceName: "repo-b"},
        {ID: "beta-one", Name: "Beta One", Collection: "beta", SourceID: "src-c", SourceName: "repo-c"},
        {ID: "standalone", Name: "Standalone", Collection: "", SourceID: "src-a", SourceName: "repo-a"},
    }

    got := ListCollections(items)
    assert.Len(t, got, 2)
    assert.Eq(t, "alpha", got[0].Name)
    assert.Eq(t, 3, got[0].SkillCount)
    assert.Eq(t, 2, got[0].SourceCount)
    assert.Eq(t, "beta", got[1].Name)
    assert.Eq(t, 1, got[1].SkillCount)
    assert.Eq(t, 1, got[1].SourceCount)
}

func TestListCollectionSkills_ReturnsSortedSkillsForCollection(t *testing.T) {
    items := []skill.Skill{
        {ID: "zeta", Name: "Zeta Skill", Description: "last", Collection: "alpha"},
        {ID: "alpha", Name: "Alpha Skill", Description: "first", Collection: "alpha"},
        {ID: "beta", Name: "Beta Skill", Description: "other", Collection: "beta"},
    }

    got, err := ListCollectionSkills(items, "alpha")
    assert.NoErr(t, err)
    assert.Len(t, got, 2)
    assert.Eq(t, "Alpha Skill", got[0].Name)
    assert.Eq(t, "Zeta Skill", got[1].Name)
}

func TestListCollectionSkills_ReturnsErrorWhenCollectionMissing(t *testing.T) {
    _, err := ListCollectionSkills([]skill.Skill{{ID: "solo", Name: "Solo"}}, "missing")
    assert.Err(t, err)
    assert.Contains(t, err.Error(), "collection not found: missing")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/infra/repoindex -run 'TestListCollections|TestListCollectionSkills'`
Expected: FAIL with undefined `ListCollections`, `ListCollectionSkills`, and `CollectionSummary`

- [ ] **Step 3: Write minimal aggregation implementation**

```go
package repoindex

import (
    "fmt"
    "sort"

    "github.com/inhere/skillc/internal/domain/skill"
)

type CollectionSummary struct {
    Name        string
    SkillCount  int
    SourceCount int
}

func ListCollections(items []skill.Skill) []CollectionSummary {
    byName := make(map[string]*CollectionSummary)
    sourcesByCollection := make(map[string]map[string]struct{})

    for _, item := range items {
        if item.Collection == "" {
            continue
        }
        summary, ok := byName[item.Collection]
        if !ok {
            summary = &CollectionSummary{Name: item.Collection}
            byName[item.Collection] = summary
            sourcesByCollection[item.Collection] = map[string]struct{}{}
        }
        summary.SkillCount++

        sourceKey := item.SourceName
        if sourceKey == "" {
            sourceKey = item.SourceID
        }
        if sourceKey != "" {
            sourcesByCollection[item.Collection][sourceKey] = struct{}{}
        }
    }

    out := make([]CollectionSummary, 0, len(byName))
    for name, summary := range byName {
        summary.SourceCount = len(sourcesByCollection[name])
        out = append(out, *summary)
    }

    sort.Slice(out, func(i, j int) bool {
        return out[i].Name < out[j].Name
    })
    return out
}

func ListCollectionSkills(items []skill.Skill, collection string) ([]skill.Skill, error) {
    matches := make([]skill.Skill, 0)
    for _, item := range items {
        if item.Collection == collection {
            matches = append(matches, item)
        }
    }
    if len(matches) == 0 {
        return nil, fmt.Errorf("collection not found: %s", collection)
    }
    sort.Slice(matches, func(i, j int) bool {
        if matches[i].Name == matches[j].Name {
            return matches[i].ID < matches[j].ID
        }
        return matches[i].Name < matches[j].Name
    })
    return matches, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/infra/repoindex -run 'TestListCollections|TestListCollectionSkills'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/infra/repoindex/collection.go internal/infra/repoindex/collection_test.go
git commit -m "feat(repoindex): add collection aggregation helpers"
```

### Task 2: Expose collection queries from searchapp

**Files:**
- Modify: `internal/app/searchapp/service.go`
- Modify: `internal/app/searchapp/service_test.go`

- [ ] **Step 1: Write the failing service tests**

Append these tests to `internal/app/searchapp/service_test.go`:

```go
func TestService_ListCollectionsReturnsEmptyWhenIndexMissing(t *testing.T) {
    service := NewService(filepath.Join(t.TempDir(), "missing.json"))

    items, err := service.ListCollections()
    assert.NoErr(t, err)
    assert.Len(t, items, 0)
}

func TestService_ListCollectionsAndSkills(t *testing.T) {
    baseDir := t.TempDir()
    indexPath := filepath.Join(baseDir, "index.json")
    store := repoindex.NewStore()
    assert.NoErr(t, store.Save(indexPath, []skill.Skill{
        {ID: "alpha-two", Name: "Alpha Two", Description: "second", Collection: "alpha", SourceID: "src-a", SourceName: "repo-a"},
        {ID: "alpha-one", Name: "Alpha One", Description: "first", Collection: "alpha", SourceID: "src-b", SourceName: "repo-b"},
        {ID: "beta-one", Name: "Beta One", Description: "beta", Collection: "beta", SourceID: "src-a", SourceName: "repo-a"},
    }))

    service := NewService(indexPath)

    collections, err := service.ListCollections()
    assert.NoErr(t, err)
    assert.Len(t, collections, 2)
    assert.Eq(t, "alpha", collections[0].Name)
    assert.Eq(t, 2, collections[0].SkillCount)
    assert.Eq(t, 2, collections[0].SourceCount)

    skills, err := service.ListCollectionSkills("alpha")
    assert.NoErr(t, err)
    assert.Len(t, skills, 2)
    assert.Eq(t, "Alpha One", skills[0].Name)
    assert.Eq(t, "Alpha Two", skills[1].Name)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/searchapp -run 'TestService_ListCollections|TestService_ListCollectionsAndSkills'`
Expected: FAIL with undefined `ListCollections` and `ListCollectionSkills`

- [ ] **Step 3: Write minimal service methods**

Update `internal/app/searchapp/service.go` with these methods:

```go
func (s *Service) ListCollections() ([]repoindex.CollectionSummary, error) {
    items, err := s.store.Load(s.indexPath)
    if err != nil {
        if os.IsNotExist(err) {
            return []repoindex.CollectionSummary{}, nil
        }
        return nil, err
    }
    return repoindex.ListCollections(items), nil
}

func (s *Service) ListCollectionSkills(collection string) ([]skill.Skill, error) {
    items, err := s.store.Load(s.indexPath)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil
        }
        return nil, err
    }
    return repoindex.ListCollectionSkills(items, collection)
}
```

Then tighten the missing-index behavior so a missing index is treated the same as an empty dataset for skill lookup:

```go
func (s *Service) ListCollectionSkills(collection string) ([]skill.Skill, error) {
    items, err := s.store.Load(s.indexPath)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, fmt.Errorf("collection not found: %s", collection)
        }
        return nil, err
    }
    return repoindex.ListCollectionSkills(items, collection)
}
```

Add the missing import:

```go
import "fmt"
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/searchapp -run 'TestService_ListCollections|TestService_ListCollectionsAndSkills'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/searchapp/service.go internal/app/searchapp/service_test.go
git commit -m "feat(searchapp): expose collection query methods"
```

### Task 3: Add the `collection` CLI command

**Files:**
- Create: `internal/cli/collection_cmd.go`
- Modify: `internal/cli/app.go`
- Modify: `internal/cli/app_test.go`

- [ ] **Step 1: Write the failing CLI tests**

Add these tests to `internal/cli/app_test.go`:

```go
func TestNewApp_RegistersCollectionCommand(t *testing.T) {
    app := newTestApp()

    collection := findCommandByName(app, "collection")
    assert.NotNil(t, collection)
    assert.Eq(t, "Browse indexed collections", collection.Desc)
}

func TestCollectionListCommand_PrintsCollectionSummary(t *testing.T) {
    baseDir := t.TempDir()
    config := cfg.DefaultConfig()
    config.IndexFile = filepath.Join(baseDir, "cache", "index.json")
    assert.NoErr(t, configstore.NewYAMLStore().Save(filepath.Join(baseDir, "skillc.yaml"), config))
    assert.NoErr(t, repoindex.NewStore().Save(config.IndexFile, []skill.Skill{
        {ID: "alpha-one", Name: "Alpha One", Collection: "alpha", SourceID: "src-a", SourceName: "repo-a"},
        {ID: "alpha-two", Name: "Alpha Two", Collection: "alpha", SourceID: "src-b", SourceName: "repo-b"},
        {ID: "beta-one", Name: "Beta One", Collection: "beta", SourceID: "src-a", SourceName: "repo-a"},
    }))

    output := runAppInDirWithStdout(t, baseDir, []string{"collection", "list"})
    assert.Contains(t, output, "alpha 2 2")
    assert.Contains(t, output, "beta 1 1")
}

func TestCollectionSkillsCommand_PrintsSkillNameAndDescription(t *testing.T) {
    baseDir := t.TempDir()
    config := cfg.DefaultConfig()
    config.IndexFile = filepath.Join(baseDir, "cache", "index.json")
    assert.NoErr(t, configstore.NewYAMLStore().Save(filepath.Join(baseDir, "skillc.yaml"), config))
    assert.NoErr(t, repoindex.NewStore().Save(config.IndexFile, []skill.Skill{
        {ID: "alpha-one", Name: "Alpha One", Description: "first skill", Collection: "alpha"},
        {ID: "alpha-two", Name: "Alpha Two", Description: "second skill", Collection: "alpha"},
    }))

    output := runAppInDirWithStdout(t, baseDir, []string{"collection", "skills", "alpha"})
    assert.Contains(t, output, "Alpha One first skill")
    assert.Contains(t, output, "Alpha Two second skill")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run 'TestNewApp_RegistersCollectionCommand|TestCollectionListCommand|TestCollectionSkillsCommand'`
Expected: FAIL because the `collection` command is not registered yet

- [ ] **Step 3: Implement CLI wiring**

Create `internal/cli/collection_cmd.go`:

```go
package cli

import (
    "fmt"

    "github.com/gookit/gcli/v3"
    "github.com/gookit/gcli/v3/show/table"
    "github.com/gookit/goutil/x/ccolor"
    "github.com/gookit/slog"
)

func buildCollectionCommand() *gcli.Command {
    cmd := &gcli.Command{
        Name: "collection",
        Desc: "Browse indexed collections",
    }

    cmd.Add(&gcli.Command{
        Name: "list",
        Desc: "List indexed collections",
        Func: func(c *gcli.Command, args []string) error {
            service := newSearchService()
            items, err := service.ListCollections()
            if err != nil {
                slog.Error(err)
                return err
            }
            if len(items) == 0 {
                ccolor.Warnln("no collections found")
                return nil
            }

            tb := table.New("Collection List").SetHeads("Collection", "Skills", "Sources")
            for _, item := range items {
                tb.AddRow(item.Name, item.SkillCount, item.SourceCount)
            }
            tb.Println()
            return nil
        },
    })

    cmd.Add(&gcli.Command{
        Name: "skills",
        Desc: "List skills in a collection",
        Func: func(c *gcli.Command, args []string) error {
            if len(args) < 1 {
                return fmt.Errorf("collection name is required")
            }
            service := newSearchService()
            items, err := service.ListCollectionSkills(args[0])
            if err != nil {
                slog.Error(err)
                return err
            }

            tb := table.New("Collection Skills").SetHeads("Name", "Description")
            for _, item := range items {
                tb.AddRow(item.Name, item.Description)
            }
            tb.Println()
            return nil
        },
    })

    return cmd
}
```

Register it in `internal/cli/app.go` inside `NewApp`:

```go
app.Add(buildCollectionCommand())
```

Place it after `buildSourceCommand()` and before `buildSearchCommand()`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run 'TestNewApp_RegistersCollectionCommand|TestCollectionListCommand|TestCollectionSkillsCommand'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/collection_cmd.go internal/cli/app.go internal/cli/app_test.go
git commit -m "feat(cli): add collection browse commands"
```

### Task 4: Sync project docs and run regression tests

**Files:**
- Modify: `arch.md`
- Modify: `plan.md`

- [ ] **Step 1: Add the architecture note for collection browsing**

Update the CLI/source/index portions of `arch.md` so they explicitly mention collection browsing. Add these bullets in the relevant sections:

```md
### 4.3 repo index

职责：
- 扫描本地源或 Git 缓存目录
- 识别顶级 Skill 与 collection 边界
- 发现 Skill 子目录
- 解析 `SKILL.md` YAML front matter
- 生成标准化 Skill 索引与限定名
- 提供 collection 聚合视图，支持集合列表与集合内 skill 浏览
```

And in the CLI-related description:

```md
- `skillc collection` 提供基于索引的集合浏览能力，不承担聚合规则本身
```

- [ ] **Step 2: Update `plan.md` with the new completed task**

Append this task block near the end of `plan.md`:

```md
### Task 18: Add collection browsing commands ✅ Completed

**Files:**
- Create: `internal/infra/repoindex/collection.go`
- Create: `internal/cli/collection_cmd.go`
- Modify: `internal/app/searchapp/service.go`
- Modify: `internal/cli/app.go`
- Modify: `internal/cli/app_test.go`
- Modify: `internal/app/searchapp/service_test.go`
- Modify: `arch.md`

- [x] **Step 1: Write the failing tests**
- [x] **Step 2: Run tests to verify they fail**
- [x] **Step 3: Write minimal implementation**
- [x] **Step 4: Run tests to verify they pass**
- [x] **Step 5: Run regression tests**
- [x] **Step 6: Commit**
```

- [ ] **Step 3: Run targeted regression tests**

Run: `go test ./internal/infra/repoindex ./internal/app/searchapp ./internal/cli`
Expected: PASS

- [ ] **Step 4: Run full regression required by project instructions**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add arch.md plan.md
git commit -m "docs: record collection command design updates"
```

## Self-Review Checklist

- **Spec coverage:**
  - `collection list` with `Collection | Skills | Sources` is implemented by Task 1 + Task 3.
  - `collection skills <collection>` with `Name | Description` is implemented by Task 1 + Task 2 + Task 3.
  - Empty collection filtering and stable sort are handled in Task 1.
  - Missing-index and missing-collection behavior are handled in Task 2 + Task 3.
  - Architecture/document sync required by repo instructions is handled in Task 4.

- **Placeholder scan:** No `TBD`, `TODO`, “similar to”, or unspecified test steps remain.

- **Type consistency:** The plan consistently uses `repoindex.CollectionSummary`, `Service.ListCollections()`, and `Service.ListCollectionSkills(collection string)`.
