#!/bin/bash

# Check if version argument is provided
if [ -z "$1" ]; then
    echo "Error: Version number required"
    echo "Usage: $0 VERSION"
    echo "Example: $0 1.0.0"
    exit 1
fi

VERSION="v$1"  # Add 'v' prefix for consistency
mkdir -p releases

# Build for Intel Macs
GOOS=darwin GOARCH=amd64 go build -o "releases/macbuds-$VERSION-darwin-amd64" .

# Build for Apple Silicon Macs
GOOS=darwin GOARCH=arm64 go build -o "releases/macbuds-$VERSION-darwin-arm64" .

echo "Built binaries for macOS:"
echo "  releases/macbuds-$VERSION-darwin-amd64"
echo "  releases/macbuds-$VERSION-darwin-arm64"