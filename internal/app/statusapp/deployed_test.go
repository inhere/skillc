package statusapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/hashx"
	"github.com/inhere/skillc/internal/infra/lockstore"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

func TestService_RunMarksLocallyModifiedCopyInstall(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	indexFile := filepath.Join(baseDir, "index.json")
	skillsRoot := filepath.Join(baseDir, ".agents", "skills")
	cleanPath := filepath.Join(skillsRoot, "clean-skill")
	editedPath := filepath.Join(skillsRoot, "edited-skill")
	for _, path := range []string{cleanPath, editedPath} {
		assert.NoErr(t, os.MkdirAll(path, 0o755))
		assert.NoErr(t, os.WriteFile(filepath.Join(path, "SKILL.md"), []byte("skill"), 0o644))
	}
	cleanSum, err := hashx.SumDeployed(cleanPath)
	assert.NoErr(t, err)
	editedSum, err := hashx.SumDeployed(editedPath)
	assert.NoErr(t, err)
	// 模拟安装后被本地改动
	assert.NoErr(t, os.WriteFile(filepath.Join(editedPath, "SKILL.md"), []byte("edited"), 0o644))

	config := cfg.DefaultConfig()
	config.LockFile = lockFile
	config.IndexFile = indexFile
	config.BackupDir = filepath.Join(baseDir, "cache", "backups")
	config.AgentTools["universal"] = cfg.AgentToolConfig{Dirname: ".agents", ProjectDir: filepath.Join(baseDir, ".agents")}
	config.Sources = []sourcepkg.Source{{ID: "gstack", Type: sourcepkg.TypeLocal, Path: filepath.Join(baseDir, "source")}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{SkillID: "clean-skill", SourceID: "gstack", Version: "1.0.0", InstallMode: "copy", InstalledChecksum: cleanSum, Agents: []string{"universal"}},
			{SkillID: "edited-skill", SourceID: "gstack", Version: "1.0.0", InstallMode: "copy", InstalledChecksum: editedSum, Agents: []string{"universal"}},
		},
	}))
	assert.NoErr(t, repoindex.NewStore().Save(indexFile, []skill.Skill{
		{ID: "clean-skill", SourceID: "gstack", Version: "1.0.0"},
		{ID: "edited-skill", SourceID: "gstack", Version: "1.0.0"},
	}))

	result, err := NewService(configFile, baseDir).Run(Req{Agent: "universal", Scope: "project", WorkDir: baseDir})

	assert.NoErr(t, err)
	modified := map[string]bool{}
	for _, item := range result.Items {
		modified[item.SkillID] = item.LocallyModified
	}
	assert.False(t, modified["clean-skill"])
	assert.True(t, modified["edited-skill"])
	assert.Eq(t, 1, result.Summary.Modified)
	assert.Eq(t, 2, result.Summary.Installed)
}
