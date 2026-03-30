package repoindex

import (
	"slices"
	"strings"

	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
)

type Query struct {
	Keyword    string
	Agent      string
	SourceType sourcepkg.Type
}

func Filter(items []skill.Skill, query Query) []skill.Skill {
	filtered := make([]skill.Skill, 0)
	for _, item := range items {
		if query.Keyword != "" {
			keyword := strings.ToLower(query.Keyword)
			if !strings.Contains(strings.ToLower(item.Name), keyword) && !strings.Contains(strings.ToLower(item.Description), keyword) {
				continue
			}
		}
		if query.Agent != "" && !contains(item.SupportedAgents, query.Agent) {
			continue
		}
		if query.SourceType != "" && item.SourceType != query.SourceType {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func FindByID(items []skill.Skill, id string) (skill.Skill, bool) {
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return skill.Skill{}, false
}

func contains(items []string, want string) bool {
	return slices.Contains(items, want)
}
