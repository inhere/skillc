package registryapp

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/registry"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/registrystore"
)

func TestLockedResolver_LatestReadsRegistryCacheWithoutMaterializing(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	cacheDir := filepath.Join(baseDir, "registry-cache")
	config := cfg.DefaultConfig()
	config.RegistryCacheDir = cacheDir
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, registrystore.NewStore().SaveFile(filepath.Join(cacheDir, "registry-index.json"), registrystore.File{
		Skills: []registry.SkillEntry{{
			ID: "go-pro", RegistryID: "team", Version: "2.0.0", Checksum: "sha256:abc123",
			SourceURL: "https://example.com/skills.git", InstallEntry: "skills/go-pro",
		}},
	}))

	got, handled, err := NewLockedResolver(configFile, baseDir).Latest(lockpkg.Record{
		SkillID: "go-pro", SourceID: "team", SourceType: "registry", RegistryEntryID: "go-pro",
	})

	assert.NoErr(t, err)
	assert.True(t, handled)
	assert.Eq(t, "go-pro", got.ID)
	assert.Eq(t, "team", got.SourceID)
	assert.Eq(t, sourcepkg.TypeRegistry, got.SourceType)
	assert.Eq(t, "2.0.0", got.Version)
	assert.Eq(t, "abc123", got.Checksum)
	assert.Eq(t, "", got.Path)
}

func TestLockedResolver_IgnoresNonRegistryRecords(t *testing.T) {
	got, handled, err := NewLockedResolver(filepath.Join(t.TempDir(), "skillc.yaml"), t.TempDir()).Latest(lockpkg.Record{
		SkillID: "go-pro", SourceID: "local", SourceType: "local",
	})

	assert.NoErr(t, err)
	assert.False(t, handled)
	assert.Eq(t, "", got.ID)
}
