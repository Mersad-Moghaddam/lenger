.PHONY: build run test lint gallery vhs check fmt-check golden
VERSION ?= dev
COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= unknown
LDFLAGS = -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/lenger ./cmd/lenger
run:
	go run ./cmd/lenger
test:
	go test -race ./...
lint:
	golangci-lint run
gallery:
	go run ./cmd/lenger --gallery
golden:
	go test ./internal/gallery -run TestGolden
fmt-check:
	@command -v gofmt >/dev/null || { echo 'gofmt is required for fmt-check'; exit 1; }
	@files="$$(gofmt -l $$(find cmd internal -name '*.go'))" || exit; test -z "$$files" || { echo 'Run gofmt on Go sources'; echo "$$files"; exit 1; }
vhs:
	@if command -v vhs >/dev/null; then for tape in vhs/*.tape; do vhs "$$tape" || exit; done; else echo 'VHS not installed; skipping recordings'; fi
check: fmt-check
	go build ./...
	go vet ./...
	$(MAKE) lint
	$(MAKE) test
	$(MAKE) golden
