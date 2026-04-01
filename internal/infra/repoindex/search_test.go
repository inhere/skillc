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

func TestResolveSkills_SupportsCollectionTargetsAndDisambiguation(t *testing.T) {
	items := []skill.Skill{
		{ID: "hello-skill", Collection: "marketplaces", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill"},
		{ID: "world-skill", Collection: "marketplaces", QualifiedName: "marketplaces/world-skill", SourceQualifiedName: "repo-a/marketplaces/world-skill"},
		{ID: "hello-skill", Collection: "marketplaces", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-b/marketplaces/hello-skill"},
	}

	matches, err := ResolveSkills(items, "repo-a/marketplaces")
	assert.NoErr(t, err)
	assert.Len(t, matches, 2)

	_, err = ResolveSkills(items, "marketplaces")
	assert.Err(t, err)
	assert.Contains(t, err.Error(), "ambiguous collection target")
}
