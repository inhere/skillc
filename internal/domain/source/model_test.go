package source

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestNewLocalSource_AssignsIDAndName(t *testing.T) {
	src, err := NewLocalSource("/tmp/skills")
	assert.NoErr(t, err)
	assert.Eq(t, TypeLocal, src.Type)
	assert.Eq(t, filepath.Clean("/tmp/skills"), src.Path)
	assert.Eq(t, "skills", src.Name)
	assert.NotEmpty(t, src.ID)
}
