package listapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/config"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/infra/lockstore"
)

// 记录里存的是别名时，读取和过滤都要按正式名称统一处理。
func TestService_ListCanonicalizesAgentAliases(t *testing.T) {
	baseDir := t.TempDir()
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	conf := config.DefaultConfig()
	conf.AgentTools["universal"] = config.AgentToolConfig{
		Dirname:    ".agents",
		ProjectDir: filepath.Join(baseDir, ".agents"),
		Aliases:    []string{"agents"},
	}
	assert.NoErr(t, os.MkdirAll(filepath.Join(baseDir, ".agents", "skills", "hello-skill"), 0o755))

	assert.NoErr(t, lockstore.NewStore().Save(lockFile, lockpkg.File{
		filepath.Clean(baseDir): {
			{
				SkillID:  "hello-skill",
				SourceID: "local-demo",
				Agents:   []string{"agents"},
			},
		},
	}))

	for _, query := range []string{"universal", "agents", ""} {
		items, err := NewService(lockFile).WithRuntime(conf, baseDir).List(query, "project")
		assert.NoErr(t, err)
		assert.Len(t, items, 1)
		assert.Eq(t, "universal", items[0].Agent)
	}
}
