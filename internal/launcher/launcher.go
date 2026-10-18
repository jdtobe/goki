// Package launcher opens the wiki in a browser or a dedicated app window.
package launcher

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

// ErrNoWindow means no way to open a dedicated window was found.
var ErrNoWindow = errors.New("no app-window backend available")

// WindowOptions describes the app window.
type WindowOptions struct {
	URL    string
	Title  string
	Width  int
	Height int
}

// Browser opens url in the user's default browser without waiting.
func Browser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// Window opens a dedicated window and blocks until the user closes it.
// It prefers the native WebKit/WebView2 window compiled in with the
// "webview" build tag, then a chromeless Chromium-family browser window.
// It returns ErrNoWindow if neither is available.
func Window(opt WindowOptions) error {
	if opt.Width == 0 {
		opt.Width, opt.Height = 1280, 860
	}
	if NativeAvailable {
		return nativeWindow(opt)
	}
	return chromiumWindow(opt)
}

// Backend names the window implementation Window will use.
func Backend() string {
	if NativeAvailable {
		return "native"
	}
	if findChromium() != "" {
		return "chromium"
	}
	return ""
}

// chromiumWindow launches a Chromium-family browser in --app mode with a
// private profile, so it runs as its own process we can wait on.
func chromiumWindow(opt WindowOptions) error {
	bin := findChromium()
	if bin == "" {
		return ErrNoWindow
	}
	profile := filepath.Join(os.TempDir(), "goki-window")
	if dir, err := os.UserCacheDir(); err == nil {
		profile = filepath.Join(dir, "goki", "window")
	}
	cmd := exec.Command(bin,
		"--app="+opt.URL,
		"--user-data-dir="+profile,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-features=Translate",
		"--class=goki",
		"--window-size="+strconv.Itoa(opt.Width)+","+strconv.Itoa(opt.Height),
	)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Wait()
}

func findChromium() string {
	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		for _, app := range []string{"Google Chrome", "Microsoft Edge", "Brave Browser", "Chromium", "Vivaldi", "Google Chrome Canary"} {
			rel := filepath.Join(app+".app", "Contents", "MacOS", app)
			candidates = append(candidates, filepath.Join("/Applications", rel))
			if home, err := os.UserHomeDir(); err == nil {
				candidates = append(candidates, filepath.Join(home, "Applications", rel))
			}
		}
	case "windows":
		for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "LocalAppData"} {
			base := os.Getenv(env)
			if base == "" {
				continue
			}
			candidates = append(candidates,
				filepath.Join(base, `Google\Chrome\Application\chrome.exe`),
				filepath.Join(base, `Microsoft\Edge\Application\msedge.exe`),
				filepath.Join(base, `BraveSoftware\Brave-Browser\Application\brave.exe`),
			)
		}
	default:
		for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "microsoft-edge", "brave-browser", "vivaldi"} {
			if p, err := exec.LookPath(name); err == nil {
				candidates = append(candidates, p)
			}
		}
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}
