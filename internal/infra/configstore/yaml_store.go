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
	ID           string            `yaml:"id"`
	Type         domainsource.Type `yaml:"type"`
	Name         string            `yaml:"name"`
	Path         string            `yaml:"path,omitempty"`
	URL          string            `yaml:"url,omitempty"`
	Ref          string            `yaml:"ref,omitempty"`
	ResolvedRef  string            `yaml:"resolved_ref,omitempty"`
	LastSyncAt   string            `yaml:"last_sync_at,omitempty"`
	Status       string            `yaml:"status,omitempty"`
	ErrorMessage string            `yaml:"error_message,omitempty"`
}

type rawConfig struct {
	ProxyURL string `yaml:"proxy_url"`
	// AgentTools is the agent tools config.
	AgentTools map[string]cfg.AgentToolConfig `yaml:"agent_tools"`
	// LockFile is the lock file path.
	LockFile         string         `yaml:"lock_file"`
	RepoCacheDir     string         `yaml:"repo_cache_dir"`
	SkillCacheDir    string         `yaml:"skill_cache_dir"`
	RegistryCacheDir string         `yaml:"registry_cache_dir"`
	IndexFile        string         `yaml:"index_file"`
	Sources          []sourceRecord `yaml:"sources"`
}

func NewYAMLStore() *YAMLStore {
	return &YAMLStore{}
}

func newYamlLoader() *gkconfig.Config {
	loader := gkconfig.NewEmpty("skillc", gkconfig.ParseEnv, gkconfig.WithTagName("yaml"))
	loader.AddDriver(gkyaml.Driver)
	return loader
}

func (s *YAMLStore) Load(path string, baseDir string) (cfg.Config, error) {
	defaults := cfg.DefaultConfig()
	if path == "" {
		return expandRuntimePaths(defaults, baseDir)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return expandRuntimePaths(defaults, baseDir)
	} else if err != nil {
		return cfg.Config{}, err
	}

	loader := newYamlLoader()
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

	mergeDefaults(&out, defaults)
	return expandRuntimePaths(out, baseDir)
}

func (s *YAMLStore) Save(path string, data cfg.Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	loader := newYamlLoader()
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
	} else if _, ok := dst.AgentTools["agents"]; !ok {
		// 通用的key agents 必定存在
		dst.AgentTools["agents"] = defaults.AgentTools["agents"]
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
