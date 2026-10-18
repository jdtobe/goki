package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultIsValid(t *testing.T) {
	c := Default()
	if err := c.Validate(); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
}

func TestLoadPrecedence(t *testing.T) {
	userDir := t.TempDir()
	t.Setenv("GOKI_CONFIG", filepath.Join(userDir, "config.yaml"))
	write(t, filepath.Join(userDir, "config.yaml"), "theme: nebula\nport: 5000\ntitle: User\n")

	wikiDir := t.TempDir()
	write(t, filepath.Join(wikiDir, "goki.yaml"), "title: Wiki\neditor:\n  layout: write\n")

	t.Setenv("GOKI_PORT", "6000")

	c, err := Load(wikiDir, "")
	if err != nil {
		t.Fatal(err)
	}
	if c.Theme != "nebula" {
		t.Errorf("theme = %q, want user value", c.Theme)
	}
	if c.Title != "Wiki" {
		t.Errorf("title = %q, want wiki value to override user value", c.Title)
	}
	if c.Port != 6000 {
		t.Errorf("port = %d, want env override", c.Port)
	}
	if c.Editor.Layout != "write" || !c.Editor.Spellcheck {
		t.Errorf("editor = %+v, want layout overridden and defaults kept", c.Editor)
	}
	if len(c.Source) != 2 {
		t.Errorf("sources = %v", c.Source)
	}
}

func TestLoadExplicitMustExist(t *testing.T) {
	t.Setenv("GOKI_CONFIG", filepath.Join(t.TempDir(), "none.yaml"))
	if _, err := Load(t.TempDir(), "/definitely/not/here.yaml"); err == nil {
		t.Fatal("expected error for missing explicit config")
	}
}

func TestLoadRejectsBadYAML(t *testing.T) {
	t.Setenv("GOKI_CONFIG", filepath.Join(t.TempDir(), "none.yaml"))
	dir := t.TempDir()
	write(t, filepath.Join(dir, "goki.yaml"), "port: [not a number\n")
	if _, err := Load(dir, ""); err == nil {
		t.Fatal("expected YAML error")
	}
}

func TestEnvBools(t *testing.T) {
	c := Default()
	env := map[string]string{"GOKI_READ_ONLY": "true", "GOKI_WATCH": "0"}
	err := c.applyEnv(func(k string) (string, bool) { v, ok := env[k]; return v, ok })
	if err != nil {
		t.Fatal(err)
	}
	if !c.ReadOnly || c.Watch {
		t.Errorf("read_only=%v watch=%v", c.ReadOnly, c.Watch)
	}
	env["GOKI_WATCH"] = "maybe"
	if err := c.applyEnv(func(k string) (string, bool) { v, ok := env[k]; return v, ok }); err == nil {
		t.Error("expected error for invalid bool")
	}
}

func TestValidate(t *testing.T) {
	c := Default()
	c.Port = 70000
	c.Open = "telepathy"
	c.AssetsDir = "../outside"
	err := c.Validate()
	if err == nil {
		t.Fatal("expected validation errors")
	}
	for _, want := range []string{"port", "open", "assets_dir"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}
}

func TestSampleParses(t *testing.T) {
	t.Setenv("GOKI_CONFIG", filepath.Join(t.TempDir(), "none.yaml"))
	dir := t.TempDir()
	write(t, filepath.Join(dir, "goki.yaml"), Sample)
	c, err := Load(dir, "")
	if err != nil {
		t.Fatalf("sample config does not load: %v", err)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("sample config invalid: %v", err)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
