package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/jdtobe/goki/internal/config"
	"github.com/jdtobe/goki/internal/wiki"
)

type bannerInfo struct {
	url      string
	cfg      config.Config
	store    *wiki.Store
	port     int
	gitNote  string
	mode     string
	loopback bool
}

// banner prints the startup summary.
func banner(w io.Writer, b bannerInfo) {
	dim := func(s string) string { return paint(w, "2", s) }
	label := func(s string) string { return paint(w, "2", fmt.Sprintf("%-10s", s)) }
	arrow := paint(w, "36", "➜")

	fmt.Fprintf(w, "\n  %s %s %s\n\n", paint(w, "36;1", "◆"), paint(w, "1", "goki"), dim(appVersion()))
	fmt.Fprintf(w, "  %s  %s %s\n", arrow, label("Local:"), paint(w, "36;1;4", b.url))
	if !b.loopback {
		for _, ip := range lanAddresses() {
			fmt.Fprintf(w, "  %s  %s %s\n", arrow, label("Network:"), paint(w, "36", fmt.Sprintf("http://%s/", net.JoinHostPort(ip, fmt.Sprint(b.port)))))
		}
	}
	pages := b.store.Count()
	noun := "pages"
	if pages == 1 {
		noun = "page"
	}
	fmt.Fprintf(w, "  %s  %s %s %s\n", arrow, label("Wiki:"), tildePath(b.store.Root()), dim(fmt.Sprintf("(%d %s)", pages, noun)))

	editing := "read & write"
	if b.cfg.ReadOnly {
		editing = "read-only"
	}
	fmt.Fprintf(w, "  %s  %s %s %s %s %s %s\n", arrow, label("Mode:"), editing,
		dim("· theme"), b.cfg.Theme, dim("· git"), b.gitNote)
	for _, src := range b.cfg.Source {
		fmt.Fprintf(w, "  %s  %s %s\n", arrow, label("Config:"), tildePath(src))
	}

	if !b.loopback && !b.cfg.Auth.Enabled() && !b.cfg.ReadOnly {
		fmt.Fprintf(w, "\n  %s %s\n", paint(w, "33;1", "!"), paint(w, "33", "Anyone on your network can edit this wiki. Set auth in goki.yaml or use --read-only."))
	}
	hint := "Press Ctrl+C to stop."
	if b.mode == config.OpenWindow {
		hint = "Close the window or press Ctrl+C to stop."
	}
	fmt.Fprintf(w, "\n  %s\n\n", dim(hint))
}

func tildePath(p string) string {
	if home, err := os.UserHomeDir(); err == nil {
		if rel, err := filepath.Rel(home, p); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.Join("~", rel)
		}
	}
	return p
}

func lanAddresses() []string {
	var out []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	for _, a := range addrs {
		ipn, ok := a.(*net.IPNet)
		if !ok || ipn.IP.IsLoopback() || ipn.IP.To4() == nil || ipn.IP.IsLinkLocalUnicast() {
			continue
		}
		out = append(out, ipn.IP.String())
	}
	return out
}

// paint wraps s in an ANSI SGR sequence when w is a color-capable terminal.
// It honors NO_COLOR (https://no-color.org) and FORCE_COLOR.
func paint(w io.Writer, code, s string) string {
	if !colorEnabled(w) {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func colorEnabled(w io.Writer) bool {
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok || os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
