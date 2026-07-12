package installapp

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/skillc/internal/app/searchapp"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
)

type skillLookupFunc func(id string) ([]skill.Skill, error)

func (f skillLookupFunc) Resolve(id string) ([]skill.Skill, error) {
	return f(id)
}

func TestService_RunInstallsIndexedSkill(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := createSkillSource(t, baseDir, "source", "hello.txt", "hello")
	service := NewService(lockFile)
	config := testConfig(baseDir)

	result, err := service.Run(config, InstallReq{
		SkillID: "hello-skill",
		Agent:   "claude-code",
		Scope:   "project",
		WorkDir: baseDir,
	}, skillLookupFunc(func(id string) ([]skill.Skill, error) {
		return []skill.Skill{testSkill(sourceDir, "hello-skill", "repo-a/marketplaces/hello-skill", "local-demo")}, nil
	}))
	assert.NoErr(t, err)
	assert.Len(t, result.Installed, 1)
	assert.Len(t, result.Restored, 0)
	assert.Eq(t, "hello-skill", result.Installed[0].SkillID)
	assert.Eq(t, "claude-code", result.Installed[0].Agent)
	assert.Eq(t, "project", result.Installed[0].Scope)
	assert.Eq(t, filepath.Join(baseDir, ".claude", "skills", "hello-skill"), result.Installed[0].InstalledPath)

	data, err := os.ReadFile(filepath.Join(result.Installed[0].InstalledPath, "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))

	projectKey, err := resolveScopeKey(agent.ScopeProject, baseDir)
	assert.NoErr(t, err)
	locks := mustLoadLockFile(t, service, lockFile)
	assert.Len(t, locks[projectKey], 1)
	assert.Eq(t, []string{"claude-code"}, locks[projectKey][0].Agents)
	assert.Eq(t, "commands", locks[projectKey][0].InstallEntry)
}

func TestService_RunInstallsUserScopeUnderGlobalKey(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := createSkillSource(t, baseDir, "source", "hello.txt", "hello")
	service := NewService(lockFile)

	result, err := service.RunResolved(testConfig(baseDir), InstallReq{
		Agent:   "claude-code",
		Scope:   "user",
		WorkDir: baseDir,
	}, []skill.Skill{testSkill(sourceDir, "hello-skill", "repo-a/marketplaces/hello-skill", "local-demo")}, nil)
	assert.NoErr(t, err)
	assert.Len(t, result.Installed, 1)
	assert.Eq(t, "user", result.Installed[0].Scope)

	locks := mustLoadLockFile(t, service, lockFile)
	assert.Len(t, locks[lockpkg.GlobalKey], 1)
	assert.Eq(t, []string{"claude-code"}, locks[lockpkg.GlobalKey][0].Agents)
}

func TestService_InstallAggregatesAgentsWithoutDuplicates(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := createSkillSource(t, baseDir, "source", "hello.txt", "hello")
	service := NewService(lockFile)
	item := testSkill(sourceDir, "hello-skill", "repo-a/marketplaces/hello-skill", "local-demo")
	projectKey, err := resolveScopeKey(agent.ScopeProject, baseDir)
	assert.NoErr(t, err)

	claudeRoot, err := agent.ResolveInstallPath(testConfig(baseDir), baseDir, "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)
	codexRoot, err := agent.ResolveInstallPath(testConfig(baseDir), baseDir, "codex", agent.ScopeProject)
	assert.NoErr(t, err)

	first, err := service.Install(item, "claude-code", agent.ScopeProject, projectKey, claudeRoot)
	assert.NoErr(t, err)
	second, err := service.Install(item, "codex", agent.ScopeProject, projectKey, codexRoot)
	assert.NoErr(t, err)
	third, err := service.Install(item, "claude-code", agent.ScopeProject, projectKey, claudeRoot)
	assert.NoErr(t, err)

	assert.NotEq(t, first.InstalledPath, second.InstalledPath)
	assert.Eq(t, first.InstalledPath, third.InstalledPath)

	locks := mustLoadLockFile(t, service, lockFile)
	assert.Len(t, locks[projectKey], 1)
	assert.Eq(t, []string{"claude-code", "codex"}, locks[projectKey][0].Agents)
}

func TestService_RunRequiresSkillLookupForInstall(t *testing.T) {
	service := NewService(filepath.Join(t.TempDir(), "skillc-install.lock"))
	_, err := service.Run(cfg.Config{}, InstallReq{
		SkillID: "hello-skill",
		Agent:   "claude-code",
		Scope:   "project",
		WorkDir: t.TempDir(),
	}, nil)
	assert.Err(t, err)
	assert.Contains(t, err.Error(), "skill lookup is required")
}

func TestService_RunReturnsLookupErrors(t *testing.T) {
	service := NewService(filepath.Join(t.TempDir(), "skillc-install.lock"))
	_, err := service.Run(cfg.Config{}, InstallReq{
		SkillID: "missing",
		Agent:   "claude-code",
		Scope:   "project",
	}, skillLookupFunc(func(id string) ([]skill.Skill, error) {
		return nil, fmt.Errorf("skill not found: %s", id)
	}))
	assert.Err(t, err)
	assert.Contains(t, err.Error(), "skill not found")
}

func TestService_InstallMultiContinuesAfterInstallFailure(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	targetRoot, err := agent.ResolveInstallPath(testConfig(baseDir), baseDir, "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)
	projectKey, err := resolveScopeKey(agent.ScopeProject, baseDir)
	assert.NoErr(t, err)
	goodSourceDir := createSkillSource(t, baseDir, filepath.Join("source", "hello-skill"), "hello.txt", "hello")

	service := NewService(lockFile)
	result, err := service.InstallMulti([]skill.Skill{
		{
			ID:                  "broken-skill",
			QualifiedName:       "marketplaces/broken-skill",
			SourceQualifiedName: "repo-a/marketplaces/broken-skill",
			Version:             "1.0.0",
			SourceID:            "local-demo",
			SourceType:          sourcepkg.TypeLocal,
			InstallEntry:        "commands",
			Path:                filepath.Join(baseDir, "missing"),
		},
		testSkill(goodSourceDir, "hello-skill", "repo-a/marketplaces/hello-skill", "local-demo"),
	}, "claude-code", agent.ScopeProject, projectKey, targetRoot, "")
	assert.NoErr(t, err)
	assert.Len(t, result.Installed, 1)
	assert.Len(t, result.Failed, 1)
	assert.Eq(t, "broken-skill", result.Failed[0].SkillID)
	assert.Contains(t, result.Failed[0].Reason, "missing")

	locks := mustLoadLockFile(t, service, lockFile)
	assert.Len(t, locks[projectKey], 1)
	assert.Eq(t, "hello-skill", locks[projectKey][0].SkillID)

	data, err := os.ReadFile(filepath.Join(result.Installed[0].InstalledPath, "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))
}

func TestService_RunResolvedReturnsInstalledAndResolveFailures(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := createSkillSource(t, baseDir, filepath.Join("source", "hello-skill"), "hello.txt", "hello")
	service := NewService(lockFile)

	result, err := service.RunResolved(testConfig(baseDir), InstallReq{
		Agent:   "claude-code",
		Scope:   "project",
		WorkDir: baseDir,
	}, []skill.Skill{testSkill(sourceDir, "hello-skill", "repo-a/marketplaces/hello-skill", "local-demo")}, []searchapp.TargetError{{
		Target: "missing-skill",
		Reason: "skill not found: missing-skill",
	}})
	assert.NoErr(t, err)
	assert.Len(t, result.Installed, 1)
	assert.Eq(t, "hello-skill", result.Installed[0].SkillID)
	assert.Len(t, result.ResolveFailed, 1)
	assert.Eq(t, "missing-skill", result.ResolveFailed[0].Target)
	assert.Len(t, result.InstallFailed, 0)
	assert.Len(t, result.Restored, 0)
}

func TestService_RunResolvedRecordsProfileName(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := createSkillSource(t, baseDir, filepath.Join("source", "go-pro"), "SKILL.md", "# Go Pro")
	service := NewService(lockFile)

	_, err := service.RunResolved(testConfig(baseDir), InstallReq{
		Agent:   "claude-code",
		Scope:   "project",
		WorkDir: baseDir,
		Profile: "go-dev",
	}, []skill.Skill{testSkill(sourceDir, "go-pro", "gstack/go-pro", "gstack")}, nil)
	assert.NoErr(t, err)

	projectKey, err := resolveScopeKey(agent.ScopeProject, baseDir)
	assert.NoErr(t, err)
	locks := mustLoadLockFile(t, service, lockFile)
	assert.Len(t, locks[projectKey], 1)
	assert.Eq(t, "go-dev", locks[projectKey][0].Profile)
}

func TestService_ReinstallAtPathUpdatesExistingLockRecord(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := createSkillSource(t, baseDir, filepath.Join("source", "hello-skill"), "hello.txt", "updated")
	service := NewService(lockFile)
	service.now = func() time.Time { return time.Date(2026, 4, 4, 9, 0, 0, 0, time.UTC) }
	projectKey, err := resolveScopeKey(agent.ScopeProject, baseDir)
	assert.NoErr(t, err)
	targetRoot, err := agent.ResolveInstallPath(testConfig(baseDir), baseDir, "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)
	targetPath := installTargetPath(testSkill(sourceDir, "hello-skill", "repo-a/marketplaces/hello-skill", "local-demo"), targetRoot)
	assert.NoErr(t, service.store.Save(lockFile, lockpkg.File{
		projectKey: {
			{
				SkillID:             "hello-skill",
				QualifiedName:       "marketplaces/hello-skill",
				SourceQualifiedName: "repo-a/marketplaces/hello-skill",
				Version:             "1.0.0",
				SourceID:            "local-demo",
				SourceType:          "local",
				Profile:             "go-dev",
				InstallEntry:        "commands",
				Agents:              []string{"claude-code"},
				InstalledAt:         time.Date(2026, 4, 1, 1, 0, 0, 0, time.UTC),
				UpdatedAt:           time.Date(2026, 4, 1, 1, 0, 0, 0, time.UTC),
			},
		},
	}))

	record, err := service.ReinstallAtPath(skill.Skill{
		ID:                  "hello-skill",
		QualifiedName:       "marketplaces/hello-skill",
		SourceQualifiedName: "repo-a/marketplaces/hello-skill",
		Version:             "1.1.0",
		SourceID:            "local-demo",
		SourceType:          sourcepkg.TypeLocal,
		InstallEntry:        "commands",
		Path:                sourceDir,
	}, "claude-code", agent.ScopeProject, projectKey, targetPath)
	assert.NoErr(t, err)
	assert.Eq(t, targetPath, record.InstalledPath)
	assert.Eq(t, "1.1.0", record.Version)
	assert.Eq(t, time.Date(2026, 4, 1, 1, 0, 0, 0, time.UTC), record.InstalledAt)
	assert.Eq(t, time.Date(2026, 4, 4, 9, 0, 0, 0, time.UTC), record.UpdatedAt)

	data, err := os.ReadFile(filepath.Join(targetPath, "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "updated", string(data))

	locks := mustLoadLockFile(t, service, lockFile)
	assert.Len(t, locks[projectKey], 1)
	assert.Eq(t, "1.1.0", locks[projectKey][0].Version)
	assert.Eq(t, "go-dev", locks[projectKey][0].Profile)
	assert.Eq(t, []string{"claude-code"}, locks[projectKey][0].Agents)
}

func TestService_InstallRejectsSameIDFromDifferentSources(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	service := NewService(lockFile)
	projectKey, err := resolveScopeKey(agent.ScopeProject, baseDir)
	assert.NoErr(t, err)
	targetRoot, err := agent.ResolveInstallPath(testConfig(baseDir), baseDir, "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)
	firstSourceDir := createSkillSource(t, baseDir, filepath.Join("source-a", "ship"), "source.txt", "repo-a")
	secondSourceDir := createSkillSource(t, baseDir, filepath.Join("source-b", "ship"), "source.txt", "repo-b")

	first, err := service.Install(skill.Skill{ID: "ship", QualifiedName: "shared/ship", SourceQualifiedName: "repo-a/shared/ship", Version: "1.0.0", SourceID: "src-a", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: firstSourceDir}, "claude-code", agent.ScopeProject, projectKey, targetRoot)
	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(targetRoot, "ship"), first.InstalledPath)

	second, err := service.Install(skill.Skill{ID: "ship", QualifiedName: "shared/ship", SourceQualifiedName: "repo-b/shared/ship", Version: "1.0.0", SourceID: "src-b", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: secondSourceDir}, "claude-code", agent.ScopeProject, projectKey, targetRoot)
	assert.Err(t, err)
	assert.Contains(t, err.Error(), "ship")
	assert.Eq(t, RuntimeRecord{}, second)

	data, err := os.ReadFile(filepath.Join(first.InstalledPath, "source.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "repo-a", string(data))

	locks := mustLoadLockFile(t, service, lockFile)
	assert.Len(t, locks[projectKey], 1)
	assert.Eq(t, "repo-a/shared/ship", locks[projectKey][0].SourceQualifiedName)
}

func TestService_InstallForceReplacesOnlyRequestedAgent(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	service := NewService(lockFile)
	projectKey, err := resolveScopeKey(agent.ScopeProject, baseDir)
	assert.NoErr(t, err)
	config := testConfig(baseDir)
	codexRoot, err := agent.ResolveInstallPath(config, baseDir, "codex", agent.ScopeProject)
	assert.NoErr(t, err)
	claudeRoot, err := agent.ResolveInstallPath(config, baseDir, "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)
	firstSourceDir := createSkillSource(t, baseDir, filepath.Join("source-a", "ship"), "source.txt", "repo-a")
	secondSourceDir := createSkillSource(t, baseDir, filepath.Join("source-b", "ship"), "source.txt", "repo-b")
	first := skill.Skill{ID: "ship", QualifiedName: "shared/ship", SourceQualifiedName: "repo-a/shared/ship", SourceID: "src-a", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: firstSourceDir}
	second := skill.Skill{ID: "ship", QualifiedName: "shared/ship", SourceQualifiedName: "repo-b/shared/ship", SourceID: "src-b", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: secondSourceDir}

	_, err = service.Install(first, "codex", agent.ScopeProject, projectKey, codexRoot)
	assert.NoErr(t, err)
	_, err = service.Install(first, "claude-code", agent.ScopeProject, projectKey, claudeRoot)
	assert.NoErr(t, err)
	_, err = service.WithForce(true).Install(second, "codex", agent.ScopeProject, projectKey, codexRoot)
	assert.NoErr(t, err)

	data, err := os.ReadFile(filepath.Join(codexRoot, "ship", "source.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "repo-b", string(data))
	data, err = os.ReadFile(filepath.Join(claudeRoot, "ship", "source.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "repo-a", string(data))
	locks := mustLoadLockFile(t, service, lockFile)
	assert.Len(t, locks[projectKey], 2)
	for _, record := range locks[projectKey] {
		if record.SourceID == "src-a" {
			assert.Eq(t, []string{"claude-code"}, record.Agents)
		} else {
			assert.Eq(t, "src-b", record.SourceID)
			assert.Eq(t, []string{"codex"}, record.Agents)
		}
	}
}

func TestService_InstallWritesDriftMetadataToLock(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	targetRoot := filepath.Join(baseDir, "target")
	sourceDir := filepath.Join(baseDir, "source", "go-pro")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("content"), 0o644))

	svc := NewService(lockFile)
	_, err := svc.Install(skill.Skill{
		ID: "go-pro", Version: "1.0.0", SourceID: "gstack", SourceType: sourcepkg.TypeGit,
		SourceResolvedRef: "deadbeefcafebabe", Checksum: "abc123", InstallEntry: ".", Path: sourceDir,
	}, "universal", agent.ScopeProject, baseDir, targetRoot)

	assert.NoErr(t, err)
	locks := mustLoadLockFile(t, svc, lockFile)
	assert.Eq(t, "abc123", locks[baseDir][0].Checksum)
	assert.Eq(t, "deadbeefcafebabe", locks[baseDir][0].SourceResolvedRef)
}

func TestService_InstallWritesRegistryProvenanceToLock(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	targetRoot := filepath.Join(baseDir, "target")
	sourceDir := filepath.Join(baseDir, "cache", "go-pro")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Go Pro"), 0o644))

	_, err := NewService(lockFile).Install(skill.Skill{
		ID: "go-pro", Version: "1.2.0", SourceID: "team", SourceType: sourcepkg.TypeRegistry,
		RegistryEntryID: "go-pro", RegistryURL: "https://example.com/registry.json",
		SourceURL: "https://example.com/skills.git", SourceRef: "main",
		DownloadURL:  "https://example.com/go-pro.zip",
		InstallEntry: ".", Path: sourceDir,
	}, "codex", agent.ScopeProject, baseDir, targetRoot)

	assert.NoErr(t, err)
	locks := mustLoadLockFile(t, NewService(lockFile), lockFile)
	assert.Eq(t, "registry", locks[baseDir][0].SourceType)
	assert.Eq(t, "go-pro", locks[baseDir][0].RegistryEntryID)
	assert.Eq(t, "https://example.com/registry.json", locks[baseDir][0].RegistryURL)
	assert.Eq(t, "https://example.com/skills.git", locks[baseDir][0].SourceURL)
	assert.Eq(t, "main", locks[baseDir][0].SourceRef)
	assert.Eq(t, "https://example.com/go-pro.zip", locks[baseDir][0].DownloadURL)
}

func TestService_RestoreUsesResolverForRegistryRecords(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	sourceDir := filepath.Join(baseDir, "cache", "go-pro")
	assert.NoErr(t, os.MkdirAll(sourceDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Go Pro"), 0o644))
	service := NewService(lockFile)
	config := testConfig(baseDir)
	config.LockFile = lockFile
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: ".agents"}
	assert.NoErr(t, service.store.Save(lockFile, lockpkg.File{
		baseDir: {{SkillID: "go-pro", SourceID: "team", SourceType: "registry", RegistryEntryID: "go-pro", InstallEntry: ".", Agents: []string{"universal"}}},
	}))
	service = service.WithRestoreResolver(func(record lockpkg.Record) (skill.Skill, bool, error) {
		return skill.Skill{ID: record.SkillID, SourceID: record.SourceID, SourceType: sourcepkg.TypeRegistry, InstallEntry: ".", Path: sourceDir}, true, nil
	})

	restored, err := service.WithRuntime(config, baseDir).Restore(map[string]string{})

	assert.NoErr(t, err)
	assert.Len(t, restored, 1)
	assert.FileExists(t, filepath.Join(baseDir, ".agents", "skills", "go-pro", "SKILL.md"))
}

func TestService_UninstallRemovesOnlyTargetAgentAndKeepsGroupedRecord(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := createSkillSource(t, baseDir, filepath.Join("source", "hello-skill"), "hello.txt", "hello")
	service := NewService(lockFile)
	config := testConfig(baseDir)
	item := testSkill(sourceDir, "hello-skill", "repo-a/marketplaces/hello-skill", "local-demo")
	projectKey, err := resolveScopeKey(agent.ScopeProject, baseDir)
	assert.NoErr(t, err)
	claudeRoot, err := agent.ResolveInstallPath(config, baseDir, "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)
	codexRoot, err := agent.ResolveInstallPath(config, baseDir, "codex", agent.ScopeProject)
	assert.NoErr(t, err)

	_, err = service.Install(item, "claude-code", agent.ScopeProject, projectKey, claudeRoot)
	assert.NoErr(t, err)
	_, err = service.Install(item, "codex", agent.ScopeProject, projectKey, codexRoot)
	assert.NoErr(t, err)

	runtimeSvc := service.WithRuntime(config, baseDir)
	assert.NoErr(t, runtimeSvc.Uninstall("marketplaces/hello-skill", "claude-code", agent.ScopeProject))

	_, err = os.Stat(installTargetPath(item, claudeRoot))
	assert.True(t, os.IsNotExist(err))
	data, err := os.ReadFile(filepath.Join(installTargetPath(item, codexRoot), "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))

	locks := mustLoadLockFile(t, service, lockFile)
	assert.Len(t, locks[projectKey], 1)
	assert.Eq(t, []string{"codex"}, locks[projectKey][0].Agents)
}

func TestService_UninstallProjectScopeDoesNotTouchOtherProjectKeys(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := createSkillSource(t, baseDir, filepath.Join("source", "hello-skill"), "hello.txt", "hello")
	service := NewService(lockFile)
	config := testConfig(baseDir)
	item := testSkill(sourceDir, "hello-skill", "repo-a/marketplaces/hello-skill", "local-demo")
	projectADir := filepath.Join(baseDir, "project-a")
	projectBDir := filepath.Join(baseDir, "project-b")
	projectAKey, err := resolveScopeKey(agent.ScopeProject, projectADir)
	assert.NoErr(t, err)
	projectBKey, err := resolveScopeKey(agent.ScopeProject, projectBDir)
	assert.NoErr(t, err)
	projectARoot, err := agent.ResolveInstallPath(config, projectADir, "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)
	projectBRoot, err := agent.ResolveInstallPath(config, projectBDir, "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)

	_, err = service.Install(item, "claude-code", agent.ScopeProject, projectAKey, projectARoot)
	assert.NoErr(t, err)
	_, err = service.Install(item, "claude-code", agent.ScopeProject, projectBKey, projectBRoot)
	assert.NoErr(t, err)

	runtimeSvc := service.WithRuntime(config, projectADir)
	assert.NoErr(t, runtimeSvc.Uninstall("marketplaces/hello-skill", "claude-code", agent.ScopeProject))

	_, err = os.Stat(installTargetPath(item, projectARoot))
	assert.True(t, os.IsNotExist(err))
	data, err := os.ReadFile(filepath.Join(installTargetPath(item, projectBRoot), "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))

	locks := mustLoadLockFile(t, service, lockFile)
	_, hasProjectA := locks[projectAKey]
	assert.False(t, hasProjectA)
	assert.Len(t, locks[projectBKey], 1)
	assert.Eq(t, []string{"claude-code"}, locks[projectBKey][0].Agents)
}

func TestService_UninstallMultiContinuesAfterFailure(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	service := NewService(lockFile)
	projectKey, err := resolveScopeKey(agent.ScopeProject, baseDir)
	assert.NoErr(t, err)
	targetRoot, err := agent.ResolveInstallPath(testConfig(baseDir), baseDir, "codex", agent.ScopeProject)
	assert.NoErr(t, err)
	sourceDir := createSkillSource(t, baseDir, filepath.Join("source", "existing"), "skill.txt", "existing")
	_, err = service.Install(testSkill(sourceDir, "existing", "repo/existing", "repo"), "codex", agent.ScopeProject, projectKey, targetRoot)
	assert.NoErr(t, err)

	err = service.WithRuntime(testConfig(baseDir), baseDir).UninstallMulti([]string{"missing", "existing"}, "codex", agent.ScopeProject)

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "missing")
	_, statErr := os.Stat(filepath.Join(targetRoot, "existing"))
	assert.True(t, os.IsNotExist(statErr))
	locks := mustLoadLockFile(t, service, lockFile)
	assert.Len(t, locks[projectKey], 0)
}

func TestService_PlanUninstallReportsInstalledTargets(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := createSkillSource(t, baseDir, filepath.Join("source", "go-pro"), "SKILL.md", "# Go Pro")
	service := NewService(lockFile)
	config := testConfig(baseDir)
	item := testSkill(sourceDir, "go-pro", "gstack/tools/go-pro", "gstack")
	projectKey, err := resolveScopeKey(agent.ScopeProject, baseDir)
	assert.NoErr(t, err)
	targetRoot, err := agent.ResolveInstallPath(config, baseDir, "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)
	_, err = service.WithRuntime(config, baseDir).Install(item, "claude-code", agent.ScopeProject, projectKey, targetRoot)
	assert.NoErr(t, err)

	plan, err := service.WithRuntime(config, baseDir).PlanUninstall(UninstallReq{
		Skills:  []string{"gstack/tools/go-pro"},
		Agent:   "claude-code",
		Scope:   "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Eq(t, "claude-code", plan.Agent)
	assert.Eq(t, "project", plan.Scope)
	assert.Len(t, plan.Items, 1)
	assert.Eq(t, "remove_record", plan.Items[0].Action)
	assert.Eq(t, "go-pro", plan.Items[0].SkillID)
	assert.Eq(t, "gstack", plan.Items[0].SourceID)
	assert.Eq(t, "1.0.0", plan.Items[0].Version)
	assert.Eq(t, filepath.Join(baseDir, ".claude", "skills", "go-pro"), plan.Items[0].InstalledPath)
}

func TestService_RunUninstallReturnsRemovedTargets(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := createSkillSource(t, baseDir, filepath.Join("source", "go-pro"), "SKILL.md", "# Go Pro")
	service := NewService(lockFile)
	config := testConfig(baseDir)
	item := testSkill(sourceDir, "go-pro", "gstack/tools/go-pro", "gstack")
	_, err := service.RunResolved(config, InstallReq{
		Agent:   "claude-code",
		Scope:   "project",
		WorkDir: baseDir,
	}, []skill.Skill{item}, nil)
	assert.NoErr(t, err)

	result, err := service.WithRuntime(config, baseDir).RunUninstall(UninstallReq{
		Skills:  []string{"go-pro"},
		Agent:   "claude-code",
		Scope:   "project",
		WorkDir: baseDir,
	})

	assert.NoErr(t, err)
	assert.Len(t, result.Removed, 1)
	assert.Eq(t, "go-pro", result.Removed[0].SkillID)
	_, err = os.Stat(filepath.Join(baseDir, ".claude", "skills", "go-pro"))
	assert.True(t, os.IsNotExist(err))
	locks := mustLoadLockFile(t, service, lockFile)
	assert.Len(t, locks, 0)
}

func testConfig(baseDir string) cfg.Config {
	return cfg.Config{
		AgentTools: map[string]cfg.AgentToolConfig{
			"claude-code": {
				UserDir:    filepath.Join(baseDir, ".claude-user"),
				ProjectDir: "./.claude",
			},
			"codex": {
				UserDir:    filepath.Join(baseDir, ".codex-user"),
				ProjectDir: "./.codex",
			},
		},
	}
}

func testSkill(sourceDir string, skillID string, sourceQualifiedName string, sourceID string) skill.Skill {
	return skill.Skill{
		ID:                  skillID,
		QualifiedName:       "marketplaces/" + skillID,
		SourceQualifiedName: sourceQualifiedName,
		Version:             "1.0.0",
		SourceID:            sourceID,
		SourceType:          sourcepkg.TypeLocal,
		InstallEntry:        "commands",
		Path:                sourceDir,
	}
}

func createSkillSource(t *testing.T, baseDir string, sourceDir string, fileName string, content string) string {
	t.Helper()
	root := filepath.Join(baseDir, sourceDir)
	commandsDir := filepath.Join(root, "commands")
	assert.NoErr(t, os.MkdirAll(commandsDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(commandsDir, fileName), []byte(content), 0o644))
	return root
}

func mustLoadLockFile(t *testing.T, service *Service, lockFile string) lockpkg.File {
	t.Helper()
	locks, err := service.store.Load(lockFile)
	assert.NoErr(t, err)
	return locks
}
