package repoindex

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
)

func TestFilter_MatchesNameDescriptionAgentAndSourceType(t *testing.T) {
	items := []skill.Skill{
		{
			ID:              "hello-skill",
			Name:            "Hello Skill",
			Description:     "Friendly greeting helper",
			SupportedAgents: []string{"claude-code"},
			SourceType:      sourcepkg.TypeLocal,
		},
		{
			ID:              "git-only",
			Name:            "Git Only",
			Description:     "Remote repo helper",
			SupportedAgents: []string{"codex"},
			SourceType:      sourcepkg.TypeGit,
		},
	}

	got := Filter(items, Query{Keyword: "greeting", Agent: "claude-code", SourceType: sourcepkg.TypeLocal})
	assert.Len(t, got, 1)
	assert.Eq(t, "hello-skill", got[0].ID)
}

func TestFindByID_ReturnsExactMatch(t *testing.T) {
	items := []skill.Skill{{ID: "hello-skill"}, {ID: "git-only"}}

	got, ok := FindByID(items, "git-only")
	assert.True(t, ok)
	assert.Eq(t, "git-only", got.ID)
}
