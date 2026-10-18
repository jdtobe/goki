---
title: Configuration
description: Every setting, command-line flag and environment variable.
tags: [reference]
---

# Configuration

goki works without any configuration. When you want to change something, settings are read from these sources — later ones win:

1. Built-in defaults
2. Your user config file
3. The wiki's config file
4. `GOKI_*` environment variables
5. Command-line flags

## Config files

**Per wiki:** the first of `goki.yaml`, `goki.yml`, `.goki.yaml` or `.goki.yml` in the served folder, or the file named by `--config`.

**Per user:** the first that applies of

- `$GOKI_CONFIG`
- `$XDG_CONFIG_HOME/goki/config.yaml`
- `~/.config/goki/config.yaml` (if it exists)
- your OS config folder: `~/Library/Application Support/goki/config.yaml` on macOS, `%AppData%\goki\config.yaml` on Windows

`goki init` writes a fully commented `goki.yaml` to start from.

## Settings

```yaml
title: Team Handbook     # sidebar and tab title (default: folder name)
theme: auto              # auto, daylight, graphite, glass, nebula, aurora, dune, ink or a custom id
accent: "#0a84ff"        # override the theme's accent color
host: 127.0.0.1          # interface to listen on
port: 4242               # 0 picks a free port
open: browser            # browser, window or none
read_only: false         # disable all editing
watch: true              # live-reload when files change on disk
ignore:                  # hide matching file and folder names
  - node_modules
  - vendor
assets_dir: assets       # where pasted files go, relative to the page ("" = beside it)
max_upload_mb: 32        # largest accepted upload

editor:
  spellcheck: true
  line_wrap: true
  tab_size: 2
  layout: split          # split, write or preview

git:
  auto_commit: false     # commit every change made through goki
  author_name: ""        # defaults to your git config
  author_email: ""

auth:                    # HTTP basic auth (recommended off localhost)
  username: ""
  password: ""
```

If the configured port is busy, goki falls back to a free port — unless you asked for that port explicitly with `--port`, in which case it stops with an error.

## Command line

```text
goki [flags] [dir]       Serve dir (default ".") as a wiki
goki init [dir]          Create goki.yaml and a welcome page in dir
goki themes              List available themes
goki version             Print the version
```

| Flag | Description |
| --- | --- |
| `-p`, `--port` | Port to listen on |
| `--host` | Interface to listen on; `0.0.0.0` shares on your network |
| `-o`, `--open` | On start, open a `browser`, an app `window` or `none` |
| `-w`, `--window` | Same as `--open window` |
| `-n`, `--no-open` | Same as `--open none` |
| `-t`, `--theme` | Default theme |
| `--title` | Wiki title |
| `-r`, `--read-only` | Disable editing |
| `--git` | Commit every change to git |
| `-c`, `--config` | Use this config file instead of the wiki's |
| `--no-watch` | Don't live-reload on file changes |
| `-v`, `--verbose` | Log every request |
| `--version` | Print the version |

## Environment variables

| Variable | Setting |
| --- | --- |
| `GOKI_TITLE` | `title` |
| `GOKI_THEME` | `theme` |
| `GOKI_ACCENT` | `accent` |
| `GOKI_HOST` | `host` |
| `GOKI_PORT` | `port` |
| `GOKI_OPEN` | `open` |
| `GOKI_READ_ONLY` | `read_only` |
| `GOKI_WATCH` | `watch` |
| `GOKI_GIT_AUTO_COMMIT` | `git.auto_commit` |
| `GOKI_AUTH_USERNAME`, `GOKI_AUTH_PASSWORD` | `auth` |
| `GOKI_CONFIG` | Path of the user config file |
| `NO_COLOR`, `FORCE_COLOR` | Disable or force colored terminal output |

## Git auto-commit

With `git.auto_commit` (or `--git`), every save, upload, move and deletion made through goki becomes a commit containing just the affected files, with a message such as `Update guides/setup.md`. If the folder isn't in a repository yet, goki runs `git init` for you. Only changes made through goki are committed — edits from other tools are left for you to commit. Requires `git` on your `PATH`.
