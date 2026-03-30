package doctorapp

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/app/configapp"
)

func TestService_CheckReportsGitAndConfigPaths(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	cfgService := configapp.NewService(configFile, baseDir)
	_, err := cfgService.Init()
	assert.NoErr(t, err)

	service := NewService(configFile, baseDir)
	service.gitLookPath = func(file string) (string, error) {
		return "/usr/bin/git", nil
	}

	result, err := service.Check()
	assert.NoErr(t, err)
	assert.True(t, result.GitAvailable)
	assert.True(t, result.ConfigOK)
	assert.NotEmpty(t, result.LockFile)
	assert.NotEmpty(t, result.RepoCacheDir)
}
