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
		ID:                  "hello-skill",
		Name:                "Hello Skill",
		QualifiedName:       "marketplaces/hello-skill",
		SourceQualifiedName: "workflow-repo/marketplaces/hello-skill",
		Version:             "1.0.0",
		SourceID:            "local-demo",
		SourceType:          sourcepkg.TypeLocal,
		InstallEntry:        "commands",
		Path:                sourceDir,
	}

	_, err := installer.Install(item, "claude-code", agent.ScopeProject, targetRoot)
	assert.NoErr(t, err)

	listed, err := listapp.NewService(lockFile).List("claude-code", "project")
	assert.NoErr(t, err)
	assert.Len(t, listed, 1)
	assert.Eq(t, "marketplaces/hello-skill", listed[0].QualifiedName)
	assert.Eq(t, "installed", listed[0].Status)

	assert.NoErr(t, installer.Uninstall("marketplaces/hello-skill", "claude-code", agent.ScopeProject))
	_, err = os.Stat(filepath.Join(targetRoot, "hello-skill"))
	assert.True(t, os.IsNotExist(err))

	listed, err = listapp.NewService(lockFile).List("claude-code", "project")
	assert.NoErr(t, err)
	assert.Len(t, listed, 0)

	_, err = installer.Install(item, "claude-code", agent.ScopeProject, targetRoot)
	assert.NoErr(t, err)
	assert.NoErr(t, os.RemoveAll(filepath.Join(targetRoot, "hello-skill")))

	listed, err = listapp.NewService(lockFile).List("claude-code", "project")
	assert.NoErr(t, err)
	assert.Len(t, listed, 1)
	assert.Eq(t, "missing", listed[0].Status)

	restored, err := installer.Restore(map[string]string{"local-demo": sourceDir})
	assert.NoErr(t, err)
	assert.Len(t, restored, 1)
	assert.Eq(t, "workflow-repo/marketplaces/hello-skill", restored[0].SourceQualifiedName)

	data, err := os.ReadFile(filepath.Join(targetRoot, "hello-skill", "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))
}
