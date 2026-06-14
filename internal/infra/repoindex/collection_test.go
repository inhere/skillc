package repoindex

import (
	"reflect"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/skill"
)

func TestListCollections_GroupsByCollectionAndCountsUniqueSources(t *testing.T) {
	items := []skill.Skill{
		{ID: "alpha-2", Collection: "alpha", SourceName: "repo-a", SourceID: "source-a"},
		{ID: "alpha-1", Collection: "alpha", SourceName: "repo-a", SourceID: "source-a"},
		{ID: "alpha-3", Collection: "alpha", SourceID: "source-b"},
		{ID: "beta-1", Collection: "beta", SourceID: "source-b"},
		{ID: "ignored", SourceName: "repo-c", SourceID: "source-c"},
	}

	got := ListCollections(items)
	want := []CollectionSummary{
		{Name: "alpha", SkillCount: 3, SourceCount: 2},
		{Name: "beta", SkillCount: 1, SourceCount: 1},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListCollections() = %#v, want %#v", got, want)
	}
}

func TestListCollectionSkills_ReturnsSortedSkillsForCollection(t *testing.T) {
	items := []skill.Skill{
		{ID: "zeta", Name: "Zulu", Collection: "alpha"},
		{ID: "alpha", Name: "Alpha", Collection: "alpha"},
		{ID: "bravo-2", Name: "Bravo", Collection: "alpha"},
		{ID: "bravo-1", Name: "Bravo", Collection: "alpha"},
		{ID: "other", Name: "Alpha", Collection: "beta"},
	}

	got, err := ListCollectionSkills(items, "alpha")
	if err != nil {
		t.Fatalf("ListCollectionSkills() error = %v", err)
	}

	want := []skill.Skill{
		{ID: "alpha", Name: "Alpha", Collection: "alpha"},
		{ID: "bravo-1", Name: "Bravo", Collection: "alpha"},
		{ID: "bravo-2", Name: "Bravo", Collection: "alpha"},
		{ID: "zeta", Name: "Zulu", Collection: "alpha"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListCollectionSkills() = %#v, want %#v", got, want)
	}
}

func TestListCollectionSkills_ReturnsErrorWhenCollectionMissing(t *testing.T) {
	_, err := ListCollectionSkills([]skill.Skill{{ID: "alpha", Collection: "alpha"}}, "missing")
	if err == nil {
		t.Fatal("ListCollectionSkills() error = nil, want error")
	}
	if err.Error() != "collection not found: missing" {
		t.Fatalf("ListCollectionSkills() error = %q, want %q", err.Error(), "collection not found: missing")
	}
}

func TestListSourceCollectionsFiltersBySource(t *testing.T) {
	items := []skill.Skill{
		{ID: "go-pro", Collection: "go", SourceID: "gstack", SourceName: "GStack"},
		{ID: "go-test", Collection: "go", SourceID: "gstack", SourceName: "GStack"},
		{ID: "py-pro", Collection: "python", SourceID: "team", SourceName: "Team"},
	}

	got := ListSourceCollections(items, "gstack")

	assert.Len(t, got, 1)
	assert.Eq(t, "go", got[0].Name)
	assert.Eq(t, "gstack", got[0].SourceID)
	assert.Eq(t, 2, got[0].SkillCount)
}

func TestListSourceSkillsFiltersByCollection(t *testing.T) {
	items := []skill.Skill{
		{ID: "go-pro", Name: "Go Pro", Collection: "go", SourceID: "gstack"},
		{ID: "review", Name: "Review", Collection: "ops", SourceID: "gstack"},
		{ID: "other", Name: "Other", Collection: "go", SourceID: "team"},
	}

	got, err := ListSourceSkills(items, "gstack", "go")

	assert.NoErr(t, err)
	assert.Len(t, got, 1)
	assert.Eq(t, "go-pro", got[0].ID)
}

func TestListSourceSkillsSortsByCollectionNameAndID(t *testing.T) {
	items := []skill.Skill{
		{ID: "a-zulu", Name: "Zulu", Collection: "go", SourceID: "gstack"},
		{ID: "z-alpha", Name: "Alpha", Collection: "go", SourceID: "gstack"},
		{ID: "bravo-2", Name: "Bravo", Collection: "go", SourceID: "gstack"},
		{ID: "bravo-1", Name: "Bravo", Collection: "go", SourceID: "gstack"},
		{ID: "ops", Name: "Ops", Collection: "ops", SourceID: "gstack"},
	}

	got, err := ListSourceSkills(items, "gstack", "")

	assert.NoErr(t, err)
	assert.Len(t, got, 5)
	assert.Eq(t, "z-alpha", got[0].ID)
	assert.Eq(t, "bravo-1", got[1].ID)
	assert.Eq(t, "bravo-2", got[2].ID)
	assert.Eq(t, "a-zulu", got[3].ID)
	assert.Eq(t, "ops", got[4].ID)
}
