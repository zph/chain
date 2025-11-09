.PHONY: help build release proto run docs tag setup test test-unit test-e2e test-coverage clean

# Default target
.DEFAULT_GOAL := help

##@ General

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

build: ## Build binary using goreleaser
	goreleaser build --single-target --skip-validate --snapshot --rm-dist

proto: ## Generate protobuf code
	buf generate

clean: ## Remove build artifacts
	rm -rf dist/ coverage.out test/chain

##@ Development

run: ## Run chain with a command (usage: make run CMD="get mychain")
	go run main.go $(CMD) chainer

docs: ## Generate documentation
	go run docs/main.go

setup: ## Install development dependencies
	go get golang.org/x/tools/cmd/godoc

##@ Testing

test: test-unit test-e2e ## Run all tests (unit + e2e)

test-unit: ## Run Go unit tests
	go test -v ./cmd/...

test-e2e: ## Run Deno e2e tests
	deno test --allow-all test/

test-coverage: ## Generate test coverage report
	go test -coverprofile=coverage.out ./cmd/...
	go tool cover -html=coverage.out

##@ Release

tag: ## Create a git tag (usage: make tag VERSION=v1.0.0)
	@if [ -z "$(VERSION)" ]; then echo "Error: VERSION is required. Usage: make tag VERSION=v1.0.0"; exit 1; fi
	git tag -a $(VERSION) -m "Release of $(VERSION)"
	@echo "Tag $(VERSION) created. Don't forget to push it: git push origin $(VERSION)"

release: ## Display release instructions
	@echo "To create a release:"
	@echo "  1. Create a tag: make tag VERSION=v1.0.0"
	@echo "  2. Push the tag: git push origin v1.0.0"
	@echo "  3. GitHub Actions will handle the release build"
