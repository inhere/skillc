package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/testutil/assert"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/projectupdateapp"
	"github.com/inhere/skillc/internal/app/statusapp"
	"github.com/inhere/skillc/internal/app/updateapp"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/profile"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/agentfs"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
	"github.com/inhere/skillc/internal/infra/termselect"
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

type projectUpdateRunnerStub struct {
	planFn func(projectupdateapp.Req) (projectupdateapp.Plan, error)
	runFn  func(projectupdateapp.Req) (projectupdateapp.Result, error)
}

func (s projectUpdateRunnerStub) Plan(req projectupdateapp.Req) (projectupdateapp.Plan, error) {
	return s.planFn(req)
}

func (s projectUpdateRunnerStub) Run(req projectupdateapp.Req) (projectupdateapp.Result, error) {
	return s.runFn(req)
}

type webServerStub struct {
	host string
	port int
}

func (s *webServerStub) Serve(host string, port int) error {
	s.host = host
	s.port = port
	return nil
}

type selectorStub struct {
	items  []termselect.Item
	got    termselect.Options
	called bool
}

func (s *selectorStub) SelectMulti(_ context.Context, opts termselect.Options) ([]termselect.Item, error) {
	s.called = true
	s.got = opts
	return s.items, nil
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

func TestNewApp_DoesNotRegisterTopLevelCollectionCommand(t *testing.T) {
	app := newTestApp()

	collection := findCommandByName(app, "collection")
	assert.Nil(t, collection)
}

func TestNewApp_RegistersUpdateCommand(t *testing.T) {
	app := newTestApp()

	update := findCommandByName(app, "update")
	assert.NotNil(t, update)
	assert.Eq(t, "Update installed skills", update.Desc)
}

func TestNewApp_RegistersStatusCommand(t *testing.T) {
	app := newTestApp()

	status := findCommandByName(app, "status")
	assert.NotNil(t, status)
	if status == nil {
		return
	}
	assert.Eq(t, "Show skill status", status.Desc)
}

func TestNewApp_RegistersProfileCommand(t *testing.T) {
	app := newTestApp()

	profileCmd := findCommandByName(app, "profile")
	assert.NotNil(t, profileCmd)
	if profileCmd == nil {
		return
	}
	assert.Eq(t, "Manage Skillc profiles", profileCmd.Desc)
}

func TestNewApp_RegistersWebCommand(t *testing.T) {
	app := newTestApp()

	web := findCommandByName(app, "web")
	assert.NotNil(t, web)
	if web == nil {
		return
	}
	assert.Eq(t, "Start local web manager", web.Desc)
}

func TestWebCommandPassesHostAndPort(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, cfg.DefaultConfig(), baseDir))
	stub := &webServerStub{}
	var gotConfigFile string
	var gotBaseDir string
	prevFactory := newWebManagerServer
	newWebManagerServer = func(configFile string, baseDir string) webManagerServer {
		gotConfigFile = configFile
		gotBaseDir = baseDir
		return stub
	}
	defer func() {
		newWebManagerServer = prevFactory
	}()

	runAppInDirWithStdout(t, baseDir, []string{"web", "--host", "127.0.0.2", "--port", "18080"})

	assert.Eq(t, "127.0.0.2", stub.host)
	assert.Eq(t, 18080, stub.port)
	assert.Eq(t, baseDir, gotBaseDir)
	assert.Eq(t, configFile, gotConfigFile)
}

func TestSkillSelectItemsUseStableSourceQualifiedTargets(t *testing.T) {
	items := skillSelectItems([]skill.Skill{{
		ID:                  "go-pro",
		Name:                "Go Pro",
		Version:             "1.2.3",
		SourceID:            "repo-a",
		Collection:          "tools",
		QualifiedName:       "tools/go-pro",
		SourceQualifiedName: "repo-a/tools/go-pro",
	}})

	assert.Len(t, items, 1)
	assert.Eq(t, "1", items[0].Key)
	assert.Eq(t, "repo-a/tools/go-pro", items[0].Value)
	assert.Contains(t, items[0].Label, "Go Pro")
	assert.Contains(t, items[0].Detail, "repo-a")
	assert.Contains(t, items[0].Detail, "tools")
	assert.Contains(t, items[0].Detail, "1.2.3")
}

func TestSkillSelectItemsKeepLabelAndDetailSeparate(t *testing.T) {
	items := skillSelectItems([]skill.Skill{{
		ID:                  "go-pro",
		Name:                "Go Pro",
		Version:             "1.2.3",
		SourceID:            "repo-a",
		Collection:          "tools",
		QualifiedName:       "tools/go-pro",
		SourceQualifiedName: "repo-a/tools/go-pro",
	}})

	assert.Len(t, items, 1)
	assert.Eq(t, "Go Pro (go-pro)", items[0].Label)
	assert.Eq(t, "source=repo-a collection=tools version=1.2.3", items[0].Detail)
}

func TestSkillTargetFallsBackToQualifiedNameWhenSourceQualifiedNameIsMissing(t *testing.T) {
	target := skillTarget(skill.Skill{
		ID:            "go-pro",
		SourceID:      "repo-a",
		QualifiedName: "tools/go-pro",
	})

	assert.Eq(t, "tools/go-pro", target)
}

func TestSelectedSkillsMapsSelectedTargetsBackToSkills(t *testing.T) {
	skills := []skill.Skill{
		{ID: "go-pro", SourceQualifiedName: "repo-a/tools/go-pro"},
		{ID: "review", SourceQualifiedName: "repo-a/ops/review"},
	}

	selected := selectedSkills(skills, []termselect.Item{{Value: "repo-a/ops/review"}})

	assert.Len(t, selected, 1)
	assert.Eq(t, "review", selected[0].ID)
}

func TestUpdateSelectItemsOnlyIncludesUpdateableStatuses(t *testing.T) {
	items := updateSelectItems([]statusapp.Item{
		{SkillID: "installed", Status: statusapp.StatusInstalled},
		{SkillID: "missing", Status: statusapp.StatusMissing},
		{SkillID: "outdated", Status: statusapp.StatusOutdated, CurrentVersion: "1.0.0", LatestVersion: "2.0.0"},
		{SkillID: "orphan", Status: statusapp.StatusOrphan},
		{SkillID: "unmanaged", Status: statusapp.StatusUnmanaged},
		{SkillID: "source-error", Status: statusapp.StatusSourceError},
	})

	assert.Len(t, items, 2)
	assert.Eq(t, "missing", items[0].Value)
	assert.Eq(t, "outdated", items[1].Value)
	assert.Contains(t, items[1].Label, "outdated")
	assert.Contains(t, items[1].Detail, "1.0.0")
	assert.Contains(t, items[1].Detail, "2.0.0")
}

func TestStatusTargetFallsBackToQualifiedNameWhenSourceQualifiedNameIsMissing(t *testing.T) {
	target := statusTarget(statusapp.Item{
		SkillID:       "go-pro",
		SourceID:      "repo-a",
		QualifiedName: "tools/go-pro",
	})

	assert.Eq(t, "tools/go-pro", target)
}

func TestStatusTargetFallsBackToSkillIDWhenQualifiedNamesAreMissing(t *testing.T) {
	target := statusTarget(statusapp.Item{
		SkillID:  "go-pro",
		SourceID: "repo-a",
	})

	assert.Eq(t, "go-pro", target)
}

func TestUpdateTargetPrefersSourceQualifiedName(t *testing.T) {
	item := statusapp.Item{
		SkillID:             "go-pro",
		QualifiedName:       "tools/go-pro",
		SourceID:            "repo-a",
		SourceQualifiedName: "repo-a/tools/go-pro",
	}

	assert.Eq(t, "repo-a/tools/go-pro", statusTarget(item))

	selected := selectedUpdateTargets([]statusapp.Item{item}, []termselect.Item{{Value: "repo-a/tools/go-pro"}})
	assert.Eq(t, []string{"repo-a/tools/go-pro"}, selected)
}

func TestUpdateCommand_AcceptsSkillArgumentAsTarget(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	config := cfg.DefaultConfig()
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))

	var gotReq updateapp.Req
	prevFactory := newUpdateService
	newUpdateService = func(configFile string, baseDir string) updateRunner {
		return updateRunnerStub{runFn: func(req updateapp.Req) (updateapp.Result, error) {
			gotReq = req
			return updateapp.Result{}, nil
		}}
	}
	defer func() {
		newUpdateService = prevFactory
	}()

	runAppInDirWithStdout(t, baseDir, []string{"update", "hello-skill"})

	assert.Eq(t, "hello-skill", gotReq.Target)
}

func TestUpdateCommand_TargetFlagTakesPrecedenceOverSkillArgument(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	config := cfg.DefaultConfig()
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))

	var gotReq updateapp.Req
	prevFactory := newUpdateService
	newUpdateService = func(configFile string, baseDir string) updateRunner {
		return updateRunnerStub{runFn: func(req updateapp.Req) (updateapp.Result, error) {
			gotReq = req
			return updateapp.Result{}, nil
		}}
	}
	defer func() {
		newUpdateService = prevFactory
	}()

	runAppInDirWithStdout(t, baseDir, []string{"update", "--target", "flag-skill", "arg-skill"})

	assert.Eq(t, "flag-skill", gotReq.Target)
}

func TestUpdateCommand_CheckPrintsCandidatesWithoutCallingUpdateRunner(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{{ID: "go-pro", SourceID: "gstack", Version: "2.0.0"}}))

	calledUpdateRunner := false
	prevFactory := newUpdateService
	newUpdateService = func(configFile string, baseDir string) updateRunner {
		return updateRunnerStub{runFn: func(req updateapp.Req) (updateapp.Result, error) {
			calledUpdateRunner = true
			return updateapp.Result{}, nil
		}}
	}
	defer func() {
		newUpdateService = prevFactory
	}()

	output := runAppInDirWithStdout(t, baseDir, []string{"update", "--check", "--agent", "universal"})

	assert.Contains(t, output, "Update Check")
	assert.Contains(t, output, "outdated")
	assert.Contains(t, output, "go-pro")
	assert.False(t, calledUpdateRunner)
}

func TestUpdateCommand_CheckHonorsTargetFilter(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "rust-pro"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}},
			{SkillID: "rust-pro", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Version: "2.0.0"},
		{ID: "rust-pro", SourceID: "gstack", Version: "2.0.0"},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"update", "--check", "--agent", "universal", "--target", "go-pro"})

	assert.Contains(t, output, "Update Check")
	assert.Contains(t, output, "go-pro")
	assert.NotContains(t, output, "rust-pro")
}

func TestUpdateCommand_CheckPrintsNoCandidatesWhenHealthy(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{{ID: "go-pro", SourceID: "gstack", Version: "1.0.0"}}))

	output := runAppInDirWithStdout(t, baseDir, []string{"update", "--check", "--agent", "universal"})

	assert.Contains(t, output, "no update candidates")
	assert.NotContains(t, output, "Update Check")
}

func TestUpdateCommandInteractiveSelectsAndUpdatesCandidates(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	installedPath := filepath.Join(baseDir, ".agents", "skills", "go-pro")
	assert.NoErr(t, os.MkdirAll(installedPath, 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{
			SkillID:             "go-pro",
			QualifiedName:       "tools/go-pro",
			SourceQualifiedName: "repo-a/tools/go-pro",
			SourceID:            "repo-a",
			Version:             "1.0.0",
			Agents:              []string{"universal"},
		}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{{
		ID:                  "go-pro",
		QualifiedName:       "tools/go-pro",
		SourceQualifiedName: "repo-a/tools/go-pro",
		SourceID:            "repo-a",
		Version:             "2.0.0",
	}}))

	stub := &selectorStub{items: []termselect.Item{{Value: "repo-a/tools/go-pro"}}}
	prevSelector := newMultiSelector
	newMultiSelector = func() multiSelector { return stub }
	defer func() { newMultiSelector = prevSelector }()

	gotReqs := make([]updateapp.Req, 0)
	prevFactory := newUpdateService
	newUpdateService = func(configFile string, baseDir string) updateRunner {
		return updateRunnerStub{runFn: func(req updateapp.Req) (updateapp.Result, error) {
			gotReqs = append(gotReqs, req)
			return updateapp.Result{
				Updated: []installapp.RuntimeRecord{{
					Record:        lockpkg.Record{SkillID: "go-pro"},
					InstalledPath: installedPath,
				}},
			}, nil
		}}
	}
	defer func() { newUpdateService = prevFactory }()

	output := runAppInDirWithStdout(t, baseDir, []string{"update", "--interactive", "--agent", "universal"})

	assert.Contains(t, stub.got.Title, "Update")
	assert.Len(t, gotReqs, 1)
	if len(gotReqs) == 0 {
		return
	}
	assert.Eq(t, "repo-a/tools/go-pro", gotReqs[0].Target)
	assert.Contains(t, output, "updated go-pro "+installedPath)
}

func TestUpdateCommandInteractiveUsesTargetFilterBeforeSelecting(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "rust-pro"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{SkillID: "go-pro", QualifiedName: "tools/go-pro", SourceQualifiedName: "repo-a/tools/go-pro", SourceID: "repo-a", Version: "1.0.0", Agents: []string{"universal"}},
			{SkillID: "rust-pro", QualifiedName: "tools/rust-pro", SourceQualifiedName: "repo-a/tools/rust-pro", SourceID: "repo-a", Version: "1.0.0", Agents: []string{"universal"}},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", QualifiedName: "tools/go-pro", SourceQualifiedName: "repo-a/tools/go-pro", SourceID: "repo-a", Version: "2.0.0"},
		{ID: "rust-pro", QualifiedName: "tools/rust-pro", SourceQualifiedName: "repo-a/tools/rust-pro", SourceID: "repo-a", Version: "2.0.0"},
	}))

	stub := &selectorStub{items: []termselect.Item{{Value: "repo-a/tools/go-pro"}}}
	prevSelector := newMultiSelector
	newMultiSelector = func() multiSelector { return stub }
	defer func() { newMultiSelector = prevSelector }()

	prevFactory := newUpdateService
	newUpdateService = func(configFile string, baseDir string) updateRunner {
		return updateRunnerStub{runFn: func(req updateapp.Req) (updateapp.Result, error) {
			return updateapp.Result{}, nil
		}}
	}
	defer func() { newUpdateService = prevFactory }()

	runAppInDirWithStdout(t, baseDir, []string{"update", "--interactive", "--agent", "universal", "--target", "go-pro"})

	assert.Len(t, stub.got.Items, 1)
	if len(stub.got.Items) == 0 {
		return
	}
	assert.Eq(t, "repo-a/tools/go-pro", stub.got.Items[0].Value)
}

func TestUpdateCommandInteractiveNoCandidatesDoesNotOpenSelector(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{SkillID: "go-pro", SourceID: "repo-a", Version: "1.0.0", Agents: []string{"universal"}}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{{ID: "go-pro", SourceID: "repo-a", Version: "1.0.0"}}))

	stub := &selectorStub{}
	prevSelector := newMultiSelector
	newMultiSelector = func() multiSelector { return stub }
	defer func() { newMultiSelector = prevSelector }()

	output := runAppInDirWithStdout(t, baseDir, []string{"update", "--interactive", "--agent", "universal"})

	assert.Contains(t, output, "no update candidates")
	assert.False(t, stub.called)
}

func TestUpdateCommandInteractiveRejectsCheckMode(t *testing.T) {
	err := validateUpdateMode(true, true, false)

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "--check and --interactive are mutually exclusive")
}

func TestStatusCommand_PrintsSkillHealth(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "go-pro"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {{SkillID: "go-pro", SourceID: "gstack", Version: "1.0.0", Profile: "go-dev", Agents: []string{"universal"}}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{{ID: "go-pro", SourceID: "gstack", Version: "2.0.0"}}))

	output := runAppInDirWithStdout(t, baseDir, []string{"status", "--agent", "universal", "--scope", "project", "--profile", "go-dev"})

	assert.Contains(t, output, "Skill Status")
	assert.Contains(t, output, "outdated")
	assert.Contains(t, output, "go-pro")
	assert.Contains(t, output, "go-dev")
	assert.Contains(t, output, "1.0.0")
	assert.Contains(t, output, "2.0.0")
}

func TestStatusCommand_HonorsAgentAndScopeFilters(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	codexUserDir := filepath.Join(baseDir, "user-codex")
	universalProjectDir := filepath.Join(baseDir, ".agents")
	assert.NoErr(t, os.MkdirAll(filepath.Join(codexUserDir, "skills", "codex-user"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(universalProjectDir, "skills", "universal-project"), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["codex"] = cfg.AgentToolConfig{Dirname: ".codex", UserDir: codexUserDir, ProjectDir: filepath.Join(baseDir, ".codex")}
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: universalProjectDir}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		lockpkg.GlobalKey: {
			{SkillID: "codex-user", SourceID: "gstack", Version: "1.0.0", Agents: []string{"codex"}},
		},
		filepath.Clean(baseDir): {
			{SkillID: "universal-project", SourceID: "gstack", Version: "1.0.0", Agents: []string{"universal"}},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "codex-user", SourceID: "gstack", Version: "1.0.0"},
		{ID: "universal-project", SourceID: "gstack", Version: "1.0.0"},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"status", "--agent", "codex", "--scope", "user"})

	assert.Contains(t, output, "Skill Status")
	assert.Contains(t, output, "codex-user")
	assert.Contains(t, output, "codex")
	assert.Contains(t, output, "user")
	assert.NotContains(t, output, "universal-project")
	assert.NotContains(t, output, "project")
}

func TestStatusCommand_PrintsNoSkillsFoundWhenEmpty(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{}))

	output := runAppInDirWithStdout(t, baseDir, []string{"status", "--agent", "universal"})

	assert.Contains(t, output, "no skills found")
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

func TestInstallCommand_InstallModeFlagInstallsIndexedSkill(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexPath := filepath.Join(baseDir, "cache", "index.json")
	sourceDir := filepath.Join(baseDir, "source", "hello-skill")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "commands", "hello.txt"), []byte("hello"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = filepath.Join(baseDir, "skillc-install.lock")
	config.IndexFile = indexPath
	config.AgentTools["codex"] = cfg.AgentToolConfig{Dirname: ".codex", ProjectDir: filepath.Join(baseDir, ".agents")}
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

	output := runAppInDirWithStdout(t, baseDir, []string{"install", "--yes", "--install-mode", "copy", "--agent", "codex", "hello-skill"})

	assert.Contains(t, output, "hello-skill")
	data, err := os.ReadFile(filepath.Join(baseDir, ".agents", "skills", "hello-skill", "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))
}

func TestManageOptions_ResolveInstallMode(t *testing.T) {
	config := cfg.DefaultConfig()
	config.InstallMode = "symlink"

	assert.Eq(t, agentfs.ModeJunction, (&ManageOptions{InstallMode: "junction"}).resolveInstallMode(config))
	assert.Eq(t, agentfs.ModeSymlink, (&ManageOptions{InstallMode: "symlink"}).resolveInstallMode(config))
	assert.Eq(t, agentfs.ModeCopy, (&ManageOptions{InstallMode: "copy"}).resolveInstallMode(config))
	assert.Eq(t, agentfs.ModeCopy, (&ManageOptions{UseCopy: true, InstallMode: "symlink"}).resolveInstallMode(config))
	assert.Eq(t, agentfs.ModeSymlink, (&ManageOptions{}).resolveInstallMode(config))
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
	assert.Contains(t, output, filepath.Join(baseDir, "project-claude", "skills", "hello-skill"))
	data, err := os.ReadFile(filepath.Join(baseDir, "project-claude", "skills", "hello-skill", "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))
}

func TestInstallCommandInteractiveSelectsAndInstallsSkills(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexPath := filepath.Join(baseDir, "cache", "index.json")
	sourceDir := filepath.Join(baseDir, "source", "go-pro")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "commands", "go.txt"), []byte("go"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = filepath.Join(baseDir, "skillc-install.lock")
	config.IndexFile = indexPath
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexPath, []skill.Skill{{
		ID:                  "go-pro",
		Name:                "Go Pro",
		Version:             "1.0.0",
		SupportedAgents:     []string{"universal"},
		SourceID:            "repo-a",
		SourceType:          sourcepkg.TypeLocal,
		Collection:          "tools",
		QualifiedName:       "tools/go-pro",
		SourceQualifiedName: "repo-a/tools/go-pro",
		InstallEntry:        "commands",
		Path:                sourceDir,
	}}))

	stub := &selectorStub{items: []termselect.Item{{Value: "repo-a/tools/go-pro"}}}
	prevSelector := newMultiSelector
	newMultiSelector = func() multiSelector { return stub }
	defer func() { newMultiSelector = prevSelector }()

	output := runAppInDirWithStdout(t, baseDir, []string{"install", "--interactive", "--yes", "--agent", "universal"})

	assert.Contains(t, output, "Will install skills: go-pro")
	assert.Contains(t, output, "installed go-pro")
	assert.Contains(t, stub.got.Title, "Install")
	data, err := os.ReadFile(filepath.Join(baseDir, ".agents", "skills", "go-pro", "go.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "go", string(data))
}

func TestInstallCommandInteractiveUsesSkillArgAsSearchKeyword(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexPath := filepath.Join(baseDir, "cache", "index.json")
	goSourceDir := filepath.Join(baseDir, "source", "go-pro")
	reviewSourceDir := filepath.Join(baseDir, "source", "review")
	assert.NoErr(t, os.MkdirAll(filepath.Join(goSourceDir, "commands"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(reviewSourceDir, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(goSourceDir, "commands", "go.txt"), []byte("go"), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(reviewSourceDir, "commands", "review.txt"), []byte("review"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = filepath.Join(baseDir, "skillc-install.lock")
	config.IndexFile = indexPath
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexPath, []skill.Skill{
		{ID: "go-pro", Name: "Go Pro", Version: "1.0.0", SupportedAgents: []string{"universal"}, SourceID: "repo-a", SourceType: sourcepkg.TypeLocal, Collection: "tools", QualifiedName: "tools/go-pro", SourceQualifiedName: "repo-a/tools/go-pro", InstallEntry: "commands", Path: goSourceDir},
		{ID: "review", Name: "Review", Version: "1.0.0", SupportedAgents: []string{"universal"}, SourceID: "repo-a", SourceType: sourcepkg.TypeLocal, Collection: "tools", QualifiedName: "tools/review", SourceQualifiedName: "repo-a/tools/review", InstallEntry: "commands", Path: reviewSourceDir},
	}))

	stub := &selectorStub{items: []termselect.Item{{Value: "repo-a/tools/go-pro"}}}
	prevSelector := newMultiSelector
	newMultiSelector = func() multiSelector { return stub }
	defer func() { newMultiSelector = prevSelector }()

	output := runAppInDirWithStdout(t, baseDir, []string{"install", "--interactive", "--yes", "--agent", "universal", "go"})

	assert.Contains(t, output, "Will install skills: go-pro")
	assert.Len(t, stub.got.Items, 1)
	if len(stub.got.Items) == 0 {
		return
	}
	assert.Eq(t, "repo-a/tools/go-pro", stub.got.Items[0].Value)
	_, err := os.Stat(filepath.Join(baseDir, ".agents", "skills", "review", "review.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestInstallCommandInteractiveNoCandidatesDoesNotOpenSelector(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexPath := filepath.Join(baseDir, "cache", "index.json")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexPath
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexPath, []skill.Skill{}))

	stub := &selectorStub{}
	prevSelector := newMultiSelector
	newMultiSelector = func() multiSelector { return stub }
	defer func() { newMultiSelector = prevSelector }()

	output := runAppInDirWithStdout(t, baseDir, []string{"install", "--interactive", "--yes", "--agent", "universal", "go"})

	assert.Contains(t, output, "no skills found")
	assert.False(t, stub.called)
	_, err := os.Stat(lockFile)
	assert.True(t, os.IsNotExist(err))
}

func TestInstallCommandInteractiveNoSelectionDoesNotInstall(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexPath := filepath.Join(baseDir, "cache", "index.json")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := filepath.Join(baseDir, "source", "go-pro")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "commands", "go.txt"), []byte("go"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexPath
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexPath, []skill.Skill{{
		ID:                  "go-pro",
		Name:                "Go Pro",
		Version:             "1.0.0",
		SupportedAgents:     []string{"universal"},
		SourceID:            "repo-a",
		SourceType:          sourcepkg.TypeLocal,
		Collection:          "tools",
		QualifiedName:       "tools/go-pro",
		SourceQualifiedName: "repo-a/tools/go-pro",
		InstallEntry:        "commands",
		Path:                sourceDir,
	}}))

	stub := &selectorStub{}
	prevSelector := newMultiSelector
	newMultiSelector = func() multiSelector { return stub }
	defer func() { newMultiSelector = prevSelector }()

	output := runAppInDirWithStdout(t, baseDir, []string{"install", "--interactive", "--yes", "--agent", "universal"})

	assert.Contains(t, output, "no skills selected")
	assert.True(t, stub.called)
	_, err := os.Stat(filepath.Join(baseDir, ".agents", "skills", "go-pro", "go.txt"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(lockFile)
	assert.True(t, os.IsNotExist(err))
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
	data, err := os.ReadFile(filepath.Join(baseDir, "project-claude", "skills", "hello-skill", "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))
	_, err = os.Stat(filepath.Join(baseDir, "project-claude", "skills", "world-skill", "hello.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestInstallCommand_DoesNotAcceptCollectionFlag(t *testing.T) {
	app := newTestApp()
	install := findCommandByName(app, "install")
	assert.NotNil(t, install)
	if install == nil {
		return
	}

	assert.Nil(t, install.Flags.FSet().Lookup("collection"))
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
	_, err := os.Stat(filepath.Join(baseDir, "project-claude", "skills", "hello-skill", "hello.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestInstallCommand_RestoresFromLockFileWhenNoArgs(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := filepath.Join(baseDir, "cache", "hello-skill")
	commandsDir := filepath.Join(sourceDir, "commands")
	claudeInstalledPath := filepath.Join(baseDir, "project-claude", "skills", "hello-skill")
	agentsInstalledPath := filepath.Join(baseDir, ".agents", "skills", "hello-skill")
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
			Agents:       []string{"universal", "claude-code"},
		}},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"install"})

	assert.Contains(t, output, "restored hello-skill  agent=universal scope=project")
	assert.Contains(t, output, "restored hello-skill  agent=claude-code scope=project")
	claudeData, err := os.ReadFile(filepath.Join(claudeInstalledPath, "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "restored", string(claudeData))
	agentsData, err := os.ReadFile(filepath.Join(agentsInstalledPath, "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "restored", string(agentsData))
}

func TestSourceCollectionsCommand_PrintsSourceScopedCollectionSummary(t *testing.T) {
	baseDir := t.TempDir()
	config := cfg.DefaultConfig()
	config.IndexFile = filepath.Join(baseDir, "cache", "index.json")
	assert.NoErr(t, configstore.NewYAMLStore().Save(filepath.Join(baseDir, "skillc.yaml"), config))
	assert.NoErr(t, repoindex.NewStore().Save(config.IndexFile, []skill.Skill{
		{ID: "go-pro", Name: "Go Pro", Collection: "go", SourceID: "gstack", SourceName: "GStack"},
		{ID: "go-test", Name: "Go Test", Collection: "go", SourceID: "gstack", SourceName: "GStack"},
		{ID: "review", Name: "Review", Collection: "ops", SourceID: "team", SourceName: "Team"},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"source", "collections", "gstack"})

	assert.Contains(t, output, "GStack")
	assert.Contains(t, output, "go")
	assert.Contains(t, output, "2")
	assert.NotContains(t, output, "ops")
}

func TestSourceSkillsCommand_PrintsSourceScopedSkills(t *testing.T) {
	baseDir := t.TempDir()
	config := cfg.DefaultConfig()
	config.IndexFile = filepath.Join(baseDir, "cache", "index.json")
	assert.NoErr(t, configstore.NewYAMLStore().Save(filepath.Join(baseDir, "skillc.yaml"), config))
	assert.NoErr(t, repoindex.NewStore().Save(config.IndexFile, []skill.Skill{
		{ID: "go-pro", Name: "Go Pro", Description: "go helper", Collection: "go", SourceID: "gstack"},
		{ID: "review", Name: "Review", Description: "review helper", Collection: "ops", SourceID: "gstack"},
		{ID: "other", Name: "Other", Description: "other helper", Collection: "go", SourceID: "team"},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"source", "skills", "gstack", "--collection", "go"})

	assert.Contains(t, output, "go-pro")
	assert.Contains(t, output, "go helper")
	assert.NotContains(t, output, "review helper")
	assert.NotContains(t, output, "other helper")
}

func TestSourceSkillsCommand_DoesNotReusePreviousCollectionFlag(t *testing.T) {
	baseDir := t.TempDir()
	config := cfg.DefaultConfig()
	config.IndexFile = filepath.Join(baseDir, "cache", "index.json")
	assert.NoErr(t, configstore.NewYAMLStore().Save(filepath.Join(baseDir, "skillc.yaml"), config))
	assert.NoErr(t, repoindex.NewStore().Save(config.IndexFile, []skill.Skill{
		{ID: "go-pro", Name: "Go Pro", Description: "go helper", Collection: "go", SourceID: "gstack"},
		{ID: "review", Name: "Review", Description: "review helper", Collection: "ops", SourceID: "gstack"},
	}))

	output := runInDirWithStdout(t, baseDir, func() error {
		app := newTestApp()
		app.Run([]string{"source", "skills", "gstack", "--collection", "go"})
		app.Run([]string{"source", "skills", "gstack"})
		return nil
	})

	assert.Contains(t, output, "go helper")
	assert.Contains(t, output, "review helper")
}

func TestProfileCreateFromCollectionCommand(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Collection: "go"},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"profile", "create", "go-dev", "--from-collection", "gstack/go"})

	assert.Contains(t, output, "profile created: go-dev")
	loaded, err := configstore.NewYAMLStore().Load(configFile, baseDir)
	assert.NoErr(t, err)
	assert.Len(t, loaded.Profiles["go-dev"].Targets, 1)
}

func TestProfileCreateCommandRequiresExactlyOneSource(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack", Collection: "go"},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"profile", "create", "go-dev", "--from-installed", "--from-collection", "gstack/go"})

	assert.Contains(t, output, "use exactly one")
	loaded, err := configstore.NewYAMLStore().Load(configFile, baseDir)
	assert.NoErr(t, err)
	_, ok := loaded.Profiles["go-dev"]
	assert.False(t, ok)
}

func TestProfileCreateInteractiveSelectsSkills(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{
			ID:                  "go-pro",
			Name:                "Go Pro",
			SourceID:            "repo-a",
			Collection:          "tools",
			QualifiedName:       "tools/go-pro",
			SourceQualifiedName: "repo-a/tools/go-pro",
			Version:             "1.0.0",
			SupportedAgents:     []string{"universal"},
		},
		{
			ID:                  "review",
			Name:                "Review",
			SourceID:            "repo-a",
			Collection:          "ops",
			QualifiedName:       "ops/review",
			SourceQualifiedName: "repo-a/ops/review",
			Version:             "1.0.0",
			SupportedAgents:     []string{"universal"},
		},
	}))

	stub := &selectorStub{items: []termselect.Item{{Value: "repo-a/tools/go-pro"}}}
	prevSelector := newMultiSelector
	newMultiSelector = func() multiSelector { return stub }
	defer func() { newMultiSelector = prevSelector }()

	output := runAppInDirWithStdout(t, baseDir, []string{"profile", "create", "go-dev", "--interactive", "--agent", "universal", "--scope", "project"})

	assert.Contains(t, stub.got.Title, "Create profile go-dev")
	assert.Len(t, stub.got.Items, 2)
	assert.Contains(t, output, "profile created: go-dev")
	loaded, err := configstore.NewYAMLStore().Load(configFile, baseDir)
	assert.NoErr(t, err)
	got := loaded.Profiles["go-dev"]
	assert.Eq(t, "universal", got.DefaultAgent)
	assert.Eq(t, "project", got.DefaultScope)
	assert.Eq(t, []profile.Target{{Source: "repo-a", Skill: "go-pro"}}, got.Targets)
}

func TestProfileCreateInteractiveIsMutuallyExclusiveWithFromInstalled(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{{ID: "go-pro", SourceID: "repo-a"}}))

	output := runAppInDirWithStdout(t, baseDir, []string{"profile", "create", "go-dev", "--interactive", "--from-installed"})

	assert.Contains(t, output, "use exactly one")
	loaded, err := configstore.NewYAMLStore().Load(configFile, baseDir)
	assert.NoErr(t, err)
	_, ok := loaded.Profiles["go-dev"]
	assert.False(t, ok)
}

func TestProfileCreateInteractiveIsMutuallyExclusiveWithFromCollection(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{{ID: "go-pro", SourceID: "repo-a", Collection: "tools"}}))

	output := runAppInDirWithStdout(t, baseDir, []string{"profile", "create", "go-dev", "--interactive", "--from-collection", "repo-a/tools"})

	assert.Contains(t, output, "use exactly one")
	loaded, err := configstore.NewYAMLStore().Load(configFile, baseDir)
	assert.NoErr(t, err)
	_, ok := loaded.Profiles["go-dev"]
	assert.False(t, ok)
}

func TestProfileCreateInteractiveNoCandidatesDoesNotOpenSelector(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{}))

	stub := &selectorStub{}
	prevSelector := newMultiSelector
	newMultiSelector = func() multiSelector { return stub }
	defer func() { newMultiSelector = prevSelector }()

	runAppInDirWithStdout(t, baseDir, []string{"profile", "create", "go-dev", "--interactive", "--agent", "universal"})

	assert.False(t, stub.called)
	loaded, err := configstore.NewYAMLStore().Load(configFile, baseDir)
	assert.NoErr(t, err)
	_, ok := loaded.Profiles["go-dev"]
	assert.False(t, ok)
}

func TestProfileCreateInteractiveNoSelectionDoesNotCreateProfile(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{{
		ID:              "go-pro",
		SourceID:        "repo-a",
		SupportedAgents: []string{"universal"},
	}}))

	stub := &selectorStub{}
	prevSelector := newMultiSelector
	newMultiSelector = func() multiSelector { return stub }
	defer func() { newMultiSelector = prevSelector }()

	runAppInDirWithStdout(t, baseDir, []string{"profile", "create", "go-dev", "--interactive", "--agent", "universal"})

	assert.True(t, stub.called)
	loaded, err := configstore.NewYAMLStore().Load(configFile, baseDir)
	assert.NoErr(t, err)
	_, ok := loaded.Profiles["go-dev"]
	assert.False(t, ok)
}

func TestProfileApplyDryRunPrintsPlan(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.Profiles = map[string]profile.Profile{
		"go-dev": {Targets: []profile.Target{{Source: "gstack", Skill: "go-pro"}}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "gstack"},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"profile", "apply", "go-dev", "--dry-run"})

	assert.Contains(t, output, "Profile Plan")
	assert.Contains(t, output, "install")
	assert.Contains(t, output, "go-pro")
}

func TestProfileApplyCommandPrintsInstallFailures(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	indexFile := filepath.Join(baseDir, "index.json")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	sourceDir := filepath.Join(baseDir, "source", "review")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Review"), 0o644))

	config := cfg.DefaultConfig()
	config.IndexFile = indexFile
	config.LockFile = lockFile
	config.Profiles = map[string]profile.Profile{
		"go-dev": {
			InstallMode: "copy",
			Targets: []profile.Target{
				{Source: "gstack", Skill: "broken"},
				{Source: "gstack", Skill: "review"},
			},
		},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "broken", SourceID: "gstack", InstallEntry: ".", Path: filepath.Join(baseDir, "missing")},
		{ID: "review", SourceID: "gstack", InstallEntry: ".", Path: sourceDir},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"profile", "apply", "go-dev", "--yes"})

	assert.Contains(t, output, "installed review")
	assert.Contains(t, output, "install failed broken")
	assert.Contains(t, output, "profile apply failed")
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

func TestSourceAddCommandAcceptsDirectPathWithCustomIDAndName(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "skills")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))

	output := runAppInDirWithStdout(t, baseDir, []string{"source", "add", "--id", "gstack", "--name", "GStack Skills", sourceDir})

	assert.Contains(t, output, "gstack added")
	assert.Contains(t, output, "GStack Skills")
	config, err := configstore.NewYAMLStore().Load(filepath.Join(baseDir, "skillc.yaml"), baseDir)
	assert.NoErr(t, err)
	assert.Len(t, config.Sources, 1)
	if len(config.Sources) == 0 {
		return
	}
	assert.Eq(t, "gstack", config.Sources[0].ID)
	assert.Eq(t, "GStack Skills", config.Sources[0].Name)
}

func TestSourceAddGitCommandAcceptsCustomIDAndName(t *testing.T) {
	baseDir := t.TempDir()

	output := runAppInDirWithStdout(t, baseDir, []string{"source", "add", "git", "--id", "acme", "--name", "Acme Skills", "https://example.com/skills.git", "main"})

	assert.Contains(t, output, "acme added")
	assert.Contains(t, output, "Acme Skills")
}

func TestSourceInfoCommandPrintsDetails(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "skills")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	runAppInDirWithStdout(t, baseDir, []string{"source", "add", "--id", "gstack", "--name", "GStack Skills", sourceDir})

	output := runAppInDirWithStdout(t, baseDir, []string{"source", "info", "gst"})

	assert.Contains(t, output, "Source Info")
	assert.Contains(t, output, "gstack")
	assert.Contains(t, output, "GStack Skills")
	assert.Contains(t, output, sourceDir)
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
	assert.Contains(t, output, "Next, please run: skillc source sync repo")
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
	assert.Contains(t, syncOutput, "Synced ")
	assert.Contains(t, syncOutput, "ready")
}

func TestListCommand_ReturnsEmptyWhenLockFileMissing(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	config := cfg.DefaultConfig()
	config.LockFile = filepath.Join(baseDir, "skillc-install.lock")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))

	output := runAppInDirWithStdout(t, baseDir, []string{"list", "--agent", "claude-code"})

	assert.Eq(t, "no skills found\n", output)
}

func TestListCommand_ListsInstalledSkills(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	installedPath := filepath.Join(baseDir, "project-claude", "skills", "hello-skill")
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
	installedPath := filepath.Join(baseDir, "project-claude", "skills", "hello-skill")
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
	helloInstalled := filepath.Join(baseDir, "project-claude", "skills", "hello-skill")
	pinnedInstalled := filepath.Join(baseDir, "project-claude", "skills", "pinned-skill")
	brokenInstalled := filepath.Join(baseDir, "project-claude", "skills", "broken-skill")
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

	cleanupInstalled := filepath.Join(baseDir, "project-claude", "skills", "shared-skill")
	prevFactory := newUpdateService
	newUpdateService = func(configFile string, baseDir string) updateRunner {
		return updateRunnerStub{runFn: func(req updateapp.Req) (updateapp.Result, error) {
			return updateapp.Result{
				Updated:       []installapp.RuntimeRecord{{Record: lockpkg.Record{SkillID: "shared-skill"}, InstalledPath: cleanupInstalled}},
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

func TestUpdateCommand_AllProjectsCheckPrintsPlan(t *testing.T) {
	baseDir := t.TempDir()
	assert.NoErr(t, configstore.NewYAMLStore().Save(filepath.Join(baseDir, "skillc.yaml"), cfg.DefaultConfig(), baseDir))
	prevFactory := newProjectUpdateService
	newProjectUpdateService = func(configFile string, baseDir string) projectUpdateRunner {
		return projectUpdateRunnerStub{
			planFn: func(req projectupdateapp.Req) (projectupdateapp.Plan, error) {
				assert.Eq(t, "go-pro", req.Target)
				return projectupdateapp.Plan{
					Agent: "universal", Scope: "project", Target: "go-pro", CandidateCount: 1,
					Projects: []projectupdateapp.ProjectPlan{{
						ProjectID: "skillc",
						Path:      baseDir,
						Items:     []statusapp.Item{{SkillID: "go-pro", Agent: "universal", Status: statusapp.StatusOutdated, CurrentVersion: "1.0.0", LatestVersion: "2.0.0"}},
					}},
				}, nil
			},
		}
	}
	defer func() { newProjectUpdateService = prevFactory }()

	output := runAppInDirWithStdout(t, baseDir, []string{"update", "--all-projects", "--check", "--target", "go-pro"})

	assert.Contains(t, output, "Cross-Project Update Check")
	assert.Contains(t, output, "skillc")
	assert.Contains(t, output, "go-pro")
	assert.Contains(t, output, "2.0.0")
}

func TestUpdateCommand_AllProjectsRequiresConfirmation(t *testing.T) {
	baseDir := t.TempDir()
	assert.NoErr(t, configstore.NewYAMLStore().Save(filepath.Join(baseDir, "skillc.yaml"), cfg.DefaultConfig(), baseDir))
	runCalled := false
	prevFactory := newProjectUpdateService
	newProjectUpdateService = func(configFile string, baseDir string) projectUpdateRunner {
		return projectUpdateRunnerStub{
			planFn: func(req projectupdateapp.Req) (projectupdateapp.Plan, error) {
				return projectupdateapp.Plan{Agent: "universal", Scope: "project", CandidateCount: 1}, nil
			},
			runFn: func(req projectupdateapp.Req) (projectupdateapp.Result, error) {
				runCalled = true
				return projectupdateapp.Result{}, nil
			},
		}
	}
	defer func() { newProjectUpdateService = prevFactory }()

	output := runAppInDirWithInput(t, baseDir, []string{"update", "--all-projects"}, "n\n")

	assert.Contains(t, output, "cross-project update cancelled")
	assert.Eq(t, false, runCalled)
}

func TestUpdateCommand_AllProjectsYesRuns(t *testing.T) {
	baseDir := t.TempDir()
	assert.NoErr(t, configstore.NewYAMLStore().Save(filepath.Join(baseDir, "skillc.yaml"), cfg.DefaultConfig(), baseDir))
	runCalled := false
	prevFactory := newProjectUpdateService
	newProjectUpdateService = func(configFile string, baseDir string) projectUpdateRunner {
		return projectUpdateRunnerStub{
			planFn: func(req projectupdateapp.Req) (projectupdateapp.Plan, error) {
				return projectupdateapp.Plan{Agent: "universal", Scope: "project", CandidateCount: 1}, nil
			},
			runFn: func(req projectupdateapp.Req) (projectupdateapp.Result, error) {
				runCalled = true
				assert.Eq(t, true, req.Confirm)
				return projectupdateapp.Result{
					Plan: projectupdateapp.Plan{Agent: "universal", Scope: "project", CandidateCount: 1},
					Results: []projectupdateapp.ProjectResult{{
						ProjectID: "skillc",
						Path:      baseDir,
						Updated:   []installapp.RuntimeRecord{{Record: lockpkg.Record{SkillID: "go-pro", Version: "2.0.0"}, Agent: "universal", Scope: "project"}},
					}},
				}, nil
			},
		}
	}
	defer func() { newProjectUpdateService = prevFactory }()

	output := runAppInDirWithStdout(t, baseDir, []string{"update", "--all-projects", "--yes"})

	assert.Eq(t, true, runCalled)
	assert.Contains(t, output, "updated skillc go-pro")
}

func TestProjectCommand_AddListRemove(t *testing.T) {
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "demo")
	assert.NoErr(t, os.MkdirAll(projectDir, 0o755))
	configFile := filepath.Join(baseDir, "skillc.yaml")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, cfg.DefaultConfig(), baseDir))

	addOutput := runAppInDirWithStdout(t, baseDir, []string{"project", "add", "--id", "demo", "--name", "Demo", projectDir})
	assert.Contains(t, addOutput, "project added: demo")

	listOutput := runAppInDirWithStdout(t, baseDir, []string{"project", "list"})
	assert.Contains(t, listOutput, "demo")
	assert.Contains(t, listOutput, "Demo")

	removeOutput := runAppInDirWithStdout(t, baseDir, []string{"project", "remove", "demo"})
	assert.Contains(t, removeOutput, "project removed: demo")
}

func TestProjectCommand_ImportLock(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	projectDir := filepath.Join(baseDir, "project-a")
	assert.NoErr(t, os.MkdirAll(projectDir, 0o755))
	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectDir: {{SkillID: "go-pro", Agents: []string{"universal"}}},
	}))

	output := runAppInDirWithStdout(t, baseDir, []string{"project", "import-lock"})

	assert.Contains(t, output, "imported project-a")
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
	ccolor.SetOutput(stdoutW)
	defer func() {
		os.Stdout = oldStdout
		ccolor.SetOutput(oldStdout)
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
