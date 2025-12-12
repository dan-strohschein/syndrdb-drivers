#!/bin/bash
# Build script for SyndrDB Go library

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Build configuration
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"
LDFLAGS="-X github.com/dan-strohschein/syndrdb-drivers/src/golang/client.Version=$VERSION"

echo "Building SyndrDB Go Driver..."
echo "Version: $VERSION"

cd "$PROJECT_ROOT"

# Run tests
echo "Running tests..."
go test ./... -v

# Build library (verify compilation)
echo "Building library..."
go build -ldflags "$LDFLAGS" ./...

# Run go vet
echo "Running go vet..."
go vet ./...

# Format code
echo "Formatting code..."
go fmt ./...

echo ""
echo "Build complete!"
echo "Version: $VERSION"
echo ""
echo "To use in your project:"
echo "  go get github.com/dan-strohschein/syndrdb-drivers/src/golang"
