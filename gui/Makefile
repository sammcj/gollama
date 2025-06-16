# Gollama GUI Makefile

# Build variables
BINARY_NAME=gollama-gui
BUILD_DIR=build
MACOS_VERSION=15.0

# Go build flags for macOS
CGO_ENABLED=1
CGO_CFLAGS=-mmacosx-version-min=$(MACOS_VERSION)
CGO_LDFLAGS=-mmacosx-version-min=$(MACOS_VERSION)

.PHONY: all build clean dev assets test

# Default target
all: assets build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	CGO_ENABLED=$(CGO_ENABLED) CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" go build -o $(BINARY_NAME)

# Build assets (CSS and JS)
assets:
	@echo "Building assets..."
	go run $(BUILD_DIR)/build.go

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f static/css/tailwind.css
	rm -f static/js/htmx.min.js
	go clean -cache

# Development build with assets
dev: assets build

# Run the application
run: dev
	./$(BINARY_NAME)

# Test the build
test:
	go test ./...

# Install dependencies
deps:
	go mod tidy
	go mod download

# Help
help:
	@echo "Available targets:"
	@echo "  all     - Build assets and application (default)"
	@echo "  build   - Build the application binary"
	@echo "  assets  - Build CSS and JS assets"
	@echo "  clean   - Clean build artifacts"
	@echo "  dev     - Development build (assets + binary)"
	@echo "  run     - Build and run the application"
	@echo "  test    - Run tests"
	@echo "  deps    - Install/update dependencies"
	@echo "  help    - Show this help message"
