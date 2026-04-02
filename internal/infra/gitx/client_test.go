package gitx

import (
	"bytes"
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

func TestCloneCommandUsesProxyAndProgressOptions(t *testing.T) {
	client := New("git")
	progress := &bytes.Buffer{}
	cmd := client.cloneCommand("https://example.com/repo.git", "/tmp/repo", "main", SyncOptions{
		ProxyURL: "http://localhost:7890",
		Progress: progress,
	})

	assert.Contains(t, cmd.Args, "--progress")
	assert.Contains(t, cmd.Env, "HTTP_PROXY=http://localhost:7890")
	assert.Contains(t, cmd.Env, "HTTPS_PROXY=http://localhost:7890")
	if cmd.Stdout != progress {
		t.Fatalf("expected stdout to use progress writer")
	}
	if cmd.Stderr != progress {
		t.Fatalf("expected stderr to use progress writer")
	}
}

func TestCloneCommandWithoutProgressLeavesWritersNil(t *testing.T) {
	client := New("git")
	cmd := client.cloneCommand("https://example.com/repo.git", "/tmp/repo", "main", SyncOptions{
		ProxyURL: "http://localhost:7890",
	})

	assert.NotContains(t, cmd.Args, "--progress")
	assert.Contains(t, cmd.Env, "HTTP_PROXY=http://localhost:7890")
	if cmd.Stdout != nil {
		t.Fatalf("expected stdout to be nil without progress writer")
	}
	if cmd.Stderr != nil {
		t.Fatalf("expected stderr to be nil without progress writer")
	}
}

func TestRevParseCommandIgnoresProxyAndProgress(t *testing.T) {
	client := New("git")
	cmd := client.revParseHeadCommand("/tmp/repo")

	assert.Eq(t, 0, len(cmd.Env))
	if cmd.Stdout != nil {
		t.Fatalf("expected rev-parse stdout to remain nil")
	}
	if cmd.Stderr != nil {
		t.Fatalf("expected rev-parse stderr to remain nil")
	}
}

func TestClient_SyncWithoutGitBinaryReturnsActionableError(t *testing.T) {
	client := New("__missing_git__")
	_, err := client.Sync("https://example.com/repo.git", filepath.Join(t.TempDir(), "repo"), "main", SyncOptions{ProxyURL: "http://localhost:7890"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git executable not found")
}
