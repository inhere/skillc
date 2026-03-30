package repoindex

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/inhere/skillc/internal/domain/skill"
)

type Store struct{}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Save(path string, items []skill.Skill) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *Store) Load(path string) ([]skill.Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var items []skill.Skill
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}
