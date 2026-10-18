# Changelog

All notable changes to goki are documented here. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and goki follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

The first release of the rebuilt goki.

### Added

- Serve any folder of Markdown as a wiki with `goki [dir]`; folders are sections and `.md` files are pages, with `index.md` / `README.md` as folder pages.
- Browser-side rendering of GitHub-flavored Markdown with front matter, wiki links, callouts, KaTeX math, Mermaid diagrams, syntax highlighting, heading anchors and an "On this page" outline.
- Split-view editor with live preview, formatting bar, smart lists, image paste and drop uploads, draft recovery and conflict detection.
- Create, rename, move (including drag and drop) and delete pages and folders.
- Interactive task lists that save when ticked.
- ⌘K command palette with fuzzy page matching, full-text search and commands.
- Seven built-in themes — Daylight, Graphite, Glass, Nebula, Aurora, Dune and Ink — plus Automatic, an accent picker and custom CSS themes.
- Live reload when files change on disk.
- Dedicated app window with `--window`: native WebKit/WebView2/WebKitGTK when built with `-tags webview`, or a Chromium app window.
- Optional git auto-commit, read-only mode and HTTP basic authentication.
- Configuration through `goki.yaml`, a user config file, `GOKI_*` environment variables and flags; `goki init` and `goki themes` commands.
- Security hardening: localhost by default, DNS-rebinding and cross-site request protection, path confinement, sanitized rendering under a strict CSP and sandboxed raw files.
- Documentation (served by goki itself), CI, release automation, Dockerfile and dev container.

[Unreleased]: https://github.com/jdtobe/goki/commits/master
