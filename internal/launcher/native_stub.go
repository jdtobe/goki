//go:build !webview

package launcher

// NativeAvailable reports whether a native webview window is compiled in.
// Build with -tags webview (requires cgo) to enable it.
const NativeAvailable = false

func nativeWindow(WindowOptions) error { return ErrNoWindow }
