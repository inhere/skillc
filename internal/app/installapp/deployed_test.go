package installapp

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/skillc/internal/app/apputil"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/skill"
	"github.com/inhere/skillc/internal/infra/agentfs"
	"github.com/inhere/skillc/internal/infra/hashx"
)

func TestService_CopyInstallRecordsDeployedChecksum(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := createSkillSource(t, baseDir, "source", "hello.txt", "hello")
	config := testConfig(baseDir)
	service := NewService(lockFile).WithRuntime(config, baseDir).WithInstallMode(agentfs.ModeCopy)
	projectKey := copyInstallScopeKey(t, baseDir)

	record, err := service.Install(copyInstallSkill(sourceDir), "claude-code", agent.ScopeProject, projectKey, copyInstallRoot(t, config, baseDir))
	assert.NoErr(t, err)
	assert.Eq(t, string(agentfs.ModeCopy), record.InstallMode)

	want, err := hashx.SumDeployed(record.InstalledPath)
	assert.NoErr(t, err)
	assert.Eq(t, want, record.InstalledChecksum)

	locks := mustLoadLockFile(t, service, lockFile)
	assert.Eq(t, want, locks[projectKey][0].InstalledChecksum)
}

func TestService_CopyInstallKeepsLocalChangesUnlessForced(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := createSkillSource(t, baseDir, "source", "hello.txt", "hello")
	config := testConfig(baseDir)
	service := NewService(lockFile).WithRuntime(config, baseDir).WithInstallMode(agentfs.ModeCopy)
	projectKey := copyInstallScopeKey(t, baseDir)
	targetRoot := copyInstallRoot(t, config, baseDir)
	item := copyInstallSkill(sourceDir)

	first, err := service.Install(item, "claude-code", agent.ScopeProject, projectKey, targetRoot)
	assert.NoErr(t, err)
	targetPath := filepath.Join(targetRoot, "hello-skill")
	assert.NoErr(t, os.WriteFile(filepath.Join(targetPath, "hello.txt"), []byte("local edit"), 0o644))

	_, err = service.Install(item, "claude-code", agent.ScopeProject, projectKey, targetRoot)
	assert.Err(t, err)
	assert.True(t, errors.Is(err, apputil.ErrLocalChanges))
	assertFileText(t, filepath.Join(targetPath, "hello.txt"), "local edit")
	assert.Eq(t, first.InstalledChecksum, mustLoadLockFile(t, service, lockFile)[projectKey][0].InstalledChecksum)

	forced, err := service.WithForce(true).Install(item, "claude-code", agent.ScopeProject, projectKey, targetRoot)
	assert.NoErr(t, err)
	assert.NotEq(t, "", forced.BackupPath)
	assertFileText(t, filepath.Join(forced.BackupPath, "hello.txt"), "local edit")
	assertFileText(t, filepath.Join(targetPath, "hello.txt"), "hello")
}

func TestService_RestoreSkipsLocallyModifiedCopy(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := createSkillSource(t, baseDir, "source", "hello.txt", "hello")
	config := testConfig(baseDir)
	service := NewService(lockFile).WithRuntime(config, baseDir).WithInstallMode(agentfs.ModeCopy)

	record, err := service.Install(copyInstallSkill(sourceDir), "claude-code", agent.ScopeProject, copyInstallScopeKey(t, baseDir), copyInstallRoot(t, config, baseDir))
	assert.NoErr(t, err)
	assert.NoErr(t, os.WriteFile(filepath.Join(record.InstalledPath, "hello.txt"), []byte("local edit"), 0o644))

	restored, skipped, err := service.Restore(map[string]string{"local-demo": sourceDir})
	assert.NoErr(t, err)
	assert.Len(t, restored, 0)
	assert.Len(t, skipped, 1)
	assert.Eq(t, "hello-skill", skipped[0].SkillID)
	assert.Contains(t, skipped[0].Reason, "local changes")
	assertFileText(t, filepath.Join(record.InstalledPath, "hello.txt"), "local edit")
}

func TestService_UninstallRefusesLocallyModifiedCopy(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := createSkillSource(t, baseDir, "source", "hello.txt", "hello")
	config := testConfig(baseDir)
	service := NewService(lockFile).WithRuntime(config, baseDir).WithInstallMode(agentfs.ModeCopy)

	record, err := service.Install(copyInstallSkill(sourceDir), "claude-code", agent.ScopeProject, copyInstallScopeKey(t, baseDir), copyInstallRoot(t, config, baseDir))
	assert.NoErr(t, err)
	assert.NoErr(t, os.WriteFile(filepath.Join(record.InstalledPath, "hello.txt"), []byte("local edit"), 0o644))

	err = service.Uninstall("hello-skill", "claude-code", agent.ScopeProject)
	assert.Err(t, err)
	assert.True(t, errors.Is(err, apputil.ErrLocalChanges))
	assertFileText(t, filepath.Join(record.InstalledPath, "hello.txt"), "local edit")

	assert.NoErr(t, service.WithForce(true).Uninstall("hello-skill", "claude-code", agent.ScopeProject))
	if _, err := os.Stat(record.InstalledPath); !os.IsNotExist(err) {
		t.Fatalf("expected %s removed, got err=%v", record.InstalledPath, err)
	}
}

func copyInstallSkill(sourceDir string) skill.Skill {
	return testSkill(sourceDir, "hello-skill", "repo-a/marketplaces/hello-skill", "local-demo")
}

func copyInstallScopeKey(t *testing.T, baseDir string) string {
	t.Helper()
	projectKey, err := resolveScopeKey(agent.ScopeProject, baseDir)
	assert.NoErr(t, err)
	return projectKey
}

func copyInstallRoot(t *testing.T, config cfg.Config, baseDir string) string {
	t.Helper()
	targetRoot, err := agent.ResolveInstallPath(config, baseDir, "claude-code", agent.ScopeProject)
	assert.NoErr(t, err)
	return targetRoot
}

func assertFileText(t *testing.T, path string, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	assert.NoErr(t, err)
	assert.Eq(t, want, string(data))
}
