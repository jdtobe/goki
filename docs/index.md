---
title: goki
description: Your folder of Markdown, as a beautiful wiki.
tags: [documentation]
---

# goki

goki turns any folder of Markdown files into a fast, beautiful, editable wiki. There is no database, no import step and no lock-in: **the folders are the structure and the `.md` files are the pages.** Close goki and your notes are exactly where they were, readable by every editor, diff tool and AI assistant you own.

```sh
cd ~/notes
goki
```

> [!TIP]
> You are reading goki's own documentation, served by goki. Run `goki docs` from a clone of the repository to browse it like this.

## Why goki

- **Zero setup.** One self-contained binary. Point it at a folder and it opens your browser.
- **Plain files.** Every page is a Markdown file you can edit anywhere. goki writes them atomically and never touches anything else.
- **Made to be looked at.** Seven hand-tuned themes, from crisp *Daylight* to the drifting glass of *Aurora*, plus your own.
- **Built for writing.** A distraction-free split editor with live preview, image paste, list continuation and draft recovery.
- **Everything you expect.** Full-text search, wiki links, callouts, diagrams, math, task lists and a ⌘K command palette.
- **Safe by default.** Listens on localhost only, defends against cross-site and DNS-rebinding attacks, and sanitizes everything it renders.

## Start here

| Guide | What's inside |
| --- | --- |
| [[Getting Started]] | Install goki and serve your first wiki |
| [[Organizing Pages]] | How folders and files become a wiki |
| [[Writing]] | Every Markdown feature goki understands |
| [[Editing]] | The editor, the sidebar and keyboard shortcuts |
| [[Themes]] | Built-in themes and how to make your own |
| [[Configuration]] | Every setting, flag and environment variable |
| [[App Window]] | Run goki in its own window |
| [[Sharing and Security]] | Serving goki to other people safely |

Building goki itself? Head to the [developer guide](developer/index.md).
