package installapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/infra/agentfs"
)

// 别名安装（claude）必须按正式名称写入 lock，并按正式名称定位目录。
func TestService_InstallWritesCanonicalAgentName(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := createSkillSource(t, baseDir, "source", "hello.txt", "hello")
	config := testConfigWithClaudeAlias(baseDir)
	service := NewService(lockFile).WithRuntime(config, baseDir).WithInstallMode(agentfs.ModeCopy)
	projectKey := copyInstallScopeKey(t, baseDir)

	record, err := service.Install(copyInstallSkill(sourceDir), "claude", agent.ScopeProject, projectKey, copyInstallRoot(t, config, baseDir))
	assert.NoErr(t, err)
	assert.Eq(t, "claude-code", record.Agent)
	assert.Eq(t, filepath.Join(baseDir, ".claude", "skills", "hello-skill"), record.InstalledPath)

	locks := mustLoadLockFile(t, service, lockFile)
	assert.Eq(t, []string{"claude-code"}, locks[projectKey][0].Agents)

	// 用别名卸载同一个正式名记录
	assert.NoErr(t, service.Uninstall("hello-skill", "claude", agent.ScopeProject))
	locks = mustLoadLockFile(t, service, lockFile)
	assert.Len(t, locks[projectKey], 0)
}

// 旧记录里的别名在写入时被统一为正式名称，不会产生重复 agent。
func TestService_InstallCanonicalizesLegacyAliasRecords(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := createSkillSource(t, baseDir, "source", "hello.txt", "hello")
	config := testConfigWithClaudeAlias(baseDir)
	projectKey := copyInstallScopeKey(t, baseDir)
	service := NewService(lockFile).WithRuntime(config, baseDir).WithInstallMode(agentfs.ModeCopy)

	assert.NoErr(t, service.store.Save(lockFile, lockpkg.File{
		projectKey: {
			{
				SkillID:             "hello-skill",
				QualifiedName:       "marketplaces/hello-skill",
				SourceQualifiedName: "repo-a/marketplaces/hello-skill",
				SourceID:            "local-demo",
				SourceType:          "local",
				InstallEntry:        "commands",
				Agents:              []string{"claude"},
			},
		},
	}))

	record, err := service.Install(copyInstallSkill(sourceDir), "claude-code", agent.ScopeProject, projectKey, copyInstallRoot(t, config, baseDir))
	assert.NoErr(t, err)
	assert.Eq(t, "claude-code", record.Agent)

	locks := mustLoadLockFile(t, service, lockFile)
	assert.Len(t, locks[projectKey], 1)
	assert.Eq(t, []string{"claude-code"}, locks[projectKey][0].Agents)

	if _, err := os.Stat(filepath.Join(baseDir, ".claude", "skills", "hello-skill", "hello.txt")); err != nil {
		t.Fatalf("expected installed file, got %v", err)
	}
}

// testConfigWithClaudeAlias 在测试配置里注册 claude → claude-code 别名。
func testConfigWithClaudeAlias(baseDir string) cfg.Config {
	config := testConfig(baseDir)
	tool := config.AgentTools["claude-code"]
	tool.Aliases = []string{"claude"}
	config.AgentTools["claude-code"] = tool
	return config
}
