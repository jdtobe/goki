---
title: Organizing Pages
description: Your filesystem is the wiki's structure. Here is exactly how goki reads it.
---

# Organizing Pages

goki has no database. It reads your folder every time, so anything you do in Finder, your editor or `git` shows up immediately — and goki's own changes are ordinary file operations.

## At a glance

```text
notes/
├── index.md                →  /                    home page
├── setup.md                →  /setup
├── guides/
│   ├── README.md           →  /guides              section page
│   ├── 01-install.md       →  /guides/01-install   titled "Install"
│   └── assets/diagram.png  →  served as-is, hidden from the sidebar
└── .private/               →  hidden, never served
```

## The rules

| On disk | In the wiki |
| --- | --- |
| `setup.md` | The page **/setup** |
| `guides/` | The section **/guides** |
| `guides/install.md` | The page **/guides/install** |
| `guides/index.md` or `guides/README.md` | The page shown at **/guides** itself |
| `index.md` or `README.md` at the root | Your home page |
| `.anything` | Hidden — dot-files and dot-folders are never shown or served |
| `diagram.png`, `report.pdf`… | Attachments, served at their own path |

When a page and a folder share a name (`guides.md` *and* `guides/`), the page wins at `/guides` and the folder's other pages still appear beneath it in the sidebar.

Folders without an index page get a generated overview listing their contents. Folders that hold only attachments, such as the `assets/` folders that pasted images go into, are left out of the sidebar to keep it tidy.

## Titles

A page's title comes from, in order:

1. a `title:` field in YAML front matter,
2. the first level-one heading (`# Like this`),
3. the file name, tidied up: `02-getting_started.md` becomes *Getting started*.

## Ordering

Folders are listed before pages, and both are sorted *naturally*, so `page2` comes before `page10`. To control the order, prefix names with numbers — goki hides the prefix in display names:

```text
01-introduction.md   → Introduction
02-installation.md   → Installation
10-faq.md            → Faq
```

## Hiding things

Besides dot-files, goki hides names matching the `ignore` patterns in your [[Configuration]]. The defaults are `node_modules`, `vendor`, `__pycache__`, `*.tmp` and `~*`.

## Links between pages

Relative links work exactly as they do on GitHub, so your wiki stays portable:

```markdown
See the [installation guide](guides/install.md).
```

Or use wiki links, which find pages by name anywhere in the wiki:

```markdown
See [[Install]] or [[guides/install|the install guide]].
```

Links to pages that don't exist yet are shown in red. Click one to create the page.

## Reserved paths

goki serves its own interface under `/-/`, so a folder named `-` at the root of your wiki can't be reached through the browser.
