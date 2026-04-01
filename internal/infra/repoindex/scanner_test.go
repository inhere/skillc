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
	items, err := scanner.Scan(source.Source{ID: "local-demo", Type: source.TypeLocal, Path: root})
	assert.NoErr(t, err)
	assert.Len(t, items, 1)
	assert.Eq(t, "skill-a", items[0].ID)
}

func TestScanner_ScanIndexesMultipleSkillDirectories(t *testing.T) {
	root := t.TempDir()
	assert.NoErr(t, os.MkdirAll(filepath.Join(root, "skill-a"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(root, "skill-a", "SKILL.md"), []byte(`---
id: skill-a
name: Skill A
version: 1.0.0
---`), 0o644))
	assert.NoErr(t, os.MkdirAll(filepath.Join(root, "skill-b"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(root, "skill-b", "SKILL.md"), []byte(`---
id: skill-b
name: Skill B
---`), 0o644))

	scanner := NewScanner()
	items, err := scanner.Scan(source.Source{ID: "git-demo", Type: source.TypeGit, Path: root, Ref: "abc12345"})
	assert.NoErr(t, err)
	assert.Len(t, items, 2)
	assert.Eq(t, "abc12345", items[1].Version)
}

func TestScanner_ScanIndexesSkillsFromNestedSkillsDirectory(t *testing.T) {
	root := t.TempDir()
	skillsRoot := filepath.Join(root, "skills")
	assert.NoErr(t, os.MkdirAll(filepath.Join(skillsRoot, "frontend-design"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(skillsRoot, "frontend-design", "SKILL.md"), []byte(`---
id: frontend-design
name: Frontend Design
---`), 0o644))

	scanner := NewScanner()
	items, err := scanner.Scan(source.Source{ID: "local-demo", Type: source.TypeLocal, Path: root})
	assert.NoErr(t, err)
	assert.Len(t, items, 1)
	assert.Eq(t, "frontend-design", items[0].ID)
}
