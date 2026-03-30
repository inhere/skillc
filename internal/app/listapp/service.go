package listapp

import (
	"os"
	"sort"

	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/infra/lockstore"
)

type Item struct {
	SkillID       string
	Agent         string
	Scope         string
	Version       string
	SourceID      string
	SourceType    string
	InstalledPath string
	Checksum      string
	UpdatedAt     string
	Status        string
}

type Service struct {
	lockFile string
	store    *lockstore.Store
}

func NewService(lockFile string) *Service {
	return &Service{
		lockFile: lockFile,
		store:    lockstore.NewStore(),
	}
}

func (s *Service) List(agentName string, scope string) ([]Item, error) {
	records, err := s.store.Load(s.lockFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Item{}, nil
		}
		return nil, err
	}

	items := make([]Item, 0)
	for _, record := range records {
		if agentName != "" && record.Agent != agentName {
			continue
		}
		if scope != "" && record.Scope != scope {
			continue
		}
		items = append(items, toItem(record))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].SkillID < items[j].SkillID
	})
	return items, nil
}

func toItem(record lockpkg.Record) Item {
	status := "installed"
	if _, err := os.Stat(record.InstalledPath); err != nil {
		status = "missing"
	}
	return Item{
		SkillID:       record.SkillID,
		Agent:         record.Agent,
		Scope:         record.Scope,
		Version:       record.Version,
		SourceID:      record.SourceID,
		SourceType:    record.SourceType,
		InstalledPath: record.InstalledPath,
		Checksum:      record.Checksum,
		UpdatedAt:     record.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		Status:        status,
	}
}
