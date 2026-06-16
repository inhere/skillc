package config

import (
	"github.com/inhere/skillc/internal/domain/project"
	"github.com/inhere/skillc/internal/domain/registry"
	domainsource "github.com/inhere/skillc/internal/domain/source"
)

func DefaultConfig() Config {
	return Config{
		AgentTools: map[string]AgentToolConfig{
			AgentToolNameUniversal: { // 通用的key universal 必定存在
				Dirname:    ".agents",
				UserDir:    "~/.agents",
				ProjectDir: ".agents",
				Aliases:    []string{"universal"},
			},
			"claude-code": {
				Dirname:    ".claude",
				Aliases:    []string{"claude"},
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
		InstallMode:      "",
		RepoCacheDir:     "~/.cache/skillc/repos",
		SkillCacheDir:    "~/.cache/skillc/skills",
		RegistryCacheDir: "~/.cache/skillc/registry",
		IndexFile:        "~/.cache/skillc/skillc-index.json",
		Sources:          []domainsource.Source{},
		Registries:       []registry.Registry{},
		Projects:         []project.Project{},
	}
}
