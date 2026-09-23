package gitx

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type SyncOptions struct {
	ProxyURL string
	Progress io.Writer
	Quiet    bool
	Verbose  bool
}

type Client struct {
	bin string
}

func New(bin string) *Client {
	if bin == "" {
		bin = "git"
	}
	return &Client{bin: bin}
}

// ErrDirtyCache 表示 git 缓存目录存在未提交的本地改动；同步会 reset/clean 或重建缓存，必须先处理它们。
var ErrDirtyCache = errors.New("git cache has local changes")

func (c *Client) Sync(url, dir, ref string, opts SyncOptions) (string, error) {
	if _, err := exec.LookPath(c.bin); err != nil {
		return "", fmt.Errorf("git executable not found: %w", err)
	}

	if reusable, err := c.canReuseCache(url, dir); err != nil {
		return "", err
	} else if reusable {
		if err := c.ensureCleanCache(dir); err != nil {
			return "", err
		}
		resolved, err := c.syncExisting(dir, ref, opts)
		if err == nil {
			return resolved, nil
		}
		if err := os.RemoveAll(dir); err != nil {
			return "", err
		}
		return c.cloneAndResolve(url, dir, ref, opts)
	}

	if err := c.ensureRemovableCache(dir); err != nil {
		return "", err
	}
	if err := os.RemoveAll(dir); err != nil {
		return "", err
	}
	return c.cloneAndResolve(url, dir, ref, opts)
}

// ensureCleanCache 在 reset --hard / clean -fd 之前确认缓存目录没有本地改动。
// 缓存目录可能被 link 安装的项目目录直接编辑，因此不能无条件丢弃内容。
func (c *Client) ensureCleanCache(dir string) error {
	out, err := c.runQuiet(dir, "status", "--porcelain")
	if err != nil {
		return err
	}
	entries := strings.Split(strings.TrimSpace(out), "\n")
	if len(entries) == 1 && entries[0] == "" {
		return nil
	}
	return fmt.Errorf("%w: %s (%d changed entries: %s); commit or discard them before syncing",
		ErrDirtyCache, dir, len(entries), strings.Join(previewEntries(entries, 3), ", "))
}

// ensureRemovableCache 在删除缓存目录前确认它不是带有本地改动的 git 工作区。
func (c *Client) ensureRemovableCache(dir string) error {
	if !c.isWorkTree(dir) {
		return nil
	}
	return c.ensureCleanCache(dir)
}

func (c *Client) isWorkTree(dir string) bool {
	if _, err := os.Stat(dir); err != nil {
		return false
	}
	_, err := c.runQuiet(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

// previewEntries 只展示前 limit 条改动，避免错误信息过长。
func previewEntries(entries []string, limit int) []string {
	out := make([]string, 0, limit)
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		out = append(out, entry)
		if len(out) == limit {
			break
		}
	}
	return out
}

func (c *Client) canReuseCache(url, dir string) (bool, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.IsDir() {
		return false, nil
	}

	if _, err := c.runQuiet(dir, "rev-parse", "--is-inside-work-tree"); err != nil {
		return false, nil
	}

	originURL, err := c.runQuiet(dir, "remote", "get-url", "origin")
	if err != nil {
		return false, nil
	}
	return originURL == url, nil
}

func (c *Client) syncExisting(dir, ref string, opts SyncOptions) (string, error) {
	if err := c.runCommand(c.fetchCommand(dir, opts), "git fetch failed"); err != nil {
		return "", err
	}

	target, err := c.resolveTarget(dir, ref)
	if err != nil {
		return "", err
	}
	if _, err := c.runQuiet(dir, "reset", "--hard", target); err != nil {
		return "", fmt.Errorf("git reset failed: %s", err.Error())
	}
	if _, err := c.runQuiet(dir, "clean", "-fd"); err != nil {
		return "", fmt.Errorf("git clean failed: %s", err.Error())
	}

	resolved, err := c.revParseHead(dir)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func (c *Client) cloneAndResolve(url, dir, ref string, opts SyncOptions) (string, error) {
	cmd := c.cloneCommand(url, dir, ref, opts)
	if err := c.runCommand(cmd, "git clone failed"); err != nil {
		return "", err
	}

	resolved, err := c.revParseHead(dir)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func (c *Client) cloneCommand(url, dir, ref string, opts SyncOptions) *exec.Cmd {
	args := []string{"clone", "-c", "core.autocrlf=false"}
	if opts.Progress != nil {
		args = append(args, "--progress")
	}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, url, dir)

	cmd := exec.Command(c.bin, args...)
	if len(opts.ProxyURL) > 0 {
		cmd.Env = buildGitEnv(os.Environ(), opts.ProxyURL)
	}
	if opts.Progress != nil {
		cmd.Stdout = opts.Progress
		cmd.Stderr = opts.Progress
	}
	return cmd
}

func (c *Client) fetchCommand(dir string, opts SyncOptions) *exec.Cmd {
	args := []string{"-C", dir, "fetch", "--prune"}
	if opts.Progress != nil {
		args = append(args, "--progress")
	}
	args = append(args, "origin")

	cmd := exec.Command(c.bin, args...)
	if len(opts.ProxyURL) > 0 {
		cmd.Env = buildGitEnv(os.Environ(), opts.ProxyURL)
	}
	if opts.Progress != nil {
		cmd.Stdout = opts.Progress
		cmd.Stderr = opts.Progress
	}
	return cmd
}

func (c *Client) resolveTarget(dir, ref string) (string, error) {
	if ref != "" {
		for _, candidate := range []string{"refs/remotes/origin/" + ref, "refs/tags/" + ref, ref} {
			if _, err := c.runQuiet(dir, "rev-parse", "--verify", candidate+"^{commit}"); err == nil {
				return candidate, nil
			}
		}
		return "", fmt.Errorf("git resolve target failed: could not resolve %q", ref)
	}

	if target, err := c.runQuiet(dir, "symbolic-ref", "refs/remotes/origin/HEAD"); err == nil && target != "" {
		return target, nil
	}
	return "HEAD", nil
}

// Pull runs `git pull` in the given directory and returns the resolved HEAD.
func (c *Client) Pull(dir string, opts SyncOptions) (string, error) {
	if _, err := exec.LookPath(c.bin); err != nil {
		return "", fmt.Errorf("git executable not found: %w", err)
	}

	args := []string{"-C", dir, "pull"}
	if opts.Progress != nil {
		args = append(args, "--progress")
	}
	cmd := exec.Command(c.bin, args...)
	if len(opts.ProxyURL) > 0 {
		cmd.Env = buildGitEnv(os.Environ(), opts.ProxyURL)
	}
	if opts.Progress != nil {
		cmd.Stdout = opts.Progress
		cmd.Stderr = opts.Progress
	}
	if err := c.runCommand(cmd, "git pull failed"); err != nil {
		return "", err
	}
	return c.revParseHead(dir)
}

func (c *Client) revParseHead(dir string) (string, error) {
	out, err := c.runQuiet(dir, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %s", err.Error())
	}
	return out, nil
}

func (c *Client) revParseHeadCommand(dir string) *exec.Cmd {
	return exec.Command(c.bin, "-C", dir, "rev-parse", "HEAD")
}

func (c *Client) runQuiet(dir string, args ...string) (string, error) {
	cmdArgs := make([]string, 0, len(args)+2)
	if dir != "" {
		cmdArgs = append(cmdArgs, "-C", dir)
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command(c.bin, cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s", string(out))
	}
	return trimOutput(string(out)), nil
}

func (c *Client) runCommand(cmd *exec.Cmd, prefix string) error {
	if cmd.Stdout != nil || cmd.Stderr != nil {
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", prefix, err)
		}
		return nil
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", prefix, string(out))
	}
	return nil
}

func buildGitEnv(base []string, proxyURL string) []string {
	if proxyURL == "" {
		return base
	}
	env := append([]string{}, base...)
	env = append(env,
		"HTTP_PROXY="+proxyURL,
		"HTTPS_PROXY="+proxyURL,
		"http_proxy="+proxyURL,
		"https_proxy="+proxyURL,
	)
	return env
}

func trimOutput(value string) string {
	for len(value) > 0 && (value[len(value)-1] == '\n' || value[len(value)-1] == '\r' || value[len(value)-1] == ' ' || value[len(value)-1] == '\t') {
		value = value[:len(value)-1]
	}
	return value
}
