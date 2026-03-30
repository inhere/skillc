package lockstore

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
)

func TestStore_SaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skillc-install.lock")
	store := NewStore()
	now := time.Unix(1710000000, 0).UTC()
	want := []lockpkg.Record{{
		SkillID:       "hello-skill",
		Agent:         "claude-code",
		Scope:         "global",
		Version:       "1.0.0",
		SourceID:      "local-demo",
		SourceType:    "local",
		InstalledPath: "/tmp/.claude/skills/hello-skill",
		Checksum:      "abc123",
		InstalledAt:   now,
		UpdatedAt:     now,
		Pinned:        true,
	}}

	assert.NoErr(t, store.Save(path, want))

	got, err := store.Load(path)
	assert.NoErr(t, err)
	assert.Eq(t, want, got)
}
