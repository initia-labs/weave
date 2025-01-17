#!/usr/bin/make -f

# Go version and build settings
GO_VERSION := 1.23
GO_SYSTEM_VERSION := $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1-2)
REQUIRE_GO_VERSION := $(GO_VERSION)

# Project version
WEAVE_VERSION := $(shell git describe --tags)

# Build directory
BUILDDIR ?= $(CURDIR)/build

# Build targets
BUILD_TARGETS := build install test

release_version=$(filter-out $@,$(MAKECMDGOALS))

# Version check
check_version:
	@if [ $(shell echo "$(GO_SYSTEM_VERSION) < $(REQUIRE_GO_VERSION)" | bc -l) -eq 1 ]; then \
		echo "ERROR: Go version $(REQUIRE_GO_VERSION) is required for Weave."; \
		exit 1; \
	fi

# Build settings
LDFLAGS := -X github.com/initia-labs/weave/cmd.Version=$(WEAVE_VERSION)

dev: check_version
	go install -ldflags "$(LDFLAGS) -X github.com/initia-labs/weave/analytics.AmplitudeKey=aba1be3e2335dd5b8b060e977d93410b" .

# Build targets
build: check_version $(BUILDDIR)
	go build -mod=readonly -ldflags "$(LDFLAGS)" -o $(BUILDDIR)/weave .

install: check_version
	go install -ldflags "$(LDFLAGS)" .

.PHONY: lint lint-fix

# Run golangci-lint to check code quality
lint: check_version
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint is required but not installed. Install it by following instructions at https://golangci-lint.run/welcome/install/"; exit 1; }
	golangci-lint run --out-format=tab --timeout=15m

# Run golangci-lint and automatically fix issues where possible (use with caution)
lint-fix: check_version
	@echo "Warning: This will automatically modify your files to fix linting issues"
	@read -p "Are you sure you want to continue? [y/N] " -n 1 -r; echo; if [[ ! $$REPLY =~ ^[Yy]$$ ]]; then exit 1; fi
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint is required but not installed. Install it by following instructions at https://golangci-lint.run/welcome/install/"; exit 1; }
	golangci-lint run --fix --out-format=tab --timeout=15m

test: check_version
	go clean -testcache
	go test -v ./...

integration-test: check_version
	go clean -testcache
	go test -v ./... -tags=integration

test-all: test integration-test

precommit: lint test-all

# Release process
release:
	@if [ -z "$(release_version)" ]; then \
		echo "ERROR: You must provide a release version. Example: make release v0.0.15"; \
		exit 1; \
	fi
	git tag -a $(release_version) -m "$(release_version)"
	git push origin $(release_version)
	@echo "Paste the release notes below (end with Ctrl+D):"
	@notes=$$(cat); \
	gh release create $(release_version) --title "$(release_version)" --notes "$$notes"

# Development purpose
local: build
	clear
	./build/weave opinit init

# Catch-all target
%:
	@:
