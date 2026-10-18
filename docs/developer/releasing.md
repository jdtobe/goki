---
title: Releasing
description: How a goki release is cut.
tags: [contributing]
---

# Releasing

Releases are built by [GoReleaser](https://goreleaser.com) in GitHub Actions whenever a version tag is pushed.

## Checklist

1. Make sure `master` is green in CI.
2. Move the **Unreleased** notes in `CHANGELOG.md` under a new version heading, following [Keep a Changelog](https://keepachangelog.com).
3. Commit, then tag and push:

   ```sh
   git tag -a v1.2.0 -m "goki v1.2.0"
   git push origin v1.2.0
   ```

4. The **Release** workflow builds portable binaries for macOS, Linux, Windows and FreeBSD on amd64 and arm64, writes checksums and publishes a draft GitHub release.
5. Review the draft's notes and publish it.

## Versioning

goki follows [Semantic Versioning](https://semver.org). The configuration format, command-line flags and the [[HTTP API]] are the public interface.

The version is injected at build time with `-ldflags "-X main.version=…"`. Builds without it fall back to the module version recorded by `go install`, or `dev`.

## Trying a release locally

```sh
make snapshot   # goreleaser release --snapshot --clean
ls dist/
```

## Native window builds

Release archives contain the portable build. The native [[App Window]] needs cgo against each platform's system libraries, so users build it with `make install-desktop` or `go install -tags webview`. CI compiles it on macOS, Linux and Windows to make sure it keeps working.
