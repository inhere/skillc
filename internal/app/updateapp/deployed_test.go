package updateapp

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/app/apputil"
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

func TestMergeEnabledModes(t *testing.T) {
	assert.True(t, mergeEnabled(Req{}))
	assert.True(t, mergeEnabled(Req{Merge: true}))
	assert.True(t, mergeEnabled(Req{Merge: true, Force: true}))
	assert.False(t, mergeEnabled(Req{Force: true}))
	assert.False(t, mergeEnabled(Req{NoMerge: true}))
	assert.False(t, mergeEnabled(Req{NoMerge: true, Force: true}))
}

func TestService_RunMergesByDefaultAndSkipsWithNoMerge(t *testing.T) {
	baseDir := t.TempDir()
	configFile, projectKey := prepareUpdateProject(t, baseDir)

	service := NewService(configFile, baseDir)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error { return nil }}
	mergeCalls := make([]bool, 0)
	service.newInstaller = func(_ string, _ cfg.Config, force bool) reinstallService {
		return reinstallServiceStub{
			reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error) {
				if !force {
					return installapp.RuntimeRecord{}, fmt.Errorf("%w: %s (rerun with --force to overwrite)", apputil.ErrLocalChanges, targetPath)
				}
				record := installapp.RuntimeRecord{Record: lockpkg.Record{SkillID: item.ID, Version: item.Version}, Agent: agentName, Scope: string(scope), InstalledPath: targetPath}
				record.BackupPath = filepath.Join(baseDir, "backups", item.ID)
				return record, nil
			},
			mergeFn: func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string, force bool) (installapp.RuntimeRecord, installapp.MergeResult, error) {
				mergeCalls = append(mergeCalls, force)
				record := installapp.RuntimeRecord{Record: lockpkg.Record{SkillID: item.ID, Version: item.Version}, Agent: agentName, Scope: string(scope), InstalledPath: targetPath}
				return record, installapp.MergeResult{
					Updated:   []string{"SKILL.md"},
					KeptLocal: []string{"run.md"},
				}, nil
			},
		}
	}

	// 默认：按文件合并，本地改动保留
	result, err := service.Run(Req{Scope: "project", WorkDir: baseDir, ProjectPaths: []string{projectKey}})
	assert.NoErr(t, err)
	assert.Eq(t, []bool{false}, mergeCalls)
	assert.Len(t, result.Updated, 1)
	assert.Len(t, result.Skipped, 0)
	assert.Len(t, result.Merged, 1)
	assert.Eq(t, []string{"run.md"}, result.Merged[0].KeptLocal)

	// --no-merge：回到跳过语义
	mergeCalls = mergeCalls[:0]
	skipped, err := service.Run(Req{Scope: "project", WorkDir: baseDir, ProjectPaths: []string{projectKey}, NoMerge: true})
	assert.NoErr(t, err)
	assert.Len(t, mergeCalls, 0)
	assert.Len(t, skipped.Updated, 0)
	assert.Len(t, skipped.Skipped, 1)
	assert.Contains(t, skipped.Skipped[0].Reason, "locally modified")
	assert.Contains(t, skipped.Skipped[0].Reason, "--force")

	// 单独 --force：整体覆盖，不合并
	forced, err := service.Run(Req{Scope: "project", WorkDir: baseDir, ProjectPaths: []string{projectKey}, Force: true})
	assert.NoErr(t, err)
	assert.Len(t, mergeCalls, 0)
	assert.Len(t, forced.Updated, 1)
	assert.Len(t, forced.BackedUp, 1)

	// --merge --force：合并但冲突取上游
	mergeCalls = mergeCalls[:0]
	forcedMerge, err := service.Run(Req{Scope: "project", WorkDir: baseDir, ProjectPaths: []string{projectKey}, Merge: true, Force: true})
	assert.NoErr(t, err)
	assert.Eq(t, []bool{true}, mergeCalls)
	assert.Len(t, forcedMerge.Updated, 1)
}

func TestService_RunReportsInstallFailuresSeparatelyFromLocalChanges(t *testing.T) {
	baseDir := t.TempDir()
	configFile, projectKey := prepareUpdateProject(t, baseDir)

	service := NewService(configFile, baseDir)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error { return nil }}
	service.newInstaller = func(_ string, _ cfg.Config, _ bool) reinstallService {
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error) {
			return installapp.RuntimeRecord{}, fmt.Errorf("copy failed: disk full")
		}}
	}

	result, err := service.Run(Req{Scope: "project", WorkDir: baseDir, ProjectPaths: []string{projectKey}})
	assert.NoErr(t, err)
	assert.Len(t, result.Skipped, 0)
	assert.Len(t, result.Failed, 1)
	assert.Contains(t, result.Failed[0].Reason, "disk full")
}

// prepareUpdateProject 准备一个「已安装 1.0.0、索引里是 2.0.0」的项目。
func prepareUpdateProject(t *testing.T, baseDir string) (string, string) {
	t.Helper()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	projectKey := filepath.Join(baseDir, "projects", "alpha")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.BackupDir = filepath.Join(baseDir, "cache", "backups")
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectKey: {
			{
				SkillID:      "hello-skill",
				SourceID:     "source-a",
				SourceType:   string(sourcepkg.TypeLocal),
				InstallEntry: "commands",
				InstallMode:  "copy",
				Agents:       []string{"claude-code"},
				Version:      "1.0.0",
			},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "hello-skill", SourceID: "source-a", SourceType: sourcepkg.TypeLocal, Version: "2.0.0", InstallEntry: "commands", Path: filepath.Join(baseDir, "source", "hello-skill")},
	}))
	return configFile, projectKey
}
