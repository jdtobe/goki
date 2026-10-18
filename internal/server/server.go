// Package server exposes a wiki.Store over HTTP: a JSON API under /-/api,
// the embedded single-page app, and raw files (images, attachments) at their
// natural paths so relative links in Markdown simply work.
package server

import (
	"bytes"
	"crypto/subtle"
	"html/template"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/jdtobe/goki/internal/config"
	"github.com/jdtobe/goki/internal/gitsync"
	"github.com/jdtobe/goki/internal/wiki"
	"github.com/jdtobe/goki/web"
)

// Prefix is the reserved URL namespace for goki itself. A wiki folder named
// "-" at the root is therefore not reachable.
const Prefix = "/-/"

// Options configures a Server.
type Options struct {
	Store    *wiki.Store
	Config   config.Config
	Git      *gitsync.Committer // nil disables auto-commit
	Version  string
	Logger   *slog.Logger
	Loopback bool // listening only on a loopback interface
}

// Server is the goki HTTP handler.
type Server struct {
	store   *wiki.Store
	cfg     config.Config
	git     *gitsync.Committer
	version string
	log     *slog.Logger
	loop    bool
	hub     *Hub
	themes  *ThemeSet
	index   *template.Template
	static  fs.FS
	dev     bool
	handler http.Handler
}

// New builds a Server.
func New(opt Options) (*Server, error) {
	if opt.Logger == nil {
		opt.Logger = slog.New(slog.DiscardHandler)
	}
	// GOKI_WEB_DIR serves the front end from disk, so edits to web/ apply
	// on reload during development.
	var static fs.FS = web.FS
	devDir := os.Getenv("GOKI_WEB_DIR")
	if devDir != "" {
		static = os.DirFS(devDir)
	}
	tmpl, err := parseIndex(static)
	if err != nil {
		return nil, err
	}
	s := &Server{
		store:   opt.Store,
		cfg:     opt.Config,
		git:     opt.Git,
		version: opt.Version,
		log:     opt.Logger,
		loop:    opt.Loopback,
		hub:     NewHub(),
		themes:  LoadThemes(opt.Store.Root(), config.UserDir()),
		index:   tmpl,
		static:  static,
		dev:     devDir != "",
	}
	s.handler = s.routes()
	return s, nil
}

// Hub returns the server's event hub so callers can attach a watcher.
func (s *Server) Hub() *Hub { return s.hub }

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	api := func(pattern string, h func(http.ResponseWriter, *http.Request) error) {
		mux.Handle(pattern, apiHandler(s.log, h))
	}
	api("GET /-/api/config", s.handleConfig)
	api("GET /-/api/tree", s.handleTree)
	api("GET /-/api/page", s.handleGetPage)
	api("PUT /-/api/page", s.mutating(s.handlePutPage))
	api("POST /-/api/move", s.mutating(s.handleMove))
	api("DELETE /-/api/entry", s.mutating(s.handleDelete))
	api("POST /-/api/folder", s.mutating(s.handleFolder))
	api("POST /-/api/upload", s.mutating(s.handleUpload))
	api("GET /-/api/search", s.handleSearch)
	mux.HandleFunc("GET /-/api/events", s.handleEvents)
	mux.HandleFunc("GET /-/themes/{file}", s.handleTheme)
	mux.Handle("GET /-/static/", http.StripPrefix("/-/static/", s.staticHandler()))
	mux.HandleFunc("GET /-/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /-/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	mux.HandleFunc("GET /", s.handleContent)
	return s.secure(mux)
}

// secure wraps every request with host validation, authentication and
// security headers.
func (s *Server) secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("X-Frame-Options", "DENY")

		// When bound to loopback, refuse foreign Host headers. This defeats
		// DNS-rebinding attacks from malicious web pages.
		if s.loop && !loopbackHost(r.Host) {
			http.Error(w, "goki: unexpected Host header", http.StatusMisdirectedRequest)
			return
		}
		if s.cfg.Auth.Enabled() {
			u, p, ok := r.BasicAuth()
			if !ok || !equal(u, s.cfg.Auth.Username) || !equal(p, s.cfg.Auth.Password) {
				h.Set("WWW-Authenticate", `Basic realm="goki", charset="UTF-8"`)
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func equal(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// loopbackHost reports whether a Host header names the local machine.
func loopbackHost(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.Trim(strings.ToLower(host), "[]")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// mutating guards state-changing endpoints. Requests must carry the
// X-Goki header, which browsers will not send cross-origin without a CORS
// preflight that goki never approves, and any Origin must match the Host.
func (s *Server) mutating(h func(http.ResponseWriter, *http.Request) error) func(http.ResponseWriter, *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		if s.cfg.ReadOnly {
			return httpError{http.StatusForbidden, "this wiki is read-only"}
		}
		if r.Header.Get("X-Goki") != "1" {
			return httpError{http.StatusForbidden, "missing X-Goki request header"}
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			if !sameOrigin(origin, r.Host) {
				return httpError{http.StatusForbidden, "cross-origin request refused"}
			}
		}
		return h(w, r)
	}
}

func sameOrigin(origin, host string) bool {
	_, rest, ok := strings.Cut(origin, "://")
	return ok && strings.EqualFold(rest, host)
}

const csp = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob: https: http:; media-src 'self' https: http:; font-src 'self' data:; " +
	"connect-src 'self'; frame-src https:; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

// handleContent serves wiki files (images and other attachments) when the
// path names one, and the single-page app for everything else.
func (s *Server) handleContent(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	ext := strings.ToLower(path.Ext(p))
	if ext != "" && ext != wiki.Ext && !strings.HasSuffix(p, "/") {
		if s.serveRaw(w, r, strings.TrimPrefix(p, "/")) {
			return
		}
	}
	if r.URL.Query().Has("raw") && ext == wiki.Ext {
		if s.serveRaw(w, r, strings.TrimPrefix(p, "/")) {
			return
		}
	}
	s.serveApp(w, r)
}

func (s *Server) serveRaw(w http.ResponseWriter, r *http.Request, rel string) bool {
	f, info, err := s.store.Open(rel)
	if err != nil {
		return false
	}
	defer f.Close()
	// Sandbox user files so an HTML or SVG file opened directly cannot run
	// script with access to the API.
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'; media-src 'self'")
	w.Header().Set("Cache-Control", "no-cache")
	if strings.EqualFold(path.Ext(rel), wiki.Ext) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
	return true
}

type indexData struct {
	Title   string
	Theme   string
	Version string
	Themes  []Theme
}

func parseIndex(static fs.FS) (*template.Template, error) {
	html, err := fs.ReadFile(static, "index.html")
	if err != nil {
		return nil, err
	}
	return template.New("index").Delims("[[", "]]").Parse(string(html))
}

func (s *Server) serveApp(w http.ResponseWriter, r *http.Request) {
	tmpl := s.index
	if s.dev {
		if t, err := parseIndex(s.static); err == nil {
			tmpl = t
		}
	}
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, indexData{
		Title:   s.title(),
		Theme:   s.cfg.Theme,
		Version: s.version,
		Themes:  s.themes.Custom(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	h.Set("Content-Security-Policy", csp)
	_, _ = w.Write(buf.Bytes())
}

func (s *Server) title() string {
	if s.cfg.Title != "" {
		return s.cfg.Title
	}
	return wiki.DisplayName(path.Base(strings.ReplaceAll(s.store.Root(), "\\", "/")))
}

// staticHandler serves embedded assets. URLs carry ?v=<version>, so they can
// be cached aggressively.
func (s *Server) staticHandler() http.Handler {
	files := http.FileServerFS(s.static)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || strings.HasSuffix(r.URL.Path, "/") || r.URL.Path == "index.html" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("v") != "" && !s.dev && s.version != "dev" {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		files.ServeHTTP(w, r)
	})
}

func (s *Server) handleTheme(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(r.PathValue("file"), ".css")
	css, ok := s.themes.CSS(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, id+".css", time.Time{}, bytes.NewReader(css))
}
