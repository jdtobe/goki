---
title: Sharing and Security
description: How goki protects your files, and how to share a wiki safely.
tags: [security]
---

# Sharing and Security

goki edits real files on your disk, so it is careful about who can reach it.

## Defaults

- **Localhost only.** goki listens on `127.0.0.1`. Nothing on your network can connect.
- **No cross-site writes.** Every change requires a custom request header that browsers refuse to send cross-origin without permission goki never grants, and any `Origin` must match. A malicious website can't edit your wiki through your browser.
- **No DNS rebinding.** While bound to localhost, goki rejects requests whose `Host` isn't a local name.
- **Confined to the folder.** Paths are cleaned, `..` is rejected, symlinks that lead outside the wiki are refused, and dot-files are never served.
- **Sanitized rendering.** Markdown is rendered through [DOMPurify](https://github.com/cure53/DOMPurify) under a strict Content Security Policy, so pages can't run scripts. Raw files such as HTML or SVG attachments are served in a sandbox.
- **Bounded uploads.** Uploaded files are capped at `max_upload_mb` (32 MB by default), and their names are sanitized.
- **Atomic writes.** Pages are written to a temporary file and renamed into place, so a crash never leaves half a file.

## Sharing on your network

To let others on your network read the wiki:

```sh
goki --host 0.0.0.0 --read-only
```

goki prints a **Network** address for each interface. If you want others to edit too, turn on authentication:

```yaml
host: 0.0.0.0
auth:
  username: team
  password: a-long-random-passphrase
```

> [!WARNING]
> goki speaks plain HTTP. Basic-auth passwords and page contents cross the network unencrypted. On networks you don't control, put goki behind a TLS-terminating reverse proxy such as [Caddy](https://caddyserver.com):
>
> ```text
> wiki.example.com {
>   reverse_proxy 127.0.0.1:4242
> }
> ```

When goki is served on a non-local address with editing enabled and no authentication, it prints a warning at startup.

## Reporting vulnerabilities

Please see [SECURITY.md](https://github.com/jdtobe/goki/blob/master/SECURITY.md).
