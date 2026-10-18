package gitsync

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	c, initialized, err := New(dir, "Test Author", "test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !initialized {
		t.Fatal("expected a new repository to be initialized")
	}
	if err := os.WriteFile(filepath.Join(dir, "page.md"), []byte("# Hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := c.Commit("Create page.md", "page.md"); err != nil {
		t.Fatal(err)
	}
	// Committing again with no changes is a no-op, not an error.
	if err := c.Commit("Nothing", "page.md"); err != nil {
		t.Fatalf("empty commit: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "page.md")); err != nil {
		t.Fatal(err)
	}
	if err := c.Commit("Delete page.md", "page.md"); err != nil {
		t.Fatalf("commit deletion: %v", err)
	}
	out, err := exec.Command("git", "-C", dir, "log", "--format=%an|%s").Output()
	if err != nil {
		t.Fatal(err)
	}
	want := "Test Author|Delete page.md\nTest Author|Create page.md"
	if got := strings.TrimSpace(string(out)); got != want {
		t.Fatalf("log = %q, want %q", got, want)
	}
}
