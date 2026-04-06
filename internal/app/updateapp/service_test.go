package updateapp

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

type sourceSyncerStub struct {
	syncFn func(id string) error
}

func (s sourceSyncerStub) Sync(id string) error {
	return s.syncFn(id)
}

type reinstallServiceStub struct {
	reinstallFn func(item skill.Skill, agentName string, scope agent.Scope, targetPath string) (lockpkg.Record, error)
}

func (s reinstallServiceStub) ReinstallAtPath(item skill.Skill, agentName string, scope agent.Scope, targetPath string) (lockpkg.Record, error) {
	return s.reinstallFn(item, agentName, scope, targetPath)
}

func TestService_RunUsesLockRecordsAndRecordedInstallPath(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	helloPath := filepath.Join(baseDir, ".claude", "skills", "hello-skill")
	worldPath := filepath.Join(baseDir, ".claude", "skills", "world-skill")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, []lockpkg.Record{
		{SkillID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", Agent: "claude-code", Scope: "project", SourceID: "source-a", InstallEntry: "commands", InstalledPath: helloPath},
		{SkillID: "world-skill", QualifiedName: "marketplaces/world-skill", SourceQualifiedName: "repo-a/marketplaces/world-skill", Agent: "claude-code", Scope: "project", SourceID: "source-a", InstallEntry: "commands", InstalledPath: worldPath},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", SourceID: "source-a", SourceType: sourcepkg.TypeLocal, Version: "2.0.0", InstallEntry: "commands", Path: filepath.Join(baseDir, "source", "hello-skill")},
		{ID: "world-skill", QualifiedName: "marketplaces/world-skill", SourceQualifiedName: "repo-a/marketplaces/world-skill", SourceID: "source-a", SourceType: sourcepkg.TypeLocal, Version: "2.1.0", InstallEntry: "commands", Path: filepath.Join(baseDir, "source", "world-skill")},
	}))

	service := NewService(configFile, baseDir)
	syncCalls := make([]string, 0)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error {
		syncCalls = append(syncCalls, id)
		return nil
	}}
	installCalls := make([]string, 0)
	service.newInstaller = func(path string) reinstallService {
		assert.Eq(t, lockFile, path)
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, targetPath string) (lockpkg.Record, error) {
			installCalls = append(installCalls, item.ID+"@"+targetPath)
			return lockpkg.Record{SkillID: item.ID, Version: item.Version, Agent: agentName, Scope: string(scope), InstalledPath: targetPath}, nil
		}}
	}

	result, err := service.Run(Req{Agent: "claude-code", Scope: "project", WorkDir: baseDir})
	assert.NoErr(t, err)
	assert.Eq(t, []string{"source-a"}, syncCalls)
	assert.Eq(t, []string{"hello-skill@" + helloPath, "world-skill@" + worldPath}, installCalls)
	assert.Len(t, result.Updated, 2)
	assert.Len(t, result.Skipped, 0)
	assert.Len(t, result.Failed, 0)
	assert.Eq(t, "2.0.0", result.Updated[0].Version)
	assert.Eq(t, "2.1.0", result.Updated[1].Version)
}

func TestService_RunLoadsIndexAfterSourceSync(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	installedPath := filepath.Join(baseDir, ".claude", "skills", "hello-skill")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, []lockpkg.Record{{
		SkillID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", Agent: "claude-code", Scope: "project", SourceID: "source-a", InstallEntry: "commands", InstalledPath: installedPath,
	}}))

	service := NewService(configFile, baseDir)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error {
		return repoindex.NewStore().Save(indexFile, []skill.Skill{{
			ID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", SourceID: "source-a", SourceType: sourcepkg.TypeLocal, Version: "3.0.0", InstallEntry: "commands", Path: filepath.Join(baseDir, "source", "hello-skill"),
		}})
	}}
	service.newInstaller = func(path string) reinstallService {
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, targetPath string) (lockpkg.Record, error) {
			return lockpkg.Record{SkillID: item.ID, Version: item.Version, Agent: agentName, Scope: string(scope), InstalledPath: targetPath}, nil
		}}
	}

	result, err := service.Run(Req{Agent: "claude-code", Scope: "project", WorkDir: baseDir})
	assert.NoErr(t, err)
	assert.Len(t, result.Updated, 1)
	assert.Eq(t, "3.0.0", result.Updated[0].Version)
	assert.Len(t, result.Failed, 0)
}

func TestService_RunAggregatesSyncAndReinstallFailures(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, []lockpkg.Record{
		{SkillID: "broken-skill", QualifiedName: "marketplaces/broken-skill", SourceQualifiedName: "repo-a/marketplaces/broken-skill", Agent: "claude-code", Scope: "project", SourceID: "source-a", InstalledPath: filepath.Join(baseDir, ".claude", "skills", "broken-skill")},
		{SkillID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", Agent: "claude-code", Scope: "project", SourceID: "source-b", InstalledPath: filepath.Join(baseDir, ".claude", "skills", "hello-skill")},
		{SkillID: "world-skill", QualifiedName: "marketplaces/world-skill", SourceQualifiedName: "repo-a/marketplaces/world-skill", Agent: "claude-code", Scope: "project", SourceID: "source-b", InstalledPath: filepath.Join(baseDir, ".claude", "skills", "world-skill")},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", SourceID: "source-b", SourceType: sourcepkg.TypeLocal, Version: "2.0.0", InstallEntry: "commands", Path: filepath.Join(baseDir, "source", "hello-skill")},
		{ID: "world-skill", QualifiedName: "marketplaces/world-skill", SourceQualifiedName: "repo-a/marketplaces/world-skill", SourceID: "source-b", SourceType: sourcepkg.TypeLocal, Version: "2.0.0", InstallEntry: "commands", Path: filepath.Join(baseDir, "source", "world-skill")},
	}))

	service := NewService(configFile, baseDir)
	syncCalls := make([]string, 0)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error {
		syncCalls = append(syncCalls, id)
		if id == "source-a" {
			return errors.New("sync failed")
		}
		return nil
	}}
	installCalls := make([]string, 0)
	service.newInstaller = func(path string) reinstallService {
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, targetPath string) (lockpkg.Record, error) {
			installCalls = append(installCalls, item.ID)
			if item.ID == "world-skill" {
				return lockpkg.Record{}, errors.New("copy failed")
			}
			return lockpkg.Record{SkillID: item.ID, Version: item.Version, Agent: agentName, Scope: string(scope), InstalledPath: targetPath}, nil
		}}
	}

	result, err := service.Run(Req{Agent: "claude-code", Scope: "project", WorkDir: baseDir})
	assert.NoErr(t, err)
	assert.Eq(t, []string{"source-a", "source-b"}, syncCalls)
	assert.Eq(t, []string{"hello-skill", "world-skill"}, installCalls)
	assert.Len(t, result.Updated, 1)
	assert.Eq(t, "hello-skill", result.Updated[0].SkillID)
	assert.Len(t, result.Failed, 2)
	assert.Eq(t, "broken-skill", result.Failed[0].SkillID)
	assert.Contains(t, result.Failed[0].Reason, "source sync failed")
	assert.Eq(t, "world-skill", result.Failed[1].SkillID)
	assert.Contains(t, result.Failed[1].Reason, "copy failed")
}


func TestService_RunFallsBackToSourceScopedInstalledDirNames(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	targetRoot := filepath.Join(baseDir, "project-claude", "skills")
	scopedDir := "repo-a--alpha--shared-skill"
	assert.NoErr(t, os.MkdirAll(filepath.Join(targetRoot, scopedDir), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "shared-skill", QualifiedName: "alpha/shared-skill", SourceQualifiedName: "repo-a/alpha/shared-skill", SourceID: "source-a", SourceType: sourcepkg.TypeLocal, Version: "1.0.0", InstallEntry: "commands", Path: filepath.Join(baseDir, "source", "shared-a")},
		{ID: "shared-skill", QualifiedName: "beta/shared-skill", SourceQualifiedName: "repo-b/beta/shared-skill", SourceID: "source-b", SourceType: sourcepkg.TypeLocal, Version: "1.0.0", InstallEntry: "commands", Path: filepath.Join(baseDir, "source", "shared-b")},
	}))

	service := NewService(configFile, baseDir)
	syncCalls := make([]string, 0)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error {
		syncCalls = append(syncCalls, id)
		return nil
	}}
	installCalls := make([]string, 0)
	service.newInstaller = func(path string) reinstallService {
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, targetPath string) (lockpkg.Record, error) {
			installCalls = append(installCalls, item.SourceID+"@"+targetPath)
			return lockpkg.Record{SkillID: item.ID, Version: item.Version, Agent: agentName, Scope: string(scope), InstalledPath: targetPath, SourceID: item.SourceID}, nil
		}}
	}

	result, err := service.Run(Req{Agent: "claude-code", Scope: "project", WorkDir: baseDir})
	assert.NoErr(t, err)
	assert.Eq(t, []string{"source-a"}, syncCalls)
	assert.Eq(t, []string{"source-a@" + filepath.Join(targetRoot, scopedDir)}, installCalls)
	assert.Len(t, result.Updated, 1)
	assert.Eq(t, "shared-skill", result.Updated[0].SkillID)
	assert.Len(t, result.Skipped, 0)
	assert.Len(t, result.Failed, 0)
}

func TestService_RunFallsBackToMixedPlainAndScopedDuplicateDirs(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	targetRoot := filepath.Join(baseDir, "project-claude", "skills")
	plainDir := "shared-skill"
	scopedDir := "repo-b--beta--shared-skill"
	assert.NoErr(t, os.MkdirAll(filepath.Join(targetRoot, plainDir), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(targetRoot, scopedDir), 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "shared-skill", QualifiedName: "alpha/shared-skill", SourceQualifiedName: "repo-a/alpha/shared-skill", SourceID: "source-a", SourceType: sourcepkg.TypeLocal, Version: "1.0.0", InstallEntry: "commands", Path: filepath.Join(baseDir, "source", "shared-a")},
		{ID: "shared-skill", QualifiedName: "beta/shared-skill", SourceQualifiedName: "repo-b/beta/shared-skill", SourceID: "source-b", SourceType: sourcepkg.TypeLocal, Version: "1.0.0", InstallEntry: "commands", Path: filepath.Join(baseDir, "source", "shared-b")},
	}))

	service := NewService(configFile, baseDir)
	syncCalls := make([]string, 0)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error {
		syncCalls = append(syncCalls, id)
		return nil
	}}
	installCalls := make([]string, 0)
	service.newInstaller = func(path string) reinstallService {
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, targetPath string) (lockpkg.Record, error) {
			installCalls = append(installCalls, item.SourceID+"@"+targetPath)
			return lockpkg.Record{SkillID: item.ID, Version: item.Version, Agent: agentName, Scope: string(scope), InstalledPath: targetPath, SourceID: item.SourceID, SourceQualifiedName: item.SourceQualifiedName}, nil
		}}
	}

	result, err := service.Run(Req{Agent: "claude-code", Scope: "project", WorkDir: baseDir})
	assert.NoErr(t, err)
	sort.Strings(syncCalls)
	sort.Strings(installCalls)
	assert.Eq(t, []string{"source-a", "source-b"}, syncCalls)
	assert.Eq(t, []string{
		"source-a@" + filepath.Join(targetRoot, plainDir),
		"source-b@" + filepath.Join(targetRoot, scopedDir),
	}, installCalls)
	assert.Len(t, result.Updated, 2)
	assert.Len(t, result.Skipped, 0)
	assert.Len(t, result.Failed, 0)
}
