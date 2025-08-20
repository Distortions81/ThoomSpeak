#!/usr/bin/env bash
set -euo pipefail

# macos_create_cert.sh - Generate a self-signed codesigning certificate for macOS.
#
# This script must be run on macOS with the `openssl` and `security` tools
# available. It creates a new certificate and private key, imports them into
# the specified keychain, and grants codesign access.
#
# Usage: scripts/macos_create_cert.sh [cert-name]
# Example: scripts/macos_create_cert.sh gothoom-dev

CERT_NAME="${1:-gothoom-dev}"
KEYCHAIN="${MAC_KEYCHAIN:-login.keychain}"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

openssl req -new -newkey rsa:2048 -nodes -keyout "$TMPDIR/${CERT_NAME}.key" \
  -x509 -days 3650 -out "$TMPDIR/${CERT_NAME}.cer" -subj "/CN=${CERT_NAME}"

security import "$TMPDIR/${CERT_NAME}.cer" -k "$KEYCHAIN" -T /usr/bin/codesign >/dev/null
security import "$TMPDIR/${CERT_NAME}.key" -k "$KEYCHAIN" -T /usr/bin/codesign >/dev/null
security set-key-partition-list -S apple-tool:,apple: -s -k "" "$KEYCHAIN" >/dev/null

echo "Created self-signed certificate '${CERT_NAME}' in $KEYCHAIN."
