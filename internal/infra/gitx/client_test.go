package gitx

import (
	"bytes"
	"os"
	"os/exec"
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

func TestClient_SyncClonesWhenCacheMissing(t *testing.T) {
	remoteDir, mainHead := createRemoteRepoWithMain(t)
	client := New("git")
	cacheDir := filepath.Join(t.TempDir(), "cache")

	resolved, err := client.Sync(remoteDir, cacheDir, "main", SyncOptions{})
	assert.Nil(t, err)
	assert.Eq(t, mainHead, resolved)
	assert.Eq(t, mainHead, gitHead(t, cacheDir))
}

func TestClient_SyncIncrementallyUpdatesHealthyCache(t *testing.T) {
	remoteDir, _ := createRemoteRepoWithMain(t)
	client := New("git")
	cacheDir := filepath.Join(t.TempDir(), "cache")

	_, err := client.Sync(remoteDir, cacheDir, "main", SyncOptions{})
	assert.Nil(t, err)

	marker := filepath.Join(cacheDir, ".git", "incremental-marker")
	writeFile(t, marker, "keep")
	updatedHead := advanceRemoteMain(t, remoteDir, "second commit")
	resolved, err := client.Sync(remoteDir, cacheDir, "main", SyncOptions{})
	assert.Nil(t, err)
	assert.Eq(t, updatedHead, resolved)
	assert.Eq(t, updatedHead, gitHead(t, cacheDir))
	assertPathExists(t, marker)
}

func TestClient_SyncFallsBackToCloneWhenCacheIsNotGitRepo(t *testing.T) {
	remoteDir, mainHead := createRemoteRepoWithMain(t)
	client := New("git")
	cacheDir := filepath.Join(t.TempDir(), "cache")
	writeFile(t, filepath.Join(cacheDir, "plain.txt"), "not a repo")

	resolved, err := client.Sync(remoteDir, cacheDir, "main", SyncOptions{})
	assert.Nil(t, err)
	assert.Eq(t, mainHead, resolved)
	assert.Eq(t, mainHead, gitHead(t, cacheDir))
}

func TestClient_SyncFallsBackToCloneWhenOriginDoesNotMatch(t *testing.T) {
	remoteDir, mainHead := createRemoteRepoWithMain(t)
	otherRemoteDir, _ := createRemoteRepoWithMain(t)
	client := New("git")
	cacheDir := filepath.Join(t.TempDir(), "cache")

	runGit(t, t.TempDir(), "clone", otherRemoteDir, cacheDir)
	marker := filepath.Join(cacheDir, ".git", "reclone-marker")
	writeFile(t, marker, "drop")

	resolved, err := client.Sync(remoteDir, cacheDir, "main", SyncOptions{})
	assert.Nil(t, err)
	assert.Eq(t, mainHead, resolved)
	assert.Eq(t, remoteDir, gitRemoteURL(t, cacheDir))
	assertPathMissing(t, marker)
}

func TestClient_SyncRemovesStaleUntrackedFilesOnIncrementalSync(t *testing.T) {
	remoteDir, _ := createRemoteRepoWithMain(t)
	client := New("git")
	cacheDir := filepath.Join(t.TempDir(), "cache")

	_, err := client.Sync(remoteDir, cacheDir, "main", SyncOptions{})
	assert.Nil(t, err)

	writeFile(t, filepath.Join(cacheDir, "stale.txt"), "stale")
	_, err = client.Sync(remoteDir, cacheDir, "main", SyncOptions{})
	assert.Nil(t, err)
	if _, statErr := os.Stat(filepath.Join(cacheDir, "stale.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("expected stale file to be removed, got %v", statErr)
	}
}

func TestClient_SyncReturnsResolvedRefFromSynchronizedHead(t *testing.T) {
	remoteDir, _ := createRemoteRepoWithMain(t)
	client := New("git")
	cacheDir := filepath.Join(t.TempDir(), "cache")

	_, err := client.Sync(remoteDir, cacheDir, "main", SyncOptions{})
	assert.Nil(t, err)

	updatedHead := advanceRemoteMain(t, remoteDir, "new head")
	resolved, err := client.Sync(remoteDir, cacheDir, "main", SyncOptions{})
	assert.Nil(t, err)
	assert.Eq(t, updatedHead, resolved)
}

func TestClient_SyncFallsBackToCloneWhenIncrementalSyncFails(t *testing.T) {
	remoteDir, mainHead := createRemoteRepoWithMain(t)
	client := New("git")
	cacheDir := filepath.Join(t.TempDir(), "cache")

	_, err := client.Sync(remoteDir, cacheDir, "main", SyncOptions{})
	assert.Nil(t, err)

	damagedFetchHead := filepath.Join(cacheDir, ".git", "FETCH_HEAD")
	if err := os.MkdirAll(damagedFetchHead, 0o755); err != nil {
		t.Fatalf("create damaged fetch head dir %s: %v", damagedFetchHead, err)
	}
	marker := filepath.Join(cacheDir, ".git", "damaged-marker")
	writeFile(t, marker, "drop")

	resolved, err := client.Sync(remoteDir, cacheDir, "main", SyncOptions{})
	assert.Nil(t, err)
	assert.Eq(t, mainHead, resolved)
	assert.Eq(t, mainHead, gitHead(t, cacheDir))
	assertPathMissing(t, damagedFetchHead)
	assertPathMissing(t, marker)
}

func TestClient_SyncReusesCacheForTagRef(t *testing.T) {
	remoteDir, tagHead := createRemoteRepoWithTag(t, "v1")
	client := New("git")
	cacheDir := filepath.Join(t.TempDir(), "cache")

	resolved, err := client.Sync(remoteDir, cacheDir, "v1", SyncOptions{})
	assert.Nil(t, err)
	assert.Eq(t, tagHead, resolved)

	marker := filepath.Join(cacheDir, ".git", "tag-marker")
	writeFile(t, marker, "keep")
	resolved, err = client.Sync(remoteDir, cacheDir, "v1", SyncOptions{})
	assert.Nil(t, err)
	assert.Eq(t, tagHead, resolved)
	assert.Eq(t, tagHead, gitHead(t, cacheDir))
	assertPathExists(t, marker)
}

func createRemoteRepoWithMain(t *testing.T) (string, string) {
	t.Helper()
	remoteDir := filepath.Join(t.TempDir(), "remote.git")
	workDir := filepath.Join(t.TempDir(), "work")

	runGit(t, t.TempDir(), "init", "--bare", remoteDir)
	runGit(t, t.TempDir(), "clone", remoteDir, workDir)
	runGit(t, workDir, "config", "user.name", "Test User")
	runGit(t, workDir, "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(workDir, "README.md"), "initial\n")
	runGit(t, workDir, "add", "README.md")
	runGit(t, workDir, "commit", "-m", "initial commit")
	runGit(t, workDir, "branch", "-M", "main")
	runGit(t, workDir, "push", "-u", "origin", "main")
	return remoteDir, gitHead(t, workDir)
}

func createRemoteRepoWithTag(t *testing.T, tag string) (string, string) {
	t.Helper()
	remoteDir, head := createRemoteRepoWithMain(t)
	workDir := filepath.Join(t.TempDir(), "work")
	runGit(t, t.TempDir(), "clone", remoteDir, workDir)
	runGit(t, workDir, "checkout", "main")
	runGit(t, workDir, "config", "user.name", "Test User")
	runGit(t, workDir, "config", "user.email", "test@example.com")
	runGit(t, workDir, "tag", tag, head)
	runGit(t, workDir, "push", "origin", tag)
	return remoteDir, head
}

func advanceRemoteMain(t *testing.T, remoteDir string, content string) string {
	t.Helper()
	workDir := filepath.Join(t.TempDir(), "work")
	runGit(t, t.TempDir(), "clone", remoteDir, workDir)
	runGit(t, workDir, "checkout", "main")
	runGit(t, workDir, "config", "user.name", "Test User")
	runGit(t, workDir, "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(workDir, "README.md"), content+"\n")
	runGit(t, workDir, "add", "README.md")
	runGit(t, workDir, "commit", "-m", content)
	runGit(t, workDir, "push", "origin", "main")
	return gitHead(t, workDir)
}

func gitHead(t *testing.T, dir string) string {
	t.Helper()
	return trimOutput(runGit(t, dir, "rev-parse", "HEAD"))
}

func gitRemoteURL(t *testing.T, dir string) string {
	t.Helper()
	return trimOutput(runGit(t, dir, "remote", "get-url", "origin"))
}

func assertPathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected path to exist: %s (%v)", path, err)
	}
}

func assertPathMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected path to be missing: %s", path)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed in %s: %v\n%s", args, dir, err, string(out))
	}
	return string(out)
}
