#!/usr/bin/env bash
set -euo pipefail

# build_mac_app.sh - Build gothoom macOS .app and zip archive.
#
# Usage: scripts/build_mac_app.sh <version>
# Example: scripts/build_mac_app.sh v1.2.3
#
# Requires Go and (optionally) codesign if signing the bundle. If both amd64
# and arm64 builds succeed a universal binary is created via lipo.

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <version>" >&2
  exit 1
fi
VERSION="$1"

APP_NAME="gothoom"
BINARY_NAME="gothoom"
IDENTIFIER="com.goThoom.client"
APP_DIR="${APP_NAME}.app"

# 1. Build binaries for both architectures

echo "Building macOS amd64 binary..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=${VERSION}" -o "${BINARY_NAME}-amd64" .

echo "Building macOS arm64 binary..."
GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=${VERSION}" -o "${BINARY_NAME}-arm64" .

# 2. Combine into a universal binary

echo "Creating universal binary..."
if command -v llvm-lipo-14 >/dev/null 2>&1; then
  llvm-lipo-14 -create "${BINARY_NAME}-amd64" "${BINARY_NAME}-arm64" -output "${BINARY_NAME}"
else
  lipo -create "${BINARY_NAME}-amd64" "${BINARY_NAME}-arm64" -output "${BINARY_NAME}"
fi
rm -f "${BINARY_NAME}-amd64" "${BINARY_NAME}-arm64"

# 3. Create .app bundle

rm -rf "${APP_DIR}"
mkdir -p "${APP_DIR}/Contents/MacOS" "${APP_DIR}/Contents/Resources"
mv "${BINARY_NAME}" "${APP_DIR}/Contents/MacOS/${BINARY_NAME}"

cat <<PLIST > "${APP_DIR}/Contents/Info.plist"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>CFBundleName</key><string>${APP_NAME}</string>
    <key>CFBundleDisplayName</key><string>${APP_NAME}</string>
    <key>CFBundleExecutable</key><string>${BINARY_NAME}</string>
    <key>CFBundleIdentifier</key><string>${IDENTIFIER}</string>
    <key>CFBundleVersion</key><string>${VERSION}</string>
    <key>CFBundlePackageType</key><string>APPL</string>
    <key>CFBundleSignature</key><string>????</string>
    <key>CFBundleInfoDictionaryVersion</key><string>6.0</string>
  </dict>
</plist>
PLIST

# 4. Codesign if available

if command -v codesign >/dev/null 2>&1; then
  IDENTITY="${MAC_SIGN_IDENTITY:--}"
  echo "Codesigning with identity ${IDENTITY}..."
  codesign --force --deep --sign "${IDENTITY}" "${APP_DIR}" || echo "codesign failed" >&2
else
  echo "codesign not found; skipping signing" >&2
fi

# 5. Zip the .app bundle

ZIP_NAME="${APP_NAME}-Mac.zip"
rm -f "$ZIP_NAME"
zip -r "$ZIP_NAME" "${APP_DIR}"

# 6. Cleanup

rm -rf "${APP_DIR}"

echo "Built ${ZIP_NAME}" 
