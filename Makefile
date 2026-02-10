# Makefile for Slack MCP Server

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint

# Binary name
BINARY_NAME=slack-mcp-server
BINARY_PATH=./$(BINARY_NAME)

# Build flags
LDFLAGS=-ldflags "-X main.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo 'dev') -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) ./cmd/server

# Run the server (requires SLACK_BOT_TOKEN environment variable)
.PHONY: run
run: build
	$(BINARY_PATH)

# Run all tests
.PHONY: test
test:
	$(GOTEST) -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Run linter
.PHONY: lint
lint:
	$(GOLINT) run ./...

# Format code
.PHONY: fmt
fmt:
	$(GOCMD) fmt ./...

# Tidy dependencies
.PHONY: tidy
tidy:
	$(GOMOD) tidy

# Clean build artifacts
.PHONY: clean
clean:
	$(GOCLEAN)
	rm -f $(BINARY_PATH)
	rm -f coverage.out coverage.html

# Verify the MCP server responds to tools/list
.PHONY: verify
verify: build
	@echo "Testing MCP tools/list response..."
	@echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | $(BINARY_PATH)

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build         - Build the slack-mcp-server binary"
	@echo "  run           - Build and run the server"
	@echo "  test          - Run all tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  lint          - Run golangci-lint"
	@echo "  fmt           - Format Go code"
	@echo "  tidy          - Tidy go.mod dependencies"
	@echo "  clean         - Remove build artifacts"
	@echo "  verify        - Test MCP tools/list response"
	@echo "  help          - Show this help message"
