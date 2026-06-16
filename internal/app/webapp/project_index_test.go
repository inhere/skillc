package webapp

import (
	"path/filepath"
	"strings"
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
				Checksum:            "abc123",
				SourceResolvedRef:   "deadbeef",
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
	assert.Eq(t, "abc123", items[0].Checksum)
	assert.Eq(t, "deadbeef", items[0].SourceResolvedRef)
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

func TestBuildProjectInstallIndexSkipsRecordsWithoutAgents(t *testing.T) {
	projectA := filepath.Clean("/work/project-a")

	t.Run("records without agents do not produce install items", func(t *testing.T) {
		records := lockpkg.File{
			projectA: {
				{
					SkillID:             "go-pro",
					QualifiedName:       "tools/go-pro",
					SourceQualifiedName: "gstack/tools/go-pro",
					SourceID:            "gstack",
					Version:             "1.0.0",
				},
			},
		}

		items := BuildProjectInstallIndex(records)

		assert.Len(t, items, 0)
	})
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

func TestBuildVersionDriftReportsSameVersionChecksumDrift(t *testing.T) {
	items := []ProjectInstall{
		{ProjectPath: "a", SkillID: "rules", SourceID: "local", Version: "1.0.0", Checksum: "oldsum"},
		{ProjectPath: "b", SkillID: "rules", SourceID: "local", Version: "1.0.0", Checksum: "newsum"},
	}
	index := []skill.Skill{{ID: "rules", SourceID: "local", Version: "1.0.0", Checksum: "newsum"}}

	groups := BuildVersionDrift(items, index)

	assert.Len(t, groups, 1)
	assert.Eq(t, "rules", groups[0].SkillID)
	assert.Contains(t, strings.Join(groups[0].DriftReasons, ","), "checksum")
}

func TestBuildVersionDriftReportsSameVersionGitRefDrift(t *testing.T) {
	items := []ProjectInstall{
		{ProjectPath: "a", SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", SourceResolvedRef: "oldcommit"},
		{ProjectPath: "b", SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", SourceResolvedRef: "newcommit"},
	}
	index := []skill.Skill{{ID: "go-pro", SourceID: "gstack", Version: "1.0.0", SourceResolvedRef: "newcommit"}}

	groups := BuildVersionDrift(items, index)

	assert.Len(t, groups, 1)
	assert.Contains(t, strings.Join(groups[0].DriftReasons, ","), "git ref")
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

func TestBuildVersionDriftSortsVersionBucketsNumerically(t *testing.T) {
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
			Version:             "1.9.0",
		},
		{
			ProjectPath:         projectB,
			Scope:               "project",
			Agent:               "codex",
			SkillID:             "go-pro",
			QualifiedName:       "tools/go-pro",
			SourceQualifiedName: "gstack/tools/go-pro",
			SourceID:            "gstack",
			Version:             "1.10.0",
		},
	}

	groups := BuildVersionDrift(items, nil)

	assert.Len(t, groups, 1)
	if len(groups) == 0 {
		return
	}
	assert.Len(t, groups[0].Versions, 2)
	if len(groups[0].Versions) < 2 {
		return
	}
	assert.Eq(t, "1.9.0", groups[0].Versions[0].Version)
	assert.Eq(t, "1.10.0", groups[0].Versions[1].Version)
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

	t.Run("same source id and skill id merge without source-qualified name", func(t *testing.T) {
		items := []ProjectInstall{
			{
				ProjectPath: projectA,
				Scope:       "project",
				Agent:       "universal",
				SkillID:     "shared-skill",
				SourceID:    "repo-a",
				Version:     "1.0.0",
			},
			{
				ProjectPath: projectB,
				Scope:       "project",
				Agent:       "codex",
				SkillID:     "shared-skill",
				SourceID:    "repo-a",
				Version:     "2.0.0",
			},
		}

		groups := BuildVersionDrift(items, nil)

		assert.Len(t, groups, 1)
		if len(groups) == 0 {
			return
		}
		assert.Eq(t, "repo-a", groups[0].SourceID)
		assert.Eq(t, "", groups[0].SourceQualifiedName)
		assert.Len(t, groups[0].Versions, 2)
	})

	t.Run("different source id does not merge when source-qualified name missing", func(t *testing.T) {
		items := []ProjectInstall{
			{
				ProjectPath: projectA,
				Scope:       "project",
				Agent:       "universal",
				SkillID:     "shared-skill",
				SourceID:    "repo-a",
				Version:     "1.0.0",
			},
			{
				ProjectPath: projectB,
				Scope:       "project",
				Agent:       "codex",
				SkillID:     "shared-skill",
				SourceID:    "repo-b",
				Version:     "2.0.0",
			},
		}

		groups := BuildVersionDrift(items, nil)

		assert.Len(t, groups, 0)
	})

	t.Run("qualified name merges when source-qualified name and source id missing", func(t *testing.T) {
		items := []ProjectInstall{
			{
				ProjectPath:   projectA,
				Scope:         "project",
				Agent:         "universal",
				SkillID:       "shared-skill",
				QualifiedName: "alpha/shared-skill",
				Version:       "1.0.0",
			},
			{
				ProjectPath:   projectB,
				Scope:         "project",
				Agent:         "codex",
				SkillID:       "shared-skill",
				QualifiedName: "alpha/shared-skill",
				Version:       "2.0.0",
			},
		}

		groups := BuildVersionDrift(items, nil)

		assert.Len(t, groups, 1)
		if len(groups) == 0 {
			return
		}
		assert.Eq(t, "alpha/shared-skill", groups[0].Versions[0].Projects[0].QualifiedName)
		assert.Len(t, groups[0].Versions, 2)
	})

	t.Run("skill id merges when all other identity fields missing", func(t *testing.T) {
		items := []ProjectInstall{
			{
				ProjectPath: projectA,
				Scope:       "project",
				Agent:       "universal",
				SkillID:     "shared-skill",
				Version:     "1.0.0",
			},
			{
				ProjectPath: projectB,
				Scope:       "project",
				Agent:       "codex",
				SkillID:     "shared-skill",
				Version:     "2.0.0",
			},
		}

		groups := BuildVersionDrift(items, nil)

		assert.Len(t, groups, 1)
		if len(groups) == 0 {
			return
		}
		assert.Eq(t, "shared-skill", groups[0].SkillID)
		assert.Len(t, groups[0].Versions, 2)
	})
}

func TestBuildVersionDriftMergesRenamedSourceQualifiedNameByStableSourceIdentity(t *testing.T) {
	projectA := filepath.Clean("/work/project-a")
	projectB := filepath.Clean("/work/project-b")
	items := []ProjectInstall{
		{
			ProjectPath:         projectA,
			Scope:               "project",
			Agent:               "universal",
			SkillID:             "shared-skill",
			SourceQualifiedName: "repo-a/alpha/shared-skill",
			SourceID:            "repo-a",
			Version:             "1.0.0",
		},
		{
			ProjectPath:         projectB,
			Scope:               "project",
			Agent:               "codex",
			SkillID:             "shared-skill",
			SourceQualifiedName: "repo-a/renamed/shared-skill",
			SourceID:            "repo-a",
			Version:             "2.0.0",
		},
	}
	index := []skill.Skill{
		{
			ID:                  "shared-skill",
			QualifiedName:       "renamed/shared-skill",
			SourceQualifiedName: "repo-a/renamed/shared-skill",
			SourceID:            "repo-a",
			Version:             "3.0.0",
		},
	}

	groups := BuildVersionDrift(items, index)

	assert.Len(t, groups, 1)
	if len(groups) == 0 {
		return
	}
	assert.Eq(t, "repo-a", groups[0].SourceID)
	assert.Eq(t, "repo-a/renamed/shared-skill", groups[0].SourceQualifiedName)
	assert.Eq(t, "3.0.0", groups[0].LatestVersion)
	assert.Len(t, groups[0].Versions, 2)
}

func TestBuildVersionDriftMatchesLatestByQualifiedNameAlias(t *testing.T) {
	projectA := filepath.Clean("/work/project-a")
	items := []ProjectInstall{
		{
			ProjectPath:   projectA,
			Scope:         "project",
			Agent:         "universal",
			SkillID:       "shared-skill",
			QualifiedName: "alpha/shared-skill",
			Version:       "1.0.0",
		},
	}
	index := []skill.Skill{
		{
			ID:                  "shared-skill",
			QualifiedName:       "alpha/shared-skill",
			SourceQualifiedName: "repo-a/alpha/shared-skill",
			SourceID:            "repo-a",
			Version:             "2.0.0",
		},
	}

	groups := BuildVersionDrift(items, index)

	assert.Len(t, groups, 1)
	if len(groups) == 0 {
		return
	}
	assert.Eq(t, "repo-a", groups[0].SourceID)
	assert.Eq(t, "repo-a/alpha/shared-skill", groups[0].SourceQualifiedName)
	assert.Eq(t, "2.0.0", groups[0].LatestVersion)
}

func TestBuildVersionDriftMatchesLatestByUniqueSkillIDAlias(t *testing.T) {
	projectA := filepath.Clean("/work/project-a")
	items := []ProjectInstall{
		{
			ProjectPath: projectA,
			Scope:       "project",
			Agent:       "universal",
			SkillID:     "shared-skill",
			Version:     "1.0.0",
		},
	}
	index := []skill.Skill{
		{
			ID:                  "shared-skill",
			QualifiedName:       "alpha/shared-skill",
			SourceQualifiedName: "repo-a/alpha/shared-skill",
			SourceID:            "repo-a",
			Version:             "2.0.0",
		},
	}

	groups := BuildVersionDrift(items, index)

	assert.Len(t, groups, 1)
	if len(groups) == 0 {
		return
	}
	assert.Eq(t, "repo-a", groups[0].SourceID)
	assert.Eq(t, "repo-a/alpha/shared-skill", groups[0].SourceQualifiedName)
	assert.Eq(t, "2.0.0", groups[0].LatestVersion)
}

func TestBuildVersionDriftLeavesAmbiguousBareSkillIDWithoutLatest(t *testing.T) {
	projectA := filepath.Clean("/work/project-a")
	items := []ProjectInstall{
		{
			ProjectPath: projectA,
			Scope:       "project",
			Agent:       "universal",
			SkillID:     "shared-skill",
			Version:     "1.0.0",
		},
	}
	index := []skill.Skill{
		{
			ID:                  "shared-skill",
			QualifiedName:       "alpha/shared-skill",
			SourceQualifiedName: "repo-a/alpha/shared-skill",
			SourceID:            "repo-a",
			Version:             "2.0.0",
		},
		{
			ID:                  "shared-skill",
			QualifiedName:       "beta/shared-skill",
			SourceQualifiedName: "repo-b/beta/shared-skill",
			SourceID:            "repo-b",
			Version:             "3.0.0",
		},
	}

	groups := BuildVersionDrift(items, index)

	assert.Len(t, groups, 0)
}

func TestBuildVersionDriftDoesNotMergeAmbiguousSourceIDSkillIDAcrossCollections(t *testing.T) {
	projectA := filepath.Clean("/work/project-a")
	projectB := filepath.Clean("/work/project-b")
	items := []ProjectInstall{
		{
			ProjectPath:         projectA,
			Scope:               "project",
			Agent:               "universal",
			SkillID:             "ship",
			QualifiedName:       "alpha/ship",
			SourceQualifiedName: "repo-a/alpha/ship",
			SourceID:            "repo-a",
			Version:             "1.0.0",
		},
		{
			ProjectPath:         projectB,
			Scope:               "project",
			Agent:               "codex",
			SkillID:             "ship",
			QualifiedName:       "beta/ship",
			SourceQualifiedName: "repo-a/beta/ship",
			SourceID:            "repo-a",
			Version:             "2.0.0",
		},
	}
	index := []skill.Skill{
		{
			ID:                  "ship",
			QualifiedName:       "alpha/ship",
			SourceQualifiedName: "repo-a/alpha/ship",
			SourceID:            "repo-a",
			Version:             "1.0.0",
		},
		{
			ID:                  "ship",
			QualifiedName:       "beta/ship",
			SourceQualifiedName: "repo-a/beta/ship",
			SourceID:            "repo-a",
			Version:             "2.0.0",
		},
	}

	groups := BuildVersionDrift(items, index)

	assert.Len(t, groups, 0)
}
