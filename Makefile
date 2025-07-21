# s3dock Makefile
BINARY_NAME=s3dock
BINARY_UNIX=$(BINARY_NAME)_unix
VERSION?=latest
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD)
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make <target>'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the binary
	go build $(LDFLAGS) -o $(BINARY_NAME) .

.PHONY: build-linux
build-linux: ## Build binary for Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_UNIX) .

.PHONY: build-all
build-all: ## Build binaries for all platforms
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe .

.PHONY: test
test: ## Run unit tests
	go test -v ./...

.PHONY: test-short
test-short: ## Run unit tests (short mode)
	go test -short -v ./...

.PHONY: test-integration
test-integration: test-infra-up ## Run integration tests
	@echo "Waiting for test infrastructure..."
	@sleep 5
	INTEGRATION_TEST=1 go test -v ./... -run Integration
	$(MAKE) test-infra-down

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-infra-up
test-infra-up: ## Start test infrastructure (minio, etc.)
	docker compose -f docker-compose.test.yml up -d

.PHONY: test-infra-down
test-infra-down: ## Stop test infrastructure
	docker compose -f docker-compose.test.yml down

.PHONY: test-image
test-image: ## Build test Docker image
	docker build -f Dockerfile.test -t s3dock-test:latest .

.PHONY: clean
clean: ## Clean build artifacts
	go clean
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out coverage.html

.PHONY: deps
deps: ## Download dependencies
	go mod download
	go mod tidy

.PHONY: lint
lint: ## Run linter
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

.PHONY: fmt
fmt: ## Format code
	go fmt ./...
	goimports -w .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: check
check: fmt vet lint test-short ## Run all checks (fmt, vet, lint, test)

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t $(BINARY_NAME):$(VERSION) .

.PHONY: install
install: build ## Install binary to $GOPATH/bin
	go install $(LDFLAGS) .

.PHONY: run-example
run-example: build test-image test-infra-up ## Run example push command
	@echo "Waiting for test infrastructure..."
	@sleep 5
	@echo "Running example push command..."
	S3DOCK_BUCKET=s3dock-test \
	AWS_ACCESS_KEY_ID=testuser \
	AWS_SECRET_ACCESS_KEY=testpass123 \
	AWS_ENDPOINT_URL=http://localhost:9000 \
	AWS_REGION=us-east-1 \
	./$(BINARY_NAME) push s3dock-test:latest || echo "Push failed (expected without proper S3 setup)"
	$(MAKE) test-infra-down

.PHONY: release
release: check build-all ## Create release build with all platforms
	@echo "Release build complete:"
	@ls -la $(BINARY_NAME)-*

.DEFAULT_GOAL := help