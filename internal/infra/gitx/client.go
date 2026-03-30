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

func (c *Client) Clone(url, dir, ref string) error {
	if _, err := exec.LookPath(c.bin); err != nil {
		return fmt.Errorf("git executable not found: %w", err)
	}

	args := []string{"clone", url, dir}
	if ref != "" {
		args = []string{"clone", "--branch", ref, url, dir}
	}
	cmd := exec.Command(c.bin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %s", string(out))
	}
	return nil
}
