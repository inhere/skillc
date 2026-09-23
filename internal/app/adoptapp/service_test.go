package adoptapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/app/apputil"
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/agentfs"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/hashx"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

func TestService_RunWritesLocalChangesBackToSource(t *testing.T) {
	env := newAdoptFixture(t)
	installed := env.install(t)

	// 本地改动：改 run.md，新增 extra.md
	assert.NoErr(t, os.WriteFile(filepath.Join(installed, "run.md"), []byte("run local"), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(installed, "extra.md"), []byte("extra"), 0o644))

	service := NewService(env.configFile, env.baseDir)
	plan, err := service.Plan(env.req(false))
	assert.NoErr(t, err)
	assert.True(t, plan.Writable)
	assert.Eq(t, env.commandsDir, plan.SourcePath)
	assert.Len(t, plan.Items, 2)
	assert.Eq(t, ActionWrite, adoptAction(plan, "run.md"))
	assert.Eq(t, ActionWrite, adoptAction(plan, "extra.md"))

	result, err := service.Run(env.req(false))
	assert.NoErr(t, err)
	assert.Eq(t, "hello-skill", result.SkillID)
	assert.NotEq(t, "", result.BackupPath)

	// 本地内容写回源目录，源文件有备份
	assertFileContent(t, filepath.Join(env.commandsDir, "run.md"), "run local")
	assertFileContent(t, filepath.Join(env.commandsDir, "extra.md"), "extra")
	assertFileContent(t, filepath.Join(result.BackupPath, "run.md"), "run v1")

	// 基线刷新为源内容：安装目录不再被判定为本地改动
	records, err := lockstore.NewStore().Load(env.lockFile)
	assert.NoErr(t, err)
	record := records[env.projectKey][0]
	incoming, err := hashx.Deployed(env.commandsDir)
	assert.NoErr(t, err)
	assert.Eq(t, incoming.Sum, record.InstalledChecksum)
	assert.Eq(t, incoming.Files, record.InstalledFiles)

	// 索引已按新源内容重建，记录跟随最新索引版本
	indexed, err := repoindex.NewStore().Load(env.indexFile)
	assert.NoErr(t, err)
	assert.Len(t, indexed, 1)
	assert.Eq(t, indexed[0].Version, record.Version)
	assert.NotEq(t, "1.0.0", record.Version)
}

func TestService_PlanDoesNotWriteSource(t *testing.T) {
	env := newAdoptFixture(t)
	installed := env.install(t)
	assert.NoErr(t, os.WriteFile(filepath.Join(installed, "run.md"), []byte("run local"), 0o644))

	service := NewService(env.configFile, env.baseDir)
	result, err := service.Run(env.req(true))
	assert.NoErr(t, err)
	assert.True(t, result.Writable)
	assert.Eq(t, "", result.BackupPath)
	assertFileContent(t, filepath.Join(env.commandsDir, "run.md"), "run v1")
}

func TestService_RefusesGitAndRegistrySources(t *testing.T) {
	cases := []struct {
		name       string
		sourceType sourcepkg.Type
		reason     string
	}{
		{name: "git source", sourceType: sourcepkg.TypeGit, reason: "git source"},
		{name: "registry source", sourceType: sourcepkg.TypeRegistry, reason: "registry skill"},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			env := newAdoptFixture(t)
			env.install(t)
			records, err := lockstore.NewStore().Load(env.lockFile)
			assert.NoErr(t, err)
			group := records[env.projectKey]
			group[0].SourceType = string(item.sourceType)
			records[env.projectKey] = group
			assert.NoErr(t, lockstore.NewStore().Save(env.lockFile, records))

			plan, err := NewService(env.configFile, env.baseDir).Plan(env.req(false))
			assert.NoErr(t, err)
			assert.False(t, plan.Writable)
			assert.Contains(t, plan.Reason, item.reason)
		})
	}
}

type adoptFixture struct {
	baseDir     string
	configFile  string
	lockFile    string
	indexFile   string
	projectKey  string
	sourceRoot  string
	skillDir    string
	commandsDir string
	config      cfg.Config
}

func (f adoptFixture) req(dryRun bool) Req {
	return Req{Target: "hello-skill", Agent: "claude-code", Scope: "project", WorkDir: f.baseDir, DryRun: dryRun}
}

// install 以 copy 模式安装一次，生成带文件清单的 lock 记录，并写入索引。
func (f adoptFixture) install(t *testing.T) string {
	t.Helper()
	config := f.config
	projectKey, err := apputil.ResolveScopeKey(agent.ScopeProject, f.baseDir)
	assert.NoErr(t, err)
	targetRoot, err := agent.ResolveInstallPath(config, f.baseDir, "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)

	installer := installapp.NewService(f.lockFile).WithRuntime(config, f.baseDir).WithInstallMode(agentfs.ModeCopy)
	record, err := installer.Install(adoptSkill(f.skillDir), "claude-code", agent.ScopeProject, projectKey, targetRoot)
	assert.NoErr(t, err)

	assert.NoErr(t, repoindex.NewStore().Save(f.indexFile, []skill.Skill{{
		ID: "hello-skill", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal,
		InstallEntry: "commands", Path: f.skillDir, Version: "1.1.0", Checksum: "checksum-1.1.0",
	}}))
	return record.InstalledPath
}

func newAdoptFixture(t *testing.T) adoptFixture {
	t.Helper()
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	sourceRoot := filepath.Join(baseDir, "source")
	skillDir := filepath.Join(sourceRoot, "hello-skill")
	commandsDir := filepath.Join(skillDir, "commands")
	writeAdoptFile(t, filepath.Join(skillDir, "SKILL.md"), "---\nname: hello-skill\ndescription: adopt fixture\ninstall_entry: commands\n---\n\n# demo\n")
	writeAdoptFile(t, filepath.Join(commandsDir, "run.md"), "run v1")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.BackupDir = filepath.Join(baseDir, "cache", "backups")
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, ".claude")}
	config.Sources = []sourcepkg.Source{{ID: "local-demo", Name: "demo", Type: sourcepkg.TypeLocal, Path: sourceRoot, Status: "ready"}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))

	projectKey, err := apputil.ResolveScopeKey(agent.ScopeProject, baseDir)
	assert.NoErr(t, err)
	return adoptFixture{
		baseDir:     baseDir,
		configFile:  configFile,
		lockFile:    lockFile,
		indexFile:   indexFile,
		projectKey:  projectKey,
		sourceRoot:  sourceRoot,
		skillDir:    skillDir,
		commandsDir: commandsDir,
		config:      config,
	}
}

func adoptSkill(skillDir string) skill.Skill {
	return skill.Skill{
		ID:                  "hello-skill",
		QualifiedName:       "marketplaces/hello-skill",
		SourceQualifiedName: "repo-a/marketplaces/hello-skill",
		Version:             "1.0.0",
		SourceID:            "local-demo",
		SourceType:          sourcepkg.TypeLocal,
		InstallEntry:        "commands",
		Path:                skillDir,
	}
}

func adoptAction(plan Plan, path string) Action {
	for _, item := range plan.Items {
		if item.Path == path {
			return item.Action
		}
	}
	return ""
}

func writeAdoptFile(t *testing.T, path string, content string) {
	t.Helper()
	assert.NoErr(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NoErr(t, os.WriteFile(path, []byte(content), 0o644))
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	assert.NoErr(t, err)
	assert.Eq(t, want, string(data))
}
