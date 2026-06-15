package webapp

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
)

func TestHistoryStoreAppendAndListRecent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	store := newHistoryStore(path)
	store.now = func() time.Time { return time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC) }

	assert.NoErr(t, store.Append(HistoryRecord{Action: "source.add", Status: "ok"}))
	items, err := store.List(10)

	assert.NoErr(t, err)
	assert.Len(t, items, 1)
	assert.Eq(t, "source.add", items[0].Action)
	assert.Eq(t, "ok", items[0].Status)
	assert.Eq(t, "2026-06-15T10:00:00Z", items[0].Time)
}

func TestHistoryStoreListReturnsNewestFirstAndSkipsInvalidLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	assert.NoErr(t, os.WriteFile(path, []byte("{bad json\n{\"time\":\"first\",\"action\":\"source.add\",\"status\":\"ok\"}\n{\"time\":\"second\",\"action\":\"profile.save\",\"status\":\"error\"}\n"), 0o644))

	items, err := newHistoryStore(path).List(1)

	assert.NoErr(t, err)
	assert.Len(t, items, 1)
	assert.Eq(t, "profile.save", items[0].Action)
	assert.Eq(t, "error", items[0].Status)
}
