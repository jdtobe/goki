# Contributing to goki

Thanks for your interest in making goki better! Bug reports, ideas, documentation fixes and code are all welcome.

## Ground rules

- Be kind. This project follows the [Code of Conduct](CODE_OF_CONDUCT.md).
- For anything bigger than a small fix, please open an issue first so we can agree on the approach.
- Security issues go through [private reporting](SECURITY.md), not public issues.

## Getting set up

You need Go 1.24+ and [golangci-lint](https://golangci-lint.run) v2. Or open the repository in the included dev container.

```sh
git clone https://github.com/jdtobe/goki.git
cd goki
make run      # serves ./docs at http://localhost:4242 with live front-end files
```

The [developer guide](docs/developer/index.md) explains the architecture, the HTTP API and how releases work.

## Making changes

1. Create a branch from `master`.
2. Keep changes focused; unrelated cleanups belong in their own pull request.
3. Add or update tests. Back-end behavior is covered by `go test`; please check UI changes by hand in at least one light and one dark theme, and at a narrow window width.
4. Update the docs in `docs/` when behavior changes, and add a line to `CHANGELOG.md` under **Unreleased**.
5. Run `make test lint` before pushing.

### Code style

- Go: `gofmt`/`goimports`, idiomatic error handling, small packages with clear ownership. Exported identifiers need doc comments.
- JavaScript: modern ES modules, no framework and no build step. Match the existing style — two-space indentation, double quotes, semicolons.
- CSS: style through the design tokens in `web/css/themes.css`; never hard-code a color that a theme should control.
- Front-end dependencies are vendored deliberately. Discuss before adding a new one.

### Commit messages

Use the imperative mood and a short summary line, optionally with a [Conventional Commits](https://www.conventionalcommits.org) prefix (`feat:`, `fix:`, `docs:`…), which groups release notes.

## Adding a built-in theme

1. Add a `[data-theme="<id>"]` block to `web/css/themes.css`.
2. Register it in `BUILTIN` in `web/js/themes.js` and `builtinThemes` in `cmd/goki/main.go` (a test checks these stay in sync).
3. If it is dark, add it to the dark callout-color selector.
4. Document it in `docs/themes.md`.

## License

By contributing, you agree that your contributions are licensed under the [MIT License](LICENSE).
