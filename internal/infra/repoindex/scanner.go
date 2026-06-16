package repoindex

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/hashx"
)

type Scanner struct{}

type scanGroup struct {
	Collection string
	RootDir    string
}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Scan(src sourcepkg.Source) ([]skill.Skill, error) {
	groups, err := s.scanGroups(src)
	if err != nil {
		return nil, err
	}

	items := make([]skill.Skill, 0)
	seen := make(map[string]struct{})
	for _, group := range groups {
		entries, err := os.ReadDir(group.RootDir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillDir := filepath.Join(group.RootDir, entry.Name())
			mdPath := filepath.Join(skillDir, "SKILL.md")
			content, err := os.ReadFile(mdPath)
			if err != nil {
				continue
			}
			candidate := src
			candidate.Path = skillDir
			parsed, err := skill.ParseSkillMarkdown(string(content), candidate)
			if err != nil {
				continue
			}
			parsed.Path = skillDir
			if checksum, err := hashx.SumDir(filepath.Join(skillDir, parsed.InstallEntry)); err == nil {
				parsed.Checksum = checksum
			}
			parsed.Collection = group.Collection
			parsed.SourceName = src.Name
			parsed.QualifiedName = qualifiedName(group.Collection, parsed.ID)
			parsed.SourceQualifiedName = sourceQualifiedName(src.Name, group.Collection, parsed.ID)
			if _, ok := seen[parsed.SourceQualifiedName]; ok {
				continue
			}
			seen[parsed.SourceQualifiedName] = struct{}{}
			items = append(items, parsed)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].QualifiedName < items[j].QualifiedName
	})
	return items, nil
}

func (s *Scanner) scanGroups(src sourcepkg.Source) ([]scanGroup, error) {
	skillsDirs := make([]string, 0)
	err := filepath.WalkDir(src.Path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == "skills" && path != src.Path {
			if has, err := hasSkillChildren(path); err != nil {
				return err
			} else if has {
				skillsDirs = append(skillsDirs, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(skillsDirs)
	if len(skillsDirs) == 1 {
		return []scanGroup{{Collection: src.Name, RootDir: skillsDirs[0]}}, nil
	}
	if len(skillsDirs) > 1 {
		groups := make([]scanGroup, 0, len(skillsDirs))
		for _, dir := range skillsDirs {
			groups = append(groups, scanGroup{
				Collection: filepath.Base(filepath.Dir(dir)),
				RootDir:    dir,
			})
		}
		return groups, nil
	}

	rootSkillCount, err := countSkillChildren(src.Path)
	if err != nil {
		return nil, err
	}
	collection := ""
	if rootSkillCount > 1 {
		collection = src.Name
	}
	return []scanGroup{{Collection: collection, RootDir: src.Path}}, nil
}

func countSkillChildren(root string) (int, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, entry.Name(), "SKILL.md")); err == nil {
			count++
		}
	}
	return count, nil
}

func hasSkillChildren(root string) (bool, error) {
	count, err := countSkillChildren(root)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func qualifiedName(collection, id string) string {
	if collection == "" {
		return id
	}
	return collection + "/" + id
}

func sourceQualifiedName(sourceName, collection, id string) string {
	if collection == "" || collection == sourceName {
		return sourceName + "/" + id
	}
	return sourceName + "/" + collection + "/" + id
}
