#!/usr/bin/env bash
# Regenerate Dart gRPC stubs from models/*.proto.
# Output goes to frontend/lib/generated/ — never committed (see .gitignore).
#
# Prereqs:
#   - protoc on PATH (brew install protobuf)
#   - protoc-gen-dart on PATH (dart pub global activate protoc_plugin)

set -euo pipefail

cd "$(dirname "$0")/.."

if ! command -v protoc >/dev/null; then
  echo "error: protoc not found. Install with: brew install protobuf" >&2
  exit 1
fi

if ! command -v protoc-gen-dart >/dev/null; then
  echo "protoc-gen-dart not found, installing..."
  dart pub global activate protoc_plugin
  PUB_BIN="$(dart pub global list 2>/dev/null >/dev/null; echo "$HOME/.pub-cache/bin")"
  if ! command -v protoc-gen-dart >/dev/null; then
    export PATH="$PUB_BIN:$PATH"
    echo "Added $PUB_BIN to PATH for this run."
    echo "Consider adding it permanently to your shell rc."
  fi
fi

OUT=frontend/lib/generated
mkdir -p "$OUT"

protoc \
  --dart_out=grpc:"$OUT" \
  -I models \
  models/*.proto

echo "Generated stubs:"
ls -1 "$OUT"
