package configapp

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestService_InitShowGetSet(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	service := NewService(configFile, baseDir)

	cfg, err := service.Init()
	assert.NoErr(t, err)
	assert.NotEmpty(t, cfg.AgentTools)

	shown, err := service.Show()
	assert.NoErr(t, err)
	assert.Eq(t, cfg.LockFile, shown.LockFile)

	lockFile, err := service.Get("lock_file")
	assert.NoErr(t, err)
	assert.Eq(t, cfg.LockFile, lockFile)

	err = service.Set("proxy_url", "http://localhost:7890")
	assert.NoErr(t, err)

	proxyURL, err := service.Get("proxy_url")
	assert.NoErr(t, err)
	assert.Eq(t, "http://localhost:7890", proxyURL)
}
