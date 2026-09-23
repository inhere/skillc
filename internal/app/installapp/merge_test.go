package installapp

import (
	installpkg "github.com/inhere/skillc/internal/domain/install"
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/skillc/internal/domain/agent"
	"github.com/inhere/skillc/internal/infra/agentfs"
	"github.com/inhere/skillc/internal/infra/hashx"
)

func TestService_MergeAtPathAppliesUpstreamAndKeepsLocalEdits(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := filepath.Join(baseDir, "source")
	commandsDir := filepath.Join(sourceDir, "commands")
	writeRestoreFile(t, filepath.Join(commandsDir, "SKILL.md"), "skill v1")
	writeRestoreFile(t, filepath.Join(commandsDir, "run.md"), "run v1")
	writeRestoreFile(t, filepath.Join(commandsDir, "docs", "old.md"), "old")

	config := testConfig(baseDir)
	service := NewService(lockFile).WithRuntime(config, baseDir).WithInstallMode(agentfs.ModeCopy)
	projectKey := copyInstallScopeKey(t, baseDir)
	item := testSkill(sourceDir, "hello-skill", "repo-a/marketplaces/hello-skill", "local-demo")

	record, err := service.Install(item, "claude-code", agent.ScopeProject, projectKey, copyInstallRoot(t, config, baseDir))
	assert.NoErr(t, err)
	assert.Len(t, record.InstalledFiles, 3)

	// 本地改动：改 run.md、新增 local.md、删除 docs/old.md
	assert.NoErr(t, os.WriteFile(filepath.Join(record.InstalledPath, "run.md"), []byte("run local"), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(record.InstalledPath, "local.md"), []byte("local only"), 0o644))
	assert.NoErr(t, os.Remove(filepath.Join(record.InstalledPath, "docs", "old.md")))

	// 上游改动：改 SKILL.md、改 run.md（与本地冲突）、新增 docs/new.md
	writeRestoreFile(t, filepath.Join(commandsDir, "SKILL.md"), "skill v2")
	writeRestoreFile(t, filepath.Join(commandsDir, "run.md"), "run v2")
	writeRestoreFile(t, filepath.Join(commandsDir, "docs", "new.md"), "new")

	merged, result, err := service.MergeAtPath(item, "claude-code", agent.ScopeProject, projectKey, record.InstalledPath, false)
	assert.NoErr(t, err)

	assert.Eq(t, []string{"SKILL.md", "docs/new.md"}, result.Updated)
	assert.Eq(t, []string{"run.md"}, result.Conflicts)
	assert.Eq(t, []string{"docs/old.md", "local.md"}, result.KeptLocal)
	assert.Len(t, result.Removed, 0)

	assertFileText(t, filepath.Join(record.InstalledPath, "SKILL.md"), "skill v2")
	assertFileText(t, filepath.Join(record.InstalledPath, "docs", "new.md"), "new")
	assertFileText(t, filepath.Join(record.InstalledPath, "run.md"), "run local")
	assertFileText(t, filepath.Join(record.InstalledPath, "run.md.incoming"), "run v2")
	assertFileText(t, filepath.Join(record.InstalledPath, "local.md"), "local only")
	if _, err := os.Stat(filepath.Join(record.InstalledPath, "docs", "old.md")); !os.IsNotExist(err) {
		t.Fatalf("expected locally deleted file to stay deleted, got %v", err)
	}

	// 基线更新为源内容，被保留的本地改动仍然算 modified
	incoming, err := hashx.Deployed(commandsDir)
	assert.NoErr(t, err)
	assert.Eq(t, incoming.Sum, merged.InstalledChecksum)
	assert.Eq(t, incoming.Files, merged.InstalledFiles)
	drift, err := installpkg.DetectDrift(merged.Record, record.InstalledPath)
	assert.NoErr(t, err)
	assert.True(t, drift.Modified)
}

func TestService_MergeAtPathForceTakesUpstreamOnConflict(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	sourceDir := filepath.Join(baseDir, "source")
	commandsDir := filepath.Join(sourceDir, "commands")
	writeRestoreFile(t, filepath.Join(commandsDir, "run.md"), "run v1")

	config := testConfig(baseDir)
	service := NewService(lockFile).WithRuntime(config, baseDir).WithInstallMode(agentfs.ModeCopy)
	projectKey := copyInstallScopeKey(t, baseDir)
	item := testSkill(sourceDir, "hello-skill", "repo-a/marketplaces/hello-skill", "local-demo")

	record, err := service.Install(item, "claude-code", agent.ScopeProject, projectKey, copyInstallRoot(t, config, baseDir))
	assert.NoErr(t, err)
	assert.NoErr(t, os.WriteFile(filepath.Join(record.InstalledPath, "run.md"), []byte("run local"), 0o644))
	writeRestoreFile(t, filepath.Join(commandsDir, "run.md"), "run v2")

	merged, result, err := service.MergeAtPath(item, "claude-code", agent.ScopeProject, projectKey, record.InstalledPath, true)
	assert.NoErr(t, err)
	assert.Eq(t, []string{"run.md"}, result.Updated)
	assert.Len(t, result.Conflicts, 0)

	assertFileText(t, filepath.Join(record.InstalledPath, "run.md"), "run v2")
	if _, err := os.Stat(filepath.Join(record.InstalledPath, "run.md.incoming")); !os.IsNotExist(err) {
		t.Fatalf("expected no .incoming file when forcing, got %v", err)
	}
	// 冲突强制取上游前会整目录备份
	assert.NotEq(t, "", merged.BackupPath)
	assertFileText(t, filepath.Join(merged.BackupPath, "run.md"), "run local")
}
