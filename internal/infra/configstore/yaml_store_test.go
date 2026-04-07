package configstore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
)

func TestStore_LoadMissingReturnsDefaultConfig(t *testing.T) {
	store := NewYAMLStore()
	got, err := store.Load("", "")
	assert.NoErr(t, err)
	assert.NotEmpty(t, got.AgentTools)
	assert.Eq(t, ".codex", got.AgentTools["codex"].Dirname)
}

func TestStore_SaveAndLoadRoundTrip(t *testing.T) {
	baseDir := t.TempDir()
	home, err := os.UserHomeDir()
	assert.NoErr(t, err)
	tmp := filepath.Join(baseDir, "skillc.yaml")
	store := NewYAMLStore()

	want := cfg.DefaultConfig()
	want.ProxyURL = "http://localhost:7890"

	err = store.Save(tmp, want)
	assert.NoErr(t, err)

	got, err := store.Load(tmp, baseDir)
	assert.NoErr(t, err)
	assert.Eq(t, want.ProxyURL, got.ProxyURL)
	assert.Eq(t, filepath.Join(home, ".config", "skillc", "skillc-install.lock"), got.LockFile)
}

func TestStore_LoadExpandsRuntimePaths(t *testing.T) {
	baseDir := t.TempDir()
	home, err := os.UserHomeDir()
	assert.NoErr(t, err)

	configFile := filepath.Join(baseDir, "skillc.yaml")
	store := NewYAMLStore()
	data := cfg.DefaultConfig()
	assert.NoErr(t, store.Save(configFile, data))

	got, err := store.Load(configFile, baseDir)
	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(home, ".config", "skillc", "skillc-install.lock"), got.LockFile)
	assert.Eq(t, filepath.Join(home, ".cache", "skillc", "repos"), got.RepoCacheDir)
	assert.Eq(t, filepath.Join(home, ".cache", "skillc", "skills"), got.SkillCacheDir)
	assert.Eq(t, filepath.Join(home, ".cache", "skillc", "registry"), got.RegistryCacheDir)
	assert.Eq(t, filepath.Join(home, ".cache", "skillc", "skillc-index.json"), got.IndexFile)
	assert.Eq(t, filepath.Join(home, ".claude"), got.AgentTools["claude-code"].UserDir)
	assert.Eq(t, filepath.Join(baseDir, ".claude"), got.AgentTools["claude-code"].ProjectDir)
}

func TestStore_SaveAndLoadSourceLastSyncAt(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	store := NewYAMLStore()
	want := cfg.DefaultConfig()
	want.Sources = []sourcepkg.Source{{
		ID:          "git-demo",
		Type:        sourcepkg.TypeGit,
		URL:         "https://example.com/repo.git",
		Ref:         "main",
		ResolvedRef: "0123456789abcdef",
		LastSyncAt:  "2024-03-09T16:00:00Z",
		Status:      "ready",
	}}

	assert.NoErr(t, store.Save(configFile, want))

	got, err := store.Load(configFile, baseDir)
	assert.NoErr(t, err)
	assert.Len(t, got.Sources, 1)
	assert.Eq(t, "2024-03-09T16:00:00Z", got.Sources[0].LastSyncAt)
	assert.Eq(t, "0123456789abcdef", got.Sources[0].ResolvedRef)
}

func TestStore_SaveAfterLoadKeepsProjectDirsPortable(t *testing.T) {
	firstBaseDir := t.TempDir()
	configFile := filepath.Join(firstBaseDir, "skillc.yaml")
	store := NewYAMLStore()

	assert.NoErr(t, store.Save(configFile, cfg.DefaultConfig()))

	loaded, err := store.Load(configFile, firstBaseDir)
	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(firstBaseDir, ".claude"), loaded.AgentTools["claude-code"].ProjectDir)

	assert.NoErr(t, store.Save(configFile, loaded))

	content, err := os.ReadFile(configFile)
	assert.NoErr(t, err)
	assert.Contains(t, string(content), "project_dir: ./.claude")

	secondBaseDir := t.TempDir()
	got, err := store.Load(configFile, secondBaseDir)
	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(secondBaseDir, ".claude"), got.AgentTools["claude-code"].ProjectDir)
}

func TestStore_SaveAfterLoadKeepsCustomRelativePathsPortable(t *testing.T) {
	firstBaseDir := t.TempDir()
	configFile := filepath.Join(firstBaseDir, "skillc.yaml")
	store := NewYAMLStore()
	data := cfg.DefaultConfig()
	data.IndexFile = "./custom-index.json"
	tool := data.AgentTools["claude-code"]
	tool.ProjectDir = "./custom-claude"
	data.AgentTools["claude-code"] = tool

	assert.NoErr(t, store.Save(configFile, data))

	loaded, err := store.Load(configFile, firstBaseDir)
	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(firstBaseDir, "custom-index.json"), loaded.IndexFile)
	assert.Eq(t, filepath.Join(firstBaseDir, "custom-claude"), loaded.AgentTools["claude-code"].ProjectDir)

	assert.NoErr(t, store.Save(configFile, loaded))

	content, err := os.ReadFile(configFile)
	assert.NoErr(t, err)
	assert.Contains(t, string(content), "index_file: ./custom-index.json")
	assert.Contains(t, string(content), "project_dir: ./custom-claude")

	secondBaseDir := t.TempDir()
	got, err := store.Load(configFile, secondBaseDir)
	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(secondBaseDir, "custom-index.json"), got.IndexFile)
	assert.Eq(t, filepath.Join(secondBaseDir, "custom-claude"), got.AgentTools["claude-code"].ProjectDir)
}


func TestStore_SaveAfterLoadKeepsRelativePathsPortableAcrossExternalConfigLocation(t *testing.T) {
	configDir := t.TempDir()
	runtimeBaseDir := t.TempDir()
	configFile := filepath.Join(configDir, "config.yaml")
	store := NewYAMLStore()
	data := cfg.DefaultConfig()
	data.IndexFile = "./custom-index.json"
	tool := data.AgentTools["claude-code"]
	tool.ProjectDir = "./custom-claude"
	data.AgentTools["claude-code"] = tool

	assert.NoErr(t, store.Save(configFile, data))

	loaded, err := store.Load(configFile, runtimeBaseDir)
	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(runtimeBaseDir, "custom-index.json"), loaded.IndexFile)
	assert.Eq(t, filepath.Join(runtimeBaseDir, "custom-claude"), loaded.AgentTools["claude-code"].ProjectDir)

	assert.NoErr(t, store.Save(configFile, loaded, runtimeBaseDir))

	content, err := os.ReadFile(configFile)
	assert.NoErr(t, err)
	assert.Contains(t, string(content), "index_file: ./custom-index.json")
	assert.Contains(t, string(content), "project_dir: ./custom-claude")
	assert.NotContains(t, string(content), filepath.Join(runtimeBaseDir, "custom-claude"))

	secondBaseDir := t.TempDir()
	got, err := store.Load(configFile, secondBaseDir)
	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(secondBaseDir, "custom-index.json"), got.IndexFile)
	assert.Eq(t, filepath.Join(secondBaseDir, "custom-claude"), got.AgentTools["claude-code"].ProjectDir)
}
