SHELL := /usr/bin/env bash
BROWSERTEST_VERSION = v0.6
LINT_VERSION = 1.50.1
GO_BIN = $(shell printf '%s/bin' "$$(go env GOPATH)")

.PHONY: all
all: lint test

.PHONY: lint-deps
lint-deps:
	@if ! which golangci-lint >/dev/null || [[ "$$(golangci-lint version 2>&1)" != *${LINT_VERSION}* ]]; then \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "${GO_BIN}" v${LINT_VERSION}; \
	fi

.PHONY: lint
lint: lint-deps
	"${GO_BIN}/golangci-lint" run
	GOOS=js GOARCH=wasm "${GO_BIN}/golangci-lint" run

.PHONY: test-deps
test-deps:
	@if [[ ! -f "${GO_BIN}/go_js_wasm_exec" ]]; then \
		set -ex; \
		go install github.com/agnivade/wasmbrowsertest@${BROWSERTEST_VERSION}; \
		mv "${GO_BIN}/wasmbrowsertest" "${GO_BIN}/go_js_wasm_exec"; \
	fi

.PHONY: test
test: test-deps
	go test wasm_tags_test.go                                                                  # Verify build tags and whatnot first
	go test -race -coverprofile=native-cover.out ./...                                         # Test non-js side
	GOOS=js GOARCH=wasm go test -coverprofile=js-cover.out -covermode=atomic ./...             # Test js side
	{ echo 'mode: atomic'; cat *-cover.out | grep -v '^mode'; } > cover.out && rm *-cover.out  # Combine JS and non-JS coverage.
	go tool cover -func cover.out | grep total:

.PHONY: test-publish-coverage
test-publish-coverage:
	go install github.com/mattn/goveralls@v0.0.11
	COVERALLS_TOKEN=$$GITHUB_TOKEN goveralls -coverprofile="cover.out" -service=github
