package install

import "sort"

// MergeAction 表示按文件三方合并时对单个文件的处理动作。
type MergeAction string

const (
	// MergeUnchanged 本地与上游内容一致，无需处理。
	MergeUnchanged MergeAction = "unchanged"
	// MergeTakeIncoming 本地未改、上游已改，取上游内容。
	MergeTakeIncoming MergeAction = "take-incoming"
	// MergeAddIncoming 源新增文件，本地没有。
	MergeAddIncoming MergeAction = "add-incoming"
	// MergeKeepLocal 本地改过、上游未改，保留本地内容。
	MergeKeepLocal MergeAction = "keep-local"
	// MergeAddLocal 本地新增文件，源里没有。
	MergeAddLocal MergeAction = "add-local"
	// MergeConflict 本地与上游都改了同一文件，需要人工确认。
	MergeConflict MergeAction = "conflict"
	// MergeRemove 上游删除且本地未改，删除本地文件。
	MergeRemove MergeAction = "remove"
	// MergeKeepRemoved 上游删除但本地改过，保留本地文件。
	MergeKeepRemoved MergeAction = "keep-removed"
	// MergeLocalDeleted 本地删除了文件，保持删除状态。
	MergeLocalDeleted MergeAction = "local-deleted"
)

// MergeItem 是单个文件的合并动作。
type MergeItem struct {
	Path         string
	Action       MergeAction
	BaselineHash string
	LocalHash    string
	IncomingHash string
}

// MergeResult 汇总一次合并的结果。
type MergeResult struct {
	Items []MergeItem
	// Updated 是取上游内容/新增的文件（已写入安装目录）。
	Updated []string
	// KeptLocal 是保留本地内容的文件（含本地新增）。
	KeptLocal []string
	// Conflicts 是双方都改动的文件，本地内容保留、上游内容写入 <path>.incoming。
	Conflicts []string
	// Removed 是跟随上游删除的文件。
	Removed []string
}

// Changed 表示合并过程中确实处理了文件（用于决定是否上报合并明细）。
func (r MergeResult) Changed() bool {
	return len(r.Updated)+len(r.KeptLocal)+len(r.Conflicts)+len(r.Removed) > 0
}

// PlanMerge 比较 baseline（安装时记录的源内容）、current（安装目录现状）、
// incoming（源当前内容）三份「相对路径 → 内容哈希」，得出每个文件的合并动作。
func PlanMerge(baseline map[string]string, current map[string]string, incoming map[string]string) []MergeItem {
	paths := make([]string, 0, len(baseline)+len(current)+len(incoming))
	seen := make(map[string]struct{}, len(baseline)+len(current)+len(incoming))
	for _, group := range []map[string]string{baseline, current, incoming} {
		for path := range group {
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)

	items := make([]MergeItem, 0, len(paths))
	for _, path := range paths {
		base, baseOK := baseline[path]
		cur, curOK := current[path]
		inc, incOK := incoming[path]
		item := MergeItem{Path: path}
		if baseOK {
			item.BaselineHash = base
		}
		if curOK {
			item.LocalHash = cur
		}
		if incOK {
			item.IncomingHash = inc
		}

		switch {
		case curOK && incOK && cur == inc:
			item.Action = MergeUnchanged
		case !baseOK && !curOK && incOK:
			item.Action = MergeAddIncoming
		case !baseOK && curOK && !incOK:
			item.Action = MergeAddLocal
		case !baseOK && curOK && incOK:
			// 双方都有但都没有基线：内容不同即冲突
			item.Action = MergeConflict
		case baseOK && !curOK:
			// 本地删除过：保持删除，不跟随上游恢复
			item.Action = MergeLocalDeleted
		case baseOK && curOK && !incOK:
			if cur == base {
				item.Action = MergeRemove
			} else {
				item.Action = MergeKeepRemoved
			}
		case baseOK && curOK && incOK && cur == base:
			item.Action = MergeTakeIncoming
		case baseOK && curOK && incOK && inc == base:
			// 上游未改、本地改过：保留本地
			item.Action = MergeKeepLocal
		default:
			item.Action = MergeConflict
		}
		items = append(items, item)
	}
	return items
}

// Summarize 汇总合并动作，便于 CLI/Web 展示。
func Summarize(items []MergeItem) MergeResult {
	result := MergeResult{Items: items}
	for _, item := range items {
		switch item.Action {
		case MergeTakeIncoming, MergeAddIncoming:
			result.Updated = append(result.Updated, item.Path)
		case MergeKeepLocal, MergeAddLocal, MergeKeepRemoved, MergeLocalDeleted:
			result.KeptLocal = append(result.KeptLocal, item.Path)
		case MergeConflict:
			result.Conflicts = append(result.Conflicts, item.Path)
		case MergeRemove:
			result.Removed = append(result.Removed, item.Path)
		}
	}
	return result
}
