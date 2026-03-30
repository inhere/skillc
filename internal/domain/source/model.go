package source

import (
	"fmt"
	"path/filepath"
)

type Type string

const (
	TypeLocal Type = "local"
	TypeGit   Type = "git"
)

type Source struct {
	ID   string
	Type Type
	Name string
	Path string
	URL  string
	Ref  string
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
