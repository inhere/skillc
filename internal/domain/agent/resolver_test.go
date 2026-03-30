package agent

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
)

func TestResolveInstallPath_UsesConfiguredScopePath(t *testing.T) {
	baseDir := t.TempDir()
	config := cfg.Config{
		AgentTools: map[string]cfg.AgentToolConfig{
			"claude-code": {
				UserDir:    "~/.claude",
				ProjectDir: "./.claude",
			},
		},
	}

	globalPath, err := ResolveInstallPath(config, baseDir, "claude-code", ScopeUser)
	assert.NoErr(t, err)
	assert.Contains(t, globalPath, filepath.Join(".claude", "skills"))

	projectPath, err := ResolveInstallPath(config, baseDir, "claude-code", ScopeProject)
	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(baseDir, ".claude", "skills"), projectPath)
}

func TestResolveInstallPath_RejectsUnknownAgent(t *testing.T) {
	_, err := ResolveInstallPath(cfg.Config{}, t.TempDir(), "unknown", ScopeUser)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported agent")
}
