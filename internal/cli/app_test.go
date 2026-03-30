package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/testutil/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

func newTestApp() *gcli.App {
	return NewApp("dev", "unknown", "unknown")
}

func TestNewApp_RegistersSearchCommand(t *testing.T) {
	app := newTestApp()

	search := findCommandByName(app, "search")
	assert.NotNil(t, search)
	assert.Eq(t, "Search indexed skills", search.Desc)

	show := findCommandByName(app, "show")
	assert.NotNil(t, show)
	assert.Eq(t, "Show indexed skill details", show.Desc)
}

func TestNewApp_RegistersInstallListAndDoctorCommands(t *testing.T) {
	app := newTestApp()

	install := findCommandByName(app, "install")
	assert.NotNil(t, install)
	assert.Eq(t, "Install skills", install.Desc)

	list := findCommandByName(app, "list")
	assert.NotNil(t, list)
	assert.Eq(t, "List installed skills", list.Desc)

	doctor := findCommandByName(app, "doctor")
	assert.NotNil(t, doctor)
	assert.Eq(t, "Check environment health", doctor.Desc)
}

func TestInstallCommand_InstallsIndexedSkill(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexPath := filepath.Join(baseDir, "skillc-index.json")
	sourceDir := filepath.Join(baseDir, "source", "hello-skill")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "commands", "hello.txt"), []byte("hello"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = filepath.Join(baseDir, "skillc-install.lock")
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexPath, []skill.Skill{{
		ID:           "hello-skill",
		Name:         "Hello Skill",
		Version:      "1.0.0",
		SourceID:     "local-demo",
		SourceType:   sourcepkg.TypeLocal,
		InstallEntry: "commands",
		Path:         sourceDir,
	}}))

	output := runInDirWithStdout(t, baseDir, func() error {
		return findCommandByName(newTestApp(), "install").Func(nil, []string{"hello-skill", "claude-code", "project"})
	})

	assert.Contains(t, output, "hello-skill")
	data, err := os.ReadFile(filepath.Join(baseDir, "project-claude", "skills", "hello-skill", "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))
}

func TestInstallCommand_RestoresFromLockFileWhenNoArgs(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := filepath.Join(baseDir, "cache", "hello-skill")
	commandsDir := filepath.Join(sourceDir, "commands")
	installedPath := filepath.Join(baseDir, "project-claude", "skills", "hello-skill")
	assert.NoErr(t, os.MkdirAll(commandsDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(commandsDir, "hello.txt"), []byte("restored"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.Sources = []sourcepkg.Source{{ID: "local-demo", Type: sourcepkg.TypeLocal, Path: sourceDir}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, []lockpkg.Record{{
		SkillID:       "hello-skill",
		Agent:         "claude-code",
		Scope:         "project",
		SourceID:      "local-demo",
		SourceType:    "local",
		InstallEntry:  "commands",
		InstalledPath: installedPath,
	}}))

	output := runInDirWithStdout(t, baseDir, func() error {
		return findCommandByName(newTestApp(), "install").Func(nil, nil)
	})

	assert.Contains(t, output, "hello-skill claude-code project")
	data, err := os.ReadFile(filepath.Join(installedPath, "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "restored", string(data))
}


func TestListCommand_ReturnsEmptyWhenLockFileMissing(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	config := cfg.DefaultConfig()
	config.LockFile = filepath.Join(baseDir, "skillc-install.lock")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))

	output := runInDirWithStdout(t, baseDir, func() error {
		return findCommandByName(newTestApp(), "list").Func(nil, []string{"claude-code", "project"})
	})

	assert.Eq(t, "", output)
}

func TestListCommand_ListsInstalledSkills(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	installedPath := filepath.Join(baseDir, "project-claude", "skills", "hello-skill")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, []lockpkg.Record{{
		SkillID:       "hello-skill",
		Agent:         "claude-code",
		Scope:         "project",
		InstalledPath: installedPath,
	}}))

	output := runInDirWithStdout(t, baseDir, func() error {
		return findCommandByName(newTestApp(), "list").Func(nil, []string{"claude-code", "project"})
	})

	assert.Contains(t, output, "hello-skill claude-code project installed")
}

func TestDoctorCommand_ReportsHealth(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	config := cfg.DefaultConfig()
	config.LockFile = filepath.Join(baseDir, "skillc-install.lock")
	config.RepoCacheDir = filepath.Join(baseDir, "repos")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))

	output := runInDirWithStdout(t, baseDir, func() error {
		return findCommandByName(newTestApp(), "doctor").Func(nil, nil)
	})

	assert.Contains(t, output, "git_available=")
	assert.Contains(t, output, "config_ok=true")
	assert.Contains(t, output, "lock_file=")
	assert.Contains(t, output, "repo_cache_dir=")
}

func findCommandByName(app *gcli.App, name string) *gcli.Command {
	for _, cmd := range app.Commands() {
		if cmd.Name == name {
			return cmd
		}
	}
	return nil
}

func runInDirWithStdout(t *testing.T, dir string, fn func() error) string {
	t.Helper()
	oldWD, err := os.Getwd()
	assert.NoErr(t, err)
	assert.NoErr(t, os.Chdir(dir))
	defer func() {
		assert.NoErr(t, os.Chdir(oldWD))
	}()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	assert.NoErr(t, err)
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	err = fn()
	assert.NoErr(t, err)
	assert.NoErr(t, w.Close())

	data, readErr := io.ReadAll(r)
	assert.NoErr(t, readErr)
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}
