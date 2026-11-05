#!/usr/bin/env bash
# Photon - Linux Production Build
#
# Usage:
#   ./build-linux.sh            build the production binary  -> bin/Photon
#   ./build-linux.sh package    binary + AppImage, .deb, .rpm and Arch package -> bin/
#
# Run it from the project root (the folder that contains Taskfile.yml).

set -u

MODE="${1:-binary}"

echo "============================================"
echo "  Photon - Linux Production Build"
echo "============================================"
echo

case "$MODE" in
    binary|package) ;;
    -h|--help)
        sed -n '2,8p' "$0" | sed 's/^# \{0,1\}//'
        exit 0
        ;;
    *)
        echo "[ERROR] Unknown option: $MODE"
        echo "Usage: $0 [binary|package]"
        exit 1
        ;;
esac

# Always run from the folder this script lives in
cd "$(dirname "$0")" || exit 1

if [ ! -f Taskfile.yml ]; then
    echo "[ERROR] Taskfile.yml not found. Run this script from the Photon project root."
    exit 1
fi

# Make sure the wails3 CLI is available
if ! command -v wails3 >/dev/null 2>&1; then
    # 'go install' puts it in GOPATH/bin, which is often not on PATH
    GOBIN_DIR="$(go env GOPATH 2>/dev/null)/bin"
    if [ -x "$GOBIN_DIR/wails3" ]; then
        export PATH="$PATH:$GOBIN_DIR"
    else
        echo "[ERROR] wails3 CLI not found in PATH."
        echo "Install it with: go install github.com/wailsapp/wails/v3/cmd/wails3@latest"
        echo "Then make sure \$(go env GOPATH)/bin is in your PATH."
        exit 1
    fi
fi

# Make sure Go is available
if ! command -v go >/dev/null 2>&1; then
    echo "[ERROR] Go not found in PATH. Install it from https://go.dev/dl/"
    exit 1
fi

# Make sure Node/npm is available for the frontend build
if ! command -v npm >/dev/null 2>&1; then
    echo "[ERROR] npm not found in PATH. Install Node.js from https://nodejs.org/"
    exit 1
fi

# The Linux build uses CGO (GTK + WebKit), so a C compiler and pkg-config are required
if ! command -v gcc >/dev/null 2>&1; then
    echo "[ERROR] gcc not found in PATH. Install a C compiler (e.g. build-essential)."
    exit 1
fi
if ! command -v pkg-config >/dev/null 2>&1; then
    echo "[ERROR] pkg-config not found in PATH. Install it (e.g. sudo apt install pkg-config)."
    exit 1
fi

# GTK3 and WebKitGTK development packages
if ! pkg-config --exists gtk+-3.0; then
    echo "[ERROR] GTK 3 development files not found."
    echo "Debian/Ubuntu: sudo apt install libgtk-3-dev"
    echo "Fedora:        sudo dnf install gtk3-devel"
    echo "Arch:          sudo pacman -S gtk3"
    exit 1
fi
if ! pkg-config --exists webkit2gtk-4.1 && ! pkg-config --exists webkit2gtk-4.0; then
    echo "[ERROR] WebKitGTK development files not found."
    echo "Debian/Ubuntu: sudo apt install libwebkit2gtk-4.1-dev"
    echo "Fedora:        sudo dnf install webkit2gtk4.1-devel"
    echo "Arch:          sudo pacman -S webkit2gtk-4.1"
    exit 1
fi

if [ "$MODE" = "package" ]; then
    echo "Running: wails3 task linux:package"
    echo
    wails3 task linux:package
    STATUS=$?
else
    echo "Running: wails3 task linux:build PRODUCTION=true"
    echo
    wails3 task linux:build PRODUCTION=true
    STATUS=$?
fi

if [ "$STATUS" -ne 0 ]; then
    echo
    echo "[ERROR] Build failed."
    exit 1
fi

echo
echo "============================================"
echo "  Build finished successfully."
echo "  Check the bin/ folder for the output."
echo "============================================"
