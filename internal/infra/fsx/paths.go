package fsx

import (
	"os"
	"path/filepath"
	"strings"
)

func ExpandPath(path string, baseDir string) (string, error) {
	expanded := os.ExpandEnv(path)
	if strings.HasPrefix(expanded, "~/") || expanded == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if expanded == "~" {
			return filepath.Clean(home), nil
		}
		expanded = filepath.Join(home, expanded[2:])
	}

	if filepath.IsAbs(expanded) {
		return filepath.Clean(expanded), nil
	}

	if baseDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		baseDir = cwd
	}

	return filepath.Clean(filepath.Join(baseDir, expanded)), nil
}
