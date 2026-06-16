package registrystore

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
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
