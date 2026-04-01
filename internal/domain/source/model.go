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
	ID           string `yaml:"id" mapstructure:"id"`
	Type         Type   `yaml:"type" mapstructure:"type"`
	Name         string `yaml:"name" mapstructure:"name"`
	Path         string `yaml:"path" mapstructure:"path"`
	URL          string `yaml:"url" mapstructure:"url"`
	Ref          string `yaml:"ref" mapstructure:"ref"`
	ResolvedRef  string `yaml:"resolved_ref" mapstructure:"resolved_ref"`
	LastSyncAt   string `yaml:"last_sync_at" mapstructure:"last_sync_at"`
	Status       string `yaml:"status" mapstructure:"status"`
	ErrorMessage string `yaml:"error_message" mapstructure:"error_message"`
}

func NewLocalSource(path string) (Source, error) {
	clean := filepath.Clean(path)
	name := filepath.Base(clean)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return Source{}, fmt.Errorf("invalid source path: %s", path)
	}

	// 特殊处理 如果name=skills 或 skill, 取出父级目录名拼接为name
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
	return Source{
		ID:   fmt.Sprintf("git-%s", name),
		Type: TypeGit,
		Name: name,
		URL:  trimmed,
		Ref:  strings.TrimSpace(ref),
	}, nil
}
