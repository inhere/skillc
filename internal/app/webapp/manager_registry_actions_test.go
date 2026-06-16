package webapp

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/skillc/internal/domain/registry"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/registrystore"
)

func TestManager_PlanRegistryInstall(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeWebManagerFixture(t, baseDir)
	writeWebRegistryFixture(t, configFile, baseDir, config)

	got, err := NewManager(configFile, baseDir).PlanRegistryInstall(WebRegistryInstallReq{
		Target:     "team/go-pro",
		ManagerReq: ManagerReq{Agent: "codex", Scope: "project", WorkDir: baseDir},
	})

	assert.NoErr(t, err)
	assert.Eq(t, "team/go-pro", got.Target)
	assert.Eq(t, "go-pro", got.SkillID)
	assert.Eq(t, "team", got.RegistryID)
	assert.Eq(t, "codex", got.Agent)
	assert.Eq(t, "project", got.Scope)
}

func TestManager_RunRegistryInstallInstallsArchiveSkill(t *testing.T) {
	baseDir := t.TempDir()
	configFile, config := writeWebManagerFixture(t, baseDir)
	archivePath := filepath.Join(baseDir, "registry-pro.zip")
	writeWebZipArchive(t, archivePath, map[string]string{"skills/registry-pro/SKILL.md": "# Registry Pro"})
	writeWebRegistryFixture(t, configFile, baseDir, config)
	addWebRegistrySkillDownload(t, configFile, baseDir, "team", "registry-pro", archivePath)

	result, err := NewManager(configFile, baseDir).RunRegistryInstall(WebRegistryInstallReq{
		Target:     "team/registry-pro",
		ManagerReq: ManagerReq{Agent: "universal", Scope: "project", WorkDir: baseDir},
	})

	assert.NoErr(t, err)
	assert.Eq(t, "", result.Error)
	assert.Len(t, result.Installed, 1)
	assert.Eq(t, "registry-pro", result.Installed[0].SkillID)
	assert.Eq(t, "team", result.Installed[0].SourceID)
	assert.FileExists(t, filepath.Join(baseDir, ".agents", "skills", "registry-pro", "SKILL.md"))
}

func addWebRegistrySkillDownload(t *testing.T, configFile string, baseDir string, registryID string, skillID string, downloadURL string) {
	t.Helper()
	config, err := configstore.NewYAMLStore().Load(configFile, baseDir)
	assert.Require(t, assert.NoErr(t, err))
	cachePath := filepath.Join(config.RegistryCacheDir, "registry-index.json")
	file, err := registrystore.NewStore().LoadFile(cachePath)
	assert.Require(t, assert.NoErr(t, err))
	file.Skills = append(file.Skills, registry.SkillEntry{
		ID: skillID, Name: skillID, Version: "1.0.0", RegistryID: registryID,
		DownloadURL: downloadURL, InstallEntry: "skills/" + skillID,
	})
	assert.Require(t, assert.NoErr(t, registrystore.NewStore().SaveFile(cachePath, file)))
}

func writeWebZipArchive(t *testing.T, path string, files map[string]string) {
	t.Helper()
	out, err := os.Create(path)
	assert.Require(t, assert.NoErr(t, err))
	defer out.Close()
	zw := zip.NewWriter(out)
	for name, body := range files {
		w, err := zw.Create(name)
		assert.Require(t, assert.NoErr(t, err))
		_, err = w.Write([]byte(body))
		assert.Require(t, assert.NoErr(t, err))
	}
	assert.Require(t, assert.NoErr(t, zw.Close()))
}
