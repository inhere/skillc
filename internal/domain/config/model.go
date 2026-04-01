package config

import domainsource "github.com/inhere/skillc/internal/domain/source"

type Config struct {
	ProxyURL         string                `yaml:"proxy_url"`
	// AgentTools is the agent tools config.
	AgentTools       map[string]AgentToolConfig `yaml:"agent_tools"`
	// LockFile is the lock file path.
	LockFile         string                `yaml:"lock_file"`
	RepoCacheDir     string                `yaml:"repo_cache_dir"`
	SkillCacheDir    string                `yaml:"skill_cache_dir"`
	RegistryCacheDir string                `yaml:"registry_cache_dir"`
	IndexFile        string                `yaml:"index_file"`
	Sources          []domainsource.Source `yaml:"sources"`
}

type AgentToolConfig struct {
	Dirname    string `yaml:"dirname"`
	UserDir    string `yaml:"user"`
	ProjectDir string `yaml:"project"`
}
