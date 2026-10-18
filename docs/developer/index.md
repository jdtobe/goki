---
title: Developer Guide
description: Build, test and contribute to goki.
tags: [contributing]
---

# Developer Guide

goki is a Go program with an embedded, build-step-free web front end. If you can run `go build`, you can work on every part of it.

## Requirements

- Go 1.24 or newer
- [golangci-lint](https://golangci-lint.run) v2 for linting
- Optional: a C toolchain to build the native [[App Window]]
- Optional: [GoReleaser](https://goreleaser.com) to produce release archives

A [dev container](https://containers.dev) definition in `.devcontainer/` sets all of this up for VS Code and GitHub Codespaces.

## Everyday commands

```sh
make run        # build and serve ./docs on http://localhost:4242
make test       # go test -race ./...
make lint       # golangci-lint run
make build      # portable binary in ./bin
make desktop    # binary with the native window (cgo)
make snapshot   # local release archives via GoReleaser
make help       # everything else
```

`make run` sets `GOKI_WEB_DIR=web`, which makes goki serve the front end from disk instead of the copy embedded in the binary. Edit any HTML, CSS or JavaScript file and simply reload the browser — no bundler, no watcher, no rebuild.

## Project layout

```text
cmd/goki/            CLI entry point, flags and startup banner
internal/config/     defaults, config files, env vars and validation
internal/wiki/       the filesystem store: paths, tree, read/write, search
internal/server/     HTTP API, security middleware, live events, themes
internal/gitsync/    optional auto-commit via the git CLI
internal/launcher/   browser and app-window launching (native via build tag)
web/                 the embedded browser app (HTML, CSS, JS, vendored libs)
docs/                this documentation — itself a goki wiki
examples/            sample custom theme
```

## Testing

`make test` runs the suite with the race detector. It covers:

- **Store:** path cleaning, symlink escapes, route resolution, conflict detection, move and delete rules, tree building and natural sorting, titles, search ranking and snippets, and uploads.
- **Configuration:** layering, environment variables, validation, and the sample `goki.yaml`.
- **Server:** the full page lifecycle over HTTP, folders, search, uploads and size limits, and custom themes.
- **Security:** missing `X-Goki` headers, cross-origin writes, DNS rebinding, traversal, read-only mode, basic auth, raw-file sandboxing and static-asset caching.
- **git and CLI:** auto-commit against a real repository, and the `init`, `themes` and `version` commands. A test also keeps the built-in theme list in sync with the CSS.

The browser app has no automated tests yet. Before sending a UI change, check it by hand in one light theme, one dark theme and at a narrow window width.

## Guidelines

- Style through the theme tokens in `web/css/themes.css`; never hard-code a color a theme should control.
- Keep the front end framework-free and build-free. Discuss before adding a vendored library.
- All filesystem access goes through `wiki.Store`, and every user-supplied path goes through `Store.Clean`.
- Mutating endpoints must be wrapped in `mutating` so they inherit the read-only, header and origin checks.
- Adding a built-in theme takes three edits: `themes.css`, `BUILTIN` in `web/js/themes.js`, and `builtinThemes` in `cmd/goki/main.go`.

## Ideas and roadmap

Contributions in these areas are especially welcome:

- Rewrite links automatically when a page is moved or renamed
- Page history and diffs from git
- A backlinks panel ("pages that link here")
- An optional search index for very large wikis
- Export to static HTML for hosting anywhere
- Automated browser tests for the front end

## Project infrastructure

| Piece | Where |
| --- | --- |
| CI: tests on Linux, macOS and Windows, lint, native-window builds, cross-compiles | `.github/workflows/ci.yml` |
| Releases via GoReleaser on `v*` tags | `.github/workflows/release.yml`, `.goreleaser.yaml` |
| Dependency updates | `.github/dependabot.yml` |
| Issue and pull request templates | `.github/` |
| Lint configuration (golangci-lint v2) | `.golangci.yml` |
| Container image | `Dockerfile` |
| Dev container | `.devcontainer/devcontainer.json` |
| Community | `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, `CHANGELOG.md` |

## Read next

- [[Architecture]] — how the pieces fit together
- [[HTTP API]] — the JSON API the front end uses
- [[Releasing]] — cutting a release

Contribution guidelines live in [CONTRIBUTING.md](https://github.com/jdtobe/goki/blob/master/CONTRIBUTING.md).
