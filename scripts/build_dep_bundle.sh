#!/usr/bin/env bash
# Build a tarball containing goThoom's system and Go module dependencies.
# The resulting archive can be unpacked on another machine to speed up
# environment setup without hitting the network.

set -euo pipefail

# Usage: scripts/build_dep_bundle.sh [output.tar.gz]
# Default output is gothoom_deps.tar.gz in the current directory.

OUT_FILE="${1:-gothoom_deps.tar.gz}"
WORK_DIR="$(mktemp -d)"
APT_DIR="$WORK_DIR/apt"
GO_DIR="$WORK_DIR/go"

mkdir -p "$APT_DIR" "$GO_DIR"

# Packages needed to build goThoom.
DEB_PACKAGES=(
  golang-go
  build-essential
  libgl1-mesa-dev
  libglu1-mesa-dev
  xorg-dev
  libxrandr-dev
  libasound2-dev
  libgtk-3-dev
)

if command -v apt-get >/dev/null 2>&1; then
  echo "Downloading Debian packages..."
  apt-get update -qq
  (
    cd "$APT_DIR"
     apt-get -o APT::Sandbox::User=root -qq download "${DEB_PACKAGES[@]}"
  )
else
  echo "apt-get not found; skipping Debian package download" >&2
fi

# Cache Go modules into a local mod cache inside the bundle.
GO_CACHE="$GO_DIR/mod"
mkdir -p "$GO_CACHE"

echo "Downloading Go modules..."
GOMODCACHE="$GO_CACHE" /usr/local/go/bin/go mod download

# Create archive.

tar -czf "$OUT_FILE" -C "$WORK_DIR" .

echo "Dependency bundle written to $OUT_FILE"
