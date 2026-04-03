package config

import domainsource "github.com/inhere/skillc/internal/domain/source"

func DefaultConfig() Config {
	return Config{
		AgentTools: map[string]AgentToolConfig{
			"agents": { // 通用的key agents 必定存在
				Dirname:    ".agents",
				UserDir:    "~/.agents",
				ProjectDir: ".agents",
			},
			"claude-code": {
				Dirname:    ".claude",
				UserDir:    "~/.claude",
				ProjectDir: "./.claude",
			},
			"opencode": {
				Dirname:    ".opencode",
				UserDir:    "~/.config/opencode",
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
		IndexFile:        "~/.cache/skillc/skillc-index.json",
		Sources:          []domainsource.Source{},
	}
}
