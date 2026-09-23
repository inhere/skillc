package config

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestConfig_CanonicalAgentNameResolvesAliases(t *testing.T) {
	config := DefaultConfig()

	assert.Eq(t, "claude-code", config.CanonicalAgentName("claude"))
	assert.Eq(t, "claude-code", config.CanonicalAgentName(" claude-code "))
	assert.Eq(t, "universal", config.CanonicalAgentName("agents"))
	assert.Eq(t, "universal", config.CanonicalAgentName("universal"))
	// 未注册的名称（目录名/自定义名称）原样保留
	assert.Eq(t, ".my-agents", config.CanonicalAgentName(".my-agents"))
	assert.Eq(t, "", config.CanonicalAgentName("  "))
}

func TestConfig_CanonicalAgentNamesSplitsAndDedupes(t *testing.T) {
	config := DefaultConfig()

	assert.Eq(t, []string{"claude-code", "universal"}, config.CanonicalAgentNames("claude, agents,claude-code"))
	assert.Eq(t, []string{"claude-code"}, config.CanonicalAgentNames("claude"))
	assert.Len(t, config.CanonicalAgentNames(" , "), 0)
}
