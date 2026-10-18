package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jdtobe/goki/internal/config"
	"github.com/jdtobe/goki/internal/wiki"
)

type harness struct {
	t    *testing.T
	srv  *Server
	root string
}

func newHarness(t *testing.T, mutate func(*config.Config)) *harness {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"index.md":        "# Home\n\nWelcome.",
		"guides/setup.md": "# Setup\n\nInstall goki.",
		"img/logo.svg":    "<svg xmlns='http://www.w3.org/2000/svg'/>",
		".secret/key.md":  "# Secret",
	}
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Default()
	if mutate != nil {
		mutate(&cfg)
	}
	store, err := wiki.New(root, cfg.Ignore)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := New(Options{Store: store, Config: cfg, Version: "test", Loopback: true})
	if err != nil {
		t.Fatal(err)
	}
	return &harness{t: t, srv: srv, root: root}
}

// do performs a request as the goki front end would.
func (h *harness) do(method, target string, body any, hdr ...string) *httptest.ResponseRecorder {
	h.t.Helper()
	var r io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		r = bytes.NewReader(data)
	}
	req := httptest.NewRequest(method, "http://localhost"+target, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet {
		req.Header.Set("X-Goki", "1")
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		if hdr[i+1] == "" {
			req.Header.Del(hdr[i])
		} else {
			req.Header.Set(hdr[i], hdr[i+1])
		}
	}
	w := httptest.NewRecorder()
	h.srv.ServeHTTP(w, req)
	return w
}

func decodeJSON[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("invalid JSON %q: %v", w.Body.String(), err)
	}
	return v
}

func TestPageLifecycle(t *testing.T) {
	h := newHarness(t, nil)

	w := h.do("GET", "/-/api/page?path=guides/setup", nil)
	if w.Code != 200 {
		t.Fatalf("get: %d %s", w.Code, w.Body)
	}
	got := decodeJSON[pageResponse](t, w)
	if got.Kind != "page" || got.Page.Title != "Setup" {
		t.Fatalf("unexpected page: %+v", got)
	}

	w = h.do("PUT", "/-/api/page", putPageRequest{File: "guides/setup.md", Content: "# Setup v2", BaseHash: got.Page.Hash})
	if w.Code != 200 {
		t.Fatalf("update: %d %s", w.Code, w.Body)
	}
	w = h.do("PUT", "/-/api/page", putPageRequest{File: "guides/setup.md", Content: "# Stale", BaseHash: got.Page.Hash})
	if w.Code != http.StatusConflict {
		t.Fatalf("stale update: %d, want 409", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Setup v2") {
		t.Errorf("conflict response should include the current page: %s", w.Body)
	}

	w = h.do("PUT", "/-/api/page", putPageRequest{File: "new/page.md", Content: "# New", Create: true})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	w = h.do("PUT", "/-/api/page", putPageRequest{File: "new/page.md", Content: "# Again", Create: true})
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate create: %d, want 409", w.Code)
	}

	w = h.do("POST", "/-/api/move", moveRequest{From: "new/page.md", To: "moved/page"})
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"route":"moved/page"`) {
		t.Fatalf("move: %d %s", w.Code, w.Body)
	}

	w = h.do("DELETE", "/-/api/entry?path=moved/page.md", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", w.Code, w.Body)
	}
	w = h.do("GET", "/-/api/page?path=moved/page", nil)
	if w.Code != 404 || decodeJSON[pageResponse](t, w).Kind != "missing" {
		t.Fatalf("deleted page: %d %s", w.Code, w.Body)
	}
}

func TestFolderAndTree(t *testing.T) {
	h := newHarness(t, nil)
	if w := h.do("POST", "/-/api/folder", folderRequest{Path: "projects/alpha"}); w.Code != http.StatusCreated {
		t.Fatalf("mkdir: %d %s", w.Code, w.Body)
	}
	w := h.do("GET", "/-/api/page?path=projects", nil)
	res := decodeJSON[pageResponse](t, w)
	if res.Kind != "dir" || len(res.Children) != 1 || res.Children[0].Name != "alpha" {
		t.Fatalf("folder view: %s", w.Body)
	}
	tree := decodeJSON[wiki.Node](t, h.do("GET", "/-/api/tree", nil))
	for _, c := range tree.Children {
		if strings.HasPrefix(c.Name, ".") {
			t.Fatalf("hidden entry leaked into tree: %s", c.Name)
		}
	}
	if w := h.do("DELETE", "/-/api/entry?path=projects", nil); w.Code != http.StatusConflict {
		t.Fatalf("non-recursive delete of non-empty folder: %d", w.Code)
	}
	if w := h.do("DELETE", "/-/api/entry?path=projects&recursive=true", nil); w.Code != http.StatusNoContent {
		t.Fatalf("recursive delete: %d", w.Code)
	}
}

func TestSecurity(t *testing.T) {
	h := newHarness(t, nil)
	body := putPageRequest{File: "x.md", Content: "x"}

	if w := h.do("PUT", "/-/api/page", body, "X-Goki", ""); w.Code != http.StatusForbidden {
		t.Errorf("missing X-Goki header: %d, want 403", w.Code)
	}
	if w := h.do("PUT", "/-/api/page", body, "Origin", "https://evil.example"); w.Code != http.StatusForbidden {
		t.Errorf("cross-origin write: %d, want 403", w.Code)
	}
	if w := h.do("PUT", "/-/api/page", body, "Origin", "http://localhost"); w.Code != 200 {
		t.Errorf("same-origin write: %d %s", w.Code, w.Body)
	}
	if w := h.do("PUT", "/-/api/page", body, "Content-Type", "text/plain"); w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("non-JSON write: %d, want 415", w.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "http://attacker.example/-/api/tree", nil)
	w := httptest.NewRecorder()
	h.srv.ServeHTTP(w, req)
	if w.Code != http.StatusMisdirectedRequest {
		t.Errorf("DNS rebinding Host: %d, want 421", w.Code)
	}

	for _, p := range []string{"../../etc/passwd", ".secret/key", "%2e%2e/x"} {
		if w := h.do("GET", "/-/api/page?path="+p, nil); w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
			t.Errorf("path %q: %d, want 400/404", p, w.Code)
		}
	}
	if w := h.do("GET", "/.secret/key.md?raw", nil); strings.Contains(w.Body.String(), "# Secret") {
		t.Error("hidden file served raw")
	}
}

func TestReadOnly(t *testing.T) {
	h := newHarness(t, func(c *config.Config) { c.ReadOnly = true })
	if w := h.do("PUT", "/-/api/page", putPageRequest{File: "x.md", Content: "x"}); w.Code != http.StatusForbidden {
		t.Fatalf("write in read-only mode: %d, want 403", w.Code)
	}
	cfg := decodeJSON[configResponse](t, h.do("GET", "/-/api/config", nil))
	if !cfg.ReadOnly {
		t.Fatal("config should report read-only")
	}
}

func TestBasicAuth(t *testing.T) {
	h := newHarness(t, func(c *config.Config) { c.Auth = config.Auth{Username: "me", Password: "pw"} })
	if w := h.do("GET", "/", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("no credentials: %d, want 401", w.Code)
	}
	req := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
	req.SetBasicAuth("me", "pw")
	w := httptest.NewRecorder()
	h.srv.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("valid credentials: %d", w.Code)
	}
}

func TestContentRouting(t *testing.T) {
	h := newHarness(t, nil)

	w := h.do("GET", "/guides/setup", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `<div class="app"`) {
		t.Fatalf("page route should serve the app: %d", w.Code)
	}
	if csp := w.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'self'") {
		t.Errorf("app CSP = %q", csp)
	}

	w = h.do("GET", "/img/logo.svg", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "<svg") {
		t.Fatalf("raw asset: %d %s", w.Code, w.Body)
	}
	if csp := w.Header().Get("Content-Security-Policy"); !strings.HasPrefix(csp, "sandbox") {
		t.Errorf("raw files must be sandboxed, CSP = %q", csp)
	}

	w = h.do("GET", "/guides/setup.md?raw", nil)
	if w.Code != 200 || w.Header().Get("Content-Type") != "text/plain; charset=utf-8" || !strings.Contains(w.Body.String(), "Install goki") {
		t.Fatalf("raw markdown: %d %q %s", w.Code, w.Header().Get("Content-Type"), w.Body)
	}

	w = h.do("GET", "/-/static/js/app.js?v=test", nil)
	if w.Code != 200 || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("static asset: %d cache=%q", w.Code, w.Header().Get("Cache-Control"))
	}
	if w := h.do("GET", "/-/nope", nil); w.Code != 404 {
		t.Errorf("unknown reserved path: %d, want 404", w.Code)
	}
}

func TestSearchEndpoint(t *testing.T) {
	h := newHarness(t, nil)
	hits := decodeJSON[[]wiki.Hit](t, h.do("GET", "/-/api/search?q=install", nil))
	if len(hits) != 1 || hits[0].Path != "guides/setup" {
		t.Fatalf("hits = %+v", hits)
	}
	empty := h.do("GET", "/-/api/search?q=", nil)
	if strings.TrimSpace(empty.Body.String()) != "[]" {
		t.Fatalf("empty query should return [], got %s", empty.Body)
	}
}

func TestUpload(t *testing.T) {
	h := newHarness(t, func(c *config.Config) { c.MaxUploadMB = 1 })
	upload := func(name string, size int) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, _ := mw.CreateFormFile("file", name)
		_, _ = fw.Write(bytes.Repeat([]byte("x"), size))
		_ = mw.Close()
		req := httptest.NewRequest(http.MethodPost, "http://localhost/-/api/upload?dir=guides", &buf)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set("X-Goki", "1")
		w := httptest.NewRecorder()
		h.srv.ServeHTTP(w, req)
		return w
	}
	w := upload("shot.png", 10)
	if w.Code != http.StatusCreated || !strings.Contains(w.Body.String(), `"path":"guides/assets/shot.png"`) {
		t.Fatalf("upload: %d %s", w.Code, w.Body)
	}
	if _, err := os.Stat(filepath.Join(h.root, "guides", "assets", "shot.png")); err != nil {
		t.Fatal(err)
	}
	if w := upload("big.bin", 2<<20); w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversize upload: %d, want 413", w.Code)
	}
}

func TestEventsHub(t *testing.T) {
	hub := NewHub()
	ch, cancel := hub.Subscribe()
	hub.Publish(Event{Type: "change", Paths: []string{"a.md"}})
	if e := <-ch; e.Paths[0] != "a.md" {
		t.Fatalf("event = %+v", e)
	}
	cancel()
	hub.Publish(Event{Type: "change"}) // must not block or panic after unsubscribe
}

func TestCustomThemes(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".goki", "themes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	css := "/* @name Solar Flare\n   @scheme dark */\n[data-theme=\"solar\"] { --accent: orange; }"
	if err := os.WriteFile(filepath.Join(dir, "solar.css"), []byte(css), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "Bad Name.css"), []byte(""), 0o644)
	ts := LoadThemes(root, "")
	got := ts.Custom()
	if len(got) != 1 || got[0] != (Theme{ID: "solar", Label: "Solar Flare", Scheme: "dark"}) {
		t.Fatalf("themes = %+v", got)
	}
	if _, ok := ts.CSS("../../etc/passwd"); ok {
		t.Fatal("theme ids must be validated")
	}
}

func TestLoopbackHost(t *testing.T) {
	for host, want := range map[string]bool{
		"localhost:4242":     true,
		"127.0.0.1:80":       true,
		"[::1]:4242":         true,
		"wiki.localhost":     true,
		"evil.example:4242":  false,
		"192.168.1.10:4242":  false,
		"localhost.evil.com": false,
	} {
		if got := loopbackHost(host); got != want {
			t.Errorf("loopbackHost(%q) = %v, want %v", host, got, want)
		}
	}
}
