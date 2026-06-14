package repoindex

import (
	"fmt"
	"slices"

	"github.com/inhere/skillc/internal/domain/skill"
)

type CollectionSummary struct {
	Name        string
	SkillCount  int
	SourceCount int
}

type SourceCollectionSummary struct {
	SourceID   string
	SourceName string
	Name       string
	SkillCount int
}

func ListCollections(items []skill.Skill) []CollectionSummary {
	groups := make(map[string]*CollectionSummary)
	sources := make(map[string]map[string]struct{})
	for _, item := range items {
		if item.Collection == "" {
			continue
		}
		summary := groups[item.Collection]
		if summary == nil {
			summary = &CollectionSummary{Name: item.Collection}
			groups[item.Collection] = summary
			sources[item.Collection] = make(map[string]struct{})
		}
		summary.SkillCount++
		key := item.SourceName
		if key == "" {
			key = item.SourceID
		}
		if _, ok := sources[item.Collection][key]; !ok {
			sources[item.Collection][key] = struct{}{}
			summary.SourceCount++
		}
	}

	result := make([]CollectionSummary, 0, len(groups))
	for _, summary := range groups {
		result = append(result, *summary)
	}
	slices.SortFunc(result, func(a, b CollectionSummary) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return result
}

func ListSourceCollections(items []skill.Skill, sourceID string) []SourceCollectionSummary {
	groups := make(map[string]*SourceCollectionSummary)
	for _, item := range items {
		if item.Collection == "" {
			continue
		}
		if sourceID != "" && item.SourceID != sourceID && item.SourceName != sourceID {
			continue
		}
		key := item.SourceID + "\x00" + item.Collection
		summary := groups[key]
		if summary == nil {
			summary = &SourceCollectionSummary{
				SourceID:   item.SourceID,
				SourceName: item.SourceName,
				Name:       item.Collection,
			}
			groups[key] = summary
		}
		summary.SkillCount++
	}

	result := make([]SourceCollectionSummary, 0, len(groups))
	for _, summary := range groups {
		result = append(result, *summary)
	}
	slices.SortFunc(result, func(a, b SourceCollectionSummary) int {
		if a.SourceID < b.SourceID {
			return -1
		}
		if a.SourceID > b.SourceID {
			return 1
		}
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return result
}

func ListCollectionSkills(items []skill.Skill, collection string) ([]skill.Skill, error) {
	matched := make([]skill.Skill, 0)
	for _, item := range items {
		if item.Collection == collection {
			matched = append(matched, item)
		}
	}
	if len(matched) == 0 {
		return nil, fmt.Errorf("collection not found: %s", collection)
	}
	slices.SortFunc(matched, func(a, b skill.Skill) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return matched, nil
}

func ListSourceSkills(items []skill.Skill, sourceID string, collection string) ([]skill.Skill, error) {
	matched := make([]skill.Skill, 0)
	for _, item := range items {
		if sourceID != "" && item.SourceID != sourceID && item.SourceName != sourceID {
			continue
		}
		if collection != "" && item.Collection != collection {
			continue
		}
		matched = append(matched, item)
	}
	if len(matched) == 0 {
		return nil, fmt.Errorf("source skills not found")
	}
	slices.SortFunc(matched, func(a, b skill.Skill) int {
		if a.Collection < b.Collection {
			return -1
		}
		if a.Collection > b.Collection {
			return 1
		}
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return matched, nil
}
