package listapp

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/config"
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

func TestService_ListReturnsStatErrors(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	conf := config.DefaultConfig()
	conf.AgentTools["claude-code"] = config.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: "./project\x00dir"}
	now := time.Unix(1710000000, 0).UTC()

	store := lockstore.NewStore()
	assert.NoErr(t, store.Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{
				SkillID:             "hello-skill",
				QualifiedName:       "marketplaces/hello-skill",
				SourceQualifiedName: "repo-a/marketplaces/hello-skill",
				Version:             "1.0.0",
				SourceID:            "local-demo",
				SourceType:          "local",
				Agents:              []string{"claude-code"},
				UpdatedAt:           now,
			},
		},
	}))

	_, err := NewService(lockFile).WithRuntime(conf, baseDir).List("claude-code", "project")
	assert.Err(t, err)
	assert.False(t, os.IsNotExist(err))
}

func TestService_ListTreatsLegacySourceScopedDirAsInstalled(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	conf := config.DefaultConfig()
	conf.AgentTools["claude-code"] = config.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: "./project-claude"}
	legacyDir := filepath.Join(baseDir, "project-claude", "skills", "repo-a--marketplaces--hello-skill")
	now := time.Unix(1710000000, 0).UTC()

	assert.NoErr(t, os.MkdirAll(legacyDir, 0o755))
	store := lockstore.NewStore()
	assert.NoErr(t, store.Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{
				SkillID:             "hello-skill",
				QualifiedName:       "marketplaces/hello-skill",
				SourceQualifiedName: "repo-a/marketplaces/hello-skill",
				Version:             "1.0.0",
				SourceID:            "local-demo",
				SourceType:          "local",
				Agents:              []string{"claude-code"},
				UpdatedAt:           now,
			},
		},
	}))

	items, err := NewService(lockFile).WithRuntime(conf, baseDir).List("claude-code", "project")
	assert.NoErr(t, err)
	assert.Len(t, items, 1)
	assert.Eq(t, legacyDir, items[0].InstalledPath)
	assert.Eq(t, "installed", items[0].Status)
}

