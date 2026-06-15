# verified-bases-api — Go → WASM → Cloudflare Workers.
#
# We use standard Go (not TinyGo) because the Cloudflare D1 driver in
# syumai/workers v0.33+ uses database/sql, which depends on reflect that
# TinyGo doesn't fully support. Trade-off: a larger WASM binary.
#
# Empirically the gzipped output is ~3 MB — at the Free-tier ceiling. If we
# add deps and overflow, the Workers Paid plan raises the limit to 10 MB
# gzipped. Re-check with `make build-stats`.

SHELL     := /bin/bash
BUILD_DIR := build
WASM      := $(BUILD_DIR)/app.wasm

.PHONY: help
help:
	@echo "verified-bases-api — common tasks:"
	@echo "  make build          — Compile Go → WASM + emit Worker runtime"
	@echo "  make build-stats    — Build and print raw + gzipped sizes"
	@echo "  make dev            — Build, then start wrangler dev (local D1+KV)"
	@echo "  make deploy         — Build, then wrangler deploy"
	@echo "  make migrate-local  — Apply migrations to local D1"
	@echo "  make migrate-remote — Apply migrations to remote D1 (PROD)"
	@echo "  make vet            — go vet"
	@echo "  make tidy           — go mod tidy"
	@echo "  make clean          — rm $(BUILD_DIR)"

.PHONY: build
build:
	@command -v go >/dev/null || { echo "go not installed (need Go 1.22+)"; exit 1; }
	# workers-assets-gen REMOVES its output dir first, so always run it
	# before go build — otherwise it wipes the freshly-built wasm.
	@go run github.com/syumai/workers/cmd/workers-assets-gen -mode=go -o $(BUILD_DIR)
	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o $(WASM) .

.PHONY: build-stats
build-stats: build
	@echo "── wasm size ──"
	@ls -lh $(WASM)
	@gzip -kf $(WASM)
	@ls -lh $(WASM).gz
	@rm $(WASM).gz

.PHONY: dev
dev: build
	npx wrangler dev --local

.PHONY: deploy
deploy: build
	npx wrangler deploy

.PHONY: migrate-local
migrate-local:
	npx wrangler d1 execute verified-bases-db --local --file=migrations/0001_init.sql

.PHONY: migrate-remote
migrate-remote:
	npx wrangler d1 execute verified-bases-db --remote --file=migrations/0001_init.sql

.PHONY: vet
vet:
	GOOS=js GOARCH=wasm go vet ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
