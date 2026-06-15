package webapp

import (
	"sort"
	"strconv"
	"strings"

	"github.com/inhere/skillc/internal/app/apputil"
	lockpkg "github.com/inhere/skillc/internal/domain/lock"
	"github.com/inhere/skillc/internal/domain/skill"
)

type ProjectInstall struct {
	ProjectPath         string `json:"project_path"`
	Scope               string `json:"scope"`
	Agent               string `json:"agent"`
	Profile             string `json:"profile,omitempty"`
	SkillID             string `json:"skill_id"`
	QualifiedName       string `json:"qualified_name,omitempty"`
	SourceQualifiedName string `json:"source_qualified_name,omitempty"`
	SourceID            string `json:"source_id,omitempty"`
	Version             string `json:"version,omitempty"`
}

type VersionDriftGroup struct {
	SkillID             string          `json:"skill_id"`
	SourceQualifiedName string          `json:"source_qualified_name,omitempty"`
	SourceID            string          `json:"source_id,omitempty"`
	LatestVersion       string          `json:"latest_version,omitempty"`
	Versions            []VersionBucket `json:"versions"`
}

type VersionBucket struct {
	Version  string           `json:"version"`
	Projects []ProjectInstall `json:"projects"`
}

func BuildProjectInstallIndex(records lockpkg.File) []ProjectInstall {
	items := make([]ProjectInstall, 0)
	for scopeKey, recordList := range records {
		scope := string(apputil.ScopeFromKey(scopeKey))
		if scopeKey == lockpkg.GlobalKey {
			scope = "global"
		}

		for _, record := range recordList {
			for _, agentName := range record.Agents {
				items = append(items, ProjectInstall{
					ProjectPath:         scopeKey,
					Scope:               scope,
					Agent:               agentName,
					Profile:             record.Profile,
					SkillID:             record.SkillID,
					QualifiedName:       record.QualifiedName,
					SourceQualifiedName: record.SourceQualifiedName,
					SourceID:            record.SourceID,
					Version:             record.Version,
				})
			}
		}
	}

	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.ProjectPath != b.ProjectPath {
			return a.ProjectPath < b.ProjectPath
		}
		if a.SkillID != b.SkillID {
			return a.SkillID < b.SkillID
		}
		return a.Agent < b.Agent
	})
	return items
}

func BuildVersionDrift(items []ProjectInstall, index []skill.Skill) []VersionDriftGroup {
	latestByKey := latestVersionByInstallKey(index)
	grouped := make(map[string][]ProjectInstall)

	for _, item := range items {
		key := projectInstallKey(item)
		grouped[key] = append(grouped[key], item)
	}

	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	groups := make([]VersionDriftGroup, 0)
	for _, key := range keys {
		projects := append([]ProjectInstall(nil), grouped[key]...)
		sortProjectInstalls(projects)

		versionBuckets := make(map[string][]ProjectInstall)
		currentVersions := make(map[string]struct{})
		for _, item := range projects {
			versionBuckets[item.Version] = append(versionBuckets[item.Version], item)
			currentVersions[item.Version] = struct{}{}
		}

		latestVersion := latestByKey[key]
		hasDriftFromLatest := false
		if latestVersion != "" {
			for version := range currentVersions {
				if version != latestVersion {
					hasDriftFromLatest = true
					break
				}
			}
		}
		if len(currentVersions) <= 1 && !hasDriftFromLatest {
			continue
		}

		versions := make([]string, 0, len(versionBuckets))
		for version := range versionBuckets {
			versions = append(versions, version)
		}
		sort.Slice(versions, func(i, j int) bool {
			return compareVersionParts(versions[i], versions[j]) < 0
		})

		buckets := make([]VersionBucket, 0, len(versions))
		for _, version := range versions {
			bucketProjects := append([]ProjectInstall(nil), versionBuckets[version]...)
			sortProjectInstalls(bucketProjects)
			buckets = append(buckets, VersionBucket{
				Version:  version,
				Projects: bucketProjects,
			})
		}

		head := projects[0]
		groups = append(groups, VersionDriftGroup{
			SkillID:             head.SkillID,
			SourceQualifiedName: head.SourceQualifiedName,
			SourceID:            head.SourceID,
			LatestVersion:       latestVersion,
			Versions:            buckets,
		})
	}

	return groups
}

func projectInstallKey(item ProjectInstall) string {
	if item.SourceQualifiedName != "" {
		return item.SourceQualifiedName
	}
	if item.SourceID != "" {
		return item.SourceID + "\x00" + item.SkillID
	}
	if item.QualifiedName != "" {
		return item.QualifiedName
	}
	return item.SkillID
}

func latestVersionByInstallKey(index []skill.Skill) map[string]string {
	result := make(map[string]string, len(index))
	for _, item := range index {
		key := skillInstallKey(item)
		if key == "" {
			continue
		}
		if result[key] == "" || compareVersionParts(result[key], item.Version) < 0 {
			result[key] = item.Version
		}
	}
	return result
}

func compareVersionParts(current string, next string) int {
	if current == next {
		return 0
	}

	currentParts := strings.Split(current, ".")
	nextParts := strings.Split(next, ".")
	maxLen := len(currentParts)
	if len(nextParts) > maxLen {
		maxLen = len(nextParts)
	}

	for i := 0; i < maxLen; i++ {
		currentPart, currentOK := versionPartAt(currentParts, i)
		nextPart, nextOK := versionPartAt(nextParts, i)

		switch {
		case currentOK && nextOK:
			if currentPart < nextPart {
				return -1
			}
			if currentPart > nextPart {
				return 1
			}
		case currentOK != nextOK:
			// Numeric dotted versions sort ahead of non-numeric forms to avoid
			// treating lexical artifacts as a higher semantic version.
			if currentOK {
				return 1
			}
			return -1
		default:
			currentRaw := rawVersionPart(currentParts, i)
			nextRaw := rawVersionPart(nextParts, i)
			if currentRaw < nextRaw {
				return -1
			}
			if currentRaw > nextRaw {
				return 1
			}
		}
	}

	if current < next {
		return -1
	}
	return 1
}

func versionPartAt(parts []string, idx int) (int, bool) {
	if idx >= len(parts) {
		return 0, true
	}
	if parts[idx] == "" {
		return 0, false
	}
	n, err := strconv.Atoi(parts[idx])
	if err != nil {
		return 0, false
	}
	return n, true
}

func rawVersionPart(parts []string, idx int) string {
	if idx >= len(parts) {
		return "0"
	}
	return parts[idx]
}

func skillInstallKey(item skill.Skill) string {
	if item.SourceQualifiedName != "" {
		return item.SourceQualifiedName
	}
	if item.SourceID != "" {
		return item.SourceID + "\x00" + item.ID
	}
	if item.QualifiedName != "" {
		return item.QualifiedName
	}
	return item.ID
}

func sortProjectInstalls(items []ProjectInstall) {
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.ProjectPath != b.ProjectPath {
			return a.ProjectPath < b.ProjectPath
		}
		if a.SkillID != b.SkillID {
			return a.SkillID < b.SkillID
		}
		return a.Agent < b.Agent
	})
}
