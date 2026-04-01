package skill

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/source"
)

func TestParseSkillMarkdown_ParsesFrontMatter(t *testing.T) {
	content := `---
id: hello-skill
name: Hello Skill
description: Say hello
version: 1.2.3
supported_agents:
  - claude-code
install_entry: prompt.md
---

# Hello`

	got, err := ParseSkillMarkdown(content, source.Source{ID: "local-demo", Type: source.TypeLocal, Path: "/skills/hello"})
	assert.NoErr(t, err)
	assert.Eq(t, "hello-skill", got.ID)
	assert.Eq(t, "Hello Skill", got.Name)
	assert.Eq(t, "1.2.3", got.Version)
	assert.Eq(t, "prompt.md", got.InstallEntry)
	assert.Eq(t, "local-demo", got.SourceID)
	assert.Eq(t, source.TypeLocal, got.SourceType)
}

func TestParseSkillMarkdown_FallsBackVersionForLocal(t *testing.T) {
	content := `---
id: hello-skill
name: Hello Skill
---

# Hello`

	got, err := ParseSkillMarkdown(content, source.Source{ID: "local-demo", Type: source.TypeLocal, Path: "/skills/hello"})
	assert.NoErr(t, err)
	assert.NotEmpty(t, got.Version)
	assert.Eq(t, 8, len(got.Version))
}

func TestParseSkillMarkdown_FallsBackToResolvedGitRef(t *testing.T) {
	content := `---
id: hello-skill
name: Hello Skill
---

# Hello`

	got, err := ParseSkillMarkdown(content, source.Source{ID: "git-demo", Type: source.TypeGit, Path: "/skills/hello", Ref: "main", ResolvedRef: "1234567890abcdef"})
	assert.NoErr(t, err)
	assert.Eq(t, "12345678", got.Version)
}

func TestParseSkillMarkdown_FallsBackToNameAsID(t *testing.T) {
	content := `---
name: frontend-design
description: Design frontend interfaces
---

# Frontend Design`

	got, err := ParseSkillMarkdown(content, source.Source{ID: "local-demo", Type: source.TypeLocal, Path: "/skills/frontend-design"})
	assert.NoErr(t, err)
	assert.Eq(t, "frontend-design", got.ID)
	assert.Eq(t, "frontend-design", got.Name)
}
