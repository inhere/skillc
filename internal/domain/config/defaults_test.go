package config

import "testing"

func TestDefaultAgentDirname_Codex(t *testing.T) {
	cfg := DefaultConfig()
	got := cfg.AgentTools["codex"].Dirname
	if got != ".codex" {
		t.Fatalf("got %q want .codex", got)
	}
}

func TestDefaultConfig_HasExpectedAgentDefaults(t *testing.T) {
	cfg := DefaultConfig()

	cases := map[string]string{
		"claude-code": ".claude",
		"opencode":    ".opencode",
		"codex":       ".codex",
	}

	for name, dirname := range cases {
		tool, ok := cfg.AgentTools[name]
		if !ok {
			t.Fatalf("missing agent tool %q", name)
		}
		if tool.Dirname != dirname {
			t.Fatalf("agent %s dirname got %q want %q", name, tool.Dirname, dirname)
		}
	}
}

func TestDefaultConfig_HasDefaultPaths(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.LockFile == "" {
		t.Fatal("expected lock file path")
	}
	if cfg.RepoCacheDir == "" {
		t.Fatal("expected repo cache dir")
	}
	if cfg.SkillCacheDir == "" {
		t.Fatal("expected skill cache dir")
	}
	if cfg.RegistryCacheDir == "" {
		t.Fatal("expected registry cache dir")
	}
}
