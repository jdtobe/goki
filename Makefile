.DEFAULT_GOAL := help

BIN      := bin/goki
PKG      := ./cmd/goki
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)
GOFLAGS  := -trimpath

# Vendored front-end libraries (see docs/developer/architecture.md).
MARKED_VERSION    := 15.0.7
PURIFY_VERSION    := 3.2.4
HLJS_VERSION      := 11.11.1
KATEX_VERSION     := 0.16.21
MERMAID_VERSION   := 11.4.1
CDN               := https://cdn.jsdelivr.net/npm

##@ Development

.PHONY: run
run: ## Serve ./docs with live front-end files (edit web/ and reload)
	GOKI_WEB_DIR=web go run $(PKG) docs

.PHONY: test
test: ## Run the test suite with the race detector
	go test -race ./...

.PHONY: cover
cover: ## Run tests and open a coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: lint
lint: ## Lint both the portable and native-window builds
	golangci-lint run ./...
	golangci-lint run --build-tags webview ./...

.PHONY: fmt
fmt: ## Format Go code
	golangci-lint fmt ./...

.PHONY: tidy
tidy: ## Tidy go.mod and go.sum
	go mod tidy

##@ Build

.PHONY: build
build: ## Build the portable binary into ./bin
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) $(PKG)

.PHONY: desktop
desktop: ## Build with the native app window (requires cgo)
	CGO_ENABLED=1 go build $(GOFLAGS) -tags webview -ldflags "$(LDFLAGS)" -o $(BIN) $(PKG)

.PHONY: install
install: ## Install the portable binary into $GOBIN
	CGO_ENABLED=0 go install $(GOFLAGS) -ldflags "$(LDFLAGS)" $(PKG)

.PHONY: install-desktop
install-desktop: ## Install with the native app window (requires cgo)
	CGO_ENABLED=1 go install $(GOFLAGS) -tags webview -ldflags "$(LDFLAGS)" $(PKG)

.PHONY: snapshot
snapshot: ## Build release archives locally with GoReleaser
	goreleaser release --snapshot --clean

.PHONY: docker
docker: ## Build the Docker image
	docker build --build-arg VERSION=$(VERSION) -t goki:$(VERSION) -t goki:latest .

.PHONY: clean
clean: ## Remove build output
	rm -rf bin dist coverage.out

##@ Maintenance

.PHONY: vendor
vendor: ## Re-download vendored front-end libraries
	curl -fsSL -o web/vendor/marked.min.js $(CDN)/marked@$(MARKED_VERSION)/marked.min.js
	curl -fsSL -o web/vendor/purify.min.js $(CDN)/dompurify@$(PURIFY_VERSION)/dist/purify.min.js
	curl -fsSL -o web/vendor/highlight.min.js $(CDN)/@highlightjs/cdn-assets@$(HLJS_VERSION)/highlight.min.js
	curl -fsSL -o web/vendor/mermaid.min.js $(CDN)/mermaid@$(MERMAID_VERSION)/dist/mermaid.min.js
	curl -fsSL -o web/vendor/katex/katex.min.js $(CDN)/katex@$(KATEX_VERSION)/dist/katex.min.js
	curl -fsSL -o web/vendor/katex/katex.min.css $(CDN)/katex@$(KATEX_VERSION)/dist/katex.min.css
	curl -fsSL -o web/vendor/katex/auto-render.min.js $(CDN)/katex@$(KATEX_VERSION)/dist/contrib/auto-render.min.js
	for f in $$(grep -oE 'fonts/[A-Za-z0-9_-]+\.woff2' web/vendor/katex/katex.min.css | sort -u); do \
		curl -fsSL -o web/vendor/katex/$$f $(CDN)/katex@$(KATEX_VERSION)/dist/$$f; done
	curl -fsSL -o web/fonts/InterVariable.woff2 https://rsms.me/inter/font-files/InterVariable.woff2

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage: make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
