package registrystore

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/inhere/skillc/internal/domain/registry"
)

type File struct {
	Skills  []registry.SkillEntry `json:"skills,omitempty"`
	Sources []registry.Entry      `json:"sources,omitempty"`
	Entries []registry.Entry      `json:"entries,omitempty"`
}

type Store struct{}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Load(path string) ([]registry.Entry, error) {
	file, err := s.LoadFile(path)
	if err != nil {
		return nil, err
	}
	return file.Sources, nil
}

func (s *Store) Save(path string, entries []registry.Entry) error {
	return s.SaveFile(path, File{Sources: entries})
}

func (s *Store) LoadFile(path string) (File, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return File{Skills: []registry.SkillEntry{}, Sources: []registry.Entry{}}, nil
	}
	if err != nil {
		return File{}, err
	}
	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return File{}, err
	}
	if file.Sources == nil && file.Entries != nil {
		file.Sources = file.Entries
	}
	if file.Skills == nil {
		file.Skills = []registry.SkillEntry{}
	}
	if file.Sources == nil {
		file.Sources = []registry.Entry{}
	}
	file.Entries = nil
	return file, nil
}

func (s *Store) SaveFile(path string, file File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file.Entries = nil
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
