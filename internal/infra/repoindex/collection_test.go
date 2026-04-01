package repoindex

import (
	"reflect"
	"testing"

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
