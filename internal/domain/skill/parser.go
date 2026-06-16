package skill

import (
	"fmt"
	"strings"

	gkconfig "github.com/gookit/config/v2"
	gkyaml "github.com/gookit/config/v2/yaml"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/hashx"
)

type frontMatter struct {
	ID              string   `yaml:"id"`
	Name            string   `yaml:"name"`
	Description     string   `yaml:"description"`
	Version         string   `yaml:"version"`
	SupportedAgents []string `yaml:"supported_agents"`
	InstallEntry    string   `yaml:"install_entry"`
}

func ParseSkillMarkdown(content string, src sourcepkg.Source) (Skill, error) {
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return Skill{}, fmt.Errorf("missing front matter")
	}

	loader := gkconfig.NewEmpty("skill", gkconfig.WithTagName("yaml"))
	loader.AddDriver(gkyaml.Driver)
	if err := loader.LoadStrings(gkconfig.Yaml, parts[1]); err != nil {
		return Skill{}, err
	}

	var meta frontMatter
	if err := loader.Decode(&meta); err != nil {
		return Skill{}, err
	}
	if meta.ID == "" {
		meta.ID = meta.Name
	}
	if meta.ID == "" {
		return Skill{}, fmt.Errorf("skill id is required")
	}
	if meta.Name == "" {
		meta.Name = meta.ID
	}
	if meta.Version == "" {
		meta.Version = fallbackVersion(content, src)
	}
	if meta.InstallEntry == "" {
		meta.InstallEntry = "."
	}

	return Skill{
		ID:                meta.ID,
		Name:              meta.Name,
		Description:       meta.Description,
		Version:           meta.Version,
		SupportedAgents:   meta.SupportedAgents,
		SourceID:          src.ID,
		SourceName:        src.Name,
		SourceType:        src.Type,
		InstallEntry:      meta.InstallEntry,
		Path:              src.Path,
		Checksum:          hashx.SumString(content),
		SourceResolvedRef: src.ResolvedRef,
	}, nil
}

func fallbackVersion(content string, src sourcepkg.Source) string {
	if src.Type == sourcepkg.TypeGit {
		if src.ResolvedRef != "" {
			return shortVersion(src.ResolvedRef)
		}
		if src.Ref != "" {
			return shortVersion(src.Ref)
		}
	}
	return shortVersion(hashx.SumString(content))
}

func shortVersion(value string) string {
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}
