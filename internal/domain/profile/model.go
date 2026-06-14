package profile

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/inhere/skillc/internal/domain/skill"
)

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type Profile struct {
	Description  string   `yaml:"description,omitempty" json:"description,omitempty"`
	DefaultAgent string   `yaml:"default_agent,omitempty" json:"default_agent,omitempty"`
	DefaultScope string   `yaml:"default_scope,omitempty" json:"default_scope,omitempty"`
	InstallMode  string   `yaml:"install_mode,omitempty" json:"install_mode,omitempty"`
	Targets      []Target `yaml:"targets" json:"targets"`
}

type NamedProfile struct {
	Name string `json:"name"`
	Profile
}

type Target struct {
	Source string `yaml:"source,omitempty" json:"source,omitempty"`
	Skill  string `yaml:"skill" json:"skill"`
	Pinned bool   `yaml:"pinned,omitempty" json:"pinned,omitempty"`
}

type ApplyPlan struct {
	Profile string          `json:"profile"`
	Agent   string          `json:"agent"`
	Scope   string          `json:"scope"`
	Items   []ApplyPlanItem `json:"items"`
}

type ApplyPlanItem struct {
	Action string      `json:"action"`
	Target Target      `json:"target"`
	Skill  skill.Skill `json:"skill,omitempty"`
	Reason string      `json:"reason,omitempty"`
}

func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name is required")
	}
	if strings.TrimSpace(name) != name {
		return fmt.Errorf("invalid profile name: %s", name)
	}
	if !namePattern.MatchString(name) {
		return fmt.Errorf("invalid profile name: %s", name)
	}
	return nil
}

func ValidateTarget(target Target) error {
	if strings.TrimSpace(target.Skill) == "" {
		return fmt.Errorf("profile target skill is required")
	}
	return nil
}

func NormalizeTargets(targets []Target) ([]Target, error) {
	seen := make(map[string]int, len(targets))
	out := make([]Target, 0, len(targets))
	for _, target := range targets {
		target.Source = strings.TrimSpace(target.Source)
		target.Skill = strings.TrimSpace(target.Skill)
		if err := ValidateTarget(target); err != nil {
			return nil, err
		}
		key := target.Source + "\x00" + target.Skill
		if idx, ok := seen[key]; ok {
			out[idx].Pinned = out[idx].Pinned || target.Pinned
			continue
		}
		seen[key] = len(out)
		out = append(out, target)
	}
	slices.SortFunc(out, func(a, b Target) int {
		if a.Source != b.Source {
			if a.Source < b.Source {
				return -1
			}
			return 1
		}
		if a.Skill < b.Skill {
			return -1
		}
		if a.Skill > b.Skill {
			return 1
		}
		return 0
	})
	return out, nil
}
