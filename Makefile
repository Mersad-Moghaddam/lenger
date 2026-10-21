.PHONY: build run test lint gallery vhs check fmt-check golden
build:
	CGO_ENABLED=0 go build -o bin/lenger ./cmd/lenger
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
	@test -z "$$(gofmt -l $$(find cmd internal -name '*.go'))" || (echo 'Run gofmt on Go sources'; exit 1)
vhs:
	@if command -v vhs >/dev/null; then for tape in vhs/*.tape; do vhs "$$tape" || exit; done; else echo 'VHS not installed; skipping recordings'; fi
check: fmt-check
	go build ./...
	go vet ./...
	$(MAKE) lint
	$(MAKE) test
	$(MAKE) golden
