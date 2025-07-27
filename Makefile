BINARY_NAME=zcd
BUILD_DIR=bin
SOURCE_DIR=.

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w"

.PHONY: all build clean test deps install uninstall help

all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(SOURCE_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Install the binary to system path
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "$(BINARY_NAME) installed to /usr/local/bin/"
	@echo ""
	@echo "To enable shell integration, add this to your shell config:"
	@echo "  source $(PWD)/shell_integration.sh"

# Uninstall the binary
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "$(BINARY_NAME) uninstalled"

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(SOURCE_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(SOURCE_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(SOURCE_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(SOURCE_DIR)
	@echo "Multi-platform build complete"

# Development build with debug info
dev:
	@echo "Building development version..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(SOURCE_DIR)
	@echo "Development build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the application
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Show help
help:
	@echo "Available targets:"
	@echo "  build      - Build the binary"
	@echo "  clean      - Clean build artifacts"
	@echo "  test       - Run tests"
	@echo "  deps       - Download dependencies"
	@echo "  install    - Install to system path"
	@echo "  uninstall  - Remove from system path"
	@echo "  build-all  - Build for multiple platforms"
	@echo "  dev        - Build development version"
	@echo "  run        - Build and run"
	@echo "  help       - Show this help"
