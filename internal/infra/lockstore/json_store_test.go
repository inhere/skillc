package lockstore

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
)

func TestStore_SaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skillc-install.lock")
	store := NewStore()
	now := time.Unix(1710000000, 0).UTC()
	projectPath := filepath.Join("/tmp", "project-a")
	want := lockpkg.File{
		lockpkg.GlobalKey: {
			{
				SkillID:             "hello-skill",
				QualifiedName:       "marketplaces/hello-skill",
				SourceQualifiedName: "workflow-repo/marketplaces/hello-skill",
				Version:             "1.0.0",
				SourceID:            "local-demo",
				SourceType:          "local",
				InstallEntry:        "commands",
				Agents:              []string{"claude-code", "codex"},
				Checksum:            "abc123",
				InstalledAt:         now,
				UpdatedAt:           now,
				Pinned:              true,
			},
		},
		projectPath: {
			{
				SkillID:             "project-skill",
				QualifiedName:       "team/project-skill",
				SourceQualifiedName: "workflow-repo/team/project-skill",
				Version:             "2.0.0",
				SourceID:            "remote-demo",
				SourceType:          "git",
				InstallEntry:        "commands/project",
				Agents:              []string{"claude-code"},
				Checksum:            "def456",
				InstalledAt:         now,
				UpdatedAt:           now,
				Pinned:              false,
			},
		},
	}

	assert.NoErr(t, store.Save(path, want))

	got, err := store.Load(path)
	assert.NoErr(t, err)
	assert.Eq(t, want, got)
}

func TestStore_NormalizesAgentNamesOnLoadAndSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skillc-install.lock")
	store := NewStore().WithAgentResolver(func(name string) string {
		switch name {
		case "claude", "claude-code":
			return "claude-code"
		case "agents", "universal":
			return "universal"
		}
		return name
	})
	projectPath := filepath.Join("/tmp", "project-a")

	assert.NoErr(t, store.Save(path, lockpkg.File{
		projectPath: {
			{
				SkillID: "hello-skill",
				Agents:  []string{"claude", "claude-code", "agents"},
			},
		},
	}))

	loaded, err := store.Load(path)
	assert.NoErr(t, err)
	assert.Eq(t, []string{"claude-code", "universal"}, loaded[projectPath][0].Agents)

	// 未注入 resolver 时保持原样，避免影响通用存储语义
	raw, err := NewStore().Load(path)
	assert.NoErr(t, err)
	assert.Eq(t, []string{"claude-code", "universal"}, raw[projectPath][0].Agents)
}
