package source

import (
	"fmt"
	pathpkg "path"
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

type SourceOptions struct {
	ID   string
	Name string
}

func NewLocalSource(path string) (Source, error) {
	return NewLocalSourceWithOptions(path, SourceOptions{})
}

func NewLocalSourceWithOptions(path string, opts SourceOptions) (Source, error) {
	clean := fsutil.ToAbsPath(filepath.Clean(path))
	name := sourceNameFromPath(clean)
	if name == "" {
		return Source{}, fmt.Errorf("invalid source path: %s", path)
	}
	if opts.Name != "" {
		name = strings.TrimSpace(opts.Name)
	}
	id := NormalizeID(firstNonEmpty(opts.ID, name))
	if id == "" {
		return Source{}, fmt.Errorf("source id is required")
	}

	return Source{
		ID:   id,
		Type: TypeLocal,
		Name: name,
		Path: clean,
	}, nil
}

func NewGitSource(url, ref string) (Source, error) {
	return NewGitSourceWithOptions(url, ref, SourceOptions{})
}

func NewGitSourceWithOptions(url, ref string, opts SourceOptions) (Source, error) {
	trimmed := strings.TrimSpace(url)
	if trimmed == "" {
		return Source{}, fmt.Errorf("invalid git source url")
	}
	name := sourceNameFromGitURL(trimmed)
	if name == "" {
		return Source{}, fmt.Errorf("invalid git source url: %s", url)
	}
	if opts.Name != "" {
		name = strings.TrimSpace(opts.Name)
	}
	id := NormalizeID(firstNonEmpty(opts.ID, name))
	if id == "" {
		return Source{}, fmt.Errorf("source id is required")
	}

	return Source{
		ID:   id,
		Type: TypeGit,
		Name: name,
		URL:  trimmed,
		Ref:  strings.TrimSpace(ref),
	}, nil
}

func sourceNameFromPath(path string) string {
	name := filepath.Base(path)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return ""
	}
	if name == "skills" || name == "skill" {
		parent := filepath.Base(filepath.Dir(path))
		if parent != "." && parent != string(filepath.Separator) && parent != "" {
			name = parent + "-" + name
		}
	}
	return name
}

func sourceNameFromGitURL(url string) string {
	trimmed := strings.TrimSpace(url)
	if before, after, ok := strings.Cut(trimmed, ":"); ok && strings.Contains(before, "@") && !strings.Contains(before, "/") {
		trimmed = after
	}
	name := strings.TrimSuffix(pathpkg.Base(trimmed), ".git")
	if name == "." || name == "/" || name == "" {
		return ""
	}
	if name == "skills" || name == "skill" {
		parent := pathpkg.Base(pathpkg.Dir(trimmed))
		if parent != "." && parent != "/" && parent != "" {
			name = parent + "-" + name
		}
	}
	return name
}

func NormalizeID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '_' || r == '-':
			b.WriteRune(r)
			lastDash = r == '-'
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-_")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
