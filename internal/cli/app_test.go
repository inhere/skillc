package cli

import (
	"testing"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/testutil/assert"
)

func TestNewApp_RegistersSearchCommand(t *testing.T) {
	app := NewApp()

	search := findCommandByName(app, "search")
	assert.NotNil(t, search)
	assert.Eq(t, "Search indexed skills", search.Desc)

	show := findCommandByName(app, "show")
	assert.NotNil(t, show)
	assert.Eq(t, "Show indexed skill details", show.Desc)
}

func findCommandByName(app *gcli.App, name string) *gcli.Command {
	for _, cmd := range app.Commands() {
		if cmd.Name == name {
			return cmd
		}
	}
	return nil
}
