package gitx

import (
	"fmt"
	"os"
	"os/exec"
)

type Client struct {
	bin string
}

func New(bin string) *Client {
	if bin == "" {
		bin = "git"
	}
	return &Client{bin: bin}
}

func (c *Client) Sync(url, dir, ref, proxyURL string) (string, error) {
	if _, err := exec.LookPath(c.bin); err != nil {
		return "", fmt.Errorf("git executable not found: %w", err)
	}

	cmd := c.cloneCommand(url, dir, ref, proxyURL)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone failed: %s", string(out))
	}

	resolved, err := c.revParseHead(dir)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func (c *Client) cloneCommand(url, dir, ref, proxyURL string) *exec.Cmd {
	args := []string{"clone", url, dir}
	if ref != "" {
		args = []string{"clone", "--branch", ref, url, dir}
	}
	cmd := exec.Command(c.bin, args...)
	cmd.Env = buildGitEnv(os.Environ(), proxyURL)
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
