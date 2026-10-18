---
title: App Window
description: Run goki in a window of its own, like a native app.
---

# App Window

Prefer an app to a browser tab? Start goki with `--window`:

```sh
goki ~/notes --window
```

goki opens a dedicated window with no address bar or tabs. Closing the window stops goki. To make it the default, set `open: window` in your [[Configuration]].

## How it works

goki picks the best window it can create:

1. **Native window** — when goki is built with the `webview` tag, it uses the operating system's own web engine: WebKit on macOS, WebView2 on Windows and WebKitGTK on Linux. On macOS it gets a proper menu bar, so <kbd>⌘</kbd><kbd>C</kbd>, <kbd>⌘</kbd><kbd>V</kbd>, <kbd>⌘</kbd><kbd>Z</kbd> and <kbd>⌘</kbd><kbd>Q</kbd> behave as you expect, and the goki icon in the Dock.
2. **Chromium app window** — otherwise goki looks for Google Chrome, Microsoft Edge, Brave, Chromium or Vivaldi and opens it in *app mode* with a private profile, so it runs separately from your normal browsing.
3. **Your browser** — if neither is available, goki says so and opens a normal browser tab.

## Building with the native window

The native window needs cgo, so it's an opt-in build:

```sh
CGO_ENABLED=1 go install -tags webview github.com/jdtobe/goki/cmd/goki@latest
# or, from a clone
make install-desktop
```

Requirements per platform:

| Platform | Needs |
| --- | --- |
| macOS | Xcode Command Line Tools (`xcode-select --install`) |
| Windows | A C compiler such as [MSYS2](https://www.msys2.org/) MinGW-w64; the WebView2 runtime ships with Windows 11 |
| Linux | `libgtk-3-dev` and `libwebkit2gtk-4.1-dev` (or your distribution's equivalents) |

The portable build without the `webview` tag needs none of these and cross-compiles anywhere.
