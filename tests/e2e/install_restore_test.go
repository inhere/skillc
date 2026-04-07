package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/listapp"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/lockstore"
)

func TestInstallListAndRestoreFlow(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := filepath.Join(baseDir, "source")
	commandsDir := filepath.Join(sourceDir, "commands")
	assert.NoErr(t, os.MkdirAll(commandsDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(commandsDir, "hello.txt"), []byte("hello"), 0o644))

	installer := installapp.NewService(lockFile)
	config := installTestConfig(baseDir)
	projectKey, err := filepath.Abs(baseDir)
	assert.NoErr(t, err)
	projectRoot, err := agent.ResolveInstallPath(config, baseDir, "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)
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

	_, err = installer.Install(item, "claude-code", agent.ScopeProject, projectKey, projectRoot)
	assert.NoErr(t, err)

	listed, err := listapp.NewService(lockFile).WithRuntime(config, baseDir).List("claude-code", "project")
	assert.NoErr(t, err)
	assert.Len(t, listed, 1)
	assert.Eq(t, "marketplaces/hello-skill", listed[0].QualifiedName)
	assert.Eq(t, "installed", listed[0].Status)

	assert.NoErr(t, installer.WithRuntime(config, baseDir).Uninstall("marketplaces/hello-skill", "claude-code", agent.ScopeProject))
	_, err = os.Stat(filepath.Join(projectRoot, "workflow-repo--marketplaces--hello-skill"))
	assert.True(t, os.IsNotExist(err))

	listed, err = listapp.NewService(lockFile).WithRuntime(config, baseDir).List("claude-code", "project")
	assert.NoErr(t, err)
	assert.Len(t, listed, 0)

	_, err = installer.Install(item, "claude-code", agent.ScopeProject, projectKey, projectRoot)
	assert.NoErr(t, err)
	assert.NoErr(t, os.RemoveAll(filepath.Join(projectRoot, "workflow-repo--marketplaces--hello-skill")))

	listed, err = listapp.NewService(lockFile).WithRuntime(config, baseDir).List("claude-code", "project")
	assert.NoErr(t, err)
	assert.Len(t, listed, 1)
	assert.Eq(t, "missing", listed[0].Status)

	restored, err := installer.WithRuntime(config, baseDir).Restore(map[string]string{"local-demo": sourceDir})
	assert.NoErr(t, err)
	assert.Len(t, restored, 1)
	assert.Eq(t, "workflow-repo/marketplaces/hello-skill", restored[0].SourceQualifiedName)

	data, err := os.ReadFile(filepath.Join(projectRoot, "workflow-repo--marketplaces--hello-skill", "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))
}

func TestRestoreUsesProjectKeyAsWorkdir(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	config := installTestConfig(baseDir)
	sourceDir := filepath.Join(baseDir, "source")
	commandsDir := filepath.Join(sourceDir, "commands")
	projectDir := filepath.Join(baseDir, "workspace")
	assert.NoErr(t, os.MkdirAll(commandsDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(commandsDir, "hello.txt"), []byte("restored"), 0o644))
	projectKey, err := filepath.Abs(projectDir)
	assert.NoErr(t, err)
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectKey: {
			{
				SkillID:             "hello-skill",
				QualifiedName:       "marketplaces/hello-skill",
				SourceQualifiedName: "workflow-repo/marketplaces/hello-skill",
				SourceID:            "local-demo",
				SourceType:          "local",
				InstallEntry:        "commands",
				Agents:              []string{"claude-code"},
			},
		},
	}))

	restored, err := installapp.NewService(lockFile).WithRuntime(config, baseDir).Restore(map[string]string{"local-demo": sourceDir})
	assert.NoErr(t, err)
	assert.Len(t, restored, 1)
	data, err := os.ReadFile(filepath.Join(projectDir, ".claude", "skills", "workflow-repo--marketplaces--hello-skill", "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "restored", string(data))
}

func installTestConfig(baseDir string) cfg.Config {
	return cfg.Config{
		AgentTools: map[string]cfg.AgentToolConfig{
			"claude-code": {UserDir: filepath.Join(baseDir, ".claude-user"), ProjectDir: "./.claude"},
			"codex":       {UserDir: filepath.Join(baseDir, ".codex-user"), ProjectDir: "./.codex"},
		},
	}
}
