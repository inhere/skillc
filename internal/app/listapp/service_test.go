package listapp

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/agent"
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

func TestService_ListIncludesProfileName(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	conf := config.DefaultConfig()
	conf.AgentTools["claude-code"] = config.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, ".claude")}
	installedPath := filepath.Join(baseDir, ".claude", "go-pro")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))

	store := lockstore.NewStore()
	assert.NoErr(t, store.Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{
				SkillID:  "go-pro",
				SourceID: "gstack",
				Agents:   []string{"claude-code"},
				Profile:  "go-dev",
			},
		},
	}))

	items, err := NewService(lockFile).WithRuntime(conf, baseDir).List("claude-code", "project")

	assert.NoErr(t, err)
	assert.Len(t, items, 1)
	assert.Eq(t, "go-dev", items[0].Profile)
}

func TestService_ListCarriesDriftMetadata(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	conf := config.DefaultConfig()
	conf.AgentTools["universal"] = config.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	installedPath := filepath.Join(baseDir, ".agents", "skills", "go-pro")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))

	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID: "go-pro", SourceID: "gstack", Agents: []string{"universal"},
			Checksum: "abc123", SourceResolvedRef: "deadbeefcafebabe",
		}},
	}))

	items, err := NewService(lockFile).WithRuntime(conf, baseDir).List("universal", "project")

	assert.NoErr(t, err)
	assert.Len(t, items, 1)
	assert.Eq(t, "abc123", items[0].Checksum)
	assert.Eq(t, "deadbeefcafebabe", items[0].SourceResolvedRef)
}

func TestService_ScanUnrecordedFiltersByRequestedAgent(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	conf := config.DefaultConfig()
	conf.AgentTools["claude-code"] = config.AgentToolConfig{Dirname: ".claude", ProjectDir: filepath.Join(baseDir, ".claude")}
	conf.AgentTools["codex"] = config.AgentToolConfig{Dirname: ".codex", ProjectDir: filepath.Join(baseDir, ".codex")}
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".claude", "skills", "claude-only"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".codex", "skills", "codex-only"), 0o755))

	groups, err := NewService(lockFile).WithRuntime(conf, baseDir).ScanUnrecorded("claude-code", agent.ScopeProject)

	assert.NoErr(t, err)
	assert.Len(t, groups, 1)
	assert.Eq(t, "claude-code", groups[0].AgentName)
	assert.Eq(t, []string{"claude-only"}, groups[0].Skills)
}
