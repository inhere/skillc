package gitx

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestBuildGitEnvAddsProxyVariables(t *testing.T) {
	got := buildGitEnv([]string{"PATH=/usr/bin"}, "http://localhost:7890")

	assert.Contains(t, got, "HTTP_PROXY=http://localhost:7890")
	assert.Contains(t, got, "HTTPS_PROXY=http://localhost:7890")
	assert.Contains(t, got, "http_proxy=http://localhost:7890")
	assert.Contains(t, got, "https_proxy=http://localhost:7890")
}

func TestBuildGitEnvLeavesEnvUnchangedWhenProxyEmpty(t *testing.T) {
	base := []string{"PATH=/usr/bin"}
	got := buildGitEnv(base, "")

	assert.Eq(t, base, got)
}

func TestCloneCommandUsesProxyButRevParseDoesNot(t *testing.T) {
	client := New("git")

	clone := client.cloneCommand("https://example.com/repo.git", "/tmp/repo", "main", "http://localhost:7890")
	assert.Contains(t, clone.Env, "HTTP_PROXY=http://localhost:7890")
	assert.Contains(t, clone.Env, "HTTPS_PROXY=http://localhost:7890")

	revParse := client.revParseHeadCommand("/tmp/repo")
	assert.Eq(t, 0, len(revParse.Env))
}

func TestClient_SyncWithoutGitBinaryReturnsActionableError(t *testing.T) {
	client := New("__missing_git__")
	_, err := client.Sync("https://example.com/repo.git", filepath.Join(t.TempDir(), "repo"), "main", "http://localhost:7890")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git executable not found")
}
