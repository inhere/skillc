package source

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/fsutil"
	"github.com/gookit/goutil/testutil/assert"
)

func TestNewLocalSource_AssignsIDAndName(t *testing.T) {
	expectedPath := fsutil.ToAbsPath(filepath.Clean("/tmp/skills"))
	src, err := NewLocalSource("/tmp/skills")
	assert.NoErr(t, err)
	assert.Eq(t, TypeLocal, src.Type)
	assert.Eq(t, expectedPath, src.Path)
	assert.Eq(t, filepath.Base(filepath.Dir(expectedPath))+"-"+filepath.Base(expectedPath), src.Name)
	assert.Eq(t, src.Name, src.ID)
}

func TestNewLocalSourceGeneratesIDWithoutTypePrefix(t *testing.T) {
	src, err := NewLocalSource(filepath.Join("work", "gstack", "skills"))

	assert.NoErr(t, err)
	assert.Eq(t, "gstack-skills", src.ID)
	assert.Eq(t, "gstack-skills", src.Name)
	assert.Eq(t, TypeLocal, src.Type)
}

func TestNewGitSource_PreservesEmptyRefForDefaultBranch(t *testing.T) {
	src, err := NewGitSource("https://example.com/repo.git", "")
	assert.NoErr(t, err)
	assert.Eq(t, TypeGit, src.Type)
	assert.Eq(t, "", src.Ref)
}

func TestNewGitSourceGeneratesIDWithoutTypePrefix(t *testing.T) {
	src, err := NewGitSource("https://github.com/acme/skills.git", "main")

	assert.NoErr(t, err)
	assert.Eq(t, "acme-skills", src.ID)
	assert.Eq(t, "acme-skills", src.Name)
	assert.Eq(t, TypeGit, src.Type)
	assert.Eq(t, "main", src.Ref)
}

func TestNewSourceWithExplicitIDAndName(t *testing.T) {
	src, err := NewGitSourceWithOptions("https://github.com/acme/skills.git", "", SourceOptions{
		ID:   "Acme Skills",
		Name: "Acme Registry",
	})

	assert.NoErr(t, err)
	assert.Eq(t, "acme-skills", src.ID)
	assert.Eq(t, "Acme Registry", src.Name)
}
