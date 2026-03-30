package config

func DefaultConfig() Config {
	return Config{
		AgentTools: map[string]AgentToolConfig{
			"claude-code": {
				Dirname:    ".claude",
				UserDir:    "~/.claude",
				ProjectDir: "./.claude",
			},
			"opencode": {
				Dirname:    ".opencode",
				UserDir:    "~/.opencode",
				ProjectDir: "./.opencode",
			},
			"codex": {
				Dirname:    ".codex",
				UserDir:    "~/.codex",
				ProjectDir: "./.codex",
			},
		},
		LockFile:         "~/.config/skillc/skillc-install.lock",
		RepoCacheDir:     "~/.cache/skillc/repos",
		SkillCacheDir:    "~/.cache/skillc/skills",
		RegistryCacheDir: "~/.cache/skillc/registry",
	}
}
