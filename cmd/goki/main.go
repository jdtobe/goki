// Command goki serves a folder of Markdown files as a beautiful wiki.
//
//	cd docs && goki .
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/jdtobe/goki/internal/config"
	"github.com/jdtobe/goki/internal/gitsync"
	"github.com/jdtobe/goki/internal/launcher"
	"github.com/jdtobe/goki/internal/server"
	"github.com/jdtobe/goki/internal/wiki"
)

// version is set at build time with -ldflags "-X main.version=v1.2.3".
var version = ""

// builtinThemes must match the themes defined in web/css/themes.css.
var builtinThemes = []struct{ id, label, scheme string }{
	{"auto", "Automatic", "follows system"},
	{"daylight", "Daylight", "light"},
	{"graphite", "Graphite", "dark"},
	{"glass", "Glass", "light"},
	{"nebula", "Nebula", "dark"},
	{"aurora", "Aurora", "dark"},
	{"dune", "Dune", "light"},
	{"ink", "Ink", "light"},
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", paint(os.Stderr, "31;1", "error:"), err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) > 0 {
		switch args[0] {
		case "version":
			fmt.Fprintf(stdout, "goki %s\n", appVersion())
			return nil
		case "init":
			return runInit(args[1:], stdout)
		case "themes":
			return runThemes(args[1:], stdout)
		case "help":
			args = []string{"--help"}
		}
	}
	return runServe(args, stdout, stderr)
}

func newFlagSet(name, usage string, stdout io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stdout)
	fs.SortFlags = false
	fs.Usage = func() {
		fmt.Fprint(stdout, usage)
		fmt.Fprintln(stdout, "\nFlags:")
		fmt.Fprint(stdout, fs.FlagUsages())
	}
	return fs
}

const serveUsage = `goki — your folder of Markdown, as a beautiful wiki.

Usage:
  goki [flags] [dir]       Serve dir (default ".") as a wiki
  goki init [dir]          Create goki.yaml and a welcome page in dir
  goki themes              List available themes
  goki version             Print the version

Examples:
  goki                     Serve the current folder and open a browser
  goki ~/notes --window    Open in a dedicated app window
  goki docs -p 8080 -n     Serve on port 8080 without opening anything
  goki . --read-only --host 0.0.0.0 --theme aurora
`

func runServe(args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("goki", serveUsage, stdout)
	var (
		port     = fs.IntP("port", "p", 0, "port to listen on (default 4242, falling back to a free port)")
		host     = fs.String("host", "", "interface to listen on (default 127.0.0.1; 0.0.0.0 shares on your network)")
		open     = fs.StringP("open", "o", "", "on start, open a \"browser\", an app \"window\" or \"none\"")
		window   = fs.BoolP("window", "w", false, "open in a dedicated app window (same as --open window)")
		noOpen   = fs.BoolP("no-open", "n", false, "don't open anything (same as --open none)")
		theme    = fs.StringP("theme", "t", "", "default theme (see \"goki themes\")")
		title    = fs.String("title", "", "wiki title (default: folder name)")
		readOnly = fs.BoolP("read-only", "r", false, "disable editing")
		gitFlag  = fs.Bool("git", false, "commit every change to git")
		cfgFile  = fs.StringP("config", "c", "", "config file (default: <dir>/goki.yaml)")
		noWatch  = fs.Bool("no-watch", false, "don't live-reload when files change on disk")
		verbose  = fs.BoolP("verbose", "v", false, "log every request")
		showVer  = fs.Bool("version", false, "print the version and exit")
	)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *showVer {
		fmt.Fprintf(stdout, "goki %s\n", appVersion())
		return nil
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("expected one directory, got %d arguments", fs.NArg())
	}
	dir := "."
	if fs.NArg() == 1 {
		dir = fs.Arg(0)
	}

	cfg, err := config.Load(dir, *cfgFile)
	if err != nil {
		return err
	}
	explicitPort := false
	if fs.Changed("port") {
		cfg.Port, explicitPort = *port, true
	}
	set := func(name string, dst *string, v string) {
		if fs.Changed(name) {
			*dst = v
		}
	}
	set("host", &cfg.Host, *host)
	set("open", &cfg.Open, *open)
	set("theme", &cfg.Theme, *theme)
	set("title", &cfg.Title, *title)
	if *window {
		cfg.Open = config.OpenWindow
	}
	if *noOpen {
		cfg.Open = config.OpenNone
	}
	if *readOnly {
		cfg.ReadOnly = true
	}
	if *gitFlag {
		cfg.Git.AutoCommit = true
	}
	if *noWatch {
		cfg.Watch = false
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	store, err := wiki.New(dir, cfg.Ignore)
	if err != nil {
		return err
	}

	logLevel := slog.LevelWarn
	if *verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: logLevel}))

	var committer *gitsync.Committer
	gitNote := "off"
	if cfg.Git.AutoCommit && !cfg.ReadOnly {
		c, initialized, err := gitsync.New(store.Root(), cfg.Git.AuthorName, cfg.Git.AuthorEmail)
		if err != nil {
			return err
		}
		committer = c
		gitNote = "auto-commit"
		if initialized {
			gitNote = "auto-commit (initialized a new repository)"
		}
	}

	ln, err := listen(cfg.Host, cfg.Port, explicitPort)
	if err != nil {
		return err
	}
	loopback := isLoopback(cfg.Host)

	srv, err := server.New(server.Options{
		Store:    store,
		Config:   cfg,
		Git:      committer,
		Version:  appVersion(),
		Logger:   logger,
		Loopback: loopback,
	})
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.Watch {
		if err := server.Watch(ctx, store.Root(), store.Hidden, srv.Hub()); err != nil {
			logger.Warn("live reload disabled", "err", err)
		}
	}

	var handler http.Handler = srv
	if *verbose {
		handler = logRequests(logger, srv)
	}
	// Request contexts derive from baseCtx so that canceling it ends
	// long-lived live-reload streams immediately on shutdown.
	baseCtx, cancelRequests := context.WithCancel(context.Background())
	defer cancelRequests()
	httpSrv := &http.Server{
		Handler:           handler,
		BaseContext:       func(net.Listener) context.Context { return baseCtx },
		ReadHeaderTimeout: 10 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- httpSrv.Serve(ln) }()

	boundPort := ln.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://localhost:%d/", boundPort)
	if !loopback && cfg.Host != "0.0.0.0" && cfg.Host != "::" && cfg.Host != "" {
		url = fmt.Sprintf("http://%s/", net.JoinHostPort(cfg.Host, strconv.Itoa(boundPort)))
	}

	mode := cfg.Open
	if mode == config.OpenWindow && launcher.Backend() == "" {
		fmt.Fprintln(stderr, paint(stderr, "33", "No app-window backend found (install Chrome/Edge or build with -tags webview); opening your browser instead."))
		mode = config.OpenBrowser
	}
	banner(stdout, bannerInfo{
		url:      url,
		cfg:      cfg,
		store:    store,
		port:     boundPort,
		gitNote:  gitNote,
		mode:     mode,
		loopback: loopback,
	})

	shutdown := func() {
		cancelRequests()
		sctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(sctx)
	}

	switch mode {
	case config.OpenWindow:
		// The window must own the main thread (Cocoa requires it), so the
		// signal handler shuts down from a goroutine instead.
		go func() {
			select {
			case <-ctx.Done():
				shutdown()
				os.Exit(0)
			case err := <-serveErr:
				fmt.Fprintf(stderr, "error: %v\n", err)
				os.Exit(1)
			}
		}()
		err := launcher.Window(launcher.WindowOptions{URL: url, Title: srvTitle(cfg, store)})
		shutdown()
		// A browser that exits non-zero when its window closes is not an error.
		var exitErr *exec.ExitError
		if err != nil && !errors.As(err, &exitErr) {
			return err
		}
		return nil
	case config.OpenBrowser:
		if err := launcher.Browser(url); err != nil {
			logger.Warn("could not open a browser", "err", err)
		}
	}

	select {
	case <-ctx.Done():
		fmt.Fprintln(stdout, paint(stdout, "2", "\n  Stopping…"))
		shutdown()
		return nil
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func srvTitle(cfg config.Config, store *wiki.Store) string {
	if cfg.Title != "" {
		return cfg.Title
	}
	return wiki.DisplayName(filepath.Base(store.Root()))
}

// listen binds host:port. Unless the port was requested explicitly, a busy
// port falls back to any free one so "goki ." always just works.
func listen(host string, port int, explicit bool) (net.Listener, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	ln, err := net.Listen("tcp", addr)
	if err == nil || explicit {
		if err != nil {
			return nil, fmt.Errorf("port %d is unavailable: %w", port, err)
		}
		return ln, nil
	}
	return net.Listen("tcp", net.JoinHostPort(host, "0"))
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func logRequests(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		log.Debug("request", "method", r.Method, "path", r.URL.RequestURI(), "status", rec.status, "duration", time.Since(start).Round(time.Microsecond))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func appVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

const welcomePage = `# Welcome

This is your new wiki. Every folder is a section and every ` + "`.md`" + ` file is a page.

> [!TIP]
> Press **⌘K** (or **Ctrl K**) to search, **E** to edit this page and **N** to create a new one.

## Getting started

- [ ] Edit this page
- [ ] Create your first page
- [ ] Try another theme from the sidebar

## Writing

goki speaks GitHub-flavored Markdown, plus a few extras:

| Feature      | Syntax                          |
|--------------|---------------------------------|
| Wiki links   | ` + "`[[Another Page]]`" + `              |
| Callouts     | ` + "`> [!NOTE]`" + `                     |
| Math         | ` + "`$e^{i\\pi} + 1 = 0$`" + `            |
| Diagrams     | fenced ` + "`mermaid`" + ` code blocks     |

Images can be pasted or dropped straight into the editor.
`

func runInit(args []string, stdout io.Writer) error {
	fs := newFlagSet("goki init", "Usage:\n  goki init [dir]\n\nCreates goki.yaml and index.md in dir (default \".\") if they don't exist.\n", stdout)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	wrote := false
	write := func(name, content string, skipIf ...string) error {
		for _, other := range append(skipIf, name) {
			if _, err := os.Stat(filepath.Join(dir, other)); err == nil {
				fmt.Fprintf(stdout, "  %s %s already exists\n", paint(stdout, "2", "•"), other)
				return nil
			}
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return err
		}
		wrote = true
		fmt.Fprintf(stdout, "  %s created %s\n", paint(stdout, "32", "✓"), name)
		return nil
	}
	if err := write("goki.yaml", config.Sample, config.FileNames...); err != nil {
		return err
	}
	if err := write("index.md", welcomePage, "README.md", "readme.md"); err != nil {
		return err
	}
	if wrote {
		target := "goki"
		if dir != "." {
			target = "goki " + dir
		}
		fmt.Fprintf(stdout, "\n  Run %s to start.\n", paint(stdout, "1", target))
	}
	return nil
}

func runThemes(_ []string, stdout io.Writer) error {
	fmt.Fprintln(stdout, paint(stdout, "1", "Built-in themes"))
	for _, t := range builtinThemes {
		fmt.Fprintf(stdout, "  %-10s %-10s %s\n", t.id, t.label, paint(stdout, "2", t.scheme))
	}
	dir, _ := os.Getwd()
	custom := server.LoadThemes(dir, config.UserDir()).Custom()
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, paint(stdout, "1", "Custom themes"))
	if len(custom) == 0 {
		fmt.Fprintf(stdout, "  %s\n", paint(stdout, "2", "none — add <id>.css to .goki/themes/ or "+filepath.Join(config.UserDir(), "themes")))
	}
	for _, t := range custom {
		fmt.Fprintf(stdout, "  %-10s %-10s %s\n", t.ID, t.Label, paint(stdout, "2", t.Scheme))
	}
	return nil
}
