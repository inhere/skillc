package registrystore

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/inhere/skillc/internal/domain/registry"
)

type File struct {
	Entries []registry.Entry `json:"entries"`
}

type Store struct{}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Load(path string) ([]registry.Entry, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []registry.Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	if file.Entries == nil {
		return []registry.Entry{}, nil
	}
	return file.Entries, nil
}

func (s *Store) Save(path string, entries []registry.Entry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(File{Entries: entries}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
