package configapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestService_InitShowGetSet(t *testing.T) {
	baseDir := t.TempDir()
	home, err := os.UserHomeDir()
	assert.NoErr(t, err)
	configFile := filepath.Join(baseDir, "skillc.yaml")
	service := NewService(configFile, baseDir)

	cfg, err := service.Init()
	assert.NoErr(t, err)
	assert.NotEmpty(t, cfg.AgentTools)

	shown, err := service.Show()
	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(home, ".config", "skillc", "skillc-install.lock"), shown.LockFile)
	assert.Eq(t, filepath.Join(home, ".cache", "skillc", "skillc-index.json"), shown.IndexFile)

	lockFile, err := service.Get("lock_file")
	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(home, ".config", "skillc", "skillc-install.lock"), lockFile)

	indexFile, err := service.Get("index_file")
	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(home, ".cache", "skillc", "skillc-index.json"), indexFile)

	err = service.Set("proxy_url", "http://localhost:7890")
	assert.NoErr(t, err)

	proxyURL, err := service.Get("proxy_url")
	assert.NoErr(t, err)
	assert.Eq(t, "http://localhost:7890", proxyURL)

	err = service.Set("index_file", filepath.Join(baseDir, "custom-index.json"))
	assert.NoErr(t, err)

	indexFile, err = service.Get("index_file")
	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(baseDir, "custom-index.json"), indexFile)
}
