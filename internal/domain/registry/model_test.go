package registry

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestNewRegistryFromLocalPath(t *testing.T) {
	got, err := New("Local Registry", "Local Registry", "./registry.json")

	assert.NoErr(t, err)
	assert.Eq(t, "local-registry", got.ID)
	assert.Eq(t, "Local Registry", got.Name)
	assert.Eq(t, TypeLocal, got.Type)
	assert.NotEmpty(t, got.Path)
}

func TestNewRegistryFromHTTPURL(t *testing.T) {
	got, err := New("official", "Official", "https://example.com/registry.json")

	assert.NoErr(t, err)
	assert.Eq(t, "official", got.ID)
	assert.Eq(t, TypeHTTP, got.Type)
	assert.Eq(t, "https://example.com/registry.json", got.URL)
}

func TestEntryValidateRequiresSourceLocation(t *testing.T) {
	err := Entry{ID: "broken", Type: "git"}.Validate()

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "registry entry git url is required")
}
