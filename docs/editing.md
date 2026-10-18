---
title: Editing
description: The editor, the sidebar, the command palette and every keyboard shortcut.
---

# Editing

## Creating pages

- Press <kbd>N</kbd>, click **New Page** in the sidebar, or use **Add a page** on any folder.
- Type a name. Slashes create folders: `guides/Deploying` creates `guides/Deploying.md`.
- Follow a red link to a page that doesn't exist yet and click **Create Page**.
- In the command palette, type any name and choose **Create page**.

A new page isn't written to disk until you save it, so abandoning one leaves nothing behind.

## The editor

Press <kbd>E</kbd> or click **Edit**. The editor has three layouts — **Write**, **Split** and **Preview** — and remembers your choice.

- **Live preview** updates as you type, with scroll kept in sync.
- **Formatting bar** for headings, emphasis, links, images, tables, lists, checklists and quotes.
- **Smart lists**: pressing <kbd>Return</kbd> continues bullets, numbers and checkboxes; on an empty item it ends the list.
- **Indentation**: <kbd>Tab</kbd> and <kbd>Shift</kbd><kbd>Tab</kbd> indent and outdent lines.
- **Images**: paste or drop images and files to upload them beside the page.
- **Drafts**: unsaved text is kept in your browser. If a tab closes unexpectedly, goki offers to restore it next time.
- **Conflict detection**: if the file changed on disk since you opened it, goki asks before overwriting.

Save with <kbd>⌘</kbd><kbd>S</kbd> (<kbd>Ctrl</kbd><kbd>S</kbd>), then press <kbd>Esc</kbd> or **Done** to return to reading. If you leave with unsaved changes, goki asks what to do.

## Organizing

Every page and folder in the sidebar has a **⋯** menu (or right-click) to create pages and folders inside it, rename or move it, copy its link, or delete it.

- **Rename or move** by editing the full location: `notes/idea` → `projects/idea`.
- **Drag and drop** pages and folders onto a folder to move them. Drop onto empty sidebar space to move to the top level.
- **Delete** asks for confirmation. Deleting a folder deletes everything in it.

> [!NOTE]
> Moving a page doesn't rewrite links that point to it. Wiki links by name (`[[Page]]`) keep working wherever the page lives.

## Live reload

goki watches your folder. Change a file in another editor and every open browser updates within a moment — including the sidebar. If you're editing the file that changed, goki loads the new version when you have no unsaved changes, and tells you otherwise.

## Command palette

Press <kbd>⌘</kbd><kbd>K</kbd> (or <kbd>/</kbd>) to jump anywhere. The palette fuzzy-matches page titles and paths instantly, searches inside every page as you type, and runs commands such as switching theme.

## Keyboard shortcuts

| Shortcut | Action |
| --- | --- |
| <kbd>⌘</kbd><kbd>K</kbd> or <kbd>/</kbd> | Search and commands |
| <kbd>E</kbd> | Edit the current page |
| <kbd>N</kbd> | New page |
| <kbd>?</kbd> | Show shortcuts |
| <kbd>⌘</kbd><kbd>\</kbd> | Toggle the sidebar |
| <kbd>⌘</kbd><kbd>S</kbd> | Save |
| <kbd>Esc</kbd> | Finish editing |
| <kbd>⌘</kbd><kbd>B</kbd> / <kbd>⌘</kbd><kbd>I</kbd> | Bold / italic |
| <kbd>⌘</kbd><kbd>E</kbd> | Inline code |
| <kbd>⌘</kbd><kbd>⇧</kbd><kbd>K</kbd> | Insert link |

On Windows and Linux, use <kbd>Ctrl</kbd> wherever you see <kbd>⌘</kbd>.

## The sidebar

Drag the sidebar's edge to resize it; double-click the edge to reset. Hide it with <kbd>⌘</kbd><kbd>\</kbd>. On small screens it slides in from the left.
