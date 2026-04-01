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
	roots, err := s.skillRoots(src.Path)
	if err != nil {
		return nil, err
	}

	items := make([]skill.Skill, 0)
	seen := make(map[string]struct{})
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillDir := filepath.Join(root, entry.Name())
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
			if _, ok := seen[parsed.ID]; ok {
				continue
			}
			seen[parsed.ID] = struct{}{}
			parsed.Path = skillDir
			items = append(items, parsed)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items, nil
}

func (s *Scanner) skillRoots(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	roots := []string{root}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == "skills" {
			roots = append(roots, filepath.Join(root, entry.Name()))
		}
	}
	return roots, nil
}
