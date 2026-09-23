package install

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestPlanMergeClassifiesThreeWayChanges(t *testing.T) {
	baseline := map[string]string{
		"unchanged":        "a",
		"local-edit":       "b",
		"both-edit":        "c",
		"upstream-edit":    "d",
		"upstream-deleted": "h",
		"local-removed":    "f",
		"kept-removed":     "g",
	}
	current := map[string]string{
		"unchanged":        "a",
		"local-edit":       "b2",
		"both-edit":        "c2",
		"upstream-edit":    "d",
		"upstream-deleted": "h",
		"kept-removed":     "g2",
		"local-only":       "x",
	}
	incoming := map[string]string{
		"unchanged":     "a",
		"local-edit":    "b",
		"both-edit":     "c3",
		"upstream-edit": "d2",
		"local-removed": "f2",
		"upstream-only": "y",
	}

	got := make(map[string]MergeAction)
	for _, item := range PlanMerge(baseline, current, incoming) {
		got[item.Path] = item.Action
	}

	assert.Eq(t, MergeUnchanged, got["unchanged"])
	assert.Eq(t, MergeKeepLocal, got["local-edit"])
	assert.Eq(t, MergeConflict, got["both-edit"])
	assert.Eq(t, MergeTakeIncoming, got["upstream-edit"])
	assert.Eq(t, MergeRemove, got["upstream-deleted"])
	assert.Eq(t, MergeLocalDeleted, got["local-removed"])
	assert.Eq(t, MergeKeepRemoved, got["kept-removed"])
	assert.Eq(t, MergeAddLocal, got["local-only"])
	assert.Eq(t, MergeAddIncoming, got["upstream-only"])
}

func TestPlanMergeTreatsUnknownSharedFileAsConflict(t *testing.T) {
	items := PlanMerge(nil, map[string]string{"new.md": "local"}, map[string]string{"new.md": "upstream"})

	assert.Len(t, items, 1)
	assert.Eq(t, MergeConflict, items[0].Action)
	assert.Eq(t, "local", items[0].LocalHash)
	assert.Eq(t, "upstream", items[0].IncomingHash)
}

func TestPlanMergeSummarizeGroupsActions(t *testing.T) {
	plan := PlanMerge(
		map[string]string{"a": "1", "b": "1", "c": "1", "d": "1"},
		map[string]string{"a": "2", "b": "1", "c": "1"},
		map[string]string{"a": "3", "b": "2", "e": "1"},
	)

	result := Summarize(plan)

	assert.Eq(t, []string{"b", "e"}, result.Updated)
	assert.Eq(t, []string{"a"}, result.Conflicts)
	assert.Eq(t, []string{"c"}, result.Removed)
	assert.Len(t, result.KeptLocal, 1)
	assert.Eq(t, "d", result.KeptLocal[0])
}
