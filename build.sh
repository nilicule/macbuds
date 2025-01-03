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

# Check if blueutil is installed
if ! command -v blueutil &> /dev/null; then
    echo "Error: blueutil is not installed"
    echo "Please install it using: brew install blueutil"
    exit 1
fi

# Function to create macOS app bundle
create_app_bundle() {
    local binary_path=$1
    local app_name=$2
    local arch=$3

    # Create app bundle directory structure
    local app_bundle="releases/MacBuds-$VERSION-$arch.app"
    mkdir -p "$app_bundle/Contents/MacOS"
    mkdir -p "$app_bundle/Contents/Resources"

    # Copy binary
    cp "$binary_path" "$app_bundle/Contents/MacOS/MacBuds"
    chmod +x "$app_bundle/Contents/MacOS/MacBuds"

    # Copy blueutil
    cp "$(which blueutil)" "$app_bundle/Contents/MacOS/blueutil"
    chmod +x "$app_bundle/Contents/MacOS/blueutil"

    # Create Info.plist
    cat > "$app_bundle/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>MacBuds</string>
    <key>CFBundleDisplayName</key>
    <string>MacBuds</string>
    <key>CFBundleIdentifier</key>
    <string>org.rc6.macbuds</string>
    <key>CFBundleVersion</key>
    <string>${VERSION#v}</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION#v}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleSignature</key>
    <string>????</string>
    <key>CFBundleExecutable</key>
    <string>MacBuds</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.13</string>
    <key>LSUIElement</key>
    <true/>
</dict>
</plist>
EOF

    echo "Created app bundle: $app_bundle"
}

# Build for Intel Macs
GOOS=darwin GOARCH=amd64 go build -o "releases/macbuds-$VERSION-darwin-amd64" .
create_app_bundle "releases/macbuds-$VERSION-darwin-amd64" "MacBuds" "amd64"
rm "releases/macbuds-$VERSION-darwin-amd64"  # Remove the standalone binary

# Build for Apple Silicon Macs
GOOS=darwin GOARCH=arm64 go build -o "releases/macbuds-$VERSION-darwin-arm64" .
create_app_bundle "releases/macbuds-$VERSION-darwin-arm64" "MacBuds" "arm64"
rm "releases/macbuds-$VERSION-darwin-arm64"  # Remove the standalone binary

echo "Built app bundles for macOS:"
echo "  releases/MacBuds-$VERSION-amd64.app"
echo "  releases/MacBuds-$VERSION-arm64.app"