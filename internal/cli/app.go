package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gookit/gcli/v3"
	"github.com/inhere/skillc/internal/app/configapp"
	"github.com/inhere/skillc/internal/app/doctorapp"
	"github.com/inhere/skillc/internal/app/searchapp"
	"github.com/inhere/skillc/internal/app/sourceapp"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
)

func NewApp(version, gitHash, buildTime string) *gcli.App {
	app := gcli.NewApp()
	app.Name = "skillc"
	app.Desc = "Skill manager for multi-agent ecosystems"
	app.Version = fmt.Sprintf("%s (Git Hash: %s, Build Time: %s)", version, gitHash, buildTime)
	app.Add(buildConfigCommand())
	app.Add(buildSourceCommand())
	app.Add(buildCollectionCommand())
	app.Add(buildSearchCommand())
	app.Add(buildShowCommand())
	app.Add(buildInstallCommand())
	app.Add(buildUpdateCommand())
	app.Add(buildUninstallCommand())
	app.Add(buildListCommand())
	app.Add(buildDoctorCommand())
	return app
}

func getWorkdir() string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return cwd
}

func newConfigService() *configapp.Service {
	cwd := getWorkdir()
	return configapp.NewService(defaultConfigFile(cwd), cwd)
}

func newSearchService() *searchapp.Service {
	config, _, err := loadConfig()
	if err != nil {
		return searchapp.NewService(filepath.Join(getWorkdir(), "skillc-index.json"))
	}
	return searchapp.NewService(config.IndexFile)
}

func newSourceService() *sourceapp.Service {
	cwd := getWorkdir()
	return sourceapp.NewService(defaultConfigFile(cwd), cwd)
}

func newDoctorService() *doctorapp.Service {
	cwd := getWorkdir()
	return doctorapp.NewService(defaultConfigFile(cwd), cwd)
}

func loadConfig() (cfg.Config, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return cfg.Config{}, "", err
	}
	config, err := configapp.NewService(defaultConfigFile(cwd), cwd).Show()
	if err != nil {
		return cfg.Config{}, "", err
	}
	return config, cwd, nil
}

func parseScope(value string) (agent.Scope, error) {
	scope := agent.Scope(value)
	switch scope {
	case agent.ScopeUser, agent.ScopeProject:
		return scope, nil
	default:
		return "", fmt.Errorf("unsupported scope: %s", value)
	}
}

func defaultConfigFile(baseDir string) string {
	localPath := filepath.Join(baseDir, "skillc.yaml")
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return localPath
	}
	return filepath.Join(home, ".config", "skillc", "config.yaml")
}
