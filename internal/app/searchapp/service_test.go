package searchapp

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

func TestService_SearchReturnsEmptyWhenIndexMissing(t *testing.T) {
	service := NewService(filepath.Join(t.TempDir(), "missing.json"))

	results, err := service.Search("design", "", "")
	assert.NoErr(t, err)
	assert.Len(t, results, 0)
}

func TestService_SearchAndShow(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{
			ID:              "hello-skill",
			Name:            "Hello Skill",
			Description:     "Friendly greeting helper",
			QualifiedName:   "marketplaces/hello-skill",
			SupportedAgents: []string{"claude-code"},
			SourceType:      sourcepkg.TypeLocal,
		},
		{
			ID:                  "git-only",
			Name:                "Git Only",
			Description:         "Remote repo helper",
			QualifiedName:       "git-only",
			SourceQualifiedName: "repo-a/git-only",
			SupportedAgents:     []string{"codex"},
			SourceType:          sourcepkg.TypeGit,
		},
	}))

	service := NewService(indexPath)

	results, err := service.Search("greeting", "claude-code", sourcepkg.TypeLocal)
	assert.NoErr(t, err)
	assert.Len(t, results, 1)
	assert.Eq(t, "marketplaces/hello-skill", results[0].QualifiedName)

	item, err := service.Show("git-only")
	assert.NoErr(t, err)
	assert.Eq(t, "git-only", item.ID)
}

func TestService_ResolveSupportsSourceCollectionTarget(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "hello-skill", Collection: "marketplaces", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill"},
		{ID: "world-skill", Collection: "marketplaces", QualifiedName: "marketplaces/world-skill", SourceQualifiedName: "repo-a/marketplaces/world-skill"},
	}))

	service := NewService(indexPath)
	items, err := service.Resolve("repo-a/marketplaces")
	assert.NoErr(t, err)
	assert.Len(t, items, 2)
}

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
