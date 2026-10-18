package server

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Theme describes a user-supplied theme.
type Theme struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Scheme string `json:"scheme"` // "light" or "dark"
}

// ThemeSet discovers custom themes. Each theme is a CSS file named <id>.css
// in <wiki>/.goki/themes or <user config dir>/themes. Files in the wiki take
// precedence. Themes are re-read on every request so edits apply on reload.
type ThemeSet struct {
	dirs []string
}

var themeID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,40}$`)

// LoadThemes returns a ThemeSet that searches the wiki and user directories.
func LoadThemes(wikiRoot, userDir string) *ThemeSet {
	ts := &ThemeSet{dirs: []string{filepath.Join(wikiRoot, ".goki", "themes")}}
	if userDir != "" {
		ts.dirs = append(ts.dirs, filepath.Join(userDir, "themes"))
	}
	return ts
}

// Custom lists the available custom themes.
func (ts *ThemeSet) Custom() []Theme {
	seen := map[string]bool{}
	var out []Theme
	for _, dir := range ts.dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			id, ok := strings.CutSuffix(e.Name(), ".css")
			if !ok || e.IsDir() || !themeID.MatchString(id) || seen[id] {
				continue
			}
			seen[id] = true
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			out = append(out, parseTheme(id, data))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

// CSS returns the stylesheet for a custom theme.
func (ts *ThemeSet) CSS(id string) ([]byte, bool) {
	if !themeID.MatchString(id) {
		return nil, false
	}
	for _, dir := range ts.dirs {
		if data, err := os.ReadFile(filepath.Join(dir, id+".css")); err == nil {
			return data, true
		}
	}
	return nil, false
}

// parseTheme reads optional metadata from the leading comment:
//
//	/* @name Solar Flare
//	   @scheme dark */
func parseTheme(id string, css []byte) Theme {
	t := Theme{ID: id, Label: strings.ToUpper(id[:1]) + strings.ReplaceAll(id[1:], "-", " "), Scheme: "light"}
	end := bytes.Index(css, []byte("*/"))
	if end < 0 {
		return t
	}
	head := css[:end]
	sc := bufio.NewScanner(bytes.NewReader(head))
	for sc.Scan() {
		line := strings.Trim(strings.TrimSpace(sc.Text()), "/* ")
		key, val, _ := strings.Cut(line, " ")
		val = strings.TrimSpace(val)
		switch key {
		case "@name":
			if val != "" {
				t.Label = val
			}
		case "@scheme":
			if val == "dark" || val == "light" {
				t.Scheme = val
			}
		}
	}
	return t
}
