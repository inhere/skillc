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

func TestNewApp_RegistersCollectionCommand(t *testing.T) {
	app := newTestApp()

	collection := findCommandByName(app, "collection")
	assert.NotNil(t, collection)
	assert.Eq(t, "Browse indexed collections", collection.Desc)
}

func TestNewApp_RegistersInstallListAndDoctorCommands(t *testing.T) {
	app := newTestApp()

	install := findCommandByName(app, "install")
	assert.NotNil(t, install)
	assert.Eq(t, "Install skills", install.Desc)

	uninstall := findCommandByName(app, "uninstall")
	assert.NotNil(t, uninstall)
	assert.Eq(t, "Uninstall skills", uninstall.Desc)

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
	indexPath := filepath.Join(baseDir, "cache", "index.json")
	sourceDir := filepath.Join(baseDir, "source", "hello-skill")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "commands", "hello.txt"), []byte("hello"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = filepath.Join(baseDir, "skillc-install.lock")
	config.IndexFile = indexPath
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

func TestCollectionListCommand_PrintsCollectionSummary(t *testing.T) {
	baseDir := t.TempDir()
	config := cfg.DefaultConfig()
	config.IndexFile = filepath.Join(baseDir, "cache", "index.json")
	assert.NoErr(t, configstore.NewYAMLStore().Save(filepath.Join(baseDir, "skillc.yaml"), config))
	assert.NoErr(t, repoindex.NewStore().Save(config.IndexFile, []skill.Skill{
		{ID: "alpha-one", Name: "Alpha One", Collection: "alpha", SourceID: "src-a", SourceName: "repo-a"},
		{ID: "alpha-two", Name: "Alpha Two", Collection: "alpha", SourceID: "src-b", SourceName: "repo-b"},
		{ID: "beta-one", Name: "Beta One", Collection: "beta", SourceID: "src-a", SourceName: "repo-a"},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"collection", "list"})
	assert.Contains(t, output, "alpha      | 2      | 2")
	assert.Contains(t, output, "beta       | 1      | 1")
}

func TestCollectionSkillsCommand_PrintsSkillNameAndDescription(t *testing.T) {
	baseDir := t.TempDir()
	config := cfg.DefaultConfig()
	config.IndexFile = filepath.Join(baseDir, "cache", "index.json")
	assert.NoErr(t, configstore.NewYAMLStore().Save(filepath.Join(baseDir, "skillc.yaml"), config))
	assert.NoErr(t, repoindex.NewStore().Save(config.IndexFile, []skill.Skill{
		{ID: "alpha-one", Name: "Alpha One", Description: "first skill", Collection: "alpha"},
		{ID: "alpha-two", Name: "Alpha Two", Description: "second skill", Collection: "alpha"},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"collection", "skills", "alpha"})
	assert.Contains(t, output, "Alpha One | first skill")
	assert.Contains(t, output, "Alpha Two | second skill")
}

func TestSearchCommand_ReturnsMatchesForQueryArgument(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexPath := filepath.Join(baseDir, "cache", "index.json")
	config := cfg.DefaultConfig()
	config.IndexFile = indexPath
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexPath, []skill.Skill{{
		ID:          "design-helper",
		Name:        "Design Helper",
		Description: "design prompts",
	}}))

	output := runAppInDirWithStdout(t, baseDir, []string{"search", "design"})

	assert.Contains(t, output, "Search Result")
	assert.Contains(t, output, "Design Helper")
	assert.Contains(t, output, "Name")
}

func TestSearchCommand_ReturnsHelpfulMessageWhenNoMatches(t *testing.T) {
	baseDir := t.TempDir()
	config := cfg.DefaultConfig()
	config.IndexFile = filepath.Join(baseDir, "cache", "index.json")
	assert.NoErr(t, configstore.NewYAMLStore().Save(filepath.Join(baseDir, "skillc.yaml"), config))

	output := runAppInDirWithStdout(t, baseDir, []string{"search", "design"})

	assert.Contains(t, output, "no skills found")
}

func TestSourceAddLocalCommand_PrintsNextSyncHint(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	config := cfg.DefaultConfig()
	config.IndexFile = filepath.Join(baseDir, "cache", "index.json")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))

	sourceRoot := filepath.Join(baseDir, "skills")
	assert.NoErr(t, os.MkdirAll(sourceRoot, 0o755))

	output := runAppInDirWithStdout(t, baseDir, []string{"source", "add", "local", sourceRoot})

	assert.Contains(t, output, "added.")
	assert.Contains(t, output, "path=")
	assert.Contains(t, output, "Next, please run: skillc source sync ")
}

func TestSourceAddLocalCommand_WithSyncRebuildsIndexForSearch(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexPath := filepath.Join(baseDir, "cache", "index.json")
	sourceRoot := filepath.Join(baseDir, "skills")
	skillDir := filepath.Join(sourceRoot, "hello-skill")
	assert.NoErr(t, os.MkdirAll(skillDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
id: hello-skill
name: Hello Skill
description: Friendly greeting helper
---
# Hello Skill
`), 0o644))

	config := cfg.DefaultConfig()
	config.IndexFile = indexPath
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))

	output := runAppInDirWithStdout(t, baseDir, []string{"source", "add", "local", "--sync", sourceRoot})
	assert.Contains(t, output, "added.")
	assert.Contains(t, output, "path=")
	assert.NotContains(t, output, "Next, please run: skillc source sync")

	items, err := repoindex.NewStore().Load(indexPath)
	assert.NoErr(t, err)
	assert.Len(t, items, 1)
	assert.Eq(t, "hello-skill", items[0].ID)
	assert.Eq(t, "Hello Skill", items[0].Name)
}

func TestSourceAddGitCommand_PrintsNextSyncHint(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	config := cfg.DefaultConfig()
	config.IndexFile = filepath.Join(baseDir, "cache", "index.json")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))

	output := runAppInDirWithStdout(t, baseDir, []string{"source", "add", "git", "https://example.com/repo.git"})

	assert.Contains(t, output, "added.")
	assert.Contains(t, output, "url=https://example.com/repo.git")
	assert.Contains(t, output, "Next, please run: skillc source sync git-repo")
}

func TestSourceSyncCommand_PrintsSourceStatusAfterSync(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexPath := filepath.Join(baseDir, "cache", "index.json")
	sourceRoot := filepath.Join(baseDir, "skills")
	skillDir := filepath.Join(sourceRoot, "hello-skill")
	assert.NoErr(t, os.MkdirAll(skillDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
id: hello-skill
name: Hello Skill
description: Friendly greeting helper
---
# Hello Skill
`), 0o644))
	localSource, err := sourcepkg.NewLocalSource(sourceRoot)
	assert.NoErr(t, err)

	config := cfg.DefaultConfig()
	config.IndexFile = indexPath
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))

	addOutput := runAppInDirWithStdout(t, baseDir, []string{"source", "add", "local", sourceRoot})
	assert.Contains(t, addOutput, "Next, please run: skillc source sync ")

	syncOutput := runAppInDirWithStdout(t, baseDir, []string{"source", "sync", localSource.ID})
	assert.Contains(t, syncOutput, "synced ")
	assert.Contains(t, syncOutput, "ready")
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

func TestUninstallCommand_RemovesInstalledSkill(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	installedPath := filepath.Join(baseDir, "project-claude", "skills", "hello-skill")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(installedPath, "hello.txt"), []byte("hello"), 0o644))

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
		return findCommandByName(newTestApp(), "uninstall").Func(nil, []string{"hello-skill", "claude-code", "project"})
	})
	assert.Contains(t, output, "ok")

	_, err := os.Stat(installedPath)
	assert.True(t, os.IsNotExist(err))

	locks, err := lockstore.NewStore().Load(lockFile)
	assert.NoErr(t, err)
	assert.Len(t, locks, 0)
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

func runAppInDirWithStdout(t *testing.T, dir string, args []string) string {
	return runInDirWithStdout(t, dir, func() error {
		newTestApp().Run(args)
		return nil
	})
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
	stdoutR, stdoutW, err := os.Pipe()
	assert.NoErr(t, err)
	os.Stdout = stdoutW
	defer func() {
		os.Stdout = oldStdout
	}()

	oldStderr := os.Stderr
	stderrR, stderrW, err := os.Pipe()
	assert.NoErr(t, err)
	os.Stderr = stderrW
	defer func() {
		os.Stderr = oldStderr
	}()

	err = fn()
	assert.NoErr(t, err)
	assert.NoErr(t, stdoutW.Close())
	assert.NoErr(t, stderrW.Close())

	stdoutData, readErr := io.ReadAll(stdoutR)
	assert.NoErr(t, readErr)
	stderrData, readErr := io.ReadAll(stderrR)
	assert.NoErr(t, readErr)
	return strings.ReplaceAll(string(stdoutData)+string(stderrData), "\r\n", "\n")
}
