// Package config loads goki settings from defaults, a user-wide config file,
// a per-wiki config file and environment variables, in that order. Command
// line flags are applied on top by the caller.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// FileNames are the per-wiki config file names, checked in order.
var FileNames = []string{"goki.yaml", "goki.yml", ".goki.yaml", ".goki.yml"}

// Open modes control what happens after the server starts.
const (
	OpenBrowser = "browser" // open the default web browser
	OpenWindow  = "window"  // open a dedicated app window
	OpenNone    = "none"    // just print the URL
)

// Config is the complete set of goki settings.
type Config struct {
	// Title is shown in the sidebar and browser tab. Defaults to the folder name.
	Title string `yaml:"title"`
	// Theme is the default theme id, or "auto" to follow the OS appearance.
	Theme string `yaml:"theme"`
	// Accent overrides the theme accent color (any CSS color).
	Accent string `yaml:"accent"`
	// Host is the interface to listen on. Use 0.0.0.0 to share on a network.
	Host string `yaml:"host"`
	// Port to listen on. 0 picks a free port. If the configured port is
	// taken, goki falls back to a free one unless the port was set explicitly.
	Port int `yaml:"port"`
	// Open is one of "browser", "window" or "none".
	Open string `yaml:"open"`
	// ReadOnly disables all editing.
	ReadOnly bool `yaml:"read_only"`
	// Ignore lists glob patterns of file and folder names to hide.
	Ignore []string `yaml:"ignore"`
	// Watch enables live reload when files change on disk.
	Watch bool `yaml:"watch"`
	// AssetsDir is where pasted and dropped files are stored, relative to the
	// page being edited. Empty stores them next to the page.
	AssetsDir string `yaml:"assets_dir"`
	// MaxUploadMB limits the size of a single uploaded file.
	MaxUploadMB int `yaml:"max_upload_mb"`

	Git    Git    `yaml:"git"`
	Auth   Auth   `yaml:"auth"`
	Editor Editor `yaml:"editor"`

	// Source records which files were loaded, for diagnostics.
	Source []string `yaml:"-"`
}

// Git controls automatic commits.
type Git struct {
	// AutoCommit commits every change made through goki.
	AutoCommit bool `yaml:"auto_commit"`
	// AuthorName and AuthorEmail override git's configured identity.
	AuthorName  string `yaml:"author_name"`
	AuthorEmail string `yaml:"author_email"`
}

// Auth enables HTTP basic authentication, recommended whenever Host is not
// a loopback address.
type Auth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Enabled reports whether authentication is configured.
func (a Auth) Enabled() bool { return a.Username != "" || a.Password != "" }

// Editor holds editor preferences passed to the browser.
type Editor struct {
	Spellcheck bool `yaml:"spellcheck" json:"spellcheck"`
	LineWrap   bool `yaml:"line_wrap" json:"lineWrap"`
	TabSize    int  `yaml:"tab_size" json:"tabSize"`
	// Layout is the default edit layout: "split", "write" or "preview".
	Layout string `yaml:"layout" json:"layout"`
}

// Default returns the built-in configuration.
func Default() Config {
	return Config{
		Theme:       "auto",
		Host:        "127.0.0.1",
		Port:        4242,
		Open:        OpenBrowser,
		Watch:       true,
		AssetsDir:   "assets",
		MaxUploadMB: 32,
		Ignore:      []string{"node_modules", "vendor", "__pycache__", "*.tmp", "~*"},
		Editor: Editor{
			Spellcheck: true,
			LineWrap:   true,
			TabSize:    2,
			Layout:     "split",
		},
	}
}

// UserFile returns the path of the user-wide config file.
// $GOKI_CONFIG wins, then $XDG_CONFIG_HOME/goki/config.yaml, then
// ~/.config/goki/config.yaml, then the OS-specific config directory.
func UserFile() string {
	if p := os.Getenv("GOKI_CONFIG"); p != "" {
		return p
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "goki", "config.yaml")
	}
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".config", "goki", "config.yaml")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "goki", "config.yaml")
	}
	return ""
}

// UserDir returns the directory holding the user-wide config and themes.
func UserDir() string {
	if f := UserFile(); f != "" {
		return filepath.Dir(f)
	}
	return ""
}

// Load builds the configuration for the wiki rooted at dir. If explicit is
// non-empty it is used instead of the per-wiki config file and must exist.
func Load(dir, explicit string) (Config, error) {
	cfg := Default()
	if uf := UserFile(); uf != "" {
		if err := cfg.merge(uf, false); err != nil {
			return cfg, err
		}
	}
	if explicit != "" {
		if err := cfg.merge(explicit, true); err != nil {
			return cfg, err
		}
	} else {
		for _, name := range FileNames {
			p := filepath.Join(dir, name)
			if _, err := os.Stat(p); err == nil {
				if err := cfg.merge(p, true); err != nil {
					return cfg, err
				}
				break
			}
		}
	}
	if err := cfg.applyEnv(os.LookupEnv); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c *Config) merge(file string, required bool) error {
	data, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) && !required {
		return nil
	}
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	// Decoding into the existing struct keeps values the file leaves unset.
	if err := yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("config: %s: %w", file, err)
	}
	c.Source = append(c.Source, file)
	return nil
}

// applyEnv reads GOKI_* overrides. lookup is os.LookupEnv in production.
func (c *Config) applyEnv(lookup func(string) (string, bool)) error {
	str := map[string]*string{
		"GOKI_TITLE":         &c.Title,
		"GOKI_THEME":         &c.Theme,
		"GOKI_ACCENT":        &c.Accent,
		"GOKI_HOST":          &c.Host,
		"GOKI_OPEN":          &c.Open,
		"GOKI_AUTH_USERNAME": &c.Auth.Username,
		"GOKI_AUTH_PASSWORD": &c.Auth.Password,
	}
	for k, p := range str {
		if v, ok := lookup(k); ok {
			*p = v
		}
	}
	if v, ok := lookup("GOKI_PORT"); ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GOKI_PORT: %w", err)
		}
		c.Port = n
	}
	bools := map[string]*bool{
		"GOKI_READ_ONLY":       &c.ReadOnly,
		"GOKI_WATCH":           &c.Watch,
		"GOKI_GIT_AUTO_COMMIT": &c.Git.AutoCommit,
	}
	for k, p := range bools {
		if v, ok := lookup(k); ok {
			b, err := strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("%s: %w", k, err)
			}
			*p = b
		}
	}
	return nil
}

// Validate checks the configuration for mistakes.
func (c *Config) Validate() error {
	var errs []error
	if c.Port < 0 || c.Port > 65535 {
		errs = append(errs, fmt.Errorf("port %d is out of range", c.Port))
	}
	switch c.Open {
	case OpenBrowser, OpenWindow, OpenNone:
	default:
		errs = append(errs, fmt.Errorf("open must be %q, %q or %q, not %q", OpenBrowser, OpenWindow, OpenNone, c.Open))
	}
	switch c.Editor.Layout {
	case "split", "write", "preview":
	default:
		errs = append(errs, fmt.Errorf("editor.layout must be split, write or preview, not %q", c.Editor.Layout))
	}
	if c.MaxUploadMB <= 0 {
		errs = append(errs, errors.New("max_upload_mb must be positive"))
	}
	if strings.ContainsAny(c.AssetsDir, `\`) || strings.Contains(c.AssetsDir, "..") || filepath.IsAbs(c.AssetsDir) {
		errs = append(errs, fmt.Errorf("assets_dir must be a relative path inside the wiki, not %q", c.AssetsDir))
	}
	if c.Editor.TabSize < 1 || c.Editor.TabSize > 8 {
		c.Editor.TabSize = 2
	}
	return errors.Join(errs...)
}

// Sample is the commented config file written by "goki init".
const Sample = `# goki configuration — https://github.com/jdtobe/goki/blob/master/docs/configuration.md
# Every setting is optional. Command-line flags override this file.

# Shown in the sidebar and the browser tab. Defaults to the folder name.
# title: My Wiki

# auto follows the system appearance (daylight / graphite).
# Built in: daylight, graphite, glass, nebula, aurora, dune, ink.
# Custom themes live in .goki/themes/<id>.css.
theme: auto

# Override the theme's accent color with any CSS color.
# accent: "#0a84ff"

# Where to listen. Use 0.0.0.0 to share on your network (set auth below!).
host: 127.0.0.1
port: 4242

# What to do on start: browser, window or none.
open: browser

# Disable all editing.
read_only: false

# Reload pages automatically when files change on disk.
watch: true

# Hide matching file and folder names (dot-files are always hidden).
ignore:
  - node_modules
  - vendor

# Pasted and dropped images are saved here, relative to the page.
assets_dir: assets
max_upload_mb: 32

editor:
  spellcheck: true
  line_wrap: true
  tab_size: 2
  layout: split   # split, write or preview

git:
  # Commit every change made through goki (requires git on your PATH).
  auto_commit: false
  # author_name: goki
  # author_email: goki@localhost

# auth:
#   username: me
#   password: change-me
`
