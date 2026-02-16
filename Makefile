# Build variables
VERSION?=latest
BUILD_DIR=build
BINARY_NAME=app
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"
BUILD_FLAGS=-trimpath -installsuffix cgo -tags sqlite_fts5
LINT_BIN_PATH?=$(shell go env GOPATH)/bin

.PHONY: build build-linux run test clean docker-build docker-up docker-down docker-logs fmt lint install-lint deps help openapi client-gen frontend-build

# Build for Linux (Docker).
build-linux:
	GOOS=linux make build

# Build the application.
build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build \
		$(BUILD_FLAGS) \
		$(LDFLAGS) \
		-o $(BUILD_DIR)/$(BINARY_NAME) \
		./cmd/app

# Run the application locally.
run: build
	./$(BUILD_DIR)/$(BINARY_NAME) --config config/config.yaml

# Run tests.
test:
	CGO_ENABLED=1 go test -v -tags sqlite_fts5 ./...

# Clean build artifacts.
clean:
	rm -rf bin/
	rm -rf $(BUILD_DIR)/$(BINARY_NAME)

# Docker
IMAGE ?= kenaz:$(VERSION)

docker-build: build-linux
	docker build -t $(IMAGE) -f $(BUILD_DIR)/Dockerfile .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Install dependencies.
deps:
	go mod download
	go mod tidy

# Format code.
fmt:
	@if ! command -v gofumpt > /dev/null; then \
		echo "gofumpt is not installed. Installing..."; \
		go install mvdan.cc/gofumpt@latest; \
	fi
	gofumpt -l -w .

# Install golangci-lint.
install-lint:
	@echo "Installing golangci-lint to $(LINT_BIN_PATH)..."
	@rm -f $(LINT_BIN_PATH)/golangci-lint
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LINT_BIN_PATH) latest
	@$(LINT_BIN_PATH)/golangci-lint version

# Run linter.
lint:
	@if [ -f "$(LINT_BIN_PATH)/golangci-lint" ]; then \
		$(LINT_BIN_PATH)/golangci-lint run ./... --new-from-rev=origin/main; \
	elif command -v golangci-lint > /dev/null; then \
		golangci-lint run ./... --new-from-rev=origin/main; \
	else \
		echo "golangci-lint is not installed. Run 'make install-lint' first."; \
		exit 1; \
	fi

# Generate TypeScript types from OpenAPI spec.
openapi: client-gen

# Generate typed frontend client from OpenAPI spec.
client-gen:
	cd frontend && npm run generate

# Build the frontend.
frontend-build:
	cd frontend && npm run build

# Help.
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  build-linux   - Build for Linux (Docker)"
	@echo "  run           - Run locally"
	@echo "  test          - Run tests"
	@echo "  lint          - Run golangci-lint"
	@echo "  install-lint  - Install golangci-lint"
	@echo "  fmt           - Format code with gofumpt"
	@echo "  clean         - Clean build artifacts"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-up     - Start containers"
	@echo "  docker-down   - Stop containers"
	@echo "  docker-logs   - View logs"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  openapi       - Generate TypeScript types from OpenAPI spec"
	@echo "  client-gen    - Generate typed frontend client"
	@echo "  frontend-build - Build frontend"
	@echo "  help          - Show this help"
