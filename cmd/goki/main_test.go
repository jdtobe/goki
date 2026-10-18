package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"version"}, &out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "goki ") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestInit(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "notes")
	var out bytes.Buffer
	if err := run([]string{"init", dir}, &out, &out); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"goki.yaml", "index.md"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("%s not created: %v", f, err)
		}
	}
	// Running again must not overwrite anything.
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := run([]string{"init", dir}, &out, &out); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "index.md")); string(data) != "mine" {
		t.Fatal("init overwrote an existing page")
	}
	if !strings.Contains(out.String(), "already exists") {
		t.Errorf("output = %q", out.String())
	}
}

func TestInitRespectsReadme(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Readme"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{"init", dir}, &out, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "index.md")); err == nil {
		t.Fatal("index.md must not be created when README.md exists")
	}
}

func TestThemes(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"themes"}, &out, &out); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"daylight", "graphite", "glass", "nebula", "aurora", "dune", "ink"} {
		if !strings.Contains(out.String(), id) {
			t.Errorf("themes output missing %q", id)
		}
	}
}

func TestServeRejectsBadArgs(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"a", "b"}, &out, &out); err == nil {
		t.Error("two directories should be an error")
	}
	if err := run([]string{"--open", "telepathy", t.TempDir()}, &out, &out); err == nil {
		t.Error("invalid --open should be an error")
	}
	if err := run([]string{filepath.Join(t.TempDir(), "missing")}, &out, &out); err == nil {
		t.Error("missing directory should be an error")
	}
}

func TestBuiltinThemesMatchCSS(t *testing.T) {
	css, err := os.ReadFile(filepath.Join("..", "..", "web", "css", "themes.css"))
	if err != nil {
		t.Fatal(err)
	}
	for _, th := range builtinThemes[1:] {
		if !bytes.Contains(css, []byte(`[data-theme="`+th.id+`"] {`)) {
			t.Errorf("theme %q is listed but not defined in themes.css", th.id)
		}
	}
}
