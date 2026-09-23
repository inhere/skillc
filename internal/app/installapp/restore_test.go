package installapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/agentfs"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

func TestService_RestoreUsesIndexedSkillDir(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	indexFile := filepath.Join(baseDir, "cache", "index.json")
	sourceRoot := filepath.Join(baseDir, "source")
	skillDir := filepath.Join(sourceRoot, "hello-skill")
	writeRestoreFile(t, filepath.Join(skillDir, "commands", "hello.txt"), "hello")
	writeRestoreFile(t, filepath.Join(sourceRoot, "other-skill", "commands", "other.txt"), "other")

	config := testConfig(baseDir)
	config.IndexFile = indexFile
	service := NewService(lockFile).WithRuntime(config, baseDir).WithInstallMode(agentfs.ModeCopy)
	projectKey := copyInstallScopeKey(t, baseDir)
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectKey: {
			{
				SkillID:      "hello-skill",
				SourceID:     "local-demo",
				SourceType:   string(sourcepkg.TypeLocal),
				InstallEntry: "commands",
				Agents:       []string{"claude-code"},
			},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "hello-skill", SourceID: "local-demo", SourceType: sourcepkg.TypeLocal, InstallEntry: "commands", Path: skillDir},
	}))

	restored, skipped, err := service.Restore(map[string]string{"local-demo": sourceRoot})
	assert.NoErr(t, err)
	assert.Len(t, skipped, 0)
	assert.Len(t, restored, 1)

	installed := restored[0].InstalledPath
	assertFileText(t, filepath.Join(installed, "hello.txt"), "hello")
	// 只能恢复目标 skill 目录，不能把整个 source 根目录复制进安装目录
	if _, err := os.Stat(filepath.Join(installed, "other-skill")); !os.IsNotExist(err) {
		t.Fatalf("expected source root not to be copied into %s, got err=%v", installed, err)
	}
}

func writeRestoreFile(t *testing.T, path string, content string) {
	t.Helper()
	assert.NoErr(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NoErr(t, os.WriteFile(path, []byte(content), 0o644))
}
