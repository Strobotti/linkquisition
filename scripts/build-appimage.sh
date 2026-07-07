#!/bin/bash
# Build an AppImage for Linkquisition.
# Requirements: linuxdeploy (downloaded automatically if not present)
# Usage: VERSION=1.0.0 ./scripts/build-appimage.sh
#
# This script expects the binary and plugins to already be built:
#   - bin/linkquisition-linux-amd64
#   - package/linux/usr/lib/linkquisition/plugins/*.so (from task build:plugins)

set -euo pipefail

VERSION="${VERSION:-0.0.0}"
ARCH="${ARCH:-x86_64}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

APPDIR="$PROJECT_ROOT/dist/AppDir"
DIST_DIR="$PROJECT_ROOT/dist"
TOOLS_DIR="$PROJECT_ROOT/dist/tools"

echo "==> Building Linkquisition AppImage v${VERSION} (${ARCH})"

# Verify binary exists
if [ ! -f "$PROJECT_ROOT/bin/linkquisition-linux-amd64" ]; then
    echo "ERROR: bin/linkquisition-linux-amd64 not found. Run 'task build' first."
    exit 1
fi

# Verify plugins exist
if ! ls "$PROJECT_ROOT/package/linux/usr/lib/linkquisition/plugins/"*.so >/dev/null 2>&1; then
    echo "ERROR: No plugins found. Run 'task build:plugins' first."
    exit 1
fi

# Download linuxdeploy if not present
mkdir -p "$TOOLS_DIR"
if [ ! -f "$TOOLS_DIR/linuxdeploy-x86_64.AppImage" ]; then
    echo "==> Downloading linuxdeploy..."
    wget -q -O "$TOOLS_DIR/linuxdeploy-x86_64.AppImage" \
        "https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/linuxdeploy-x86_64.AppImage"
    chmod +x "$TOOLS_DIR/linuxdeploy-x86_64.AppImage"
fi

# Clean and create AppDir structure
rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin"
mkdir -p "$APPDIR/usr/lib/linkquisition/plugins"
mkdir -p "$APPDIR/usr/share/applications"
mkdir -p "$APPDIR/usr/share/icons/hicolor/256x256/apps"
mkdir -p "$APPDIR/usr/share/man/man1"
mkdir -p "$APPDIR/usr/share/man/man5"

# Install binary
cp "$PROJECT_ROOT/bin/linkquisition-linux-amd64" "$APPDIR/usr/bin/linkquisition"

# Install plugins
cp "$PROJECT_ROOT/package/linux/usr/lib/linkquisition/plugins/"*.so \
    "$APPDIR/usr/lib/linkquisition/plugins/"

# Install desktop file
cp "$PROJECT_ROOT/templates/linkquisition.desktop" "$APPDIR/usr/share/applications/"

# Install icon (must be exactly 256x256 for linuxdeploy)
cp "$PROJECT_ROOT/Icon.png" "$APPDIR/usr/share/icons/hicolor/256x256/apps/linkquisition.png"

# Install man pages if available
if [ -f "$PROJECT_ROOT/package/linux/usr/share/man/man1/linkquisition.1" ]; then
    cp "$PROJECT_ROOT/package/linux/usr/share/man/man1/linkquisition.1" "$APPDIR/usr/share/man/man1/"
fi
if [ -f "$PROJECT_ROOT/package/linux/usr/share/man/man5/linkquisition.5" ]; then
    cp "$PROJECT_ROOT/package/linux/usr/share/man/man5/linkquisition.5" "$APPDIR/usr/share/man/man5/"
fi

# Run linuxdeploy to bundle shared library dependencies and create AppImage
# --appimage-extract-and-run avoids FUSE requirement (needed in CI/containers)
echo "==> Running linuxdeploy to bundle dependencies..."
mkdir -p "$DIST_DIR"

export OUTPUT="$DIST_DIR/Linkquisition-${VERSION}-${ARCH}.AppImage"
export VERSION

"$TOOLS_DIR/linuxdeploy-x86_64.AppImage" --appimage-extract-and-run \
    --appdir "$APPDIR" \
    --executable "$APPDIR/usr/bin/linkquisition" \
    --desktop-file "$APPDIR/usr/share/applications/linkquisition.desktop" \
    --icon-file "$APPDIR/usr/share/icons/hicolor/256x256/apps/linkquisition.png" \
    --output appimage

echo "==> AppImage created: $OUTPUT"
