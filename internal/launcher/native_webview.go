//go:build webview

package launcher

import (
	"runtime"

	webview "github.com/webview/webview_go"
)

// NativeAvailable reports whether a native webview window is compiled in.
const NativeAvailable = true

func init() {
	// Cocoa and GTK require the UI to run on the main OS thread.
	runtime.LockOSThread()
}

func nativeWindow(opt WindowOptions) error {
	w := webview.New(false)
	defer w.Destroy()
	platformSetup(opt.Title)
	w.SetTitle(opt.Title)
	w.SetSize(opt.Width, opt.Height, webview.HintNone)
	w.Navigate(opt.URL)
	w.Run()
	return nil
}
