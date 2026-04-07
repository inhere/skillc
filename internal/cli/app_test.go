package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/testutil/assert"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/updateapp"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

func newTestApp() *gcli.App {
	ccolor.Disable()
	return NewApp("dev", "unknown", "unknown")
}

type updateRunnerStub struct {
	runFn func(updateapp.Req) (updateapp.Result, error)
}

func (s updateRunnerStub) Run(req updateapp.Req) (updateapp.Result, error) {
	return s.runFn(req)
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

func TestNewApp_RegistersUpdateCommand(t *testing.T) {
	app := newTestApp()

	update := findCommandByName(app, "update")
	assert.NotNil(t, update)
	assert.Eq(t, "Update installed skills", update.Desc)
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

	output := runAppInDirWithStdout(t, baseDir, []string{"install", "--yes", "--agent", "claude-code", "hello-skill"})

	assert.Contains(t, output, "hello-skill")
	assert.Contains(t, output, filepath.Join(baseDir, "project-claude", "skills", "local-demo--hello-skill"))
	data, err := os.ReadFile(filepath.Join(baseDir, "project-claude", "skills", "local-demo--hello-skill", "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))
}

func TestInstallCommand_BatchTargetsWithYesReportsResolveAndInstallFailures(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexPath := filepath.Join(baseDir, "cache", "index.json")
	goodSourceDir := filepath.Join(baseDir, "source", "hello-skill")
	assert.NoErr(t, os.MkdirAll(filepath.Join(goodSourceDir, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(goodSourceDir, "commands", "hello.txt"), []byte("hello"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = filepath.Join(baseDir, "skillc-install.lock")
	config.IndexFile = indexPath
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexPath, []skill.Skill{
		{ID: "hello-skill", Name: "Hello Skill", Version: "1.0.0", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: goodSourceDir},
		{ID: "world-skill", Name: "World Skill", Version: "1.0.0", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: filepath.Join(baseDir, "missing")},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"install", "--yes", "--agent", "claude-code", "hello-skill,world-*,missing-skill"})

	assert.Contains(t, output, "hello-skill")
	assert.Contains(t, output, "resolve failed missing-skill")
	assert.Contains(t, output, "install failed world-skill")
	data, err := os.ReadFile(filepath.Join(baseDir, "project-claude", "skills", "local-demo--hello-skill", "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))
	_, err = os.Stat(filepath.Join(baseDir, "project-claude", "skills", "local-demo--world-skill", "hello.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestInstallCommand_CollectionModeInstallsCollectionTarget(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexPath := filepath.Join(baseDir, "cache", "index.json")
	firstSourceDir := filepath.Join(baseDir, "source", "hello-skill")
	secondSourceDir := filepath.Join(baseDir, "source", "world-skill")
	assert.NoErr(t, os.MkdirAll(filepath.Join(firstSourceDir, "commands"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(secondSourceDir, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(firstSourceDir, "commands", "hello.txt"), []byte("hello"), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(secondSourceDir, "commands", "world.txt"), []byte("world"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = filepath.Join(baseDir, "skillc-install.lock")
	config.IndexFile = indexPath
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexPath, []skill.Skill{
		{ID: "hello-skill", Name: "Hello Skill", Collection: "marketplaces", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", Version: "1.0.0", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: firstSourceDir},
		{ID: "world-skill", Name: "World Skill", Collection: "marketplaces", QualifiedName: "marketplaces/world-skill", SourceQualifiedName: "repo-a/marketplaces/world-skill", Version: "1.0.0", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: secondSourceDir},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"install", "--yes", "--collection", "--agent", "claude-code", "repo-a/marketplaces"})

	assert.Contains(t, output, "hello-skill")
	assert.Contains(t, output, "world-skill")
	helloData, err := os.ReadFile(filepath.Join(baseDir, "project-claude", "skills", "repo-a--marketplaces--hello-skill", "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(helloData))
	worldData, err := os.ReadFile(filepath.Join(baseDir, "project-claude", "skills", "repo-a--marketplaces--world-skill", "world.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "world", string(worldData))
}

func TestInstallCommand_PromptsBeforeInstallWithoutYes(t *testing.T) {
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

	output := runAppInDirWithInput(t, baseDir, []string{"install", "--agent", "claude-code", "hello-skill"}, "n\n")

	assert.Contains(t, output, "Continue? [y/N]")
	assert.Contains(t, output, "install cancelled")
	_, err := os.Stat(filepath.Join(baseDir, "project-claude", "skills", "local-demo--hello-skill", "hello.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestInstallCommand_RestoresFromLockFileWhenNoArgs(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := filepath.Join(baseDir, "cache", "hello-skill")
	commandsDir := filepath.Join(sourceDir, "commands")
	claudeInstalledPath := filepath.Join(baseDir, "project-claude", "skills", "local-demo--hello-skill")
	agentsInstalledPath := filepath.Join(baseDir, ".agents", "skills", "local-demo--hello-skill")
	assert.NoErr(t, os.MkdirAll(commandsDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(commandsDir, "hello.txt"), []byte("restored"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	config.Sources = []sourcepkg.Source{{ID: "local-demo", Type: sourcepkg.TypeLocal, Path: sourceDir}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID:      "hello-skill",
			SourceID:     "local-demo",
			SourceType:   "local",
			InstallEntry: "commands",
			Agents:       []string{"agents", "claude-code"},
		}},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"install"})

	assert.Contains(t, output, "hello-skill agents project")
	assert.Contains(t, output, "hello-skill claude-code project")
	claudeData, err := os.ReadFile(filepath.Join(claudeInstalledPath, "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "restored", string(claudeData))
	agentsData, err := os.ReadFile(filepath.Join(agentsInstalledPath, "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "restored", string(agentsData))
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
	assert.Contains(t, output, "design-helper")
	assert.Contains(t, output, "Target")
}

func TestSearchCommand_ShowsResolvableQualifiedName(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexPath := filepath.Join(baseDir, "cache", "index.json")
	config := cfg.DefaultConfig()
	config.IndexFile = indexPath
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexPath, []skill.Skill{{
		ID:            "ship-skill",
		Name:          "ship",
		Description:   "Ship workflow",
		Collection:    "gstack",
		QualifiedName: "gstack/ship",
	}}))

	output := runAppInDirWithStdout(t, baseDir, []string{"search", "ship"})

	assert.Contains(t, output, "gstack/ship")
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

	output := runAppInDirWithStdout(t, baseDir, []string{"list", "--agent", "claude-code"})

	assert.Eq(t, "", output)
}

func TestListCommand_ListsInstalledSkills(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	installedPath := filepath.Join(baseDir, "project-claude", "skills", "local-demo--hello-skill")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(installedPath, "hello.txt"), []byte("hello"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID:  "hello-skill",
			SourceID: "local-demo",
			Agents:   []string{"claude-code"},
		}},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"list", "--agent", "claude-code"})

	assert.Contains(t, output, "hello-skill")
	assert.Contains(t, output, "claude-code")
	assert.Contains(t, output, "project")
	assert.Contains(t, output, "installed")
}

func TestUninstallCommand_RemovesInstalledSkill(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	installedPath := filepath.Join(baseDir, "project-claude", "skills", "local-demo--hello-skill")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(installedPath, "hello.txt"), []byte("hello"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID:  "hello-skill",
			SourceID: "local-demo",
			Agents:   []string{"claude-code"},
		}},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"uninstall", "--agent", "claude-code", "hello-skill"})
	_ = output

	_, err := os.Stat(installedPath)
	assert.True(t, os.IsNotExist(err))

	locks, err := lockstore.NewStore().Load(lockFile)
	assert.NoErr(t, err)
	assert.Len(t, locks, 0)
}

func TestUpdateCommand_PrintsUpdatedSkippedAndFailedItems(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	helloSource := filepath.Join(baseDir, "source", "hello-skill")
	helloCommands := filepath.Join(helloSource, "commands")
	helloInstalled := filepath.Join(baseDir, "project-claude", "skills", "local-demo--hello-skill")
	pinnedInstalled := filepath.Join(baseDir, "project-claude", "skills", "local-demo--pinned-skill")
	brokenInstalled := filepath.Join(baseDir, "project-claude", "skills", "local-demo--broken-skill")
	assert.NoErr(t, os.MkdirAll(helloCommands, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(helloCommands, "hello.txt"), []byte("updated"), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(helloSource, "SKILL.md"), []byte(`---
id: hello-skill
name: Hello Skill
description: Friendly greeting helper
version: 2.0.0
install_entry: commands
---
# Hello Skill
`), 0o644))
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, "project-claude", "skills"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.Sources = []sourcepkg.Source{{ID: "local-demo", Name: "local-demo", Type: sourcepkg.TypeLocal, Path: filepath.Join(baseDir, "source")}}
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{SkillID: "hello-skill", QualifiedName: "hello-skill", SourceQualifiedName: "local-demo/hello-skill", SourceID: "local-demo", InstallEntry: "commands", Agents: []string{"claude-code"}},
			{SkillID: "pinned-skill", QualifiedName: "pinned-skill", SourceQualifiedName: "local-demo/pinned-skill", SourceID: "local-demo", InstallEntry: "commands", Agents: []string{"claude-code"}, Pinned: true},
			{SkillID: "broken-skill", QualifiedName: "broken-skill", SourceQualifiedName: "local-demo/broken-skill", SourceID: "local-demo", InstallEntry: "commands", Agents: []string{"claude-code"}},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "hello-skill", QualifiedName: "hello-skill", SourceQualifiedName: "local-demo/hello-skill", Version: "2.0.0", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: helloSource},
		{ID: "pinned-skill", QualifiedName: "pinned-skill", SourceQualifiedName: "local-demo/pinned-skill", Version: "2.0.0", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: filepath.Join(baseDir, "source", "pinned-skill")},
	}))
	assert.NoErr(t, os.MkdirAll(helloInstalled, 0o755))
	assert.NoErr(t, os.MkdirAll(pinnedInstalled, 0o755))
	assert.NoErr(t, os.MkdirAll(brokenInstalled, 0o755))

	output := runAppInDirWithStdout(t, baseDir, []string{"update", "--agent", "claude-code"})

	assert.Contains(t, output, "updated hello-skill "+helloInstalled)
	assert.Contains(t, output, "skipped pinned-skill pinned")
	assert.Contains(t, output, "update failed broken-skill installed skill not found in source index: broken-skill")
}
func TestUpdateCommand_PrintsCleanupFailuresWithoutDroppingSuccessfulUpdate(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	config := cfg.DefaultConfig()
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))

	cleanupInstalled := filepath.Join(baseDir, "project-claude", "skills", "local-demo--renamed--shared-skill")
	prevFactory := newUpdateService
	newUpdateService = func(configFile string, baseDir string) updateRunner {
		return updateRunnerStub{runFn: func(req updateapp.Req) (updateapp.Result, error) {
			return updateapp.Result{
				Updated: []installapp.RuntimeRecord{{Record: lockpkg.Record{SkillID: "shared-skill"}, InstalledPath: cleanupInstalled}},
				CleanupFailed: []updateapp.FailedItem{{SkillID: "shared-skill", Reason: "cleanup failed"}},
			}, nil
		}}
	}
	defer func() {
		newUpdateService = prevFactory
	}()

	output := runAppInDirWithStdout(t, baseDir, []string{"update", "--agent", "claude-code"})

	assert.Contains(t, output, "updated shared-skill "+cleanupInstalled)
	assert.Contains(t, output, "cleanup failed shared-skill cleanup failed")
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

func runAppInDirWithInput(t *testing.T, dir string, args []string, input string) string {
	return runInDirWithIO(t, dir, input, func() error {
		newTestApp().Run(args)
		return nil
	})
}

func runInDirWithStdout(t *testing.T, dir string, fn func() error) string {
	return runInDirWithIO(t, dir, "", fn)
}

func runInDirWithIO(t *testing.T, dir string, input string, fn func() error) string {
	t.Helper()
	oldWD, err := os.Getwd()
	assert.NoErr(t, err)
	assert.NoErr(t, os.Chdir(dir))
	defer func() {
		assert.NoErr(t, os.Chdir(oldWD))
	}()

	oldStdin := os.Stdin
	stdinR, stdinW, err := os.Pipe()
	assert.NoErr(t, err)
	if input != "" {
		_, err = stdinW.Write([]byte(input))
		assert.NoErr(t, err)
	}
	assert.NoErr(t, stdinW.Close())
	os.Stdin = stdinR
	defer func() {
		os.Stdin = oldStdin
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
