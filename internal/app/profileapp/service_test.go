package profileapp

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/profile"
	"github.com/inhere/skillc/internal/infra/configstore"
)

func TestService_ListAndShowProfiles(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	config := cfg.DefaultConfig()
	config.Profiles = map[string]profile.Profile{
		"review": {Description: "Review", Targets: []profile.Target{{Source: "gstack", Skill: "review"}}},
		"go-dev": {Description: "Go dev", Targets: []profile.Target{{Source: "gstack", Skill: "go-pro"}}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))

	svc := NewService(configFile, baseDir)
	list, err := svc.List()

	assert.NoErr(t, err)
	assert.Len(t, list, 2)
	assert.Eq(t, "go-dev", list[0].Name)
	assert.Eq(t, "review", list[1].Name)

	got, err := svc.Show("go-dev")
	assert.NoErr(t, err)
	assert.Eq(t, "Go dev", got.Description)
}

func TestService_ShowMissingProfileReturnsError(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, cfg.DefaultConfig(), baseDir))

	svc := NewService(configFile, baseDir)
	_, err := svc.Show("missing")

	assert.Err(t, err)
	assert.Contains(t, err.Error(), "profile not found: missing")
}

func TestService_SaveProfileNormalizesTargets(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, cfg.DefaultConfig(), baseDir))

	svc := NewService(configFile, baseDir)
	err := svc.Save("go-dev", profile.Profile{
		Description: "Go dev",
		Targets: []profile.Target{
			{Source: "gstack", Skill: "review"},
			{Source: "gstack", Skill: "go-pro"},
			{Source: "gstack", Skill: "review"},
		},
	})
	assert.NoErr(t, err)

	got, err := svc.Show("go-dev")
	assert.NoErr(t, err)
	assert.Eq(t, "Go dev", got.Description)
	assert.Len(t, got.Targets, 2)
	assert.Eq(t, "go-pro", got.Targets[0].Skill)
	assert.Eq(t, "review", got.Targets[1].Skill)
}

func TestService_CreateProfileRejectsExistingName(t *testing.T) {
	baseDir := t.TempDir()
	configFile := filepath.Join(baseDir, "skillc.yaml")
	config := cfg.DefaultConfig()
	config.Profiles = map[string]profile.Profile{
		"go-dev": {Description: "Existing", Targets: []profile.Target{{Source: "gstack", Skill: "go-pro"}}},
	}
	assert.NoErr(t, configstore.NewYAMLStore().Save(configFile, config, baseDir))

	svc := NewService(configFile, baseDir)
	got, err := svc.Create("new-dev", profile.Profile{
		Description: "New dev",
		Targets:     []profile.Target{{Source: "gstack", Skill: "review"}},
	})
	assert.NoErr(t, err)
	assert.Eq(t, "New dev", got.Description)

	_, err = svc.Create("go-dev", profile.Profile{
		Description: "Overwrite",
		Targets:     []profile.Target{{Source: "gstack", Skill: "review"}},
	})
	assert.Err(t, err)
	assert.Contains(t, err.Error(), "profile already exists: go-dev")

	existing, err := svc.Show("go-dev")
	assert.NoErr(t, err)
	assert.Eq(t, "Existing", existing.Description)
}
