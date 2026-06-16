package registrystore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/skillc/internal/domain/registry"
)

func TestStoreLoadSaveEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry-index.json")
	store := NewStore()
	entries := []registry.Entry{{ID: "gstack", Name: "GStack", Type: "git", URL: "https://example.com/skills.git", RegistryID: "official"}}

	assert.NoErr(t, store.Save(path, entries))
	got, err := store.Load(path)

	assert.NoErr(t, err)
	assert.Len(t, got, 1)
	assert.Eq(t, "gstack", got[0].ID)
	assert.Eq(t, "official", got[0].RegistryID)
}

func TestStore_SaveAndLoadSkillsAndSources(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry-index.json")
	store := NewStore()
	file := File{
		Skills:  []registry.SkillEntry{{ID: "go-pro", RegistryID: "team", SourceURL: "https://example.com/skills.git", InstallEntry: "skills/go-pro"}},
		Sources: []registry.Entry{{ID: "gstack", RegistryID: "team", Type: "git", URL: "https://example.com/gstack.git"}},
	}

	assert.NoErr(t, store.SaveFile(path, file))
	got, err := store.LoadFile(path)

	assert.NoErr(t, err)
	assert.Len(t, got.Skills, 1)
	assert.Eq(t, "go-pro", got.Skills[0].ID)
	assert.Len(t, got.Sources, 1)
	assert.Eq(t, "gstack", got.Sources[0].ID)
}

func TestStore_LoadLegacyEntriesAsSources(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry-index.json")
	assert.NoErr(t, os.WriteFile(path, []byte(`{"entries":[{"id":"gstack","registry_id":"team","type":"git","url":"https://example.com/gstack.git"}]}`), 0o644))

	got, err := NewStore().LoadFile(path)

	assert.NoErr(t, err)
	assert.Len(t, got.Sources, 1)
	assert.Eq(t, "gstack", got.Sources[0].ID)
}
