package sourceapp

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestService_AddListRemoveLocalSource(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	service := NewService(configFile, baseDir)

	src, err := service.AddLocal(filepath.Join(baseDir, "skills"))
	assert.NoErr(t, err)
	assert.NotEmpty(t, src.ID)
	assert.Eq(t, "skills", src.Name)

	_, err = service.AddLocal(filepath.Join(baseDir, "skills"))
	assert.Error(t, err)

	list, err := service.List()
	assert.NoErr(t, err)
	assert.Len(t, list, 1)
	assert.Eq(t, src.ID, list[0].ID)

	err = service.Remove(src.ID)
	assert.NoErr(t, err)

	list, err = service.List()
	assert.NoErr(t, err)
	assert.Len(t, list, 0)
}
