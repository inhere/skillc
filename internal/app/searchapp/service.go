package searchapp

import (
	"fmt"
	"os"

	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/repoindex"
)

type Service struct {
	indexPath string
	store     *repoindex.Store
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

func (s *Service) Show(id string) (skill.Skill, error) {
	items, err := s.store.Load(s.indexPath)
	if err != nil {
		return skill.Skill{}, err
	}
	return repoindex.ResolveSkill(items, id)
}
