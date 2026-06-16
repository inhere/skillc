package updateapp

import (
	"errors"
	"os"
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

type sourceSyncerStub struct {
	syncFn func(id string) error
}

func (s sourceSyncerStub) Sync(id string) error {
	return s.syncFn(id)
}

type reinstallServiceStub struct {
	reinstallFn func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error)
}

func (s reinstallServiceStub) ReinstallAtPath(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error) {
	return s.reinstallFn(item, agentName, scope, scopeKey, targetPath)
}

func TestService_RunExpandsGroupedLockRecordsPerAgentAndProjectScopePath(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	projectKey := filepath.Join(baseDir, "projects", "alpha")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	config.AgentTools["codex"] = cfg.AgentToolConfig{Dirname: ".codex", UserDir: filepath.Join(baseDir, "user-codex"), ProjectDir: filepath.Join(baseDir, "project-codex")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectKey: {
			{
				SkillID:             "hello-skill",
				QualifiedName:       "marketplaces/hello-skill",
				SourceQualifiedName: "repo-a/marketplaces/hello-skill",
				SourceID:            "source-a",
				SourceType:          string(sourcepkg.TypeLocal),
				InstallEntry:        "commands",
				Agents:              []string{"claude-code", "codex"},
			},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", SourceID: "source-a", SourceType: sourcepkg.TypeLocal, Version: "2.0.0", InstallEntry: "commands", Path: filepath.Join(baseDir, "source", "hello-skill")},
	}))

	service := NewService(configFile, baseDir)
	syncCalls := make([]string, 0)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error {
		syncCalls = append(syncCalls, id)
		return nil
	}}
	installCalls := make([]string, 0)
	service.newInstaller = func(path string, _ cfg.Config) reinstallService {
		assert.Eq(t, lockFile, path)
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error) {
			installCalls = append(installCalls, agentName+"|"+string(scope)+"|"+scopeKey+"|"+targetPath)
			return installapp.RuntimeRecord{Record: lockpkg.Record{SkillID: item.ID, Version: item.Version}, Agent: agentName, Scope: string(scope), InstalledPath: targetPath}, nil
		}}
	}

	result, err := service.Run(Req{Scope: "project", WorkDir: baseDir, ProjectPaths: []string{projectKey}})
	assert.NoErr(t, err)

	claudeRoot, err := agent.ResolveInstallPath(config, projectKey, "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)
	codexRoot, err := agent.ResolveInstallPath(config, projectKey, "codex", agent.ScopeProject)
	assert.NoErr(t, err)
	installDir := "hello-skill"
	sort.Strings(installCalls)
	assert.Eq(t, []string{"source-a"}, syncCalls)
	assert.Eq(t, []string{
		"claude-code|project|" + projectKey + "|" + filepath.Join(claudeRoot, installDir),
		"codex|project|" + projectKey + "|" + filepath.Join(codexRoot, installDir),
	}, installCalls)
	assert.Len(t, result.Candidates, 2)
	assert.Len(t, result.Updated, 2)
	assert.Len(t, result.Failed, 0)
}

func TestService_RunUsesSourceAwareCandidateMatchingForGroupedRecords(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	projectKey := filepath.Join(baseDir, "projects", "alpha")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	config.AgentTools["codex"] = cfg.AgentToolConfig{Dirname: ".codex", UserDir: filepath.Join(baseDir, "user-codex"), ProjectDir: filepath.Join(baseDir, "project-codex")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectKey: {
			{
				SkillID:             "shared-skill",
				QualifiedName:       "beta/shared-skill",
				SourceQualifiedName: "repo-b/beta/shared-skill",
				SourceID:            "source-b",
				SourceType:          string(sourcepkg.TypeLocal),
				InstallEntry:        "commands",
				Agents:              []string{"claude-code", "codex"},
			},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "shared-skill", QualifiedName: "alpha/shared-skill", SourceQualifiedName: "repo-a/alpha/shared-skill", SourceID: "source-a", SourceType: sourcepkg.TypeLocal, Version: "9.9.9", InstallEntry: "commands", Path: filepath.Join(baseDir, "source", "shared-a")},
		{ID: "shared-skill", QualifiedName: "beta/shared-skill", SourceQualifiedName: "repo-b/beta/shared-skill", SourceID: "source-b", SourceType: sourcepkg.TypeLocal, Version: "2.0.0", InstallEntry: "commands", Path: filepath.Join(baseDir, "source", "shared-b")},
	}))

	service := NewService(configFile, baseDir)
	syncCalls := make([]string, 0)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error {
		syncCalls = append(syncCalls, id)
		return nil
	}}
	installCalls := make([]string, 0)
	service.newInstaller = func(path string, _ cfg.Config) reinstallService {
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error) {
			installCalls = append(installCalls, item.SourceID+"|"+item.Version+"|"+agentName)
			return installapp.RuntimeRecord{Record: lockpkg.Record{SkillID: item.ID, Version: item.Version, SourceID: item.SourceID}, Agent: agentName, Scope: string(scope), InstalledPath: targetPath}, nil
		}}
	}

	result, err := service.Run(Req{Target: "shared-skill", Scope: "project", WorkDir: baseDir, ProjectPaths: []string{projectKey}})
	assert.NoErr(t, err)

	sort.Strings(installCalls)
	assert.Eq(t, []string{"source-b"}, syncCalls)
	assert.Eq(t, []string{"source-b|2.0.0|claude-code", "source-b|2.0.0|codex"}, installCalls)
	assert.Len(t, result.Updated, 2)
	assert.Eq(t, "source-b", result.Updated[0].SourceID)
}

func TestService_RunUsesGlobalScopeKeyForUserScopeUpdates(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		lockpkg.GlobalKey: {
			{
				SkillID:             "hello-skill",
				QualifiedName:       "marketplaces/hello-skill",
				SourceQualifiedName: "repo-a/marketplaces/hello-skill",
				SourceID:            "source-a",
				SourceType:          string(sourcepkg.TypeLocal),
				InstallEntry:        "commands",
				Agents:              []string{"claude-code"},
			},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", SourceID: "source-a", SourceType: sourcepkg.TypeLocal, Version: "2.0.0", InstallEntry: "commands", Path: filepath.Join(baseDir, "source", "hello-skill")},
	}))

	service := NewService(configFile, baseDir)
	installCalls := make([]string, 0)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error { return nil }}
	service.newInstaller = func(path string, _ cfg.Config) reinstallService {
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error) {
			installCalls = append(installCalls, string(scope)+"|"+scopeKey+"|"+targetPath)
			return installapp.RuntimeRecord{Record: lockpkg.Record{SkillID: item.ID, Version: item.Version}, Agent: agentName, Scope: string(scope), InstalledPath: targetPath}, nil
		}}
	}

	result, err := service.Run(Req{Agent: "claude-code", Scope: "user", WorkDir: filepath.Join(baseDir, "project-a")})
	assert.NoErr(t, err)

	userRoot, err := agent.ResolveInstallPath(config, filepath.Join(baseDir, "project-a"), "claude-code", agent.ScopeUser)
	assert.NoErr(t, err)
	assert.Eq(t, []string{"user|" + lockpkg.GlobalKey + "|" + filepath.Join(userRoot, "hello-skill")}, installCalls)
	assert.Len(t, result.Updated, 1)
}

func TestService_RunKeepsInstalledPathWhenQualifiedNameChanges(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	projectKey := filepath.Join(baseDir, "projects", "alpha")
	oldQualifiedName := "repo-a/alpha/shared-skill"
	newQualifiedName := "repo-a/renamed/shared-skill"

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectKey: {
			{
				SkillID:             "shared-skill",
				QualifiedName:       "alpha/shared-skill",
				SourceQualifiedName: oldQualifiedName,
				SourceID:            "source-a",
				SourceType:          string(sourcepkg.TypeLocal),
				InstallEntry:        "commands",
				Version:             "1.0.0",
				Agents:              []string{"claude-code"},
			},
		},
	}))
	targetRoot, err := agent.ResolveInstallPath(config, projectKey, "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)
	oldPath := filepath.Join(targetRoot, "shared-skill")
	assert.NoErr(t, os.MkdirAll(oldPath, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(oldPath, "stale.txt"), []byte("stale"), 0o644))
	sourceDir := createSkillSource(t, baseDir, filepath.Join("source", "shared-skill"), "payload.txt", "updated")
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "shared-skill", QualifiedName: "renamed/shared-skill", SourceQualifiedName: newQualifiedName, SourceID: "source-a", SourceType: sourcepkg.TypeLocal, Version: "2.0.0", InstallEntry: "commands", Path: sourceDir},
	}))

	service := NewService(configFile, baseDir)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error { return nil }}

	result, err := service.Run(Req{Agent: "claude-code", Scope: "project", WorkDir: baseDir, ProjectPaths: []string{projectKey}})
	assert.NoErr(t, err)
	assert.Len(t, result.Updated, 1)

	newPath := filepath.Join(targetRoot, "shared-skill")
	assert.Eq(t, newPath, result.Updated[0].InstalledPath)
	data, err := os.ReadFile(filepath.Join(newPath, "payload.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "updated", string(data))

	locks, err := lockstore.NewStore().Load(lockFile)
	assert.NoErr(t, err)
	assert.Len(t, locks[projectKey], 1)
	assert.Eq(t, "renamed/shared-skill", locks[projectKey][0].QualifiedName)
	assert.Eq(t, newQualifiedName, locks[projectKey][0].SourceQualifiedName)
	assert.Eq(t, []string{"claude-code"}, locks[projectKey][0].Agents)
}

func TestService_RunKeepsInstalledPathWhenReinstallFailsAfterQualifiedNameChange(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	projectKey := filepath.Join(baseDir, "projects", "alpha")
	oldQualifiedName := "repo-a/alpha/shared-skill"
	newQualifiedName := "repo-a/renamed/shared-skill"

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectKey: {
			{
				SkillID:             "shared-skill",
				QualifiedName:       "alpha/shared-skill",
				SourceQualifiedName: oldQualifiedName,
				SourceID:            "source-a",
				SourceType:          string(sourcepkg.TypeLocal),
				InstallEntry:        "commands",
				Version:             "1.0.0",
				Agents:              []string{"claude-code"},
			},
		},
	}))
	targetRoot, err := agent.ResolveInstallPath(config, projectKey, "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)
	oldPath := filepath.Join(targetRoot, "shared-skill")
	assert.NoErr(t, os.MkdirAll(oldPath, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(oldPath, "stale.txt"), []byte("stale"), 0o644))
	sourceDir := createSkillSource(t, baseDir, filepath.Join("source", "shared-skill"), "payload.txt", "updated")
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "shared-skill", QualifiedName: "renamed/shared-skill", SourceQualifiedName: newQualifiedName, SourceID: "source-a", SourceType: sourcepkg.TypeLocal, Version: "2.0.0", InstallEntry: "commands", Path: sourceDir},
	}))

	service := NewService(configFile, baseDir)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error { return nil }}
	service.newInstaller = func(path string, _ cfg.Config) reinstallService {
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error) {
			return installapp.RuntimeRecord{}, errors.New("copy failed")
		}}
	}

	result, err := service.Run(Req{Agent: "claude-code", Scope: "project", WorkDir: baseDir, ProjectPaths: []string{projectKey}})
	assert.NoErr(t, err)
	assert.Len(t, result.Updated, 0)
	assert.Len(t, result.UpdateFailed, 1)
	assert.Eq(t, "shared-skill", result.UpdateFailed[0].SkillID)

	data, err := os.ReadFile(filepath.Join(oldPath, "stale.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "stale", string(data))
	_, err = os.Stat(filepath.Join(targetRoot, "shared-skill"))
	assert.NoErr(t, err)

	locks, err := lockstore.NewStore().Load(lockFile)
	assert.NoErr(t, err)
	assert.Len(t, locks[projectKey], 1)
	assert.Eq(t, "alpha/shared-skill", locks[projectKey][0].QualifiedName)
	assert.Eq(t, oldQualifiedName, locks[projectKey][0].SourceQualifiedName)
}

func TestService_RunReportsCleanupFailureWhenInstalledPathChanges(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	projectKey := filepath.Join(baseDir, "projects", "alpha")
	oldQualifiedName := "repo-a/alpha/shared-skill"
	newQualifiedName := "repo-a/renamed/shared-skill"

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.AgentTools["claude-code"] = cfg.AgentToolConfig{Dirname: ".claude", UserDir: filepath.Join(baseDir, "user-claude"), ProjectDir: filepath.Join(baseDir, "project-claude")}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectKey: {
			{
				SkillID:             "shared-skill",
				QualifiedName:       "alpha/shared-skill",
				SourceQualifiedName: oldQualifiedName,
				SourceID:            "source-a",
				SourceType:          string(sourcepkg.TypeLocal),
				InstallEntry:        "commands",
				Version:             "1.0.0",
				Agents:              []string{"claude-code"},
			},
		},
	}))
	targetRoot, err := agent.ResolveInstallPath(config, projectKey, "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)
	oldPath := filepath.Join(targetRoot, "shared-skill")
	assert.NoErr(t, os.MkdirAll(oldPath, 0o755))
	sourceDir := createSkillSource(t, baseDir, filepath.Join("source", "shared-skill"), "payload.txt", "updated")
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "shared-skill", QualifiedName: "renamed/shared-skill", SourceQualifiedName: newQualifiedName, SourceID: "source-a", SourceType: sourcepkg.TypeLocal, Version: "2.0.0", InstallEntry: "commands", Path: sourceDir},
	}))

	service := NewService(configFile, baseDir)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error { return nil }}
	service.removeAll = func(path string) error {
		if path == oldPath {
			return errors.New("cleanup failed")
		}
		return os.RemoveAll(path)
	}

	result, err := service.Run(Req{Agent: "claude-code", Scope: "project", WorkDir: baseDir, ProjectPaths: []string{projectKey}})
	assert.NoErr(t, err)
	assert.Len(t, result.Updated, 1)
	assert.Len(t, result.UpdateFailed, 0)
	assert.Len(t, result.CleanupFailed, 0)

	data, err := os.ReadFile(filepath.Join(oldPath, "payload.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "updated", string(data))

	locks, err := lockstore.NewStore().Load(lockFile)
	assert.NoErr(t, err)
	assert.Len(t, locks[projectKey], 1)
	assert.Eq(t, "renamed/shared-skill", locks[projectKey][0].QualifiedName)
	assert.Eq(t, newQualifiedName, locks[projectKey][0].SourceQualifiedName)
}

func TestService_RunSkipsPinnedGroupedRecordForRequestedAgent(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	projectKey := filepath.Join(baseDir, "projects", "alpha")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectKey: {
			{
				SkillID:             "pinned-skill",
				QualifiedName:       "marketplaces/pinned-skill",
				SourceQualifiedName: "repo-a/marketplaces/pinned-skill",
				SourceID:            "source-a",
				Agents:              []string{"claude-code", "codex"},
				Pinned:              true,
			},
		},
	}))

	service := NewService(configFile, baseDir)
	syncCalls := make([]string, 0)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error {
		syncCalls = append(syncCalls, id)
		return nil
	}}
	installCalled := false
	service.newInstaller = func(path string, _ cfg.Config) reinstallService {
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error) {
			installCalled = true
			return installapp.RuntimeRecord{}, nil
		}}
	}

	result, err := service.Run(Req{Target: "pinned-skill", Agent: "claude-code", Scope: "project", WorkDir: baseDir, ProjectPaths: []string{projectKey}})
	assert.NoErr(t, err)
	assert.Len(t, result.Updated, 0)
	assert.Len(t, result.Skipped, 1)
	assert.Eq(t, "pinned-skill", result.Skipped[0].SkillID)
	assert.Eq(t, []string{}, syncCalls)
	assert.Eq(t, false, installCalled)
}

func TestService_RunAggregatesGroupedSyncAndReinstallFailures(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	projectKey := filepath.Join(baseDir, "projects", "alpha")

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectKey: {
			{SkillID: "broken-skill", QualifiedName: "marketplaces/broken-skill", SourceQualifiedName: "repo-a/marketplaces/broken-skill", SourceID: "source-a", Agents: []string{"claude-code", "codex"}},
			{SkillID: "hello-skill", QualifiedName: "marketplaces/hello-skill", SourceQualifiedName: "repo-a/marketplaces/hello-skill", SourceID: "source-b", Agents: []string{"claude-code"}},
			{SkillID: "world-skill", QualifiedName: "marketplaces/world-skill", SourceQualifiedName: "repo-a/marketplaces/world-skill", SourceID: "source-b", Agents: []string{"codex"}},
		},
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
	service.newInstaller = func(path string, _ cfg.Config) reinstallService {
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error) {
			installCalls = append(installCalls, item.ID+"|"+agentName)
			if item.ID == "world-skill" {
				return installapp.RuntimeRecord{}, errors.New("copy failed")
			}
			return installapp.RuntimeRecord{Record: lockpkg.Record{SkillID: item.ID, Version: item.Version}, Agent: agentName, Scope: string(scope), InstalledPath: targetPath}, nil
		}}
	}

	result, err := service.Run(Req{Scope: "project", WorkDir: baseDir, ProjectPaths: []string{projectKey}})
	assert.NoErr(t, err)
	assert.Eq(t, []string{"source-a", "source-b"}, syncCalls)
	assert.Eq(t, []string{"hello-skill|claude-code", "world-skill|codex"}, installCalls)
	assert.Len(t, result.Updated, 1)
	assert.Eq(t, "hello-skill", result.Updated[0].SkillID)
	assert.Len(t, result.Failed, 3)
	assert.Eq(t, "broken-skill", result.Failed[0].SkillID)
	assert.Eq(t, "broken-skill", result.Failed[1].SkillID)
	assert.Eq(t, "world-skill", result.Failed[2].SkillID)
}

func TestService_RunProjectScopeDefaultsToCurrentWorkDirOnly(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	projectA := filepath.Join(baseDir, "project-a")
	projectB := filepath.Join(baseDir, "project-b")
	assert.NoErr(t, os.MkdirAll(projectA, 0o755))
	assert.NoErr(t, os.MkdirAll(projectB, 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectA: {{SkillID: "go-pro", SourceID: "source-a", Agents: []string{"universal"}}},
		projectB: {{SkillID: "review", SourceID: "source-a", Agents: []string{"universal"}}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "source-a", Version: "2.0.0", Path: createSkillSource(t, baseDir, filepath.Join("source", "go-pro"), "payload.txt", "go")},
		{ID: "review", SourceID: "source-a", Version: "2.0.0", Path: createSkillSource(t, baseDir, filepath.Join("source", "review"), "payload.txt", "review")},
	}))

	service := NewService(configFile, baseDir)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error { return nil }}
	updated := make([]string, 0)
	service.newInstaller = func(path string, _ cfg.Config) reinstallService {
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error) {
			updated = append(updated, scopeKey+"|"+item.ID)
			return installapp.RuntimeRecord{Record: lockpkg.Record{SkillID: item.ID, Version: item.Version}, Agent: agentName, Scope: string(scope), InstalledPath: targetPath}, nil
		}}
	}

	result, err := service.Run(Req{Scope: "project", WorkDir: projectA})

	assert.NoErr(t, err)
	assert.Len(t, result.Updated, 1)
	assert.Eq(t, []string{projectA + "|go-pro"}, updated)
}

func TestService_RunProjectScopeAllUsesExplicitProjectPaths(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	projectA := filepath.Join(baseDir, "project-a")
	projectB := filepath.Join(baseDir, "project-b")
	assert.NoErr(t, os.MkdirAll(projectA, 0o755))
	assert.NoErr(t, os.MkdirAll(projectB, 0o755))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectA: {{SkillID: "go-pro", SourceID: "source-a", Agents: []string{"universal"}}},
		projectB: {{SkillID: "review", SourceID: "source-a", Agents: []string{"universal"}}},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "go-pro", SourceID: "source-a", Version: "2.0.0", Path: createSkillSource(t, baseDir, filepath.Join("source", "go-pro-2"), "payload.txt", "go")},
		{ID: "review", SourceID: "source-a", Version: "2.0.0", Path: createSkillSource(t, baseDir, filepath.Join("source", "review-2"), "payload.txt", "review")},
	}))

	service := NewService(configFile, baseDir)
	service.syncer = sourceSyncerStub{syncFn: func(id string) error { return nil }}
	updated := make([]string, 0)
	service.newInstaller = func(path string, _ cfg.Config) reinstallService {
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error) {
			updated = append(updated, scopeKey+"|"+item.ID)
			return installapp.RuntimeRecord{Record: lockpkg.Record{SkillID: item.ID, Version: item.Version}, Agent: agentName, Scope: string(scope), InstalledPath: targetPath}, nil
		}}
	}

	result, err := service.Run(Req{Scope: "project", WorkDir: baseDir, All: true, ProjectPaths: []string{projectB}})

	assert.NoErr(t, err)
	assert.Len(t, result.Updated, 1)
	assert.Eq(t, []string{projectB + "|review"}, updated)
}

func TestService_RunRejectsAmbiguousDuplicatePlainDirs(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	targetRoot := filepath.Join(baseDir, "project-claude", "skills")
	plainDir := "shared-skill"
	assert.NoErr(t, os.MkdirAll(filepath.Join(targetRoot, plainDir), 0o755))

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
	service.newInstaller = func(path string, _ cfg.Config) reinstallService {
		return reinstallServiceStub{reinstallFn: func(item skill.Skill, agentName string, scope agent.Scope, scopeKey string, targetPath string) (installapp.RuntimeRecord, error) {
			installCalls = append(installCalls, item.SourceID+"@"+targetPath)
			return installapp.RuntimeRecord{Record: lockpkg.Record{SkillID: item.ID, Version: item.Version, SourceID: item.SourceID, SourceQualifiedName: item.SourceQualifiedName}, Agent: agentName, Scope: string(scope), InstalledPath: targetPath}, nil
		}}
	}

	result, err := service.Run(Req{Agent: "claude-code", Scope: "project", WorkDir: baseDir})
	assert.NoErr(t, err)
	assert.Eq(t, []string{}, syncCalls)
	assert.Eq(t, []string{}, installCalls)
	assert.Len(t, result.Updated, 0)
	assert.Len(t, result.Skipped, 1)
	assert.Eq(t, "shared-skill", result.Skipped[0].SkillID)
	assert.Eq(t, "ambiguous index match", result.Skipped[0].Reason)
	assert.Len(t, result.Failed, 0)
}

func createSkillSource(t *testing.T, baseDir string, sourceDir string, fileName string, content string) string {
	t.Helper()
	root := filepath.Join(baseDir, sourceDir)
	commandsDir := filepath.Join(root, "commands")
	assert.NoErr(t, os.MkdirAll(commandsDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(commandsDir, fileName), []byte(content), 0o644))
	return root
}
