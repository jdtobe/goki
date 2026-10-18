---
title: HTTP API
description: The JSON API behind goki's interface.
tags: [reference, contributing]
---

# HTTP API

goki's interface is built entirely on this API, so anything the UI can do, a script can do too. All endpoints live under `/-/api/`.

> [!IMPORTANT]
> Requests that change anything (`PUT`, `POST`, `DELETE`) must send the header `X-Goki: 1`, and JSON bodies must use `Content-Type: application/json`. Any `Origin` header must match the host. If authentication is configured, use HTTP basic auth.

Errors are returned as `{"error": "message"}` with a meaningful status: `400` invalid path, `403` forbidden or read-only, `404` not found, `409` conflict or already exists, `413` upload too large.

## Read

### `GET /-/api/config`

Wiki title, default theme, read-only flag, editor preferences, custom themes and version.

### `GET /-/api/tree`

The complete page tree.

```json
{
  "type": "dir", "name": "", "path": "", "file": "index.md", "title": "",
  "children": [
    { "type": "dir", "name": "guides", "path": "guides", "title": "Guides", "children": [
      { "type": "page", "name": "setup", "path": "guides/setup", "file": "guides/setup.md",
        "title": "Setup", "modified": "2026-10-17T09:41:00Z" }
    ]}
  ]
}
```

### `GET /-/api/page?path=<route>`

Resolves a route. `kind` is `page`, `dir` or `missing` (with status `404`). Folders include their direct `children`; a folder with an index page includes it as `page`.

```json
{
  "kind": "page",
  "path": "guides/setup",
  "page": {
    "path": "guides/setup", "file": "guides/setup.md", "title": "Setup",
    "content": "# Setup\n…", "hash": "9f2c…", "modified": "…", "size": 1234
  }
}
```

### `GET /-/api/search?q=<query>&limit=<n>`

Full-text search. Every word must match the title, path or body. Results are ranked with title matches first.

```json
[{ "path": "guides/setup", "file": "guides/setup.md", "title": "Setup",
   "snippet": "…Install goki with…", "score": 13.5 }]
```

### `GET /-/api/events`

A [Server-Sent Events](https://developer.mozilla.org/docs/Web/API/Server-sent_events) stream. After a `hello` event, goki sends `change` events whenever files change, whether through goki or on disk:

```text
event: change
data: {"type":"change","paths":["guides/setup.md"]}
```

## Write

### `PUT /-/api/page`

Creates or updates a page.

```json
{ "file": "guides/setup.md", "content": "# Setup\n", "create": false, "baseHash": "9f2c…" }
```

- `create: true` fails with `409` if the page exists.
- `baseHash` (from a previous read) makes the write fail with `409` if the file has changed since; the response then includes the current `page`.

Returns the saved page.

### `POST /-/api/move`

Renames or moves a page or folder. Pages keep their `.md` extension; missing parent folders are created; existing targets are never overwritten.

```json
{ "from": "notes/idea.md", "to": "projects/idea" }
```

Returns `{"path": "projects/idea.md", "route": "projects/idea"}`.

### `DELETE /-/api/entry?path=<path>&recursive=<bool>`

Deletes a page file or folder. Non-empty folders require `recursive=true`. Returns `204`.

### `POST /-/api/folder`

Creates a folder: `{"path": "projects/alpha"}`.

### `POST /-/api/upload?dir=<folder>`

Uploads one file as `multipart/form-data` field `file`. It is stored in `<dir>/<assets_dir>/` with a sanitized, unique name. Returns `{"path": "guides/assets/shot.png", "name": "shot.png"}`.

## Other endpoints

| Endpoint | Purpose |
| --- | --- |
| `GET /<path>` | A raw file when one exists at that path, otherwise the app |
| `GET /<page>.md?raw` | A page's Markdown source as plain text |
| `GET /-/themes/<id>.css` | A custom theme stylesheet |
| `GET /-/healthz` | Liveness check; returns `ok` |

## Example

```sh
curl -s localhost:4242/-/api/page?path=index | jq -r .page.content

curl -s -X PUT localhost:4242/-/api/page \
  -H 'X-Goki: 1' -H 'Content-Type: application/json' \
  -d '{"file": "inbox/from-curl.md", "content": "# Hello from curl\n", "create": true}'
```
