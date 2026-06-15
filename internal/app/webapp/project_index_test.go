package webapp

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
)

func TestBuildProjectInstallIndexExpandsAgentsAndProfiles(t *testing.T) {
	projectA := filepath.Clean("/work/project-a")
	records := lockpkg.File{
		projectA: {
			{
				SkillID:             "go-pro",
				QualifiedName:       "tools/go-pro",
				SourceQualifiedName: "gstack/tools/go-pro",
				SourceID:            "gstack",
				Version:             "1.0.0",
				Profile:             "go-dev",
				Agents:              []string{"universal", "codex"},
			},
		},
	}

	items := BuildProjectInstallIndex(records)

	assert.Len(t, items, 2)
	assert.Eq(t, projectA, items[0].ProjectPath)
	assert.Eq(t, "project", items[0].Scope)
	assert.Eq(t, "codex", items[0].Agent)
	assert.Eq(t, "go-dev", items[0].Profile)
	assert.Eq(t, "gstack/tools/go-pro", items[0].SourceQualifiedName)
	assert.Eq(t, "universal", items[1].Agent)
}

func TestBuildProjectInstallIndexKeepsGlobalScopeSeparate(t *testing.T) {
	records := lockpkg.File{
		lockpkg.GlobalKey: {
			{
				SkillID:  "review",
				SourceID: "gstack",
				Version:  "1.0.0",
				Agents:   []string{"universal"},
			},
		},
	}

	items := BuildProjectInstallIndex(records)

	assert.Len(t, items, 1)
	assert.Eq(t, lockpkg.GlobalKey, items[0].ProjectPath)
	assert.Eq(t, "global", items[0].Scope)
}

func TestBuildVersionDriftGroupsBySourceQualifiedIdentity(t *testing.T) {
	projectA := filepath.Clean("/work/project-a")
	projectB := filepath.Clean("/work/project-b")
	items := []ProjectInstall{
		{
			ProjectPath:         projectA,
			Scope:               "project",
			Agent:               "universal",
			SkillID:             "go-pro",
			QualifiedName:       "tools/go-pro",
			SourceQualifiedName: "gstack/tools/go-pro",
			SourceID:            "gstack",
			Version:             "1.0.0",
		},
		{
			ProjectPath:         projectB,
			Scope:               "project",
			Agent:               "codex",
			SkillID:             "go-pro",
			QualifiedName:       "tools/go-pro",
			SourceQualifiedName: "gstack/tools/go-pro",
			SourceID:            "gstack",
			Version:             "2.0.0",
		},
	}
	index := []skill.Skill{
		{
			ID:                  "go-pro",
			QualifiedName:       "tools/go-pro",
			SourceQualifiedName: "gstack/tools/go-pro",
			SourceID:            "gstack",
			Version:             "2.0.0",
		},
	}

	groups := BuildVersionDrift(items, index)

	assert.Len(t, groups, 1)
	assert.Eq(t, "go-pro", groups[0].SkillID)
	assert.Eq(t, "gstack/tools/go-pro", groups[0].SourceQualifiedName)
	assert.Eq(t, "2.0.0", groups[0].LatestVersion)
	assert.Len(t, groups[0].Versions, 2)
	assert.Eq(t, "1.0.0", groups[0].Versions[0].Version)
	assert.Eq(t, "2.0.0", groups[0].Versions[1].Version)
}

func TestBuildVersionDriftChoosesLatestVersionNumerically(t *testing.T) {
	projectA := filepath.Clean("/work/project-a")
	items := []ProjectInstall{
		{
			ProjectPath:         projectA,
			Scope:               "project",
			Agent:               "universal",
			SkillID:             "go-pro",
			QualifiedName:       "tools/go-pro",
			SourceQualifiedName: "gstack/tools/go-pro",
			SourceID:            "gstack",
			Version:             "1.9.0",
		},
	}
	index := []skill.Skill{
		{
			ID:                  "go-pro",
			QualifiedName:       "tools/go-pro",
			SourceQualifiedName: "gstack/tools/go-pro",
			SourceID:            "gstack",
			Version:             "1.9.0",
		},
		{
			ID:                  "go-pro",
			QualifiedName:       "tools/go-pro",
			SourceQualifiedName: "gstack/tools/go-pro",
			SourceID:            "gstack",
			Version:             "1.10.0",
		},
	}

	groups := BuildVersionDrift(items, index)

	assert.Len(t, groups, 1)
	if len(groups) == 0 {
		return
	}
	assert.Eq(t, "1.10.0", groups[0].LatestVersion)
}

func TestBuildVersionDriftDoesNotMergeUnrelatedSourcesWithSameSkillID(t *testing.T) {
	projectA := filepath.Clean("/work/project-a")
	projectB := filepath.Clean("/work/project-b")

	t.Run("source-qualified identities stay separate", func(t *testing.T) {
		items := []ProjectInstall{
			{
				ProjectPath:         projectA,
				Scope:               "project",
				Agent:               "universal",
				SkillID:             "shared-skill",
				QualifiedName:       "alpha/shared-skill",
				SourceQualifiedName: "repo-a/alpha/shared-skill",
				SourceID:            "repo-a",
				Version:             "1.0.0",
			},
			{
				ProjectPath:         projectB,
				Scope:               "project",
				Agent:               "universal",
				SkillID:             "shared-skill",
				QualifiedName:       "beta/shared-skill",
				SourceQualifiedName: "repo-b/beta/shared-skill",
				SourceID:            "repo-b",
				Version:             "2.0.0",
			},
		}

		groups := BuildVersionDrift(items, nil)

		assert.Len(t, groups, 0)
	})
}
