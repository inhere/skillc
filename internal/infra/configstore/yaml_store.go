package configstore

import (
	"os"
	"path/filepath"

	gkconfig "github.com/gookit/config/v2"
	gkyaml "github.com/gookit/config/v2/yaml"
	cfg "github.com/inhere/skillc/internal/domain/config"
	domainsource "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/fsx"
)

type YAMLStore struct{}

type sourceRecord struct {
	ID           string             `yaml:"id" mapstructure:"id"`
	Type         domainsource.Type  `yaml:"type" mapstructure:"type"`
	Name         string             `yaml:"name" mapstructure:"name"`
	Path         string             `yaml:"path" mapstructure:"path"`
	URL          string             `yaml:"url" mapstructure:"url"`
	Ref          string             `yaml:"ref" mapstructure:"ref"`
	ResolvedRef  string             `yaml:"resolved_ref" mapstructure:"resolved_ref"`
	LastSyncAt   string             `yaml:"last_sync_at" mapstructure:"last_sync_at"`
	Status       string             `yaml:"status" mapstructure:"status"`
	ErrorMessage string             `yaml:"error_message" mapstructure:"error_message"`
}

type rawConfig struct {
	ProxyURL         string                `mapstructure:"proxy_url"`
	AgentTools       map[string]cfg.AgentToolConfig `mapstructure:"agent_tools"`
	LockFile         string                `mapstructure:"lock_file"`
	RepoCacheDir     string                `mapstructure:"repo_cache_dir"`
	SkillCacheDir    string                `mapstructure:"skill_cache_dir"`
	RegistryCacheDir string                `mapstructure:"registry_cache_dir"`
	IndexFile        string                `mapstructure:"index_file"`
	Sources          []sourceRecord        `mapstructure:"sources"`
}

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

	var raw rawConfig
	if err := loader.Decode(&raw); err != nil {
		return cfg.Config{}, err
	}

	out, err := fromRawConfig(raw)
	if err != nil {
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
		"index_file":         data.IndexFile,
		"sources":            toSourceRecords(data.Sources),
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
	if dst.IndexFile == "" {
		dst.IndexFile = defaults.IndexFile
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
	data.IndexFile, err = fsx.ExpandPath(data.IndexFile, baseDir)
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

func toSourceRecords(sources []domainsource.Source) []sourceRecord {
	if len(sources) == 0 {
		return []sourceRecord{}
	}
	records := make([]sourceRecord, 0, len(sources))
	for _, src := range sources {
		record := sourceRecord{
			ID:           src.ID,
			Type:         src.Type,
			Name:         src.Name,
			Path:         src.Path,
			URL:          src.URL,
			Ref:          src.Ref,
			ResolvedRef:  src.ResolvedRef,
			LastSyncAt:   src.LastSyncAt,
			Status:       src.Status,
			ErrorMessage: src.ErrorMessage,
		}
		records = append(records, record)
	}
	return records
}

func fromRawConfig(raw rawConfig) (cfg.Config, error) {
	sources := make([]domainsource.Source, 0, len(raw.Sources))
	for _, src := range raw.Sources {
		item := domainsource.Source{
			ID:           src.ID,
			Type:         src.Type,
			Name:         src.Name,
			Path:         src.Path,
			URL:          src.URL,
			Ref:          src.Ref,
			ResolvedRef:  src.ResolvedRef,
			LastSyncAt:   src.LastSyncAt,
			Status:       src.Status,
			ErrorMessage: src.ErrorMessage,
		}
		sources = append(sources, item)
	}
	return cfg.Config{
		ProxyURL:         raw.ProxyURL,
		AgentTools:       raw.AgentTools,
		LockFile:         raw.LockFile,
		RepoCacheDir:     raw.RepoCacheDir,
		SkillCacheDir:    raw.SkillCacheDir,
		RegistryCacheDir: raw.RegistryCacheDir,
		IndexFile:        raw.IndexFile,
		Sources:          sources,
	}, nil
}
