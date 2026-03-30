package configstore

import (
	"os"
	"path/filepath"

	gkconfig "github.com/gookit/config/v2"
	gkyaml "github.com/gookit/config/v2/yaml"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/infra/fsx"
)

type YAMLStore struct{}

func NewYAMLStore() *YAMLStore {
	return &YAMLStore{}
}

func (s *YAMLStore) Load(path string, baseDir string) (cfg.Config, error) {
	if path == "" {
		data := cfg.DefaultConfig()
		return expandRuntimePaths(data, baseDir)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		data := cfg.DefaultConfig()
		return expandRuntimePaths(data, baseDir)
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
	return expandRuntimePaths(out, baseDir)
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

func expandRuntimePaths(data cfg.Config, baseDir string) (cfg.Config, error) {
	var err error
	data.LockFile, err = fsx.ExpandPath(data.LockFile, baseDir)
	if err != nil {
		return cfg.Config{}, err
	}
	data.RepoCacheDir, err = fsx.ExpandPath(data.RepoCacheDir, baseDir)
	if err != nil {
		return cfg.Config{}, err
	}
	data.SkillCacheDir, err = fsx.ExpandPath(data.SkillCacheDir, baseDir)
	if err != nil {
		return cfg.Config{}, err
	}
	data.RegistryCacheDir, err = fsx.ExpandPath(data.RegistryCacheDir, baseDir)
	if err != nil {
		return cfg.Config{}, err
	}

	for name, tool := range data.AgentTools {
		if tool.UserDir != "" {
			tool.UserDir, err = fsx.ExpandPath(tool.UserDir, baseDir)
			if err != nil {
				return cfg.Config{}, err
			}
		}
		if tool.ProjectDir != "" {
			tool.ProjectDir, err = fsx.ExpandPath(tool.ProjectDir, baseDir)
			if err != nil {
				return cfg.Config{}, err
			}
		}
		data.AgentTools[name] = tool
	}

	for i, src := range data.Sources {
		if src.Path == "" {
			continue
		}
		src.Path, err = fsx.ExpandPath(src.Path, baseDir)
		if err != nil {
			return cfg.Config{}, err
		}
		data.Sources[i] = src
	}

	return data, nil
}
