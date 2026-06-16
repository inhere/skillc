package registryapp

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/registry"
	"github.com/inhere/skillc/internal/infra/configstore"
	"github.com/inhere/skillc/internal/infra/registrystore"
)

func TestService_SyncLocalRegistryAndSearch(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeRegistryAppConfig(t, baseDir)
	catalogPath := filepath.Join(baseDir, "registry.json")
	assert.NoErr(t, os.WriteFile(catalogPath, []byte(`{"sources":[{"id":"gstack","name":"GStack Skills","description":"Go workflow","type":"git","url":"https://example.com/gstack.git","ref":"main","tags":["go"]}]}`), 0o644))
	service := NewService(configFile, baseDir)

	item, err := service.Add(AddReq{ID: "local", Name: "Local", Value: catalogPath})
	assert.NoErr(t, err)
	assert.Eq(t, "local", item.ID)
	assert.NoErr(t, service.Sync("local"))

	results, err := service.Search("go")
	assert.NoErr(t, err)
	assert.Len(t, results, 1)
	assert.Eq(t, "gstack", results[0].ID)
	assert.Eq(t, "local", results[0].RegistryID)
}

func TestService_SyncHTTPRegistryAndSearch(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeRegistryAppConfig(t, baseDir)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"sources":[{"id":"remote","name":"Remote Skills","description":"Remote Go workflow","type":"git","url":"https://example.com/remote.git","tags":["go"]}]}`))
	}))
	defer server.Close()
	service := NewService(configFile, baseDir)

	_, err := service.Add(AddReq{ID: "official", Name: "Official", Value: server.URL})
	assert.NoErr(t, err)
	assert.NoErr(t, service.Sync("official"))

	results, err := service.Search("remote")
	assert.NoErr(t, err)
	assert.Len(t, results, 1)
	assert.Eq(t, "remote", results[0].ID)
	assert.Eq(t, "official", results[0].RegistryID)
}

func TestService_SyncJSONRegistryCachesSkillEntries(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeRegistryAppConfig(t, baseDir)
	catalogPath := filepath.Join(baseDir, "registry.json")
	assert.NoErr(t, os.WriteFile(catalogPath, []byte(`{"skills":[{"id":"go-pro","name":"Go Pro","description":"Go helper","version":"1.2.0","source_url":"https://example.com/skills.git","source_ref":"main","install_entry":"skills/go-pro","tags":["go"]}]}`), 0o644))

	service := NewService(configFile, baseDir)
	_, err := service.Add(AddReq{ID: "team", Name: "Team", Value: catalogPath})
	assert.NoErr(t, err)
	assert.NoErr(t, service.Sync("team"))

	results, err := service.SearchSkills(SearchReq{Keyword: "go"})

	assert.NoErr(t, err)
	assert.Len(t, results, 1)
	assert.Eq(t, "go-pro", results[0].ID)
	assert.Eq(t, "team", results[0].RegistryID)
	assert.Eq(t, catalogPath, results[0].RegistryURL)
}

func TestService_InfoSkillRequiresRegistryWhenAmbiguous(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeRegistryAppConfig(t, baseDir)
	config, err := configstore.NewYAMLStore().Load(configFile, baseDir)
	assert.NoErr(t, err)
	service := NewService(configFile, baseDir)
	assert.NoErr(t, service.cache.SaveFile(filepath.Join(config.RegistryCacheDir, "registry-index.json"), registrystore.File{
		Skills: []registry.SkillEntry{
			{ID: "go-pro", RegistryID: "team-a", SourceURL: "https://example.com/a.git", InstallEntry: "."},
			{ID: "go-pro", RegistryID: "team-b", SourceURL: "https://example.com/b.git", InstallEntry: "."},
		},
	}))

	_, err = service.InfoSkill("go-pro")

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "ambiguous registry skill")
}

func TestService_AddSourceFromRegistryEntry(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeRegistryAppConfig(t, baseDir)
	catalogPath := filepath.Join(baseDir, "registry.json")
	assert.NoErr(t, os.WriteFile(catalogPath, []byte(`{"sources":[{"id":"gstack","name":"GStack Skills","type":"git","url":"https://example.com/gstack.git","ref":"main"}]}`), 0o644))
	service := NewService(configFile, baseDir)
	_, err := service.Add(AddReq{ID: "local", Value: catalogPath})
	assert.NoErr(t, err)
	assert.NoErr(t, service.Sync("local"))

	src, err := service.AddSource(AddSourceReq{EntryID: "gstack"})

	assert.NoErr(t, err)
	assert.Eq(t, "gstack", src.ID)
	assert.Eq(t, "GStack Skills", src.Name)
	assert.Eq(t, "https://example.com/gstack.git", src.URL)
}

func TestService_InfoRejectsAmbiguousEntryID(t *testing.T) {
	baseDir := t.TempDir()
	configFile := writeRegistryAppConfig(t, baseDir)
	firstPath := filepath.Join(baseDir, "first.json")
	secondPath := filepath.Join(baseDir, "second.json")
	assert.NoErr(t, os.WriteFile(firstPath, []byte(`{"sources":[{"id":"shared","name":"First","type":"git","url":"https://example.com/first.git"}]}`), 0o644))
	assert.NoErr(t, os.WriteFile(secondPath, []byte(`{"sources":[{"id":"shared","name":"Second","type":"git","url":"https://example.com/second.git"}]}`), 0o644))
	service := NewService(configFile, baseDir)
	_, err := service.Add(AddReq{ID: "first", Value: firstPath})
	assert.NoErr(t, err)
	_, err = service.Add(AddReq{ID: "second", Value: secondPath})
	assert.NoErr(t, err)
	assert.NoErr(t, service.SyncAll())

	_, err = service.Info("shared")

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "ambiguous registry entry")
}

func writeRegistryAppConfig(t *testing.T, baseDir string) string {
	t.Helper()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	config := cfg.DefaultConfig()
	config.RegistryCacheDir = filepath.Join(baseDir, "cache", "registry")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))
	return configFile
}
