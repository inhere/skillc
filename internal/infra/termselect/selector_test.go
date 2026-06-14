package termselect

import (
	"context"
	"testing"

	"github.com/gookit/cliui/interact"
	"github.com/gookit/cliui/interact/backend"
	"github.com/gookit/goutil/testutil/assert"
)

func TestCliUISelectorSelectMultiWithFilter(t *testing.T) {
	be := interact.NewUIFakeBackend(
		backend.Event{Type: backend.EventKey, Text: "go"},
		backend.Event{Type: backend.EventKey, Key: backend.KeySpace},
		backend.Event{Type: backend.EventKey, Key: backend.KeyCtrlU},
		backend.Event{Type: backend.EventKey, Key: backend.KeyEnter},
	)
	selector := NewCliUISelectorWithBackend(be)

	got, err := selector.SelectMulti(context.Background(), Options{
		Title: "Choose skills",
		Items: []Item{
			{Key: "1", Label: "Go Pro repo-a/tools v1.0.0", Value: "repo-a/tools/go-pro"},
			{Key: "2", Label: "Review repo-a/tools v1.0.0", Value: "repo-a/tools/review"},
		},
	})

	assert.NoErr(t, err)
	assert.Len(t, got, 1)
	assert.Eq(t, "repo-a/tools/go-pro", got[0].Value)
}

func TestCliUISelectorSelectMultiWithTypedKeys(t *testing.T) {
	be := interact.NewUIFakeBackend(
		backend.Event{Type: backend.EventKey, Key: backend.KeyEnter, Text: "api,web"},
	)
	selector := NewCliUISelectorWithBackend(be)

	got, err := selector.SelectMulti(context.Background(), Options{
		Title: "Choose services",
		Items: []Item{
			{Key: "api", Label: "API", Value: "services/api"},
			{Key: "job", Label: "Job Worker", Value: "services/job"},
			{Key: "web", Label: "Web", Value: "services/web"},
		},
	})

	assert.NoErr(t, err)
	assert.Len(t, got, 2)
	if len(got) != 2 {
		t.FailNow()
	}
	assert.Eq(t, "services/api", got[0].Value)
	assert.Eq(t, "services/web", got[1].Value)
}

func TestCliUISelectorReturnsEmptyForNoItems(t *testing.T) {
	selector := NewCliUISelectorWithBackend(interact.NewUIFakeBackend())

	got, err := selector.SelectMulti(context.Background(), Options{
		Title: "Choose skills",
	})

	assert.NoErr(t, err)
	assert.Len(t, got, 0)
}
