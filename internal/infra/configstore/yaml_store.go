package configstore

import (
	"os"
	"path/filepath"

	gkconfig "github.com/gookit/config/v2"
	gkyaml "github.com/gookit/config/v2/yaml"
	cfg "github.com/inhere/skillc/internal/domain/config"
)

type YAMLStore struct{}

func NewYAMLStore() *YAMLStore {
	return &YAMLStore{}
}

func (s *YAMLStore) Load(path string, baseDir string) (cfg.Config, error) {
	if path == "" {
		return cfg.DefaultConfig(), nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg.DefaultConfig(), nil
	} else if err != nil {
		return cfg.Config{}, err
	}

	loader := gkconfig.NewEmpty("skillc", gkconfig.ParseEnv)
	loader.AddDriver(gkyaml.Driver)
	if err := loader.LoadFiles(path); err != nil {
		return cfg.Config{}, err
	}

	var out cfg.Config
	if err := loader.Decode(&out); err != nil {
		return cfg.Config{}, err
	}

	defaults := cfg.DefaultConfig()
	mergeDefaults(&out, defaults)
	return out, nil
}

func (s *YAMLStore) Save(path string, data cfg.Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	loader := gkconfig.NewEmpty("skillc")
	loader.AddDriver(gkyaml.Driver)
	loader.SetData(map[string]any{
		"proxy_url":          data.ProxyURL,
		"agent_tools":        data.AgentTools,
		"lock_file":          data.LockFile,
		"repo_cache_dir":     data.RepoCacheDir,
		"skill_cache_dir":    data.SkillCacheDir,
		"registry_cache_dir": data.RegistryCacheDir,
		"sources":            data.Sources,
	})
	return loader.DumpToFile(path, gkconfig.Yaml)
}

func mergeDefaults(dst *cfg.Config, defaults cfg.Config) {
	if dst.AgentTools == nil {
		dst.AgentTools = defaults.AgentTools
	}
	if dst.LockFile == "" {
		dst.LockFile = defaults.LockFile
	}
	if dst.RepoCacheDir == "" {
		dst.RepoCacheDir = defaults.RepoCacheDir
	}
	if dst.SkillCacheDir == "" {
		dst.SkillCacheDir = defaults.SkillCacheDir
	}
	if dst.RegistryCacheDir == "" {
		dst.RegistryCacheDir = defaults.RegistryCacheDir
	}
	if dst.Sources == nil {
		dst.Sources = defaults.Sources
	}
}
