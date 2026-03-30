package sourcestore

import (
	cfg "github.com/inhere/skillc/internal/domain/config"
	domainsource "github.com/inhere/skillc/internal/domain/source"
)

func List(config cfg.Config) []domainsource.Source {
	if len(config.Sources) == 0 {
		return []domainsource.Source{}
	}
	return append([]domainsource.Source{}, config.Sources...)
}

func Add(config *cfg.Config, src domainsource.Source) {
	config.Sources = append(config.Sources, src)
}

func Remove(config *cfg.Config, id string) bool {
	for i, src := range config.Sources {
		if src.ID == id {
			config.Sources = append(config.Sources[:i], config.Sources[i+1:]...)
			return true
		}
	}
	return false
}

func ExistsByPath(config cfg.Config, path string) bool {
	for _, src := range config.Sources {
		if src.Path == path {
			return true
		}
	}
	return false
}
