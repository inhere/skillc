package listapp

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/infra/lockstore"
)

func TestService_ListReturnsEmptyWhenLockFileMissing(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")

	service := NewService(lockFile)
	items, err := service.List("claude-code", "project")
	assert.NoErr(t, err)
	assert.Len(t, items, 0)
}

func TestService_ListProjectsInstalledAndMissingStatus(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	installedPath := filepath.Join(baseDir, ".claude", "skills", "hello-skill")
	missingPath := filepath.Join(baseDir, ".claude", "skills", "gone-skill")
	now := time.Unix(1710000000, 0).UTC()

	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))
	store := lockstore.NewStore()
	assert.NoErr(t, store.Save(lockFile, []lockpkg.Record{
		{
			SkillID:             "hello-skill",
			QualifiedName:       "marketplaces/hello-skill",
			SourceQualifiedName: "repo-a/marketplaces/hello-skill",
			Agent:               "claude-code",
			Scope:               "project",
			Version:             "1.0.0",
			SourceID:            "local-demo",
			SourceType:          "local",
			InstalledPath:       installedPath,
			Checksum:            "abc123",
			UpdatedAt:           now,
		},
		{
			SkillID:             "gone-skill",
			QualifiedName:       "gone-skill",
			SourceQualifiedName: "repo-a/gone-skill",
			Agent:               "claude-code",
			Scope:               "project",
			Version:             "1.1.0",
			SourceID:            "local-demo",
			SourceType:          "local",
			InstalledPath:       missingPath,
			Checksum:            "def456",
			UpdatedAt:           now,
		},
	}))

	service := NewService(lockFile)
	items, err := service.List("claude-code", "project")
	assert.NoErr(t, err)
	assert.Len(t, items, 2)
	assert.Eq(t, "gone-skill", items[0].QualifiedName)
	assert.Eq(t, "missing", items[0].Status)
	assert.Eq(t, "marketplaces/hello-skill", items[1].QualifiedName)
	assert.Eq(t, "installed", items[1].Status)
}
