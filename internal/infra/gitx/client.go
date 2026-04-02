package gitx

import (
	"fmt"
	"io"
	"os"
	"os/exec"
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

func (c *Client) Sync(url, dir, ref string, opts SyncOptions) (string, error) {
	if _, err := exec.LookPath(c.bin); err != nil {
		return "", fmt.Errorf("git executable not found: %w", err)
	}

	reuse, err := c.canReuseCache(url, dir)
	if err != nil {
		return "", err
	}
	if !reuse {
		if err := c.replaceWithClone(url, dir, ref, opts); err != nil {
			return "", err
		}
	} else {
		if err := c.incrementalSync(dir, ref, opts); err != nil {
			if cloneErr := c.replaceWithClone(url, dir, ref, opts); cloneErr != nil {
				return "", err
			}
		}
	}

	resolved, err := c.revParseHead(dir)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func (c *Client) canReuseCache(url, dir string) (bool, error) {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat cache dir failed: %w", err)
	}
	if !c.isGitRepo(dir) {
		return false, nil
	}
	originURL, err := c.remoteGetURL(dir)
	if err != nil {
		return false, nil
	}
	return originURL == url, nil
}

func (c *Client) isGitRepo(dir string) bool {
	cmd := exec.Command(c.bin, "-C", dir, "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func (c *Client) remoteGetURL(dir string) (string, error) {
	cmd := exec.Command(c.bin, "-C", dir, "remote", "get-url", "origin")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git remote get-url failed: %s", string(out))
	}
	return trimOutput(string(out)), nil
}

func (c *Client) incrementalSync(dir, ref string, opts SyncOptions) error {
	if err := c.runGitCommand(c.fetchCommand(dir, opts)); err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}
	target, err := c.resolveSyncTarget(dir, ref)
	if err != nil {
		return err
	}
	if out, err := exec.Command(c.bin, "-C", dir, "reset", "--hard", target).CombinedOutput(); err != nil {
		return fmt.Errorf("git reset --hard failed: %s", string(out))
	}
	if out, err := exec.Command(c.bin, "-C", dir, "clean", "-fd").CombinedOutput(); err != nil {
		return fmt.Errorf("git clean -fd failed: %s", string(out))
	}
	return nil
}

func (c *Client) resolveSyncTarget(dir, ref string) (string, error) {
	if ref == "" {
		return c.revParseTarget(dir, "origin/HEAD")
	}

	targets := []string{"origin/" + ref, "refs/tags/" + ref}
	for _, target := range targets {
		resolved, err := c.revParseTarget(dir, target)
		if err == nil {
			return resolved, nil
		}
	}
	return "", fmt.Errorf("git rev-parse target failed: ref %q not found in origin branches or tags", ref)
}

func (c *Client) revParseTarget(dir, target string) (string, error) {
	out, err := exec.Command(c.bin, "-C", dir, "rev-parse", target).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse target failed: %s", string(out))
	}
	return trimOutput(string(out)), nil
}

func (c *Client) replaceWithClone(url, dir, ref string, opts SyncOptions) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove cache dir failed: %w", err)
	}
	if err := c.runClone(url, dir, ref, opts); err != nil {
		return err
	}
	return nil
}

func (c *Client) runClone(url, dir, ref string, opts SyncOptions) error {
	cmd := c.cloneCommand(url, dir, ref, opts)
	if err := c.runGitCommand(cmd); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}
	return nil
}

func (c *Client) runGitCommand(cmd *exec.Cmd) error {
	if cmd.Stdout != nil || cmd.Stderr != nil {
		if err := cmd.Run(); err != nil {
			return err
		}
		return nil
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(out))
	}
	return nil
}

func (c *Client) cloneCommand(url, dir, ref string, opts SyncOptions) *exec.Cmd {
	args := []string{"clone"}
	if opts.Progress != nil {
		args = append(args, "--progress")
	}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, url, dir)

	cmd := exec.Command(c.bin, args...)
	cmd.Env = buildGitEnv(os.Environ(), opts.ProxyURL)
	if opts.Progress != nil {
		cmd.Stdout = opts.Progress
		cmd.Stderr = opts.Progress
	}
	return cmd
}

func (c *Client) fetchCommand(dir string, opts SyncOptions) *exec.Cmd {
	cmd := exec.Command(c.bin, "-C", dir, "fetch", "--prune", "origin")
	cmd.Env = buildGitEnv(os.Environ(), opts.ProxyURL)
	if opts.Progress != nil {
		cmd.Stdout = opts.Progress
		cmd.Stderr = opts.Progress
	}
	return cmd
}

func (c *Client) revParseHead(dir string) (string, error) {
	cmd := c.revParseHeadCommand(dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %s", string(out))
	}
	return trimOutput(string(out)), nil
}

func (c *Client) revParseHeadCommand(dir string) *exec.Cmd {
	return exec.Command(c.bin, "-C", dir, "rev-parse", "HEAD")
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
