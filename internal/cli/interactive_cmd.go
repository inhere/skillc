package cli

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/inhere/skillc/internal/app/statusapp"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/skill"
	"github.com/inhere/skillc/internal/infra/termselect"
)

type multiSelector interface {
	SelectMulti(ctx context.Context, opts termselect.Options) ([]termselect.Item, error)
}

var newMultiSelector = func() multiSelector {
	return termselect.NewCliUISelector()
}

func skillSelectItems(skills []skill.Skill) []termselect.Item {
	items := make([]termselect.Item, 0, len(skills))
	for idx, item := range skills {
		items = append(items, termselect.Item{
			Key:    strconv.Itoa(idx + 1),
			Label:  skillLabel(item),
			Value:  skillTarget(item),
			Detail: skillDetail(item),
		})
	}
	return items
}

func selectedSkills(skills []skill.Skill, selected []termselect.Item) []skill.Skill {
	wanted := selectedValues(selected)
	out := make([]skill.Skill, 0, len(selected))
	for _, item := range skills {
		if wanted[skillTarget(item)] {
			out = append(out, item)
		}
	}
	return out
}

func updateSelectItems(items []statusapp.Item) []termselect.Item {
	out := make([]termselect.Item, 0, len(items))
	for _, item := range items {
		if item.Status != statusapp.StatusOutdated && item.Status != statusapp.StatusMissing {
			continue
		}
		out = append(out, termselect.Item{
			Key:    strconv.Itoa(len(out) + 1),
			Label:  updateLabel(item),
			Value:  statusTarget(item),
			Detail: updateDetail(item),
		})
	}
	return out
}

func selectedUpdateTargets(items []statusapp.Item, selected []termselect.Item) []string {
	wanted := selectedValues(selected)
	out := make([]string, 0, len(selected))
	for _, item := range items {
		target := statusTarget(item)
		if wanted[target] {
			out = append(out, target)
		}
	}
	return out
}

func agentSelectItems(config cfg.Config) []termselect.Item {
	names := make([]string, 0, len(config.AgentTools))
	for name := range config.AgentTools {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]termselect.Item, 0, len(names))
	for _, name := range names {
		tool := config.AgentTools[name]
		items = append(items, termselect.Item{
			Key:    strconv.Itoa(len(items) + 1),
			Label:  name,
			Value:  name,
			Detail: "dir=" + tool.Dirname,
		})
	}
	return items
}

func selectedAgentNames(selected []termselect.Item) []string {
	names := make([]string, 0, len(selected))
	for _, item := range selected {
		if item.Value != "" {
			names = append(names, item.Value)
		}
	}
	return names
}

func skillTarget(item skill.Skill) string {
	if item.SourceQualifiedName != "" {
		return item.SourceQualifiedName
	}
	return firstNonEmpty(item.QualifiedName, item.ID)
}

func statusTarget(item statusapp.Item) string {
	return firstNonEmpty(item.SourceQualifiedName, item.QualifiedName, item.SkillID)
}

func selectedValues(selected []termselect.Item) map[string]bool {
	out := make(map[string]bool, len(selected))
	for _, item := range selected {
		if item.Value != "" {
			out[item.Value] = true
		}
	}
	return out
}

func skillLabel(item skill.Skill) string {
	name := firstNonEmpty(item.Name, item.ID)
	if item.ID != "" && item.Name != "" && item.Name != item.ID {
		name = fmt.Sprintf("%s (%s)", item.Name, item.ID)
	}
	return name
}

func skillDetail(item skill.Skill) string {
	parts := []string{}
	if item.SourceID != "" {
		parts = append(parts, "source="+item.SourceID)
	}
	if item.Collection != "" {
		parts = append(parts, "collection="+item.Collection)
	}
	if item.Version != "" {
		parts = append(parts, "version="+item.Version)
	}
	return strings.Join(parts, " ")
}

func updateLabel(item statusapp.Item) string {
	name := firstNonEmpty(item.QualifiedName, item.SkillID)
	if item.Status != "" {
		name = fmt.Sprintf("%s %s", name, item.Status)
	}
	return name
}

func updateDetail(item statusapp.Item) string {
	parts := []string{}
	if item.SourceID != "" {
		parts = append(parts, "source="+item.SourceID)
	}
	if item.Agent != "" {
		parts = append(parts, "agent="+item.Agent)
	}
	if item.CurrentVersion != "" || item.LatestVersion != "" {
		parts = append(parts, fmt.Sprintf("version=%s->%s", item.CurrentVersion, item.LatestVersion))
	}
	return strings.Join(parts, " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
