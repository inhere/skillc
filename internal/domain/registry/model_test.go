package registry

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
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

func TestNewRegistryFromProviderURL(t *testing.T) {
	got, err := NewWithProvider("skillsmp", "SkillsMP", "https://skillsmp.com/", "skillsmp")

	assert.NoErr(t, err)
	assert.Eq(t, "skillsmp", got.ID)
	assert.Eq(t, "SkillsMP", got.Name)
	assert.Eq(t, TypeProvider, got.Type)
	assert.Eq(t, "skillsmp", got.Provider)
	assert.Eq(t, "https://skillsmp.com", got.URL)
}

func TestNewRegistryRejectsUnsupportedProvider(t *testing.T) {
	_, err := NewWithProvider("bad", "Bad", "https://example.com", "unknown")

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "unsupported registry provider")
}

func TestEntryValidateRequiresSourceLocation(t *testing.T) {
	err := Entry{ID: "broken", Type: "git"}.Validate()

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "registry entry git url is required")
}

func TestSkillEntryValidateRequiresInstallSource(t *testing.T) {
	err := SkillEntry{ID: "go-pro", Name: "Go Pro"}.Validate()

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "source_url or download_url is required")
}

func TestNormalizeSkillEntryDefaultsInstallEntryAndName(t *testing.T) {
	entry, err := NormalizeSkillEntry(SkillEntry{
		ID:        "Go Pro",
		SourceURL: "https://github.com/acme/skills.git",
	}, "skills-sh")

	assert.NoErr(t, err)
	assert.Eq(t, "go-pro", entry.ID)
	assert.Eq(t, "go-pro", entry.Name)
	assert.Eq(t, ".", entry.InstallEntry)
	assert.Eq(t, "skills-sh", entry.RegistryID)
}
