package repoindex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/source"
)

func TestScanner_ScanSkipsDirectoriesWithoutSkillMarkdown(t *testing.T) {
	root := t.TempDir()
	assert.NoErr(t, os.MkdirAll(filepath.Join(root, "skill-a"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(root, "skill-a", "SKILL.md"), []byte(`---
id: skill-a
name: Skill A
---`), 0o644))
	assert.NoErr(t, os.MkdirAll(filepath.Join(root, "notes"), 0o755))

	scanner := NewScanner()
	items, err := scanner.Scan(source.Source{ID: "local-demo", Name: "local-demo", Type: source.TypeLocal, Path: root})
	assert.NoErr(t, err)
	assert.Len(t, items, 1)
	assert.Eq(t, "skill-a", items[0].ID)
	assert.Eq(t, "skill-a", items[0].QualifiedName)
}

func TestScanner_ScanTreatsSingleRootSkillAsTopLevel(t *testing.T) {
	root := t.TempDir()
	assert.NoErr(t, os.MkdirAll(filepath.Join(root, "skill-a"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(root, "skill-a", "SKILL.md"), []byte(`---
id: skill-a
name: Skill A
---`), 0o644))

	items, err := NewScanner().Scan(source.Source{ID: "local-demo", Name: "workflow-repo", Type: source.TypeLocal, Path: root})
	assert.NoErr(t, err)
	assert.Len(t, items, 1)
	assert.Eq(t, "", items[0].Collection)
	assert.Eq(t, "skill-a", items[0].QualifiedName)
	assert.Eq(t, "workflow-repo/skill-a", items[0].SourceQualifiedName)
}

func TestScanner_ScanAssignsSourceNameCollectionWhenRootHasMultipleSkills(t *testing.T) {
	root := t.TempDir()
	assert.NoErr(t, os.MkdirAll(filepath.Join(root, "skill-a"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(root, "skill-a", "SKILL.md"), []byte(`---
id: skill-a
name: Skill A
---`), 0o644))
	assert.NoErr(t, os.MkdirAll(filepath.Join(root, "skill-b"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(root, "skill-b", "SKILL.md"), []byte(`---
id: skill-b
name: Skill B
---`), 0o644))

	items, err := NewScanner().Scan(source.Source{ID: "git-demo", Name: "workflow-repo", Type: source.TypeGit, Path: root, Ref: "abc12345"})
	assert.NoErr(t, err)
	assert.Len(t, items, 2)
	assert.Eq(t, "workflow-repo", items[0].Collection)
	assert.Eq(t, "workflow-repo/skill-a", items[0].QualifiedName)
	assert.Eq(t, "workflow-repo/skill-a", items[0].SourceQualifiedName)
	assert.Eq(t, "abc12345", items[1].Version)
}

func TestScanner_ScanAssignsSourceNameCollectionForSingleSkillsDirectory(t *testing.T) {
	root := t.TempDir()
	skillsRoot := filepath.Join(root, "skills")
	assert.NoErr(t, os.MkdirAll(filepath.Join(skillsRoot, "frontend-design"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(skillsRoot, "frontend-design", "SKILL.md"), []byte(`---
id: frontend-design
name: Frontend Design
---`), 0o644))

	items, err := NewScanner().Scan(source.Source{ID: "local-demo", Name: "workflow-repo", Type: source.TypeLocal, Path: root})
	assert.NoErr(t, err)
	assert.Len(t, items, 1)
	assert.Eq(t, "workflow-repo", items[0].Collection)
	assert.Eq(t, "workflow-repo/frontend-design", items[0].QualifiedName)
	assert.Eq(t, "workflow-repo/frontend-design", items[0].SourceQualifiedName)
}

func TestScanner_ScanUsesParentNameForMultipleSkillsDirectories(t *testing.T) {
	root := t.TempDir()
	marketplaces := filepath.Join(root, "plugins", "marketplaces", "skills", "skill-a")
	productivity := filepath.Join(root, "plugins", "productivity", "skills", "skill-a")
	assert.NoErr(t, os.MkdirAll(marketplaces, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(marketplaces, "SKILL.md"), []byte(`---
id: skill-a
name: Skill A
---`), 0o644))
	assert.NoErr(t, os.MkdirAll(productivity, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(productivity, "SKILL.md"), []byte(`---
id: skill-a
name: Skill A
---`), 0o644))

	items, err := NewScanner().Scan(source.Source{ID: "local-demo", Name: "workflow-repo", Type: source.TypeLocal, Path: root})
	assert.NoErr(t, err)
	assert.Len(t, items, 2)
	assert.Eq(t, "marketplaces", items[0].Collection)
	assert.Eq(t, "marketplaces/skill-a", items[0].QualifiedName)
	assert.Eq(t, "productivity", items[1].Collection)
	assert.Eq(t, "productivity/skill-a", items[1].QualifiedName)
}
