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
	repo := newGitRemoteFixture(t)
	want := repo.commitFile(t, "README.md", "first\n", "initial commit")
	cacheDir := filepath.Join(t.TempDir(), "cache")

	got, err := New("git").Sync(repo.remoteDir, cacheDir, "main", SyncOptions{})
	assert.NoErr(t, err)
	assert.Eq(t, want, got)
	assertFileContent(t, filepath.Join(cacheDir, "README.md"), "first\n")
	assertExists(t, filepath.Join(cacheDir, ".git"))
}

func TestClient_SyncReusesValidCacheForIncrementalSync(t *testing.T) {
	repo := newGitRemoteFixture(t)
	_ = repo.commitFile(t, "README.md", "first\n", "initial commit")
	cacheDir := filepath.Join(t.TempDir(), "cache")
	client := New("git")

	_, err := client.Sync(repo.remoteDir, cacheDir, "main", SyncOptions{})
	assert.NoErr(t, err)
	runGit(t, cacheDir, "config", "--local", "skillc.marker", "keep")

	want := repo.commitFile(t, "README.md", "second\n", "update commit")
	got, err := client.Sync(repo.remoteDir, cacheDir, "main", SyncOptions{})
	assert.NoErr(t, err)
	assert.Eq(t, want, got)
	assertFileContent(t, filepath.Join(cacheDir, "README.md"), "second\n")

	marker, ok := gitConfigValue(cacheDir, "skillc.marker")
	if !ok || marker != "keep" {
		t.Fatalf("expected reusable cache marker to persist, got %q ok=%v", marker, ok)
	}
}

func TestClient_SyncFallsBackToCloneWhenCacheIsNotGitRepo(t *testing.T) {
	repo := newGitRemoteFixture(t)
	want := repo.commitFile(t, "README.md", "first\n", "initial commit")
	cacheDir := filepath.Join(t.TempDir(), "cache")
	assert.NoErr(t, os.MkdirAll(cacheDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(cacheDir, "junk.txt"), []byte("junk"), 0o644))

	got, err := New("git").Sync(repo.remoteDir, cacheDir, "main", SyncOptions{})
	assert.NoErr(t, err)
	assert.Eq(t, want, got)
	assertFileContent(t, filepath.Join(cacheDir, "README.md"), "first\n")
	assertNotExists(t, filepath.Join(cacheDir, "junk.txt"))
}

func TestClient_SyncFallsBackToCloneWhenCacheOriginMismatches(t *testing.T) {
	firstRepo := newGitRemoteFixture(t)
	_ = firstRepo.commitFile(t, "README.md", "first\n", "initial commit")
	secondRepo := newGitRemoteFixture(t)
	want := secondRepo.commitFile(t, "README.md", "second\n", "initial commit")
	cacheDir := filepath.Join(t.TempDir(), "cache")
	client := New("git")

	_, err := client.Sync(firstRepo.remoteDir, cacheDir, "main", SyncOptions{})
	assert.NoErr(t, err)
	runGit(t, cacheDir, "config", "--local", "skillc.marker", "stale")

	got, err := client.Sync(secondRepo.remoteDir, cacheDir, "main", SyncOptions{})
	assert.NoErr(t, err)
	assert.Eq(t, want, got)
	assertFileContent(t, filepath.Join(cacheDir, "README.md"), "second\n")

	if marker, ok := gitConfigValue(cacheDir, "skillc.marker"); ok {
		t.Fatalf("expected fallback clone to replace repo and drop marker, got %q", marker)
	}
}

func TestClient_SyncRemovesStaleUntrackedFilesDuringIncrementalSync(t *testing.T) {
	repo := newGitRemoteFixture(t)
	_ = repo.commitFile(t, "README.md", "first\n", "initial commit")
	cacheDir := filepath.Join(t.TempDir(), "cache")
	client := New("git")

	_, err := client.Sync(repo.remoteDir, cacheDir, "main", SyncOptions{})
	assert.NoErr(t, err)
	staleFile := filepath.Join(cacheDir, "stale.txt")
	assert.NoErr(t, os.WriteFile(staleFile, []byte("stale"), 0o644))

	_, err = client.Sync(repo.remoteDir, cacheDir, "main", SyncOptions{})
	assert.NoErr(t, err)
	assertNotExists(t, staleFile)
}

func TestClient_SyncReturnsResolvedRefFromSynchronizedHead(t *testing.T) {
	repo := newGitRemoteFixture(t)
	first := repo.commitFile(t, "README.md", "first\n", "initial commit")
	cacheDir := filepath.Join(t.TempDir(), "cache")
	client := New("git")

	got, err := client.Sync(repo.remoteDir, cacheDir, "main", SyncOptions{})
	assert.NoErr(t, err)
	assert.Eq(t, first, got)

	want := repo.commitFile(t, "README.md", "second\n", "update commit")
	got, err = client.Sync(repo.remoteDir, cacheDir, "main", SyncOptions{})
	assert.NoErr(t, err)
	assert.Eq(t, want, got)
	assert.Eq(t, want, runGit(t, cacheDir, "rev-parse", "HEAD"))
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

type gitRemoteFixture struct {
	remoteDir string
	workDir   string
}

func newGitRemoteFixture(t *testing.T) gitRemoteFixture {
	t.Helper()
	requireGit(t)

	baseDir := t.TempDir()
	remoteDir := filepath.Join(baseDir, "remote.git")
	workDir := filepath.Join(baseDir, "work")

	runGit(t, "", "init", "--bare", remoteDir)
	runGit(t, "", "init", workDir)
	runGit(t, workDir, "config", "user.name", "Test User")
	runGit(t, workDir, "config", "user.email", "test@example.com")
	runGit(t, workDir, "checkout", "-b", "main")
	runGit(t, workDir, "remote", "add", "origin", remoteDir)

	return gitRemoteFixture{remoteDir: remoteDir, workDir: workDir}
}

func (r gitRemoteFixture) commitFile(t *testing.T, name, content, message string) string {
	t.Helper()

	path := filepath.Join(r.workDir, name)
	assert.NoErr(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NoErr(t, os.WriteFile(path, []byte(content), 0o644))

	runGit(t, r.workDir, "add", name)
	runGit(t, r.workDir, "commit", "-m", message)
	runGit(t, r.workDir, "push", "-u", "origin", "main")
	return runGit(t, r.workDir, "rev-parse", "HEAD")
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
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
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
	return trimOutput(string(out))
}

func gitConfigValue(dir, key string) (string, bool) {
	cmd := exec.Command("git", "-C", dir, "config", "--local", "--get", key)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", false
	}
	return trimOutput(string(out)), true
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	assert.NoErr(t, err)
	assert.Eq(t, want, string(data))
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be absent, err=%v", path, err)
	}
}
