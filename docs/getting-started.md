---
title: Getting Started
description: Install goki and serve your first wiki in under a minute.
---

# Getting Started

## Install

goki is a single binary with no runtime dependencies.

### With Go

```sh
go install github.com/jdtobe/goki/cmd/goki@latest
```

This builds the portable version. To include the native [[App Window]] (recommended on macOS), enable cgo and the `webview` tag:

```sh
CGO_ENABLED=1 go install -tags webview github.com/jdtobe/goki/cmd/goki@latest
```

### Prebuilt binaries

Download an archive for your platform from the [releases page](https://github.com/jdtobe/goki/releases), extract it and put `goki` somewhere on your `PATH`.

### From source

```sh
git clone https://github.com/jdtobe/goki.git
cd goki
make install          # portable build
make install-desktop  # with the native app window
```

### Docker

```sh
docker build -t goki .
docker run --rm -p 4242:4242 -v "$PWD:/wiki" --user "$(id -u):$(id -g)" goki
```

## Serve a folder

From any folder that contains (or will contain) Markdown:

```sh
goki
```

goki prints its address and opens your browser:

```text
  ◆ goki v1.0.0

  ➜  Local:     http://localhost:4242/
  ➜  Wiki:      ~/notes (42 pages)
  ➜  Mode:      read & write · theme auto · git off

  Press Ctrl+C to stop.
```

Serve a different folder by naming it — `goki ~/notes` — or keep the browser closed with `goki -n`. If port 4242 is busy, goki quietly picks a free one.

## Start a new wiki

`goki init` creates a commented `goki.yaml` and a welcome page:

```sh
goki init my-wiki
goki my-wiki
```

It never overwrites existing files, and skips the welcome page if the folder already has an `index.md` or `README.md`.

## Recipes

```sh
goki                                # serve the current folder, open a browser
goki ~/notes --window               # open in a dedicated app window
goki docs -p 8080 -n                # port 8080, don't open anything
goki . --theme aurora               # choose the default theme
goki . --read-only --host 0.0.0.0   # share read-only on your network
goki . --git                        # commit every change to git
goki init my-wiki && goki my-wiki   # start a brand-new wiki
GOKI_THEME=dune goki                # settings work as environment variables too
```

Every flag is listed in [[Configuration]].

## Next steps

- Learn how [[Organizing Pages|folders and files map to pages]].
- Explore what you can do when [[Writing]].
- Pick a look in [[Themes]].
