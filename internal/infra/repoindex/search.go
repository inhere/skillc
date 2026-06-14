package repoindex

import (
	"fmt"
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
	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))
	for _, item := range items {
		if keyword != "" && !matchesKeyword(item, keyword) {
			continue
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

func matchesKeyword(item skill.Skill, keyword string) bool {
	fields := []string{
		item.ID,
		item.Name,
		item.Description,
		item.Collection,
		item.QualifiedName,
		item.SourceQualifiedName,
		item.SourceID,
		item.SourceName,
		string(item.SourceType),
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), keyword) {
			return true
		}
	}
	return false
}

func ResolveSkills(items []skill.Skill, target string) ([]skill.Skill, error) {
	exact := make([]skill.Skill, 0)
	for _, item := range items {
		if item.SourceQualifiedName == target || item.QualifiedName == target {
			exact = append(exact, item)
		}
	}
	if len(exact) > 1 {
		return nil, fmt.Errorf("ambiguous skill target: %s; use source/collection/skill", target)
	}
	if len(exact) == 1 {
		return exact, nil
	}

	if !strings.Contains(target, "/") {
		exactID := make([]skill.Skill, 0)
		for _, item := range items {
			if item.ID == target {
				exactID = append(exactID, item)
			}
		}
		if len(exactID) > 1 {
			return nil, fmt.Errorf("ambiguous skill target: %s; use source/collection/skill", target)
		}
		if len(exactID) == 1 {
			return exactID, nil
		}

		tailMatches := make([]skill.Skill, 0)
		for _, item := range items {
			if idx := strings.LastIndex(item.QualifiedName, "/"); idx >= 0 && idx < len(item.QualifiedName)-1 && item.QualifiedName[idx+1:] == target {
				tailMatches = append(tailMatches, item)
			}
		}
		if len(tailMatches) > 1 {
			return nil, fmt.Errorf("ambiguous skill target: %s; use source/collection/skill", target)
		}
		if len(tailMatches) == 1 {
			return tailMatches, nil
		}
	}

	collectionMatches := make([]skill.Skill, 0)
	if strings.Contains(target, "/") {
		prefix := target + "/"
		for _, item := range items {
			if strings.HasPrefix(item.SourceQualifiedName, prefix) {
				collectionMatches = append(collectionMatches, item)
			}
		}
	} else {
		for _, item := range items {
			if item.Collection == target || strings.HasPrefix(item.QualifiedName, target+"/") {
				collectionMatches = append(collectionMatches, item)
			}
		}
		if len(collectionMatches) > 0 && hasMultipleSources(collectionMatches) {
			return nil, fmt.Errorf("ambiguous collection target: %s; use source/collection", target)
		}
	}
	if len(collectionMatches) == 0 {
		return nil, fmt.Errorf("skill not found: %s", target)
	}
	return collectionMatches, nil
}

func ResolveSkill(items []skill.Skill, target string) (skill.Skill, error) {
	matches, err := ResolveSkills(items, target)
	if err != nil {
		return skill.Skill{}, err
	}
	if len(matches) != 1 {
		return skill.Skill{}, fmt.Errorf("target resolves multiple skills: %s; use a specific skill name", target)
	}
	return matches[0], nil
}

func FindByID(items []skill.Skill, id string) (skill.Skill, bool) {
	item, err := ResolveSkill(items, id)
	if err != nil {
		return skill.Skill{}, false
	}
	return item, true
}

func hasMultipleSources(items []skill.Skill) bool {
	sources := make(map[string]struct{})
	for _, item := range items {
		key := item.SourceName
		if key == "" && item.SourceQualifiedName != "" {
			key = strings.SplitN(item.SourceQualifiedName, "/", 2)[0]
		}
		if key == "" {
			key = item.SourceID
		}
		sources[key] = struct{}{}
		if len(sources) > 1 {
			return true
		}
	}
	return false
}

func contains(items []string, want string) bool {
	return slices.Contains(items, want)
}
