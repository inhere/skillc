package configstore

import (
	"os"
	"path/filepath"

	gkconfig "github.com/gookit/config/v2"
	gkyaml "github.com/gookit/config/v2/yaml"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/profile"
	"github.com/inhere/skillc/internal/domain/project"
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

type projectRecord struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name,omitempty"`
	Path        string `yaml:"path"`
	Description string `yaml:"description,omitempty"`
}

type rawConfig struct {
	ProxyURL string `yaml:"proxy_url"`
	// AgentTools is the agent tools config.
	AgentTools  map[string]cfg.AgentToolConfig `yaml:"agent_tools"`
	InstallMode string                         `yaml:"install_mode"`
	// LockFile is the lock file path.
	LockFile         string                     `yaml:"lock_file"`
	RepoCacheDir     string                     `yaml:"repo_cache_dir"`
	SkillCacheDir    string                     `yaml:"skill_cache_dir"`
	RegistryCacheDir string                     `yaml:"registry_cache_dir"`
	IndexFile        string                     `yaml:"index_file"`
	Sources          []sourceRecord             `yaml:"sources"`
	Profiles         map[string]profile.Profile `yaml:"profiles,omitempty"`
	Projects         []projectRecord            `yaml:"projects,omitempty"`
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

func (s *YAMLStore) Save(path string, data cfg.Config, runtimeBaseDir ...string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	baseDir := filepath.Dir(path)
	if len(runtimeBaseDir) > 0 && runtimeBaseDir[0] != "" {
		baseDir = runtimeBaseDir[0]
	}

	existingRaw, hasExisting, err := loadRawConfig(path)
	if err != nil {
		return err
	}
	persisted, err := compactRuntimePaths(cloneConfig(data), baseDir, existingRaw, hasExisting)
	if err != nil {
		return err
	}

	loader := newYamlLoader()
	out := map[string]any{
		"proxy_url":          persisted.ProxyURL,
		"agent_tools":        persisted.AgentTools,
		"install_mode":       persisted.InstallMode,
		"lock_file":          persisted.LockFile,
		"repo_cache_dir":     persisted.RepoCacheDir,
		"skill_cache_dir":    persisted.SkillCacheDir,
		"registry_cache_dir": persisted.RegistryCacheDir,
		"index_file":         persisted.IndexFile,
		"sources":            toSourceRecords(persisted.Sources),
	}
	if len(persisted.Profiles) > 0 {
		out["profiles"] = persisted.Profiles
	}
	if len(persisted.Projects) > 0 {
		out["projects"] = toProjectRecords(persisted.Projects)
	}
	loader.SetData(out)
	return loader.DumpToFile(path, gkconfig.Yaml)
}

func mergeDefaults(dst *cfg.Config, defaults cfg.Config) {
	if dst.AgentTools == nil {
		dst.AgentTools = defaults.AgentTools
	} else if _, ok := dst.AgentTools[cfg.AgentToolNameUniversal]; !ok {
		// 通用的key universal 必定存在
		dst.AgentTools[cfg.AgentToolNameUniversal] = defaults.AgentTools[cfg.AgentToolNameUniversal]
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
	if dst.InstallMode == "" {
		dst.InstallMode = defaults.InstallMode
	}
	if dst.Sources == nil {
		dst.Sources = defaults.Sources
	}
}

func loadRawConfig(path string) (rawConfig, bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return rawConfig{}, false, nil
	} else if err != nil {
		return rawConfig{}, false, err
	}

	loader := newYamlLoader()
	if err := loader.LoadFiles(path); err != nil {
		return rawConfig{}, false, err
	}

	var raw rawConfig
	if err := loader.Decode(&raw); err != nil {
		return rawConfig{}, false, err
	}
	return raw, true, nil
}

func cloneConfig(data cfg.Config) cfg.Config {
	clone := data
	if data.AgentTools != nil {
		clone.AgentTools = make(map[string]cfg.AgentToolConfig, len(data.AgentTools))
		for name, tool := range data.AgentTools {
			clone.AgentTools[name] = tool
		}
	}
	if data.Sources != nil {
		clone.Sources = append([]domainsource.Source(nil), data.Sources...)
	}
	if data.Profiles != nil {
		clone.Profiles = make(map[string]profile.Profile, len(data.Profiles))
		for name, item := range data.Profiles {
			item.Targets = append([]profile.Target(nil), item.Targets...)
			clone.Profiles[name] = item
		}
	}
	if data.Projects != nil {
		clone.Projects = append([]project.Project(nil), data.Projects...)
	}
	return clone
}

func compactRuntimePaths(data cfg.Config, baseDir string, existing rawConfig, hasExisting bool) (cfg.Config, error) {
	defaults := cfg.DefaultConfig()
	var err error

	data.LockFile, err = compactPath(data.LockFile, existing.LockFile, defaults.LockFile, baseDir, hasExisting)
	if err != nil {
		return cfg.Config{}, err
	}
	data.RepoCacheDir, err = compactPath(data.RepoCacheDir, existing.RepoCacheDir, defaults.RepoCacheDir, baseDir, hasExisting)
	if err != nil {
		return cfg.Config{}, err
	}
	data.SkillCacheDir, err = compactPath(data.SkillCacheDir, existing.SkillCacheDir, defaults.SkillCacheDir, baseDir, hasExisting)
	if err != nil {
		return cfg.Config{}, err
	}
	data.RegistryCacheDir, err = compactPath(data.RegistryCacheDir, existing.RegistryCacheDir, defaults.RegistryCacheDir, baseDir, hasExisting)
	if err != nil {
		return cfg.Config{}, err
	}
	data.IndexFile, err = compactPath(data.IndexFile, existing.IndexFile, defaults.IndexFile, baseDir, hasExisting)
	if err != nil {
		return cfg.Config{}, err
	}

	for name, tool := range data.AgentTools {
		var existingTool cfg.AgentToolConfig
		if hasExisting && existing.AgentTools != nil {
			existingTool = existing.AgentTools[name]
		}
		defaultTool, hasDefault := defaults.AgentTools[name]
		if tool.UserDir != "" {
			defaultRaw := ""
			if hasDefault {
				defaultRaw = defaultTool.UserDir
			}
			tool.UserDir, err = compactPath(tool.UserDir, existingTool.UserDir, defaultRaw, baseDir, hasExisting)
			if err != nil {
				return cfg.Config{}, err
			}
		}
		if tool.ProjectDir != "" {
			defaultRaw := ""
			if hasDefault {
				defaultRaw = defaultTool.ProjectDir
			}
			tool.ProjectDir, err = compactPath(tool.ProjectDir, existingTool.ProjectDir, defaultRaw, baseDir, hasExisting)
			if err != nil {
				return cfg.Config{}, err
			}
		}
		data.AgentTools[name] = tool
	}

	for i, item := range data.Projects {
		existingRaw := ""
		if hasExisting {
			for _, existingProject := range existing.Projects {
				if existingProject.ID == item.ID {
					existingRaw = existingProject.Path
					break
				}
			}
		}
		item.Path, err = compactPath(item.Path, existingRaw, "", baseDir, hasExisting)
		if err != nil {
			return cfg.Config{}, err
		}
		data.Projects[i] = item
	}

	return data, nil
}

func compactPath(current string, existingRaw string, defaultRaw string, baseDir string, hasExisting bool) (string, error) {
	if current == "" {
		return "", nil
	}
	if hasExisting && existingRaw != "" {
		expandedExisting, err := fsx.ExpandPath(existingRaw, baseDir)
		if err != nil {
			return "", err
		}
		if current == expandedExisting {
			return existingRaw, nil
		}
	}
	if defaultRaw != "" {
		expandedDefault, err := fsx.ExpandPath(defaultRaw, baseDir)
		if err != nil {
			return "", err
		}
		if current == expandedDefault {
			return defaultRaw, nil
		}
	}
	return current, nil
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

	for i, item := range data.Projects {
		if item.Path == "" {
			continue
		}
		item.Path, err = fsx.ExpandPath(item.Path, baseDir)
		if err != nil {
			return cfg.Config{}, err
		}
		item.Path = filepath.Clean(item.Path)
		data.Projects[i] = item
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

func toProjectRecords(projects []project.Project) []projectRecord {
	if len(projects) == 0 {
		return nil
	}
	records := make([]projectRecord, 0, len(projects))
	for _, item := range projects {
		records = append(records, projectRecord{
			ID:          item.ID,
			Name:        item.Name,
			Path:        item.Path,
			Description: item.Description,
		})
	}
	return records
}

func fromProjectRecords(records []projectRecord) []project.Project {
	if len(records) == 0 {
		return []project.Project{}
	}
	items := make([]project.Project, 0, len(records))
	for _, record := range records {
		items = append(items, project.Project{
			ID:          record.ID,
			Name:        record.Name,
			Path:        record.Path,
			Description: record.Description,
		})
	}
	return items
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
		InstallMode:      raw.InstallMode,
		LockFile:         raw.LockFile,
		RepoCacheDir:     raw.RepoCacheDir,
		SkillCacheDir:    raw.SkillCacheDir,
		RegistryCacheDir: raw.RegistryCacheDir,
		IndexFile:        raw.IndexFile,
		Sources:          sources,
		Profiles:         raw.Profiles,
		Projects:         fromProjectRecords(raw.Projects),
	}, nil
}
