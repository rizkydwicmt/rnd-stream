# Makefile for Stream Application
# Provides convenient commands for development, testing, and building

# ========================================================================
# Variables
# ========================================================================
APP_NAME := stream
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)

GO := go
GOFLAGS := -v
GOBUILD := $(GO) build $(GOFLAGS)
GOTEST := $(GO) test $(GOFLAGS)
GOLINT := golangci-lint

BIN_DIR := bin
COVERAGE_DIR := coverage

# ========================================================================
# Default target
# ========================================================================
.DEFAULT_GOAL := help

# ========================================================================
# Help
# ========================================================================
.PHONY: help
help: ## Display this help message
	@echo "Stream Application - Available Commands:"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "\033[36m\033[0m"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""

# ========================================================================
##@ Development
# ========================================================================

.PHONY: dev
dev: ## Run application in development mode
	$(GO) run .

.PHONY: watch
watch: ## Run application with auto-reload (requires air)
	@which air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air

.PHONY: deps
deps: ## Download dependencies
	$(GO) mod download
	$(GO) mod verify

.PHONY: tidy
tidy: ## Tidy up go.mod and go.sum
	$(GO) mod tidy

.PHONY: vendor
vendor: ## Vendor dependencies
	$(GO) mod vendor

# ========================================================================
##@ Building
# ========================================================================

.PHONY: build
build: ## Build application binary
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) .
	@echo "Build complete: $(BIN_DIR)/$(APP_NAME)"

.PHONY: build-all
build-all: ## Build for all platforms
	@mkdir -p $(BIN_DIR)
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-windows-amd64.exe .
	@echo "Multi-platform build complete!"
	@ls -lh $(BIN_DIR)

.PHONY: install
install: ## Install application to $GOPATH/bin
	$(GO) install -ldflags="$(LDFLAGS)" .

# ========================================================================
##@ Testing
# ========================================================================

.PHONY: test
test: ## Run unit tests
	$(GOTEST) -race -coverprofile=coverage.out ./...

.PHONY: test-verbose
test-verbose: ## Run unit tests with verbose output
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

.PHONY: test-short
test-short: ## Run short tests only
	$(GOTEST) -short ./...

.PHONY: test-integration
test-integration: ## Run integration tests
	$(GOTEST) -v -tags=integration ./tests/integration/...

.PHONY: coverage
coverage: test ## Generate coverage report
	@mkdir -p $(COVERAGE_DIR)
	$(GO) tool cover -html=coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

.PHONY: coverage-func
coverage-func: test ## Show coverage by function
	$(GO) tool cover -func=coverage.out

# ========================================================================
##@ Benchmarking
# ========================================================================

.PHONY: bench
bench: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem -benchtime=100000x ./...

.PHONY: bench-all
bench-all: ## Run all benchmarks with detailed output
	$(GOTEST) -bench=. -benchmem -benchtime=1000000x -cpuprofile=cpu.prof -memprofile=mem.prof ./...

.PHONY: bench-compare
bench-compare: ## Compare benchmark results (requires benchstat)
	@which benchstat > /dev/null || (echo "Installing benchstat..." && go install golang.org/x/perf/cmd/benchstat@latest)
	$(GOTEST) -bench=. -benchmem -count=5 ./... | tee new.txt
	@echo "Compare with: benchstat old.txt new.txt"

# ========================================================================
##@ Code Quality
# ========================================================================

.PHONY: lint
lint: ## Run linters
	@which $(GOLINT) > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	$(GOLINT) run --config=.golangci.yml

.PHONY: lint-fix
lint-fix: ## Run linters with auto-fix
	$(GOLINT) run --config=.golangci.yml --fix

.PHONY: fmt
fmt: ## Format code
	$(GO) fmt ./...
	gofmt -s -w .

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: staticcheck
staticcheck: ## Run staticcheck
	@which staticcheck > /dev/null || (echo "Installing staticcheck..." && go install honnef.co/go/tools/cmd/staticcheck@latest)
	staticcheck ./...

.PHONY: check
check: fmt lint vet test ## Run all quality checks

# ========================================================================
##@ Security
# ========================================================================

.PHONY: sec
sec: ## Run security scanner (gosec)
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	gosec ./...

.PHONY: vuln
vuln: ## Check for vulnerabilities
	@which govulncheck > /dev/null || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

.PHONY: audit
audit: sec vuln ## Run full security audit

# ========================================================================
##@ Docker
# ========================================================================

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(APP_NAME):$(VERSION) \
		-t $(APP_NAME):latest \
		.

.PHONY: docker-run
docker-run: ## Run Docker container
	docker run --rm -p 8080:8080 $(APP_NAME):latest

.PHONY: docker-push
docker-push: docker-build ## Push Docker image to registry
	docker tag $(APP_NAME):$(VERSION) ghcr.io/$(GITHUB_REPOSITORY)/$(APP_NAME):$(VERSION)
	docker push ghcr.io/$(GITHUB_REPOSITORY)/$(APP_NAME):$(VERSION)

# ========================================================================
##@ Cleaning
# ========================================================================

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)
	rm -rf $(COVERAGE_DIR)
	rm -f coverage.out coverage.html
	rm -f *.prof
	rm -f *.txt
	$(GO) clean -cache -testcache -modcache

.PHONY: clean-deps
clean-deps: ## Remove vendor directory
	rm -rf vendor/

# ========================================================================
##@ Release
# ========================================================================

.PHONY: tag
tag: ## Create a new git tag (usage: make tag VERSION=v1.0.0)
	@if [ -z "$(VERSION)" ]; then echo "Usage: make tag VERSION=v1.0.0"; exit 1; fi
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
	@echo "Tag $(VERSION) created and pushed!"

.PHONY: changelog
changelog: ## Generate changelog
	@which git-chglog > /dev/null || (echo "Installing git-chglog..." && go install github.com/git-chglog/git-chglog/cmd/git-chglog@latest)
	git-chglog -o CHANGELOG.md

# ========================================================================
##@ CI/CD Local Testing
# ========================================================================

.PHONY: ci-lint
ci-lint: ## Run CI lint checks locally
	$(MAKE) lint
	@echo "Checking go mod tidy..."
	$(GO) mod tidy
	@git diff --exit-code go.mod go.sum || (echo "go.mod or go.sum is not tidy" && exit 1)
	@echo "Checking formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Code is not formatted" && gofmt -d . && exit 1)
	$(MAKE) vet

.PHONY: ci-test
ci-test: ## Run CI test suite locally
	$(MAKE) test-verbose
	$(MAKE) bench

.PHONY: ci-build
ci-build: ## Run CI build checks locally
	$(MAKE) build
	$(MAKE) build-all

.PHONY: ci-security
ci-security: ## Run CI security checks locally
	$(MAKE) audit
	$(MAKE) lint

.PHONY: ci-all
ci-all: ci-lint ci-test ci-build ci-security ## Run all CI checks locally
	@echo "All CI checks passed! ✅"

# ========================================================================
##@ Information
# ========================================================================

.PHONY: version
version: ## Show version information
	@echo "Version:    $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

.PHONY: info
info: ## Show project information
	@echo "App Name:   $(APP_NAME)"
	@echo "Version:    $(VERSION)"
	@echo "Go Version: $$(go version)"
	@echo "Git Branch: $$(git branch --show-current 2>/dev/null || echo 'N/A')"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

# ========================================================================
# Special Targets
# ========================================================================

.PHONY: all
all: clean deps lint test build ## Run full build pipeline

.PHONY: quick
quick: fmt test build ## Quick build without linting

.PHONY: pre-commit
pre-commit: fmt lint test ## Run pre-commit checks
	@echo "Pre-commit checks passed! ✅"

.PHONY: pre-push
pre-push: check bench ## Run pre-push checks
	@echo "Pre-push checks passed! ✅"
