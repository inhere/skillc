package repoindex

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
)

type Scanner struct{}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Scan(src sourcepkg.Source) ([]skill.Skill, error) {
	entries, err := os.ReadDir(src.Path)
	if err != nil {
		return nil, err
	}

	items := make([]skill.Skill, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillDir := filepath.Join(src.Path, entry.Name())
		mdPath := filepath.Join(skillDir, "SKILL.md")
		content, err := os.ReadFile(mdPath)
		if err != nil {
			continue
		}

		candidate := src
		candidate.Path = skillDir
		parsed, err := skill.ParseSkillMarkdown(string(content), candidate)
		if err != nil {
			continue
		}
		parsed.Path = skillDir
		items = append(items, parsed)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items, nil
}
