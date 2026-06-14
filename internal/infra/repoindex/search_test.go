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

func TestFilter_MatchesIdentitySourceAndCollectionFields(t *testing.T) {
	items := []skill.Skill{{
		ID:                  "go-pro",
		Name:                "Go Pro",
		Collection:          "tools",
		QualifiedName:       "tools/go-pro",
		SourceQualifiedName: "repo-a/tools/go-pro",
		SourceID:            "source-a",
		SourceName:          "workflow-repo",
	}}

	tests := []struct {
		name    string
		keyword string
	}{
		{name: "id", keyword: "go-pro"},
		{name: "collection", keyword: "tools"},
		{name: "qualified name", keyword: "tools/go-pro"},
		{name: "source qualified name", keyword: "repo-a/tools"},
		{name: "source id", keyword: "source-a"},
		{name: "source name", keyword: "workflow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Filter(items, Query{Keyword: tt.keyword})
			if assert.Len(t, got, 1) {
				assert.Eq(t, "go-pro", got[0].ID)
			}
		})
	}
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

func TestResolveSkill_SupportsUniqueQualifiedNameTail(t *testing.T) {
	items := []skill.Skill{
		{ID: "ship-skill", Name: "ship", Collection: "gstack", QualifiedName: "gstack/ship", SourceQualifiedName: "repo-a/gstack/ship"},
	}

	item, err := ResolveSkill(items, "ship")
	assert.NoErr(t, err)
	assert.Eq(t, "ship-skill", item.ID)
}

func TestResolveSkill_PrefersExactIDOverQualifiedNameTail(t *testing.T) {
	items := []skill.Skill{
		{ID: "ship", Name: "ship", QualifiedName: "ship"},
		{ID: "ship-skill", Name: "ship", Collection: "gstack", QualifiedName: "gstack/ship", SourceQualifiedName: "repo-a/gstack/ship"},
	}

	item, err := ResolveSkill(items, "ship")
	assert.NoErr(t, err)
	assert.Eq(t, "ship", item.ID)
}

func TestResolveSkill_RejectsAmbiguousQualifiedNameTail(t *testing.T) {
	items := []skill.Skill{
		{ID: "ship-gstack", Name: "ship", Collection: "gstack", QualifiedName: "gstack/ship", SourceQualifiedName: "repo-a/gstack/ship"},
		{ID: "ship-other", Name: "ship", Collection: "other", QualifiedName: "other/ship", SourceQualifiedName: "repo-b/other/ship"},
	}

	_, err := ResolveSkill(items, "ship")
	assert.Err(t, err)
	assert.Contains(t, err.Error(), "ambiguous skill target")
}
