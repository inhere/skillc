package config

type Config struct {
	ProxyURL         string                      `yaml:"proxy_url" mapstructure:"proxy_url"`
	AgentTools       map[string]AgentToolConfig `yaml:"agent_tools" mapstructure:"agent_tools"`
	LockFile         string                      `yaml:"lock_file" mapstructure:"lock_file"`
	RepoCacheDir     string                      `yaml:"repo_cache_dir" mapstructure:"repo_cache_dir"`
	SkillCacheDir    string                      `yaml:"skill_cache_dir" mapstructure:"skill_cache_dir"`
	RegistryCacheDir string                      `yaml:"registry_cache_dir" mapstructure:"registry_cache_dir"`
}

type AgentToolConfig struct {
	Dirname    string `yaml:"dirname" mapstructure:"dirname"`
	UserDir    string `yaml:"user_dir" mapstructure:"user_dir"`
	ProjectDir string `yaml:"project_dir" mapstructure:"project_dir"`
}
