package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/listapp"
	"github.com/inhere/skillc/internal/domain/agent"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
)

func TestInstallListAndRestoreFlow(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	targetRoot := filepath.Join(baseDir, ".claude", "skills")
	sourceDir := filepath.Join(baseDir, "source")
	commandsDir := filepath.Join(sourceDir, "commands")
	assert.NoErr(t, os.MkdirAll(commandsDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(commandsDir, "hello.txt"), []byte("hello"), 0o644))

	installer := installapp.NewService(lockFile)
	item := skill.Skill{
		ID:           "hello-skill",
		Name:         "Hello Skill",
		Version:      "1.0.0",
		SourceID:     "local-demo",
		SourceType:   sourcepkg.TypeLocal,
		InstallEntry: "commands",
		Path:         sourceDir,
	}

	_, err := installer.Install(item, "claude-code", agent.ScopeProject, targetRoot)
	assert.NoErr(t, err)

	listed, err := listapp.NewService(lockFile).List("claude-code", "project")
	assert.NoErr(t, err)
	assert.Len(t, listed, 1)
	assert.Eq(t, "installed", listed[0].Status)

	assert.NoErr(t, os.RemoveAll(filepath.Join(targetRoot, "hello-skill")))

	listed, err = listapp.NewService(lockFile).List("claude-code", "project")
	assert.NoErr(t, err)
	assert.Len(t, listed, 1)
	assert.Eq(t, "missing", listed[0].Status)

	restored, err := installer.Restore(map[string]string{"local-demo": commandsDir})
	assert.NoErr(t, err)
	assert.Len(t, restored, 1)

	data, err := os.ReadFile(filepath.Join(targetRoot, "hello-skill", "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))
}
