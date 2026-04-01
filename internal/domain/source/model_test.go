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
	assert.Eq(t, "local-"+src.Name, src.ID)
}

func TestNewGitSource_PreservesEmptyRefForDefaultBranch(t *testing.T) {
	src, err := NewGitSource("https://example.com/repo.git", "")
	assert.NoErr(t, err)
	assert.Eq(t, TypeGit, src.Type)
	assert.Eq(t, "", src.Ref)
}
