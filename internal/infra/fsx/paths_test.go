package fsx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandPath_HomeDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	got, err := ExpandPath("~/skillc-test", filepath.Join(home, "project"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "skillc-test")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExpandPath_EnvVar(t *testing.T) {
	if err := os.Setenv("SKILLC_TMP", "env-dir"); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("SKILLC_TMP")

	got, err := ExpandPath("$SKILLC_TMP/config.yaml", "/tmp/base")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, filepath.Join("env-dir", "config.yaml")) {
		t.Fatalf("expected expanded env path, got %q", got)
	}
}

func TestExpandPath_RelativeToBaseDir(t *testing.T) {
	base := filepath.Join(string(filepath.Separator), "workspace", "skillc")
	got, err := ExpandPath("./configs/app.yaml", base)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(base, "configs", "app.yaml")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
