package installapp

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/agent"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
)

func TestService_InstallCopiesFilesAndWritesLock(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "source")
	lockFile := filepath.Join(baseDir, "skillc-install.lock")
	targetRoot := filepath.Join(baseDir, ".claude", "skills")
	assert.NoErr(t, os.MkdirAll(filepath.Join(sourceDir, "commands"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(sourceDir, "commands", "hello.txt"), []byte("hello"), 0o644))

	item := skill.Skill{
		ID:           "hello-skill",
		Name:         "Hello Skill",
		Version:      "1.0.0",
		SourceID:     "local-demo",
		SourceType:   sourcepkg.TypeLocal,
		InstallEntry: "commands",
		Path:         sourceDir,
	}

	service := NewService(lockFile)
	service.now = func() time.Time { return time.Unix(1710000000, 0).UTC() }

	record, err := service.Install(item, "claude-code", agent.ScopeProject, targetRoot)
	assert.NoErr(t, err)
	assert.Eq(t, "hello-skill", record.SkillID)
	assert.Eq(t, filepath.Join(targetRoot, "hello-skill"), record.InstalledPath)

	data, err := os.ReadFile(filepath.Join(targetRoot, "hello-skill", "hello.txt"))
	assert.NoErr(t, err)
	assert.Eq(t, "hello", string(data))

	locks, err := service.store.Load(lockFile)
	assert.NoErr(t, err)
	assert.Len(t, locks, 1)
	assert.Eq(t, record.SkillID, locks[0].SkillID)
}
