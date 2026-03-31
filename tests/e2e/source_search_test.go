package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/app/configapp"
	"github.com/inhere/skillc/internal/app/searchapp"
	"github.com/inhere/skillc/internal/app/sourceapp"
	skillpkg "github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
)

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

	item, err := searchService.Show("hello-skill")
	assert.NoErr(t, err)
	assert.Eq(t, skillpkg.Skill{ID: "hello-skill", Name: "Hello Skill", Description: "Friendly greeting helper", Version: results[0].Version, SupportedAgents: []string{"claude-code"}, SourceID: listedSources[0].ID, SourceType: sourcepkg.TypeLocal, InstallEntry: ".", Path: skillDir}, item)
}
