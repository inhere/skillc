package gitx

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestClient_SyncWithoutGitBinaryReturnsActionableError(t *testing.T) {
	client := New("__missing_git__")
	_, err := client.Sync("https://example.com/repo.git", filepath.Join(t.TempDir(), "repo"), "main")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git executable not found")
}
