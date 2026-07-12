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

func TestService_ResolveInstallTargetsMixedTargets(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "hello-skill", Collection: "marketplaces", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill"},
		{ID: "world-skill", Collection: "marketplaces", QualifiedName: "marketplaces/world-skill", SourceQualifiedName: "repo-a/marketplaces/world-skill"},
	}))

	service := NewService(indexPath)
	result, err := service.ResolveInstallTargets([]string{"hello-skill", "world-*", "missing-skill"}, false)
	assert.NoErr(t, err)
	assert.Len(t, result.Resolved, 2)
	assert.Eq(t, "hello-skill", result.Resolved[0].ID)
	assert.Eq(t, "world-skill", result.Resolved[1].ID)
	assert.Len(t, result.Failed, 1)
	assert.Eq(t, "missing-skill", result.Failed[0].Target)
	assert.Contains(t, result.Failed[0].Reason, "skill not found")
}

func TestService_ResolveInstallTargetsCollectionModeExpandsCollections(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "hello-skill", Collection: "marketplaces", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill"},
		{ID: "world-skill", Collection: "marketplaces", QualifiedName: "marketplaces/world-skill", SourceQualifiedName: "repo-a/marketplaces/world-skill"},
	}))

	service := NewService(indexPath)
	result, err := service.ResolveInstallTargets([]string{"repo-a/marketplaces", "repo-a/missing"}, true)
	assert.NoErr(t, err)
	assert.Len(t, result.Resolved, 2)
	assert.Eq(t, "hello-skill", result.Resolved[0].ID)
	assert.Eq(t, "world-skill", result.Resolved[1].ID)
	assert.Len(t, result.Failed, 1)
	assert.Eq(t, "repo-a/missing", result.Failed[0].Target)
	assert.Contains(t, result.Failed[0].Reason, "not found")
}

func TestService_ResolveInstallTargetsDoesNotAutoExpandCollectionWithoutFlag(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "hello-skill", Collection: "marketplaces", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill"},
		{ID: "world-skill", Collection: "marketplaces", QualifiedName: "marketplaces/world-skill", SourceQualifiedName: "repo-a/marketplaces/world-skill"},
	}))

	service := NewService(indexPath)
	result, err := service.ResolveInstallTargets([]string{"repo-a/marketplaces"}, false)
	assert.NoErr(t, err)
	assert.Len(t, result.Resolved, 0)
	assert.Len(t, result.Failed, 1)
	assert.Eq(t, "repo-a/marketplaces", result.Failed[0].Target)
	assert.Contains(t, result.Failed[0].Reason, "not found")
}

func TestService_ResolveInstallTargetsDeduplicatesRepeatedMatches(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "hello-skill", Collection: "marketplaces", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill"},
	}))

	service := NewService(indexPath)
	result, err := service.ResolveInstallTargets([]string{"hello-skill", "hello-*"}, false)
	assert.NoErr(t, err)
	assert.Len(t, result.Resolved, 1)
	assert.Eq(t, "hello-skill", result.Resolved[0].ID)
	assert.Len(t, result.Failed, 0)
}

func TestService_ResolveInstallTargetsDeduplicatesWithoutSourceQualifiedName(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "hello-skill", QualifiedName: "marketplaces/hello-skill"},
		{ID: "plain-skill"},
	}))

	service := NewService(indexPath)
	qualified, err := service.ResolveInstallTargets([]string{"hello-skill", "hello-*"}, false)
	assert.NoErr(t, err)
	assert.Len(t, qualified.Resolved, 1)
	assert.Eq(t, "hello-skill", qualified.Resolved[0].ID)

	plain, err := service.ResolveInstallTargets([]string{"plain-skill", "plain-*"}, false)
	assert.NoErr(t, err)
	assert.Len(t, plain.Resolved, 1)
	assert.Eq(t, "plain-skill", plain.Resolved[0].ID)
}

func TestService_ResolveInstallTargetsDeduplicatesWithSourceQualifiedNamePrecedence(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "ship", Collection: "shared", QualifiedName: "shared/ship", SourceQualifiedName: "repo-a/shared/ship"},
		{ID: "ship", Collection: "shared", QualifiedName: "shared/ship", SourceQualifiedName: "repo-b/shared/ship"},
	}))

	service := NewService(indexPath)
	result, err := service.ResolveInstallTargets([]string{"shared"}, true)
	assert.NoErr(t, err)
	assert.Len(t, result.Resolved, 2)
	assert.Eq(t, "repo-a/shared/ship", result.Resolved[0].SourceQualifiedName)
	assert.Eq(t, "repo-b/shared/ship", result.Resolved[1].SourceQualifiedName)
	assert.Len(t, result.Failed, 0)
}

func TestService_ResolveInstallTargetsCollectionModeRejectsPlainSkillTarget(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "hello-skill", Collection: "marketplaces", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill"},
	}))

	service := NewService(indexPath)
	result, err := service.ResolveInstallTargets([]string{"hello-skill"}, true)
	assert.NoErr(t, err)
	assert.Len(t, result.Resolved, 0)
	assert.Len(t, result.Failed, 1)
	assert.Eq(t, "hello-skill", result.Failed[0].Target)
}

func TestService_ResolveInstallTargetsDoesNotAutoExpandSingleSkillCollectionWithoutFlag(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "solo-skill", Collection: "solo", QualifiedName: "solo/solo-skill", SourceQualifiedName: "repo-a/solo/solo-skill"},
	}))

	service := NewService(indexPath)
	result, err := service.ResolveInstallTargets([]string{"repo-a/solo"}, false)
	assert.NoErr(t, err)
	assert.Len(t, result.Resolved, 0)
	assert.Len(t, result.Failed, 1)
	assert.Eq(t, "repo-a/solo", result.Failed[0].Target)
}

func TestService_ResolveInstallTargetsPrefixMatchesOnlySkillID(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "ship", Collection: "hello-tools", QualifiedName: "hello-tools/ship", SourceQualifiedName: "repo-a/hello-tools/ship"},
	}))

	service := NewService(indexPath)
	result, err := service.ResolveInstallTargets([]string{"hello-*"}, false)
	assert.NoErr(t, err)
	assert.Len(t, result.Resolved, 0)
	assert.Len(t, result.Failed, 1)
	assert.Contains(t, result.Failed[0].Reason, "skill not found")
}

func TestService_ResolveInstallTargetsGlobPatterns(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "flutter-core", QualifiedName: "mobile/flutter-core", SourceQualifiedName: "repo-a/mobile/flutter-core", SourceID: "repo-a"},
		{ID: "flutter-x-pro", QualifiedName: "mobile/flutter-x-pro", SourceQualifiedName: "repo-a/mobile/flutter-x-pro", SourceID: "repo-a"},
		{ID: "dart-testing", QualifiedName: "mobile/dart-testing", SourceQualifiedName: "repo-a/mobile/dart-testing", SourceID: "repo-a"},
		{ID: "review", QualifiedName: "testing/review", SourceQualifiedName: "superpowers/testing/review", SourceID: "superpowers"},
		{ID: "brainstorming", QualifiedName: "core/brainstorming", SourceQualifiedName: "superpowers/core/brainstorming", SourceID: "superpowers"},
	}))
	service := NewService(indexPath)

	tests := []struct {
		name     string
		targets  []string
		expected []string
	}{
		{name: "id prefix", targets: []string{"flutter-*"}, expected: []string{"flutter-core", "flutter-x-pro"}},
		{name: "id suffix", targets: []string{"*-testing"}, expected: []string{"dart-testing"}},
		{name: "question", targets: []string{"flutter-?-pro"}, expected: []string{"flutter-x-pro"}},
		{name: "character range", targets: []string{"flutter-[a-z]*"}, expected: []string{"flutter-core", "flutter-x-pro"}},
		{name: "qualified name", targets: []string{"mobile/flutter-*"}, expected: []string{"flutter-core", "flutter-x-pro"}},
		{name: "source qualified name", targets: []string{"repo-a/mobile/*"}, expected: []string{"flutter-core", "flutter-x-pro", "dart-testing"}},
		{name: "all source skills", targets: []string{"superpowers/*"}, expected: []string{"review", "brainstorming"}},
		{name: "deduplicate patterns", targets: []string{"flutter-*", "mobile/flutter-core"}, expected: []string{"flutter-core", "flutter-x-pro"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ResolveInstallTargets(tt.targets, false)
			assert.NoErr(t, err)
			assert.Len(t, result.Failed, 0)
			ids := make([]string, 0, len(result.Resolved))
			for _, item := range result.Resolved {
				ids = append(ids, item.ID)
			}
			assert.Eq(t, tt.expected, ids)
		})
	}

	t.Run("invalid pattern", func(t *testing.T) {
		result, err := service.ResolveInstallTargets([]string{"flutter-["}, false)
		assert.NoErr(t, err)
		assert.Len(t, result.Resolved, 0)
		assert.Len(t, result.Failed, 1)
		assert.Contains(t, result.Failed[0].Reason, "glob")
	})
}

func TestService_SearchInstallCandidatesGlob(t *testing.T) {
	indexPath := filepath.Join(t.TempDir(), "index.json")
	assert.NoErr(t, repoindex.NewStore().Save(indexPath, []skill.Skill{
		{ID: "flutter-core", SourceID: "repo-a", QualifiedName: "mobile/flutter-core", SourceQualifiedName: "repo-a/mobile/flutter-core", SupportedAgents: []string{"universal"}},
		{ID: "flutter-testing", SourceID: "repo-a", QualifiedName: "mobile/flutter-testing", SourceQualifiedName: "repo-a/mobile/flutter-testing", SupportedAgents: []string{"codex"}},
		{ID: "review", SourceID: "superpowers", QualifiedName: "testing/review", SourceQualifiedName: "superpowers/testing/review", SupportedAgents: []string{"universal"}},
		{ID: "brainstorming", SourceID: "superpowers", QualifiedName: "core/brainstorming", SourceQualifiedName: "superpowers/core/brainstorming", SupportedAgents: []string{"codex"}},
	}))
	service := NewService(indexPath)

	t.Run("source and agent filter", func(t *testing.T) {
		items, err := service.SearchInstallCandidates([]string{"superpowers/*"}, "universal")
		assert.NoErr(t, err)
		assert.Len(t, items, 1)
		assert.Eq(t, "review", items[0].ID)
	})

	t.Run("multiple targets deduplicate", func(t *testing.T) {
		items, err := service.SearchInstallCandidates([]string{"flutter-*", "*-core"}, "")
		assert.NoErr(t, err)
		assert.Len(t, items, 2)
		assert.Eq(t, "flutter-core", items[0].ID)
		assert.Eq(t, "flutter-testing", items[1].ID)
	})

	t.Run("empty targets list all", func(t *testing.T) {
		items, err := service.SearchInstallCandidates(nil, "")
		assert.NoErr(t, err)
		assert.Len(t, items, 4)
	})

	for _, pattern := range []string{"*", "flutter-["} {
		t.Run("reject "+pattern, func(t *testing.T) {
			items, err := service.SearchInstallCandidates([]string{pattern}, "")
			assert.Err(t, err)
			assert.Len(t, items, 0)
		})
	}
}

func TestService_ResolveInstallTargetsRejectsBareWildcard(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "hello-skill", QualifiedName: "marketplaces/hello-skill"},
	}))

	service := NewService(indexPath)
	result, err := service.ResolveInstallTargets([]string{"*"}, false)
	assert.NoErr(t, err)
	assert.Len(t, result.Resolved, 0)
	assert.Len(t, result.Failed, 1)
	assert.Contains(t, result.Failed[0].Reason, "bare wildcard")
}

func TestService_ResolveInstallTargetsFailsAmbiguousPlainTarget(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "shared-skill", QualifiedName: "alpha/shared-skill", SourceQualifiedName: "repo-a/alpha/shared-skill"},
		{ID: "shared-skill", QualifiedName: "beta/shared-skill", SourceQualifiedName: "repo-b/beta/shared-skill"},
	}))

	service := NewService(indexPath)
	result, err := service.ResolveInstallTargets([]string{"shared-skill"}, false)
	assert.NoErr(t, err)
	assert.Len(t, result.Resolved, 0)
	assert.Len(t, result.Failed, 1)
	assert.Contains(t, result.Failed[0].Reason, "ambiguous skill target")
}

func TestService_ResolveInstallTargetsReturnsEmptyWhenIndexMissing(t *testing.T) {
	service := NewService(filepath.Join(t.TempDir(), "missing.json"))

	result, err := service.ResolveInstallTargets([]string{"hello-skill"}, false)
	assert.NoErr(t, err)
	assert.Len(t, result.Resolved, 0)
	assert.Len(t, result.Failed, 0)
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

func TestService_ListSourceCollectionsAndSkills(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{ID: "go-pro", Name: "Go Pro", Description: "go helper", Collection: "go", SourceID: "gstack", SourceName: "GStack"},
		{ID: "go-test", Name: "Go Test", Description: "go test helper", Collection: "go", SourceID: "gstack", SourceName: "GStack"},
		{ID: "review", Name: "Review", Description: "review helper", Collection: "ops", SourceID: "team", SourceName: "Team"},
	}))

	service := NewService(indexPath)

	collections, err := service.ListSourceCollections("gstack")
	assert.NoErr(t, err)
	assert.Len(t, collections, 1)
	assert.Eq(t, "go", collections[0].Name)
	assert.Eq(t, 2, collections[0].SkillCount)

	skills, err := service.ListSourceSkills("gstack", "go")
	assert.NoErr(t, err)
	assert.Len(t, skills, 2)
	assert.Eq(t, "go-pro", skills[0].ID)
	assert.Eq(t, "go-test", skills[1].ID)
}
