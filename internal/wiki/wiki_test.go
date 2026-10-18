package wiki

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// newTestStore creates a store over a temp directory populated with files.
func newTestStore(t *testing.T, files map[string]string) *Store {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s, err := New(dir, []string{"node_modules"})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestClean(t *testing.T) {
	s := newTestStore(t, nil)
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", "", false},
		{"/", "", false},
		{"guides/setup", "guides/setup", false},
		{"/guides//setup/", "guides/setup", false},
		{`guides\setup`, "guides/setup", false},
		{"./a/./b", "a/b", false},
		{"../etc/passwd", "", true},
		{"a/../../b", "", true},
		{"a/../b", "", true},
		{".git/config", "", true},
		{"docs/.secret", "", true},
		{"node_modules/x", "", true},
		{"a\x00b", "", true},
	}
	for _, tt := range tests {
		got, err := s.Clean(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("Clean(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("Clean(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSymlinkEscapeIsRejected(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.md"), []byte("# Secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := newTestStore(t, map[string]string{"index.md": "# Home"})
	if err := os.Symlink(outside, filepath.Join(s.Root(), "escape")); err != nil {
		t.Skip("symlinks unsupported:", err)
	}
	if _, err := s.Read("escape/secret.md"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("Read through escaping symlink: err = %v, want ErrInvalidPath", err)
	}
	if _, err := s.Write("escape/new.md", "x", WriteOptions{}); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("Write through escaping symlink: err = %v, want ErrInvalidPath", err)
	}
}

func TestResolve(t *testing.T) {
	s := newTestStore(t, map[string]string{
		"index.md":              "# Home",
		"guides/README.md":      "# Guides",
		"guides/setup.md":       "# Setup",
		"guides.md":             "# Guides page wins",
		"empty/.keep":           "",
		"notes/ideas.md":        "# Ideas",
		"notes/assets/logo.png": "png",
	})
	tests := []struct {
		route, kind, file, dir string
	}{
		{"", "dir", "index.md", ""},
		{"guides/setup", "page", "guides/setup.md", ""},
		{"guides/setup.md", "page", "guides/setup.md", ""},
		{"guides", "page", "guides.md", ""}, // file beats folder
		{"notes", "dir", "", "notes"},
		{"empty", "dir", "", "empty"},
		{"missing", "", "", ""},
	}
	for _, tt := range tests {
		r, err := s.Resolve(tt.route)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", tt.route, err)
		}
		if r.Kind != tt.kind || r.File != tt.file || r.Dir != tt.dir {
			t.Errorf("Resolve(%q) = %+v, want kind=%q file=%q dir=%q", tt.route, r, tt.kind, tt.file, tt.dir)
		}
	}
}

func TestReadWriteConflict(t *testing.T) {
	s := newTestStore(t, map[string]string{"page.md": "# One"})
	p, err := s.Read("page.md")
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "One" || p.Path != "page" {
		t.Fatalf("unexpected page: %+v", p)
	}
	p2, err := s.Write("page", "# Two", WriteOptions{BaseHash: p.Hash})
	if err != nil {
		t.Fatalf("write with matching hash: %v", err)
	}
	if p2.Content != "# Two" {
		t.Fatalf("content = %q", p2.Content)
	}
	if _, err := s.Write("page.md", "# Three", WriteOptions{BaseHash: p.Hash}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale hash: err = %v, want ErrConflict", err)
	}
	if _, err := s.Write("page.md", "# New", WriteOptions{Create: true}); !errors.Is(err, ErrExists) {
		t.Fatalf("create existing: err = %v, want ErrExists", err)
	}
	if _, err := s.Write("deep/new/page", "# Deep", WriteOptions{Create: true}); err != nil {
		t.Fatalf("create nested: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.Root(), "deep", "new", "page.md")); err != nil {
		t.Fatalf("nested page not on disk: %v", err)
	}
	if _, err := s.Read("nope.md"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: err = %v, want ErrNotFound", err)
	}
}

func TestWritePreservesMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permissions only")
	}
	s := newTestStore(t, map[string]string{"page.md": "x"})
	full := filepath.Join(s.Root(), "page.md")
	if err := os.Chmod(full, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Write("page.md", "y", WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(full)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
	}
	entries, _ := os.ReadDir(s.Root())
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temporary file left behind: %s", e.Name())
		}
	}
}

func TestMoveAndDelete(t *testing.T) {
	s := newTestStore(t, map[string]string{
		"a.md":       "# A",
		"b.md":       "# B",
		"dir/c.md":   "# C",
		"other/d.md": "# D",
	})
	if dst, err := s.Move("a.md", "moved/renamed"); err != nil || dst != "moved/renamed.md" {
		t.Fatalf("Move page = %q, %v", dst, err)
	}
	if _, err := s.Move("b.md", "other/d"); !errors.Is(err, ErrExists) {
		t.Fatalf("Move onto existing: err = %v, want ErrExists", err)
	}
	if _, err := s.Move("dir", "dir/sub"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("Move into self: err = %v, want ErrInvalidPath", err)
	}
	if _, err := s.Move("dir", "renamed-dir"); err != nil {
		t.Fatalf("Move dir: %v", err)
	}
	if _, err := s.Read("renamed-dir/c.md"); err != nil {
		t.Fatalf("moved dir content: %v", err)
	}
	if err := s.Delete("other", false); !errors.Is(err, ErrNotEmpty) {
		t.Fatalf("Delete non-empty: err = %v, want ErrNotEmpty", err)
	}
	if err := s.Delete("other", true); err != nil {
		t.Fatalf("Delete recursive: %v", err)
	}
	if err := s.Delete("", true); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("Delete root: err = %v, want ErrInvalidPath", err)
	}
	if err := s.Delete("missing.md", false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete missing: err = %v, want ErrNotFound", err)
	}
}

func TestTree(t *testing.T) {
	s := newTestStore(t, map[string]string{
		"index.md":               "# Home",
		"page10.md":              "# Ten",
		"page2.md":               "# Two",
		"guides/README.md":       "# The Guides",
		"guides/01-intro.md":     "no heading",
		"node_modules/x/y.md":    "# Hidden",
		".git/HEAD.md":           "# Hidden",
		"empty-section/.gitkeep": "",
		"notes.txt":              "not a page",
		"guides/assets/shot.png": "png",
		"media/deep/clip.mp4":    "mp4",
	})
	tree, err := s.Tree()
	if err != nil {
		t.Fatal(err)
	}
	if tree.File != "index.md" {
		t.Errorf("root index = %q", tree.File)
	}
	var names []string
	for _, c := range tree.Children {
		names = append(names, c.Type+":"+c.Name)
	}
	want := "dir:empty-section dir:guides page:page2 page:page10"
	if got := strings.Join(names, " "); got != want {
		t.Errorf("children = %q, want %q", got, want)
	}
	guides := tree.Children[1]
	if guides.Title != "The Guides" || guides.File != "guides/README.md" {
		t.Errorf("guides node = %+v", guides)
	}
	if len(guides.Children) != 1 || guides.Children[0].Title != "Intro" {
		t.Errorf("guides children = %+v", guides.Children)
	}
}

func TestTitleOf(t *testing.T) {
	tests := []struct{ content, file, want string }{
		{"# Hello", "x.md", "Hello"},
		{"---\ntitle: From Front Matter\n---\n# Heading", "x.md", "From Front Matter"},
		{"---\ntitle: \"Quoted\"\n---\n", "x.md", "Quoted"},
		{"```\n# not a heading\n```\n# Real", "x.md", "Real"},
		{"## Second level only", "my-page.md", "My page"},
		{"", "guides/index.md", "Guides"},
		{"", "README.md", "Home"},
		{"\uFEFF# BOM", "x.md", "BOM"},
		{"# Closed #", "x.md", "Closed"},
	}
	for _, tt := range tests {
		if got := TitleOf([]byte(tt.content), tt.file); got != tt.want {
			t.Errorf("TitleOf(%q, %q) = %q, want %q", tt.content, tt.file, got, tt.want)
		}
	}
}

func TestDisplayName(t *testing.T) {
	tests := map[string]string{
		"02-getting_started.md": "Getting started",
		"api-reference":         "Api reference",
		"10":                    "10",
		"ünicode":               "Ünicode",
	}
	for in, want := range tests {
		if got := DisplayName(in); got != want {
			t.Errorf("DisplayName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNaturalLess(t *testing.T) {
	if !naturalLess("page2", "page10") || naturalLess("page10", "page2") {
		t.Error("numeric runs must compare numerically")
	}
	if !naturalLess("a", "b") || !naturalLess("a", "ab") {
		t.Error("plain strings must compare lexically")
	}
	if !naturalLess("v01", "v2") {
		t.Error("leading zeros must be ignored")
	}
}

func TestRouteFor(t *testing.T) {
	tests := map[string]string{
		"index.md":         "",
		"README.md":        "",
		"guides/index.md":  "guides",
		"guides/setup.md":  "guides/setup",
		"guides/readme.md": "guides",
	}
	for in, want := range tests {
		if got := RouteFor(in); got != want {
			t.Errorf("RouteFor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSearch(t *testing.T) {
	s := newTestStore(t, map[string]string{
		"kubernetes.md":  "---\ntags: [ops]\n---\n# Kubernetes\n\nContainers at scale.",
		"notes/misc.md":  "# Misc\n\nWe run kubernetes in production.",
		"unrelated.md":   "# Cooking\n\nPasta.",
		"multi/terms.md": "# Deploying\n\nDeploy with kubernetes and helm charts.",
	})
	hits, err := s.Search("kubernetes", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 3 {
		t.Fatalf("got %d hits, want 3: %+v", len(hits), hits)
	}
	if hits[0].Path != "kubernetes" {
		t.Errorf("title match should rank first, got %q", hits[0].Path)
	}
	if strings.Contains(hits[0].Snippet, "tags:") {
		t.Errorf("snippet should skip front matter: %q", hits[0].Snippet)
	}
	if got := frontMatterEnd([]byte("---\na: b\n---\nbody")); got != 13 {
		t.Errorf("frontMatterEnd = %d, want 13", got)
	}
	hits, _ = s.Search("kubernetes helm", 10)
	if len(hits) != 1 || hits[0].Path != "multi/terms" {
		t.Errorf("all terms must match: %+v", hits)
	}
	if !strings.Contains(hits[0].Snippet, "kubernetes") {
		t.Errorf("snippet should include match: %q", hits[0].Snippet)
	}
	if hits, _ := s.Search("   ", 10); hits != nil {
		t.Errorf("blank query should return nil, got %+v", hits)
	}
}

func TestSaveAsset(t *testing.T) {
	s := newTestStore(t, nil)
	p1, err := s.SaveAsset("guides/assets", "Screen Shot.png", strings.NewReader("a"))
	if err != nil {
		t.Fatal(err)
	}
	p2, err := s.SaveAsset("guides/assets", "Screen Shot.png", strings.NewReader("b"))
	if err != nil {
		t.Fatal(err)
	}
	if p1 != "guides/assets/Screen Shot.png" || p2 != "guides/assets/Screen Shot-2.png" {
		t.Fatalf("paths = %q, %q", p1, p2)
	}
	if _, err := s.SaveAsset("", "../../evil.png", strings.NewReader("x")); err != nil {
		t.Fatalf("traversal in name should be neutralized, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.Root(), "evil.png")); err != nil {
		t.Fatalf("sanitized asset should land in root: %v", err)
	}
	if p, err := s.SaveAsset("", ".htaccess", strings.NewReader("x")); err != nil || p != "htaccess" {
		t.Fatalf("hidden names must be made visible: %q, %v", p, err)
	}
	if _, err := s.SaveAsset("", "...", strings.NewReader("x")); err == nil {
		t.Fatal("names that sanitize to nothing must be rejected")
	}
}
