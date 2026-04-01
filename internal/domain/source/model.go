package source

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gookit/goutil/fsutil"
)

type Type string

const (
	TypeLocal Type = "local"
	TypeGit   Type = "git"
)

type Source struct {
	ID           string `yaml:"id" `
	Type         Type   `yaml:"type"`
	Name         string `yaml:"name"`
	Path         string `yaml:"path,omitempty"`
	URL          string `yaml:"url,omitempty"`
	Ref          string `yaml:"ref,omitempty"`
	ResolvedRef  string `yaml:"resolved_ref,omitempty"`
	LastSyncAt   string `yaml:"last_sync_at,omitempty"`
	Status       string `yaml:"status,omitempty"`
	ErrorMessage string `yaml:"error_message,omitempty"`
}

func NewLocalSource(path string) (Source, error) {
	clean := fsutil.ToAbsPath(filepath.Clean(path))
	name := filepath.Base(clean)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return Source{}, fmt.Errorf("invalid source path: %s", path)
	}

	// note: 如果 name = skills or skill 则使用 parent dir name + name
	if name == "skills" || name == "skill" {
		name = filepath.Base(filepath.Dir(clean)) + "-" + name
	}

	return Source{
		ID:   fmt.Sprintf("local-%s", name),
		Type: TypeLocal,
		Name: name,
		Path: clean,
	}, nil
}

func NewGitSource(url, ref string) (Source, error) {
	trimmed := strings.TrimSpace(url)
	if trimmed == "" {
		return Source{}, fmt.Errorf("invalid git source url")
	}
	name := strings.TrimSuffix(filepath.Base(trimmed), ".git")
	if name == "." || name == "" || name == "/" {
		return Source{}, fmt.Errorf("invalid git source url: %s", url)
	}

	// note: 如果 name = skills or skill 则使用 parent dir name + name
	if name == "skills" || name == "skill" {
		name = filepath.Base(filepath.Dir(trimmed)) + "-" + name
	}

	return Source{
		ID:   fmt.Sprintf("git-%s", name),
		Type: TypeGit,
		Name: name,
		URL:  trimmed,
		Ref:  strings.TrimSpace(ref),
	}, nil
}
