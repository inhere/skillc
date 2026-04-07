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
