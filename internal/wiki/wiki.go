// Package wiki implements a Markdown wiki whose structure is the filesystem.
//
// Every directory under the root is a section and every *.md file is a page.
// A directory's own page is its index.md or README.md. Nothing is stored
// anywhere except in the files themselves.
package wiki

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// Ext is the file extension that identifies a page.
const Ext = ".md"

// Errors returned by Store methods. Callers can test for them with errors.Is.
var (
	ErrNotFound    = errors.New("not found")
	ErrExists      = errors.New("already exists")
	ErrInvalidPath = errors.New("invalid path")
	ErrConflict    = errors.New("modified since it was loaded")
	ErrNotEmpty    = errors.New("folder is not empty")
)

// indexNames are the file names, in priority order, that act as a folder's page.
var indexNames = []string{"index.md", "README.md", "readme.md", "Readme.md"}

// Node is an entry in the page tree.
type Node struct {
	Type     string    `json:"type"` // "page" or "dir"
	Name     string    `json:"name"` // base name without extension
	Path     string    `json:"path"` // slash-separated route, without extension
	File     string    `json:"file,omitempty"`
	Title    string    `json:"title"`
	Modified time.Time `json:"modified,omitzero"`
	Children []*Node   `json:"children,omitempty"`
}

// Page is a single Markdown document.
type Page struct {
	Path     string    `json:"path"`
	File     string    `json:"file"`
	Title    string    `json:"title"`
	Content  string    `json:"content"`
	Hash     string    `json:"hash"`
	Modified time.Time `json:"modified"`
	Size     int64     `json:"size"`
}

// Store provides safe access to a wiki rooted at a directory.
type Store struct {
	root   string
	ignore []string

	mu     sync.Mutex
	titles map[string]cachedTitle
}

type cachedTitle struct {
	mod   time.Time
	title string
}

// New returns a Store rooted at dir. Entries whose base name matches any of
// the ignore glob patterns are invisible to the wiki, as are dot-files.
func New(dir string, ignore []string) (*Store, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}
	return &Store{root: abs, ignore: ignore, titles: map[string]cachedTitle{}}, nil
}

// Root returns the absolute path of the wiki root.
func (s *Store) Root() string { return s.root }

// Clean normalizes a user-supplied relative path and rejects anything that
// would escape the root or touch a hidden or ignored entry. The result is
// slash-separated with no leading slash; the root itself is "".
func (s *Store) Clean(p string) (string, error) {
	p = strings.ReplaceAll(p, "\\", "/")
	if strings.ContainsRune(p, 0) {
		return "", ErrInvalidPath
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return "", ErrInvalidPath
		}
	}
	p = strings.Trim(path.Clean("/"+p), "/")
	if p == "" {
		return "", nil
	}
	for _, seg := range strings.Split(p, "/") {
		if s.Hidden(seg) {
			return "", ErrInvalidPath
		}
	}
	return p, nil
}

// Hidden reports whether a single path segment is excluded from the wiki.
func (s *Store) Hidden(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	for _, pat := range s.ignore {
		if ok, _ := path.Match(pat, name); ok {
			return true
		}
	}
	return false
}

// abs converts a cleaned relative path into an absolute filesystem path and
// verifies, after resolving symlinks, that it is still inside the root.
func (s *Store) abs(rel string) (string, error) {
	full := filepath.Join(s.root, filepath.FromSlash(rel))
	// Resolve the deepest existing ancestor so symlinks cannot point outside.
	probe := full
	for {
		resolved, err := filepath.EvalSymlinks(probe)
		if err == nil {
			if resolved != s.root && !strings.HasPrefix(resolved, s.root+string(filepath.Separator)) {
				return "", ErrInvalidPath
			}
			break
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			break
		}
		probe = parent
	}
	return full, nil
}

// Resolved describes what a route points at.
type Resolved struct {
	Kind string // "page", "dir" or "" when nothing exists
	File string // relative page file, if any
	Dir  string // relative directory, if Kind is "dir"
}

// Resolve maps a route (e.g. "guides/setup") to a page file or folder.
// A page file takes precedence over a folder of the same name.
func (s *Store) Resolve(route string) (Resolved, error) {
	rel, err := s.Clean(strings.TrimSuffix(route, Ext))
	if err != nil {
		return Resolved{}, err
	}
	if rel != "" {
		if f, err := s.abs(rel + Ext); err == nil {
			if info, err := os.Stat(f); err == nil && info.Mode().IsRegular() {
				return Resolved{Kind: "page", File: rel + Ext}, nil
			}
		}
	}
	dir, err := s.abs(rel)
	if err != nil {
		return Resolved{}, err
	}
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		r := Resolved{Kind: "dir", Dir: rel}
		if idx := s.indexFile(rel); idx != "" {
			r.File = idx
		}
		return r, nil
	}
	return Resolved{}, nil
}

// indexFile returns the relative path of a folder's index page, or "".
func (s *Store) indexFile(dir string) string {
	for _, name := range indexNames {
		rel := path.Join(dir, name)
		full, err := s.abs(rel)
		if err != nil {
			continue
		}
		if info, err := os.Stat(full); err == nil && info.Mode().IsRegular() {
			return rel
		}
	}
	return ""
}

// IsIndex reports whether a file name is one of the folder index names.
func IsIndex(name string) bool {
	for _, n := range indexNames {
		if strings.EqualFold(n, name) {
			return true
		}
	}
	return false
}

// RouteFor returns the URL route for a relative page file.
func RouteFor(file string) string {
	dir, base := path.Split(file)
	if IsIndex(base) {
		return strings.TrimSuffix(dir, "/")
	}
	return strings.TrimSuffix(file, Ext)
}

// Read loads a page by its relative file path.
func (s *Store) Read(file string) (*Page, error) {
	rel, err := s.pageFile(file)
	if err != nil {
		return nil, err
	}
	full, err := s.abs(rel)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(full)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return nil, err
	}
	return &Page{
		Path:     RouteFor(rel),
		File:     rel,
		Title:    TitleOf(data, rel),
		Content:  string(data),
		Hash:     Hash(data),
		Modified: info.ModTime(),
		Size:     info.Size(),
	}, nil
}

// WriteOptions controls Write.
type WriteOptions struct {
	// Create fails with ErrExists if the page already exists.
	Create bool
	// BaseHash, when set, must match the current content hash or the write
	// fails with ErrConflict. This prevents silently clobbering edits.
	BaseHash string
}

// Write creates or replaces a page, creating parent folders as needed.
func (s *Store) Write(file, content string, opt WriteOptions) (*Page, error) {
	rel, err := s.pageFile(file)
	if err != nil {
		return nil, err
	}
	full, err := s.abs(rel)
	if err != nil {
		return nil, err
	}
	existing, readErr := os.ReadFile(full)
	exists := readErr == nil
	if opt.Create && exists {
		return nil, ErrExists
	}
	if opt.BaseHash != "" && exists && Hash(existing) != opt.BaseHash {
		return nil, ErrConflict
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return nil, err
	}
	if err := writeAtomic(full, []byte(content)); err != nil {
		return nil, err
	}
	return s.Read(rel)
}

// writeAtomic writes data to a temporary file and renames it into place so a
// crash never leaves a half-written page behind.
func writeAtomic(full string, data []byte) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(full); err == nil {
		mode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(full), ".goki-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), full)
}

// pageFile cleans a page file path, appending the extension if missing.
func (s *Store) pageFile(file string) (string, error) {
	rel, err := s.Clean(file)
	if err != nil {
		return "", err
	}
	if rel == "" {
		return "", ErrInvalidPath
	}
	if !strings.EqualFold(path.Ext(rel), Ext) {
		rel += Ext
	}
	return rel, nil
}

// Mkdir creates a folder (and any missing parents).
func (s *Store) Mkdir(dir string) (string, error) {
	rel, err := s.Clean(dir)
	if err != nil {
		return "", err
	}
	if rel == "" {
		return "", ErrInvalidPath
	}
	full, err := s.abs(rel)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(full); err == nil {
		return "", ErrExists
	}
	return rel, os.MkdirAll(full, 0o755)
}

// Move renames a page or folder. Pages keep their extension; parent folders
// are created as needed. Existing destinations are never overwritten.
func (s *Store) Move(from, to string) (string, error) {
	src, err := s.Clean(from)
	if err != nil {
		return "", err
	}
	dst, err := s.Clean(to)
	if err != nil {
		return "", err
	}
	if src == "" || dst == "" {
		return "", ErrInvalidPath
	}
	srcFull, err := s.abs(src)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(srcFull)
	if errors.Is(err, fs.ErrNotExist) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if info.Mode().IsRegular() && strings.EqualFold(path.Ext(src), Ext) && !strings.EqualFold(path.Ext(dst), Ext) {
		dst += Ext
	}
	if info.IsDir() && (dst == src || strings.HasPrefix(dst, src+"/")) {
		return "", fmt.Errorf("%w: cannot move a folder into itself", ErrInvalidPath)
	}
	dstFull, err := s.abs(dst)
	if err != nil {
		return "", err
	}
	// Allow pure case changes on case-insensitive filesystems.
	if _, err := os.Stat(dstFull); err == nil && !strings.EqualFold(src, dst) {
		return "", ErrExists
	}
	if err := os.MkdirAll(filepath.Dir(dstFull), 0o755); err != nil {
		return "", err
	}
	return dst, os.Rename(srcFull, dstFull)
}

// Delete removes a page or folder. Non-empty folders require recursive.
func (s *Store) Delete(p string, recursive bool) error {
	rel, err := s.Clean(p)
	if err != nil {
		return err
	}
	if rel == "" {
		return ErrInvalidPath
	}
	full, err := s.abs(rel)
	if err != nil {
		return err
	}
	info, err := os.Lstat(full)
	if errors.Is(err, fs.ErrNotExist) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		if recursive {
			return os.RemoveAll(full)
		}
		entries, err := os.ReadDir(full)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return ErrNotEmpty
		}
	}
	return os.Remove(full)
}

// SaveAsset stores an uploaded file in dir, choosing a unique name if one is
// taken. It returns the relative path of the stored file.
func (s *Store) SaveAsset(dir, name string, r io.Reader) (string, error) {
	rel, err := s.Clean(dir)
	if err != nil {
		return "", err
	}
	name = sanitizeName(name)
	if name == "" || s.Hidden(name) {
		return "", ErrInvalidPath
	}
	dirFull, err := s.abs(rel)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dirFull, 0o755); err != nil {
		return "", err
	}
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		f, err := os.OpenFile(filepath.Join(dirFull, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			name = fmt.Sprintf("%s-%d%s", stem, i+1, ext)
			continue
		}
		if err != nil {
			return "", err
		}
		_, err = io.Copy(f, r)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			os.Remove(f.Name())
			return "", err
		}
		return path.Join(rel, name), nil
	}
}

var unsafeName = regexp.MustCompile(`[^\p{L}\p{N}._ -]+`)

func sanitizeName(name string) string {
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	name = unsafeName.ReplaceAllString(name, "-")
	name = strings.Trim(name, " .-")
	if len(name) > 120 {
		ext := path.Ext(name)
		name = name[:120-len(ext)] + ext
	}
	return name
}

// Open returns a non-page file (image, attachment) for raw serving.
func (s *Store) Open(file string) (*os.File, fs.FileInfo, error) {
	rel, err := s.Clean(file)
	if err != nil {
		return nil, nil, err
	}
	full, err := s.abs(rel)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(full)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		f.Close()
		return nil, nil, ErrNotFound
	}
	return f, info, nil
}

// Tree returns the complete page tree. Folders are listed before pages, and
// both are sorted naturally by name. Folders holding only attachments (such
// as pasted images) are omitted; completely empty folders are kept so newly
// created sections are visible.
func (s *Store) Tree() (*Node, error) {
	root := &Node{Type: "dir", Name: "", Path: "", Title: ""}
	if idx := s.indexFile(""); idx != "" {
		root.File = idx
	}
	if _, err := s.fill(root, ""); err != nil {
		return nil, err
	}
	return root, nil
}

// fill adds the children of rel to node. It reports how many visible
// non-page files the folder contains.
func (s *Store) fill(node *Node, rel string) (attachments int, err error) {
	full, err := s.abs(rel)
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		name := e.Name()
		if s.Hidden(name) {
			continue
		}
		childRel := path.Join(rel, name)
		isDir := e.IsDir()
		if e.Type()&fs.ModeSymlink != 0 {
			target, err := s.abs(childRel)
			if err != nil {
				continue
			}
			info, err := os.Stat(target)
			if err != nil {
				continue
			}
			isDir = info.IsDir()
		}
		switch {
		case isDir:
			child := &Node{Type: "dir", Name: name, Path: childRel, Title: DisplayName(name)}
			if idx := s.indexFile(childRel); idx != "" {
				child.File = idx
				if t := s.cachedTitle(idx); t != "" {
					child.Title = t
				}
			}
			others, err := s.fill(child, childRel)
			if err != nil {
				return 0, err
			}
			if len(child.Children) == 0 && child.File == "" && others > 0 {
				attachments++
				continue
			}
			node.Children = append(node.Children, child)
		case strings.EqualFold(path.Ext(name), Ext):
			if IsIndex(name) {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			node.Children = append(node.Children, &Node{
				Type:     "page",
				Name:     strings.TrimSuffix(name, path.Ext(name)),
				Path:     strings.TrimSuffix(childRel, path.Ext(name)),
				File:     childRel,
				Title:    s.cachedTitle(childRel),
				Modified: info.ModTime(),
			})
		default:
			attachments++
		}
	}
	sort.SliceStable(node.Children, func(i, j int) bool {
		a, b := node.Children[i], node.Children[j]
		if a.Type != b.Type {
			return a.Type == "dir"
		}
		return naturalLess(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return attachments, nil
}

// cachedTitle reads a page title, reusing the previous result while the
// file's modification time is unchanged.
func (s *Store) cachedTitle(rel string) string {
	full, err := s.abs(rel)
	if err != nil {
		return DisplayName(path.Base(rel))
	}
	info, err := os.Stat(full)
	if err != nil {
		return DisplayName(path.Base(rel))
	}
	s.mu.Lock()
	c, ok := s.titles[rel]
	s.mu.Unlock()
	if ok && c.mod.Equal(info.ModTime()) {
		return c.title
	}
	f, err := os.Open(full)
	if err != nil {
		return DisplayName(path.Base(rel))
	}
	head := make([]byte, 8192)
	n, _ := io.ReadFull(f, head)
	f.Close()
	title := TitleOf(head[:n], rel)
	s.mu.Lock()
	s.titles[rel] = cachedTitle{mod: info.ModTime(), title: title}
	s.mu.Unlock()
	return title
}

var orderPrefix = regexp.MustCompile(`^\d+[-_. ]+`)

// DisplayName turns a file or folder name into a human-friendly label:
// "02-getting_started.md" becomes "Getting started".
func DisplayName(name string) string {
	if strings.EqualFold(path.Ext(name), Ext) {
		name = strings.TrimSuffix(name, path.Ext(name))
	}
	if stripped := orderPrefix.ReplaceAllString(name, ""); stripped != "" {
		name = stripped
	}
	name = strings.NewReplacer("-", " ", "_", " ").Replace(name)
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return "Untitled"
	}
	r := []rune(name)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// TitleOf extracts a page title from YAML front matter ("title:") or the first
// level-one heading, falling back to the file name.
func TitleOf(content []byte, file string) string {
	sc := bufio.NewScanner(bytes.NewReader(content))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	first := true
	inFront, inFence := false, false
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		trimmed := strings.TrimSpace(line)
		if first {
			first = false
			line = strings.TrimPrefix(line, "\uFEFF")
			if strings.TrimSpace(line) == "---" {
				inFront = true
				continue
			}
		}
		if inFront {
			if trimmed == "---" || trimmed == "..." {
				inFront = false
				continue
			}
			if k, v, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(k) == "title" {
				if v = strings.Trim(strings.TrimSpace(v), `"'`); v != "" {
					return v
				}
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if !inFence && strings.HasPrefix(line, "# ") {
			if t := strings.TrimSpace(strings.Trim(strings.TrimSpace(line[2:]), "#")); t != "" {
				return t
			}
		}
	}
	base := path.Base(file)
	if IsIndex(base) {
		dir := path.Dir(file)
		if dir == "." || dir == "/" {
			return "Home"
		}
		return DisplayName(path.Base(dir))
	}
	return DisplayName(base)
}

// Hash returns a short, stable content fingerprint.
func Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:12])
}

// naturalLess compares strings so that "page2" sorts before "page10".
func naturalLess(a, b string) bool {
	for a != "" && b != "" {
		ca, cb := a[0], b[0]
		if isDigit(ca) && isDigit(cb) {
			na, ra := leadingDigits(a)
			nb, rb := leadingDigits(b)
			ta, tb := strings.TrimLeft(na, "0"), strings.TrimLeft(nb, "0")
			if len(ta) != len(tb) {
				return len(ta) < len(tb)
			}
			if ta != tb {
				return ta < tb
			}
			a, b = ra, rb
			continue
		}
		if ca != cb {
			return ca < cb
		}
		a, b = a[1:], b[1:]
	}
	return len(a) < len(b)
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func leadingDigits(s string) (string, string) {
	i := 0
	for i < len(s) && isDigit(s[i]) {
		i++
	}
	return s[:i], s[i:]
}

// Count returns the number of pages in the wiki.
func (s *Store) Count() int {
	n := 0
	_ = s.Walk(func(string) error { n++; return nil })
	return n
}

// Walk calls fn with the relative path of every page file.
func (s *Store) Walk(fn func(file string) error) error {
	return filepath.WalkDir(s.root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == s.root {
				return err
			}
			return nil
		}
		if p == s.root {
			return nil
		}
		if s.Hidden(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(p), Ext) {
			return nil
		}
		rel, err := filepath.Rel(s.root, p)
		if err != nil {
			return nil
		}
		return fn(filepath.ToSlash(rel))
	})
}
