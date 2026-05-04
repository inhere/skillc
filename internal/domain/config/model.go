package config

import (
	"slices"

	domainsource "github.com/inhere/skillc/internal/domain/source"
)

const (
	AgentToolNameUniversal = "universal"
)

type Config struct {
	ProxyURL string `yaml:"proxy_url"`
	// AgentTools is the agent tools config.
	//  - 通用的key universal 必定存在
	AgentTools map[string]AgentToolConfig `yaml:"agent_tools"`
	// InstallMode 控制 skill 安装方式: "symlink"、"junction" 或 "copy"；空值使用平台默认。
	// 平台默认：Windows 使用 junction，其他系统使用 symlink。
	InstallMode string `yaml:"install_mode"`
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
		if len(t.Aliases) > 0 && slices.Contains(t.Aliases, nameOrAlias) {
			return name, t, true
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
