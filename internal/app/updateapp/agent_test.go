package updateapp

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

// 旧记录里存的是别名 agents/claude 时，-a 用正式名或别名都能匹配到。
func TestService_RunMatchesLegacyAgentAliases(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	projectKey := filepath.Join(baseDir, "projects", "alpha")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.BackupDir = filepath.Join(baseDir, "cache", "backups")
	universal := config.AgentTools["universal"]
	universal.UserDir = filepath.Join(baseDir, "user-agents")
	universal.ProjectDir = filepath.Join(baseDir, "project-agents")
	config.AgentTools["universal"] = universal
	claude := config.AgentTools["claude-code"]
	claude.UserDir = filepath.Join(baseDir, "user-claude")
	claude.ProjectDir = filepath.Join(baseDir, "project-claude")
	config.AgentTools["claude-code"] = claude
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectKey: {
			{
				SkillID:      "hello-skill",
				SourceID:     "source-a",
				SourceType:   string(sourcepkg.TypeLocal),
				InstallEntry: "commands",
				Agents:       []string{"agents", "claude"},
				Version:      "1.0.0",
			},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "hello-skill", SourceID: "source-a", SourceType: sourcepkg.TypeLocal, Version: "2.0.0", InstallEntry: "commands", Path: filepath.Join(baseDir, "source", "hello-skill")},
	}))

	cases := []struct {
		query string
		want  []string
	}{
		{query: "universal", want: []string{"universal"}},
		{query: "agents", want: []string{"universal"}},
		{query: "claude", want: []string{"claude-code"}},
		{query: "claude-code", want: []string{"claude-code"}},
		{query: "claude,agents", want: []string{"claude-code", "universal"}},
		{query: "", want: []string{"claude-code", "universal"}},
	}
	for _, item := range cases {
		t.Run("query="+item.query, func(t *testing.T) {
			service := NewService(configFile, baseDir)
			service.syncer = sourceSyncerStub{syncFn: func(id string) error { return nil }}
			installed := make([]string, 0)
			service.newInstaller = func(_ string, _ cfg.Config, _ bool) reinstallService {
				return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error) {
					installed = append(installed, agentName)
					return installapp.RuntimeRecord{Record: lockpkg.Record{SkillID: item.ID, Version: item.Version}, Agent: agentName, Scope: string(scope), InstalledPath: targetPath}, nil
				}}
			}

			result, err := service.Run(Req{Agent: item.query, Scope: "project", WorkDir: baseDir, ProjectPaths: []string{projectKey}})
			assert.NoErr(t, err)
			assert.Len(t, result.Updated, len(item.want))
			assert.Len(t, result.Skipped, 0)
			sort.Strings(installed)
			assert.Eq(t, item.want, installed)
		})
	}
}
