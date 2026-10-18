// Package web holds the browser application, embedded into the goki binary.
// There is no build step: the files here are served exactly as written.
package web

import "embed"

// FS contains index.html and every static asset.
//
//go:embed index.html css js vendor fonts img
var FS embed.FS
