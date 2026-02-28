.PHONY: build test test-e2e test-all vet fmt lint check clean help

## build: compile clihub binary into bin/
build:
	go build -o bin/clihub .

## test: run unit tests
test:
	go test ./...

## test-e2e: run end-to-end tests
test-e2e:
	go test ./e2e/... -v -count=1 -timeout 180s

## test-all: run unit and E2E tests
test-all: test test-e2e

## vet: run go vet
vet:
	go vet ./...

## fmt: check formatting (mirrors pre-push gofmt check)
fmt:
	@unformatted=$$(gofmt -l $$(git ls-files '*.go')); \
	if [ -n "$$unformatted" ]; then \
		echo "Unformatted Go files:"; \
		echo "$$unformatted"; \
		echo "Run: gofmt -w <files>"; \
		exit 1; \
	fi

## lint: run golangci-lint (if installed)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi

## check: full pre-push validation (fmt + vet + lint + test-all)
check: fmt vet lint test-all

## clean: remove build artifacts
clean:
	rm -rf bin/

## help: list available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //' | column -t -s ':'
