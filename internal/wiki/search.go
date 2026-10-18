package wiki

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// Hit is a single search result.
type Hit struct {
	Path    string  `json:"path"`
	File    string  `json:"file"`
	Title   string  `json:"title"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

// maxSearchFile bounds how much of a single file is scanned.
const maxSearchFile = 2 << 20

// Search performs a case-insensitive full-text search across all pages.
// Every whitespace-separated term must appear in the title, path or body.
// Results are ranked with title and path matches above body matches.
func (s *Store) Search(query string, limit int) ([]Hit, error) {
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	var hits []Hit
	err := s.Walk(func(file string) error {
		full := filepath.Join(s.root, filepath.FromSlash(file))
		info, err := os.Stat(full)
		if err != nil || info.Size() > maxSearchFile {
			return nil
		}
		data, err := os.ReadFile(full)
		if err != nil {
			return nil
		}
		title := TitleOf(data, file)
		data = data[frontMatterEnd(data):]
		lowTitle := strings.ToLower(title)
		lowPath := strings.ToLower(file)
		body := string(data)
		lowBody := strings.ToLower(body)

		score := 0.0
		first := -1
		for _, t := range terms {
			inTitle := strings.Contains(lowTitle, t)
			inPath := strings.Contains(lowPath, t)
			idx := strings.Index(lowBody, t)
			if !inTitle && !inPath && idx < 0 {
				return nil
			}
			if inTitle {
				score += 10
				if strings.HasPrefix(lowTitle, t) {
					score += 5
				}
			}
			if inPath {
				score += 3
			}
			if idx >= 0 {
				score += 1 + min(float64(strings.Count(lowBody, t)), 20)*0.25
				if first < 0 || idx < first {
					first = idx
				}
			}
		}
		if lowTitle == strings.Join(terms, " ") {
			score += 20
		}
		hits = append(hits, Hit{
			Path:    RouteFor(file),
			File:    file,
			Title:   title,
			Snippet: snippet(body, lowBody, first),
			Score:   score,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].Path < hits[j].Path
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

// snippet returns roughly 160 characters of context around a match.
// lower must be strings.ToLower(body); offsets are mapped proportionally
// because lower-casing can change byte lengths for some scripts.
func snippet(body, lower string, at int) string {
	if at < 0 {
		at = 0
	}
	if len(lower) != len(body) && len(lower) > 0 {
		at = at * len(body) / len(lower)
	}
	start := max(0, at-60)
	end := min(len(body), at+100)
	for start > 0 && !utf8.RuneStart(body[start]) {
		start--
	}
	for end < len(body) && !utf8.RuneStart(body[end]) {
		end++
	}
	out := strings.Join(strings.Fields(body[start:end]), " ")
	if start > 0 {
		out = "…" + out
	}
	if end < len(body) {
		out += "…"
	}
	return out
}

// frontMatterEnd returns the offset just past a leading YAML front matter
// block, or 0 if there is none.
func frontMatterEnd(data []byte) int {
	s := strings.TrimPrefix(string(data), "\uFEFF")
	skip := len(data) - len(s)
	if !strings.HasPrefix(s, "---\n") && !strings.HasPrefix(s, "---\r\n") {
		return 0
	}
	rest := s[strings.Index(s, "\n")+1:]
	for off := len(s) - len(rest); rest != ""; {
		line, after, _ := strings.Cut(rest, "\n")
		off += len(line) + 1
		if t := strings.TrimSpace(line); t == "---" || t == "..." {
			return skip + min(off, len(s))
		}
		rest = after
	}
	return 0
}
