package skill

import (
	"fmt"
	"strings"

	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/hashx"
	gkconfig "github.com/gookit/config/v2"
	gkyaml "github.com/gookit/config/v2/yaml"
)

type frontMatter struct {
	ID              string   `mapstructure:"id"`
	Name            string   `mapstructure:"name"`
	Description     string   `mapstructure:"description"`
	Version         string   `mapstructure:"version"`
	SupportedAgents []string `mapstructure:"supported_agents"`
	InstallEntry    string   `mapstructure:"install_entry"`
}

func ParseSkillMarkdown(content string, src sourcepkg.Source) (Skill, error) {
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return Skill{}, fmt.Errorf("missing front matter")
	}

	loader := gkconfig.NewEmpty("skill")
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
		ID:              meta.ID,
		Name:            meta.Name,
		Description:     meta.Description,
		Version:         meta.Version,
		SupportedAgents: meta.SupportedAgents,
		SourceID:        src.ID,
		SourceName:      src.Name,
		SourceType:      src.Type,
		InstallEntry:    meta.InstallEntry,
		Path:            src.Path,
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
