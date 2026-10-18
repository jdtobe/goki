package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Event is pushed to browsers over Server-Sent Events.
type Event struct {
	Type  string   `json:"type"`
	Paths []string `json:"paths,omitempty"`
}

// Hub fans events out to connected browsers.
type Hub struct {
	mu   sync.Mutex
	subs map[chan Event]struct{}
}

// NewHub returns an empty Hub.
func NewHub() *Hub { return &Hub{subs: map[chan Event]struct{}{}} }

// Subscribe registers a listener. Call the returned function to unsubscribe.
func (h *Hub) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, 16)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}
}

// Publish delivers an event to every listener without blocking; slow
// listeners miss events rather than stalling the server.
func (h *Hub) Publish(e Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- e:
		default:
		}
	}
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")

	events, cancel := s.hub.Subscribe()
	defer cancel()
	fmt.Fprintf(w, "retry: 2000\nevent: hello\ndata: {\"version\":%q}\n\n", s.version)
	flusher.Flush()

	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case e := <-events:
			data, _ := json.Marshal(e)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", e.Type, data)
			flusher.Flush()
		}
	}
}

// Watch observes the wiki directory and publishes debounced change events
// until ctx is canceled. hidden reports names to skip.
func Watch(ctx context.Context, root string, hidden func(string) bool, hub *Hub) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	addTree := func(dir string) {
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}
			if p != root && hidden(d.Name()) {
				return filepath.SkipDir
			}
			_ = w.Add(p)
			return nil
		})
	}
	addTree(root)

	go func() {
		defer w.Close()
		pending := map[string]struct{}{}
		var timer <-chan time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				rel, err := filepath.Rel(root, ev.Name)
				if err != nil || skipPath(rel, hidden) {
					continue
				}
				if ev.Has(fsnotify.Create) {
					if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
						addTree(ev.Name)
					}
				}
				pending[filepath.ToSlash(rel)] = struct{}{}
				if timer == nil {
					timer = time.After(150 * time.Millisecond)
				}
			case <-timer:
				paths := make([]string, 0, len(pending))
				for p := range pending {
					paths = append(paths, p)
				}
				sort.Strings(paths)
				clear(pending)
				timer = nil
				hub.Publish(Event{Type: "change", Paths: paths})
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	return nil
}

func skipPath(rel string, hidden func(string) bool) bool {
	for _, seg := range strings.Split(filepath.ToSlash(rel), "/") {
		if seg == ".." || hidden(seg) {
			return true
		}
	}
	return false
}
