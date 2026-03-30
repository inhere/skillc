package sourceapp

import (
	"errors"
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

func TestService_AddGitAndSyncStatus(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	service := NewService(configFile, baseDir)
	service.git = gitRunnerFunc(func(url, dir, ref string) error {
		return nil
	})

	src, err := service.AddGit("https://example.com/repo.git", "main")
	assert.NoErr(t, err)
	assert.Eq(t, "main", src.Ref)

	err = service.Sync(src.ID)
	assert.NoErr(t, err)

	list, err := service.List()
	assert.NoErr(t, err)
	assert.Len(t, list, 1)
	assert.Eq(t, "ready", list[0].Status)
	assert.NotEmpty(t, list[0].Path)
}

func TestService_SyncMissingGitSetsSourceError(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	service := NewService(configFile, baseDir)
	service.git = gitRunnerFunc(func(url, dir, ref string) error {
		return errors.New("git executable not found")
	})

	src, err := service.AddGit("https://example.com/repo.git", "main")
	assert.NoErr(t, err)

	err = service.Sync(src.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git executable not found")

	list, err := service.List()
	assert.NoErr(t, err)
	assert.Eq(t, "error", list[0].Status)
	assert.Contains(t, list[0].ErrorMessage, "git executable not found")
}
