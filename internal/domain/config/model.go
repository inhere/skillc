package config

import domainsource "github.com/inhere/skillc/internal/domain/source"

type Config struct {
	ProxyURL string `yaml:"proxy_url"`
	// AgentTools is the agent tools config.
	//  - 通用的key agents 必定存在
	AgentTools map[string]AgentToolConfig `yaml:"agent_tools"`
	// LockFile is the lock file path.
	LockFile         string                `yaml:"lock_file"`
	RepoCacheDir     string                `yaml:"repo_cache_dir"`
	SkillCacheDir    string                `yaml:"skill_cache_dir"`
	RegistryCacheDir string                `yaml:"registry_cache_dir"`
	IndexFile        string                `yaml:"index_file"`
	Sources          []domainsource.Source `yaml:"sources"`
}

type AgentToolConfig struct {
	Dirname    string   `yaml:"dirname"`
	Aliases    []string `yaml:"aliases,omitempty"`
	UserDir    string   `yaml:"user_dir,omitempty"`
	ProjectDir string   `yaml:"project_dir,omitempty"`
}

// ResolveAgentTool looks up an agent tool by name or alias.
// Returns the canonical name and config, or ok=false if not found.
func (c *Config) ResolveAgentTool(nameOrAlias string) (canonicalName string, tool AgentToolConfig, ok bool) {
	if t, exists := c.AgentTools[nameOrAlias]; exists {
		return nameOrAlias, t, true
	}
	for name, t := range c.AgentTools {
		for _, alias := range t.Aliases {
			if alias == nameOrAlias {
				return name, t, true
			}
		}
	}
	return "", AgentToolConfig{}, false
}

func (atc *AgentToolConfig) GetUserDir() string {
	if atc.UserDir == "" {
		return "~/" + atc.Dirname
	}
	return atc.UserDir
}

func (atc *AgentToolConfig) GetProjectDir() string {
	if atc.ProjectDir == "" {
		return atc.Dirname
	}
	return atc.ProjectDir
}
