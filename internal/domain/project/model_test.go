package project

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestNewProjectNormalizesIDAndName(t *testing.T) {
	got, err := New("", "", filepath.Join("work", "My App"))

	assert.NoErr(t, err)
	assert.Eq(t, "my-app", got.ID)
	assert.Eq(t, "My App", got.Name)
	assert.Contains(t, got.Path, filepath.Join("work", "My App"))
}

func TestNewProjectUsesExplicitIDAndName(t *testing.T) {
	got, err := New("demo_api", "Demo API", filepath.Join("work", "demo"))

	assert.NoErr(t, err)
	assert.Eq(t, "demo_api", got.ID)
	assert.Eq(t, "Demo API", got.Name)
}

func TestNewProjectRejectsEmptyPath(t *testing.T) {
	_, err := New("demo", "", "")

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "project path is required")
}
