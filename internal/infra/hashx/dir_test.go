package hashx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestSumDirChangesWhenNestedFileContentChanges(t *testing.T) {
	root := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("skill"), 0o644))
	assert.NoErr(t, os.MkdirAll(filepath.Join(root, "commands"), 0o755))
	nested := filepath.Join(root, "commands", "run.md")
	assert.NoErr(t, os.WriteFile(nested, []byte("first"), 0o644))

	first, err := SumDir(root)
	assert.NoErr(t, err)
	assert.NoErr(t, os.WriteFile(nested, []byte("second"), 0o644))
	second, err := SumDir(root)

	assert.NoErr(t, err)
	assert.NotEq(t, first, second)
}

func TestSumDirSkipsGitDirectory(t *testing.T) {
	root := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("skill"), 0o644))
	gitDir := filepath.Join(root, ".git")
	assert.NoErr(t, os.MkdirAll(gitDir, 0o755))

	first, err := SumDir(root)
	assert.NoErr(t, err)
	assert.NoErr(t, os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("changed"), 0o644))
	second, err := SumDir(root)

	assert.NoErr(t, err)
	assert.Eq(t, first, second)
}
