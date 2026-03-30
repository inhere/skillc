package source

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Type string

const (
	TypeLocal Type = "local"
	TypeGit   Type = "git"
)

type Source struct {
	ID           string
	Type         Type
	Name         string
	Path         string
	URL          string
	Ref          string
	Status       string
	ErrorMessage string
}

func NewLocalSource(path string) (Source, error) {
	clean := filepath.Clean(path)
	name := filepath.Base(clean)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return Source{}, fmt.Errorf("invalid source path: %s", path)
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
	if ref == "" {
		ref = "HEAD"
	}
	return Source{
		ID:   fmt.Sprintf("git-%s", name),
		Type: TypeGit,
		Name: name,
		URL:  trimmed,
		Ref:  ref,
	}, nil
}
