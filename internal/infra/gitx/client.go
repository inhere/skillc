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

	if reusable, err := c.canReuseCache(url, dir); err != nil {
		return "", err
	} else if reusable {
		resolved, err := c.syncExisting(dir, ref, opts)
		if err == nil {
			return resolved, nil
		}
		if err := os.RemoveAll(dir); err != nil {
			return "", err
		}
		return c.cloneAndResolve(url, dir, ref, opts)
	}

	if err := os.RemoveAll(dir); err != nil {
		return "", err
	}
	return c.cloneAndResolve(url, dir, ref, opts)
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
	args := []string{"-C", dir, "fetch", "--prune"}
	if opts.Progress != nil {
		args = append(args, "--progress")
	}
	args = append(args, "origin")

	cmd := exec.Command(c.bin, args...)
	cmd.Env = buildGitEnv(os.Environ(), opts.ProxyURL)
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
