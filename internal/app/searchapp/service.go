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

func (s *Service) Show(id string) (skill.Skill, error) {
	items, err := s.store.Load(s.indexPath)
	if err != nil {
		return skill.Skill{}, err
	}
	item, ok := repoindex.FindByID(items, id)
	if !ok {
		return skill.Skill{}, fmt.Errorf("skill not found: %s", id)
	}
	return item, nil
}
