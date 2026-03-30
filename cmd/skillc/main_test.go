package main

import (
	"testing"

	"github.com/inhere/skillc/internal/cli"
)

func TestNewApp_HasRootCommandName(t *testing.T) {
	app := cli.NewApp()
	if app == nil {
		t.Fatal("expected app")
	}
	if got := app.Name; got != "skillc" {
		t.Fatalf("got %q want %q", got, "skillc")
	}
}
