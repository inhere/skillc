package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/app/configapp"
	"github.com/inhere/skillc/internal/app/searchapp"
	"github.com/inhere/skillc/internal/app/sourceapp"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
)

func TestMain(m *testing.M) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	testHome := filepath.Join(cwd, "testdata", "home")
	if err := os.RemoveAll(testHome); err != nil {
		panic(err)
	}
	if err := os.MkdirAll(testHome, 0o755); err != nil {
		panic(err)
	}
	if err := os.Setenv("HOME", testHome); err != nil {
		panic(err)
	}
	if err := os.Setenv("USERPROFILE", testHome); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func TestConfigInit_UsesLocalTestHomePaths(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")

	configService := configapp.NewService(configFile, baseDir)
	cfg, err := configService.Init()
	assert.NoErr(t, err)
	assert.Contains(t, cfg.IndexFile, filepath.Join("testdata", "home"))
	assert.Contains(t, cfg.RepoCacheDir, filepath.Join("testdata", "home"))
}

func TestLocalSourceToSearchFlow(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	sourceRoot := filepath.Join(baseDir, "skills")
	skillDir := filepath.Join(sourceRoot, "hello-skill")
	assert.NoErr(t, os.MkdirAll(skillDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
id: hello-skill
name: Hello Skill
description: Friendly greeting helper
supported_agents:
  - claude-code
install_entry: .
---
# Hello Skill
`), 0o644))

	configService := configapp.NewService(configFile, baseDir)
	_, err := configService.Init()
	assert.NoErr(t, err)
	cfg, err := configService.Show()
	assert.NoErr(t, err)

	sourceService := sourceapp.NewService(configFile, baseDir)
	src, err := sourceService.AddLocal(sourceRoot)
	assert.NoErr(t, err)
	assert.NoErr(t, sourceService.Sync(src.ID))

	listedSources, err := sourceService.List()
	assert.NoErr(t, err)
	assert.Len(t, listedSources, 1)
	assert.Eq(t, "ready", listedSources[0].Status)

	searchService := searchapp.NewService(cfg.IndexFile)
	results, err := searchService.Search("greeting", "claude-code", sourcepkg.TypeLocal)
	assert.NoErr(t, err)
	assert.Len(t, results, 1)
	assert.Eq(t, "hello-skill", results[0].ID)
	assert.Eq(t, "hello-skill", results[0].QualifiedName)
	assert.Eq(t, src.Name+"/hello-skill", results[0].SourceQualifiedName)

	item, err := searchService.Show("hello-skill")
	assert.NoErr(t, err)
	assert.Eq(t, "hello-skill", item.ID)
	assert.Eq(t, "Hello Skill", item.Name)
	assert.Eq(t, "Friendly greeting helper", item.Description)
	assert.Eq(t, results[0].Version, item.Version)
	assert.Eq(t, []string{"claude-code"}, item.SupportedAgents)
	assert.Eq(t, listedSources[0].ID, item.SourceID)
	assert.Eq(t, listedSources[0].Name, item.SourceName)
	assert.Eq(t, sourcepkg.TypeLocal, item.SourceType)
	assert.Eq(t, "hello-skill", item.QualifiedName)
	assert.Eq(t, src.Name+"/hello-skill", item.SourceQualifiedName)
	assert.Eq(t, ".", item.InstallEntry)
	assert.Eq(t, skillDir, item.Path)
	assert.NotEmpty(t, item.Checksum)
}
