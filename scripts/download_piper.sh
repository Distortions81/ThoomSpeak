#!/usr/bin/env bash
set -euo pipefail

# Determine repository root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PIPER_DIR="$ROOT_DIR/data/piper"
VOICE_DIR="$PIPER_DIR/voices"

mkdir -p "$PIPER_DIR" "$VOICE_DIR"

# Download piper binaries
PIPER_RELEASE="https://github.com/rhasspy/piper/releases/latest/download"
BINS=(
  "piper_linux_x86_64.tar.gz"
  "piper_linux_aarch64.tar.gz"
  "piper_linux_armv7l.tar.gz"
  "piper_macos_x64.tar.gz"
  "piper_macos_aarch64.tar.gz"
  "piper_windows_amd64.zip"
)

for f in "${BINS[@]}"; do
  echo "Downloading $f..."
  curl -L "$PIPER_RELEASE/$f" -o "$PIPER_DIR/$f"
done

# Download voices
VOICE_BASE="https://huggingface.co/rhasspy/piper-voices/resolve/main"
declare -A VOICES=(
  [en_US-hfc_female-medium]="en/en_US/hfc_female/medium"
  [en_US-hfc_male-medium]="en/en_US/hfc_male/medium"
)

for name in "${!VOICES[@]}"; do
  path="${VOICES[$name]}"
  vdir="$VOICE_DIR/$name"
  mkdir -p "$vdir"
  echo "Downloading voice $name..."
  curl -L "$VOICE_BASE/$path/$name.onnx" -o "$vdir/$name.onnx"
  curl -L "$VOICE_BASE/$path/$name.onnx.json" -o "$vdir/$name.onnx.json"
  curl -L "$VOICE_BASE/$path/MODEL_CARD" -o "$vdir/MODEL_CARD"
done

echo "Piper binaries downloaded to $PIPER_DIR and voices to $VOICE_DIR"
