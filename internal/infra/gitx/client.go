package gitx

import (
	"fmt"
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

func (c *Client) Sync(url, dir, ref string) (string, error) {
	if _, err := exec.LookPath(c.bin); err != nil {
		return "", fmt.Errorf("git executable not found: %w", err)
	}

	args := []string{"clone", url, dir}
	if ref != "" {
		args = []string{"clone", "--branch", ref, url, dir}
	}
	cmd := exec.Command(c.bin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone failed: %s", string(out))
	}

	resolved, err := c.revParseHead(dir)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func (c *Client) revParseHead(dir string) (string, error) {
	cmd := exec.Command(c.bin, "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %s", string(out))
	}
	return trimOutput(string(out)), nil
}

func trimOutput(value string) string {
	for len(value) > 0 && (value[len(value)-1] == '\n' || value[len(value)-1] == '\r' || value[len(value)-1] == ' ' || value[len(value)-1] == '\t') {
		value = value[:len(value)-1]
	}
	return value
}
