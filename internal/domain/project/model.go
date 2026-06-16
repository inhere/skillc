package project

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

type Project struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name,omitempty" json:"name,omitempty"`
	Path        string `yaml:"path" json:"path"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

func New(id string, name string, path string) (Project, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Project{}, fmt.Errorf("project path is required")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Project{}, err
	}
	cleanPath := filepath.Clean(absPath)
	baseName := filepath.Base(cleanPath)
	if name == "" {
		name = baseName
	}
	if id == "" {
		id = NormalizeID(baseName)
	} else {
		id = NormalizeID(id)
	}
	if id == "" {
		return Project{}, fmt.Errorf("project id is required")
	}
	return Project{ID: id, Name: name, Path: cleanPath}, nil
}

func NormalizeID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
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
