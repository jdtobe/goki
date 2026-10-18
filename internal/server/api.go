package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/jdtobe/goki/internal/config"
	"github.com/jdtobe/goki/internal/wiki"
)

// httpError is an error with an HTTP status code.
type httpError struct {
	status int
	msg    string
}

func (e httpError) Error() string { return e.msg }

// apiHandler adapts an error-returning handler, translating wiki errors into
// JSON error responses with appropriate status codes.
func apiHandler(log *slog.Logger, h func(http.ResponseWriter, *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		err := h(w, r)
		if err == nil {
			return
		}
		var he httpError
		var mbe *http.MaxBytesError
		status := http.StatusInternalServerError
		switch {
		case errors.As(err, &he):
			status = he.status
		case errors.As(err, &mbe):
			status = http.StatusRequestEntityTooLarge
		case errors.Is(err, wiki.ErrNotFound):
			status = http.StatusNotFound
		case errors.Is(err, wiki.ErrInvalidPath):
			status = http.StatusBadRequest
		case errors.Is(err, wiki.ErrExists), errors.Is(err, wiki.ErrConflict), errors.Is(err, wiki.ErrNotEmpty):
			status = http.StatusConflict
		}
		if status == http.StatusInternalServerError {
			log.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
		}
		writeError(w, status, err.Error())
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	_ = writeJSON(w, status, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	ct := r.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		return httpError{http.StatusUnsupportedMediaType, "expected application/json"}
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 16<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return httpError{http.StatusBadRequest, "invalid JSON: " + err.Error()}
	}
	return nil
}

type configResponse struct {
	Title    string          `json:"title"`
	Theme    string          `json:"theme"`
	Accent   string          `json:"accent,omitempty"`
	ReadOnly bool            `json:"readOnly"`
	Git      bool            `json:"git"`
	Watch    bool            `json:"watch"`
	Version  string          `json:"version"`
	Assets   string          `json:"assetsDir"`
	MaxMB    int             `json:"maxUploadMB"`
	Editor   config.Editor   `json:"editor"`
	Themes   []Theme         `json:"themes"`
	Root     string          `json:"root"`
	Features map[string]bool `json:"features"`
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) error {
	return writeJSON(w, http.StatusOK, configResponse{
		Title:    s.title(),
		Theme:    s.cfg.Theme,
		Accent:   s.cfg.Accent,
		ReadOnly: s.cfg.ReadOnly,
		Git:      s.git != nil,
		Watch:    s.cfg.Watch,
		Version:  s.version,
		Assets:   s.cfg.AssetsDir,
		MaxMB:    s.cfg.MaxUploadMB,
		Editor:   s.cfg.Editor,
		Themes:   s.themes.Custom(),
		Root:     s.store.Root(),
		Features: map[string]bool{"upload": !s.cfg.ReadOnly},
	})
}

func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) error {
	tree, err := s.store.Tree()
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, tree)
}

// pageResponse describes whatever a route resolves to.
type pageResponse struct {
	Kind     string      `json:"kind"` // page, dir or missing
	Path     string      `json:"path"`
	Dir      string      `json:"dir,omitempty"`
	Page     *wiki.Page  `json:"page,omitempty"`
	Children []childInfo `json:"children,omitempty"`
}

func (s *Server) handleGetPage(w http.ResponseWriter, r *http.Request) error {
	route := r.URL.Query().Get("path")
	res, err := s.store.Resolve(route)
	if err != nil {
		return err
	}
	clean, _ := s.store.Clean(strings.TrimSuffix(route, wiki.Ext))
	out := pageResponse{Kind: res.Kind, Path: clean, Dir: res.Dir}
	if res.Kind == "" {
		out.Kind = "missing"
		return writeJSON(w, http.StatusNotFound, out)
	}
	if res.File != "" {
		page, err := s.store.Read(res.File)
		if err != nil {
			return err
		}
		out.Page = page
	}
	if res.Kind == "dir" {
		tree, err := s.store.Tree()
		if err != nil {
			return err
		}
		if n := findNode(tree, res.Dir); n != nil {
			out.Children = shallow(n.Children)
		}
	}
	return writeJSON(w, http.StatusOK, out)
}

func findNode(n *wiki.Node, p string) *wiki.Node {
	if n.Path == p {
		return n
	}
	for _, c := range n.Children {
		if c.Type == "dir" && (c.Path == p || strings.HasPrefix(p, c.Path+"/")) {
			return findNode(c, p)
		}
	}
	return nil
}

// childInfo summarizes a folder entry for folder overview pages.
type childInfo struct {
	wiki.Node
	Count int `json:"count,omitempty"`
}

// shallow copies nodes without their descendants, recording child counts.
func shallow(nodes []*wiki.Node) []childInfo {
	out := make([]childInfo, 0, len(nodes))
	for _, n := range nodes {
		c := childInfo{Node: *n, Count: len(n.Children)}
		c.Children = nil
		out = append(out, c)
	}
	return out
}

type putPageRequest struct {
	File     string `json:"file"`
	Content  string `json:"content"`
	Create   bool   `json:"create"`
	BaseHash string `json:"baseHash"`
}

func (s *Server) handlePutPage(w http.ResponseWriter, r *http.Request) error {
	var req putPageRequest
	if err := decode(r, &req); err != nil {
		return err
	}
	page, err := s.store.Write(req.File, req.Content, wiki.WriteOptions{Create: req.Create, BaseHash: req.BaseHash})
	if errors.Is(err, wiki.ErrConflict) {
		current, rerr := s.store.Read(req.File)
		if rerr != nil {
			return err
		}
		return writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "page": current})
	}
	if err != nil {
		return err
	}
	verb := "Update"
	if req.Create {
		verb = "Create"
	}
	s.changed(fmt.Sprintf("%s %s", verb, page.File), page.File)
	status := http.StatusOK
	if req.Create {
		status = http.StatusCreated
	}
	return writeJSON(w, status, page)
}

type moveRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (s *Server) handleMove(w http.ResponseWriter, r *http.Request) error {
	var req moveRequest
	if err := decode(r, &req); err != nil {
		return err
	}
	dst, err := s.store.Move(req.From, req.To)
	if err != nil {
		return err
	}
	from, _ := s.store.Clean(req.From)
	s.changed(fmt.Sprintf("Move %s to %s", from, dst), from, dst)
	route := dst
	if strings.EqualFold(path.Ext(dst), wiki.Ext) {
		route = wiki.RouteFor(dst)
	}
	return writeJSON(w, http.StatusOK, map[string]string{"path": dst, "route": route})
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	p := q.Get("path")
	recursive, _ := strconv.ParseBool(q.Get("recursive"))
	if err := s.store.Delete(p, recursive); err != nil {
		return err
	}
	clean, _ := s.store.Clean(p)
	s.changed("Delete "+clean, clean)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

type folderRequest struct {
	Path string `json:"path"`
}

func (s *Server) handleFolder(w http.ResponseWriter, r *http.Request) error {
	var req folderRequest
	if err := decode(r, &req); err != nil {
		return err
	}
	rel, err := s.store.Mkdir(req.Path)
	if err != nil {
		return err
	}
	// Empty folders are invisible to git, so there is nothing to commit.
	s.hub.Publish(Event{Type: "change", Paths: []string{rel}})
	return writeJSON(w, http.StatusCreated, map[string]string{"path": rel})
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) error {
	limit := int64(s.cfg.MaxUploadMB) << 20
	r.Body = http.MaxBytesReader(w, r.Body, limit+1<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return httpError{http.StatusRequestEntityTooLarge, fmt.Sprintf("files are limited to %d MB", s.cfg.MaxUploadMB)}
		}
		return httpError{http.StatusBadRequest, "invalid upload: " + err.Error()}
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()
	file, hdr, err := r.FormFile("file")
	if err != nil {
		return httpError{http.StatusBadRequest, "missing file"}
	}
	defer file.Close()
	if hdr.Size > limit {
		return httpError{http.StatusRequestEntityTooLarge, fmt.Sprintf("files are limited to %d MB", s.cfg.MaxUploadMB)}
	}
	dir := r.URL.Query().Get("dir")
	if s.cfg.AssetsDir != "" {
		dir = path.Join(dir, s.cfg.AssetsDir)
	}
	stored, err := s.store.SaveAsset(dir, hdr.Filename, file)
	if err != nil {
		return err
	}
	s.changed("Add "+stored, stored)
	return writeJSON(w, http.StatusCreated, map[string]string{"path": stored, "name": path.Base(stored)})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	hits, err := s.store.Search(q.Get("q"), limit)
	if err != nil {
		return err
	}
	if hits == nil {
		hits = []wiki.Hit{}
	}
	return writeJSON(w, http.StatusOK, hits)
}

// changed notifies browsers and, if enabled, records a git commit.
func (s *Server) changed(message string, paths ...string) {
	s.hub.Publish(Event{Type: "change", Paths: paths})
	if s.git == nil {
		return
	}
	if err := s.git.Commit(message, paths...); err != nil {
		s.log.Warn("git commit failed", "err", err)
	}
}
