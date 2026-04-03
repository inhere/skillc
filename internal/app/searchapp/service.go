package searchapp

import (
	"fmt"
	"os"
	"strings"

	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

type Service struct {
	indexPath string
	store     *repoindex.Store
}

type InstallTargetResolveResult struct {
	Resolved []skill.Skill
	Failed   []TargetError
}

type TargetError struct {
	Target string
	Reason string
}

func NewService(indexPath string) *Service {
	return &Service{
		indexPath: indexPath,
		store:     repoindex.NewStore(),
	}
}

func (s *Service) Search(keyword string, agent string, sourceType sourcepkg.Type) ([]skill.Skill, error) {
	items, err := s.store.Load(s.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []skill.Skill{}, nil
		}
		return nil, err
	}
	return repoindex.Filter(items, repoindex.Query{
		Keyword:    keyword,
		Agent:      agent,
		SourceType: sourceType,
	}), nil
}

func (s *Service) ListCollections() ([]repoindex.CollectionSummary, error) {
	items, err := s.store.Load(s.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []repoindex.CollectionSummary{}, nil
		}
		return nil, err
	}
	return repoindex.ListCollections(items), nil
}

func (s *Service) ListCollectionSkills(collection string) ([]skill.Skill, error) {
	items, err := s.store.Load(s.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("collection not found: %s", collection)
		}
		return nil, err
	}
	return repoindex.ListCollectionSkills(items, collection)
}

func (s *Service) Resolve(target string) ([]skill.Skill, error) {
	items, err := s.store.Load(s.indexPath)
	if err != nil {
		return nil, err
	}
	return repoindex.ResolveSkills(items, target)
}

func (s *Service) ResolveInstallTargets(targets []string, collectionMode bool) (InstallTargetResolveResult, error) {
	items, err := s.store.Load(s.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return InstallTargetResolveResult{}, nil
		}
		return InstallTargetResolveResult{}, err
	}

	result := InstallTargetResolveResult{
		Resolved: make([]skill.Skill, 0),
		Failed:   make([]TargetError, 0),
	}
	seen := make(map[string]struct{})
	for _, target := range targets {
		matches, err := resolveInstallTargetMatches(items, target, collectionMode)
		if err != nil {
			result.Failed = append(result.Failed, TargetError{Target: target, Reason: err.Error()})
			continue
		}
		for _, item := range matches {
			key := skillIdentityKey(item)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result.Resolved = append(result.Resolved, item)
		}
	}
	return result, nil
}

func (s *Service) Show(id string) (skill.Skill, error) {
	items, err := s.store.Load(s.indexPath)
	if err != nil {
		return skill.Skill{}, err
	}
	return repoindex.ResolveSkill(items, id)
}

func resolveInstallTargetMatches(items []skill.Skill, target string, collectionMode bool) ([]skill.Skill, error) {
	if collectionMode {
		return resolveCollectionTargets(items, target)
	}
	if prefix, ok := strings.CutSuffix(target, "*"); ok {
		return resolveSkillIDPrefix(items, prefix, target)
	}

	return resolveSingleSkillTarget(items, target)
}

func resolveSingleSkillTarget(items []skill.Skill, target string) ([]skill.Skill, error) {
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

	if strings.Contains(target, "/") {
		return nil, fmt.Errorf("skill not found: %s", target)
	}

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

	return nil, fmt.Errorf("skill not found: %s", target)
}

func resolveCollectionTargets(items []skill.Skill, target string) ([]skill.Skill, error) {
	matches := make([]skill.Skill, 0)
	if strings.Contains(target, "/") {
		prefix := target + "/"
		for _, item := range items {
			if strings.HasPrefix(item.SourceQualifiedName, prefix) {
				matches = append(matches, item)
			}
		}
	} else {
		for _, item := range items {
			if item.Collection == target || strings.HasPrefix(item.QualifiedName, target+"/") {
				matches = append(matches, item)
			}
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("collection not found: %s", target)
	}
	return matches, nil
}

func resolveSkillIDPrefix(items []skill.Skill, prefix string, target string) ([]skill.Skill, error) {
	if prefix == "" {
		return nil, fmt.Errorf("skill not found: %s", target)
	}

	matches := make([]skill.Skill, 0)
	for _, item := range items {
		if strings.HasPrefix(item.ID, prefix) {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("skill not found: %s", target)
	}
	return matches, nil
}

func skillIdentityKey(item skill.Skill) string {
	if item.SourceQualifiedName != "" {
		return item.SourceQualifiedName
	}
	if item.QualifiedName != "" {
		return item.QualifiedName
	}
	return item.ID
}
