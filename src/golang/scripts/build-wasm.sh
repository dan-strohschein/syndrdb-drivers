#!/bin/bash
# Build script for SyndrDB WASM binary

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
WASM_DIR="$PROJECT_ROOT/wasm"

# Build configuration
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"
LDFLAGS="-s -w -X github.com/dan-strohschein/syndrdb-drivers/src/golang/client.Version=$VERSION"

echo "Building SyndrDB WASM Driver..."
echo "Version: $VERSION"

cd "$WASM_DIR"

# Set WASM build environment
export GOOS=js
export GOARCH=wasm

# Build WASM binary
echo "Compiling WASM binary..."
go build -ldflags "$LDFLAGS" -o syndrdb.wasm main.go

# Get file size
WASM_SIZE=$(du -h syndrdb.wasm | cut -f1)
echo "WASM binary size: $WASM_SIZE"

# Compress for web delivery
echo "Compressing for web delivery..."
gzip -9 -k -f syndrdb.wasm

GZIP_SIZE=$(du -h syndrdb.wasm.gz | cut -f1)
echo "Compressed size: $GZIP_SIZE"

# Copy wasm_exec.js if not present
if [ ! -f "$WASM_DIR/wasm_exec.js" ]; then
    echo "Copying wasm_exec.js..."
    GOROOT=$(go env GOROOT)
    if [ -f "$GOROOT/misc/wasm/wasm_exec.js" ]; then
        cp "$GOROOT/misc/wasm/wasm_exec.js" "$WASM_DIR/"
        echo "✓ wasm_exec.js copied"
    else
        echo "⚠ Warning: wasm_exec.js not found in GOROOT"
    fi
fi

echo ""
echo "WASM build complete!"
echo "Version: $VERSION"
echo "Binary: $WASM_DIR/syndrdb.wasm ($WASM_SIZE)"
echo "Compressed: $WASM_DIR/syndrdb.wasm.gz ($GZIP_SIZE)"
echo ""
echo "Usage:"
echo "  1. Copy wasm_exec.js and syndrdb.wasm to your web directory"
echo "  2. See wasm/README.md for integration examples"
