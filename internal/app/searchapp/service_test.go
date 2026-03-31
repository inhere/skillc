package searchapp

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

func TestService_SearchReturnsEmptyWhenIndexMissing(t *testing.T) {
	service := NewService(filepath.Join(t.TempDir(), "missing.json"))

	results, err := service.Search("design", "", "")
	assert.NoErr(t, err)
	assert.Len(t, results, 0)
}

func TestService_SearchAndShow(t *testing.T) {
	baseDir := t.TempDir()
	indexPath := filepath.Join(baseDir, "index.json")
	store := repoindex.NewStore()
	assert.NoErr(t, store.Save(indexPath, []skill.Skill{
		{
			ID:              "hello-skill",
			Name:            "Hello Skill",
			Description:     "Friendly greeting helper",
			SupportedAgents: []string{"claude-code"},
			SourceType:      sourcepkg.TypeLocal,
		},
		{
			ID:              "git-only",
			Name:            "Git Only",
			Description:     "Remote repo helper",
			SupportedAgents: []string{"codex"},
			SourceType:      sourcepkg.TypeGit,
		},
	}))

	service := NewService(indexPath)

	results, err := service.Search("greeting", "claude-code", sourcepkg.TypeLocal)
	assert.NoErr(t, err)
	assert.Len(t, results, 1)
	assert.Eq(t, "hello-skill", results[0].ID)

	item, err := service.Show("git-only")
	assert.NoErr(t, err)
	assert.Eq(t, "git-only", item.ID)
}
