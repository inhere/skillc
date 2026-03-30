package configstore

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
)

func TestStore_LoadMissingReturnsDefaultConfig(t *testing.T) {
	store := NewYAMLStore()
	got, err := store.Load("", "")
	assert.NoErr(t, err)
	assert.NotEmpty(t, got.AgentTools)
	assert.Eq(t, ".codex", got.AgentTools["codex"].Dirname)
}

func TestStore_SaveAndLoadRoundTrip(t *testing.T) {
	tmp := t.TempDir() + "/skillc.yaml"
	store := NewYAMLStore()

	want := cfg.DefaultConfig()
	want.ProxyURL = "http://localhost:7890"

	err := store.Save(tmp, want)
	assert.NoErr(t, err)

	got, err := store.Load(tmp, t.TempDir())
	assert.NoErr(t, err)
	assert.Eq(t, want.ProxyURL, got.ProxyURL)
	assert.Eq(t, want.LockFile, got.LockFile)
}
