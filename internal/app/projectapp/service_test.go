package projectapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/project"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/lockstore"
)

func TestService_AddListRemoveProject(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	projectDir := filepath.Join(baseDir, "demo")
	assert.NoErr(t, os.MkdirAll(projectDir, 0o755))
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, cfg.DefaultConfig(), baseDir))

	service := NewService(configFile, baseDir)
	added, err := service.Add(AddReq{ID: "demo", Name: "Demo", Path: projectDir})
	assert.NoErr(t, err)
	assert.Eq(t, "demo", added.ID)

	items, err := service.List()
	assert.NoErr(t, err)
	assert.Len(t, items, 1)
	assert.Eq(t, projectDir, items[0].Path)

	assert.NoErr(t, service.Remove("demo"))
	items, err = service.List()
	assert.NoErr(t, err)
	assert.Len(t, items, 0)
}

func TestService_AddRejectsDuplicatePath(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	projectDir := filepath.Join(baseDir, "demo")
	assert.NoErr(t, os.MkdirAll(projectDir, 0o755))
	data := cfg.DefaultConfig()
	data.Projects = []project.Project{{ID: "demo", Path: projectDir}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, data, baseDir))

	_, err := NewService(configFile, baseDir).Add(AddReq{ID: "copy", Path: projectDir})

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "project path already registered")
}

func TestService_AddRejectsDuplicateExplicitID(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	firstDir := filepath.Join(baseDir, "first")
	secondDir := filepath.Join(baseDir, "second")
	assert.NoErr(t, os.MkdirAll(firstDir, 0o755))
	assert.NoErr(t, os.MkdirAll(secondDir, 0o755))
	data := cfg.DefaultConfig()
	data.Projects = []project.Project{{ID: "demo", Path: firstDir}}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, data, baseDir))

	_, err := NewService(configFile, baseDir).Add(AddReq{ID: "demo", Path: secondDir})

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "project id already registered")
}

func TestService_ImportFromLockRegistersExistingProjectKeys(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	lockFile := filepath.Join(baseDir, "skillc.lock.json")
	projectDir := filepath.Join(baseDir, "project-a")
	missingDir := filepath.Join(baseDir, "missing-project")
	assert.NoErr(t, os.MkdirAll(projectDir, 0o755))
	data := cfg.DefaultConfig()
	data.LockFile = lockFile
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, data, baseDir))
	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		projectDir:        {{SkillID: "go-pro", Agents: []string{"universal"}}},
		missingDir:        {{SkillID: "review", Agents: []string{"universal"}}},
		lockpkg.GlobalKey: {{SkillID: "global", Agents: []string{"universal"}}},
	}))

	result, err := NewService(configFile, baseDir).ImportFromLock()

	assert.NoErr(t, err)
	assert.Len(t, result.Added, 1)
	assert.Eq(t, "project-a", result.Added[0].ID)
	assert.Len(t, result.Skipped, 2)
}
