#!/usr/bin/env bash
# Build a stripped, static binary into ./bin/profilepulse.
# Usage: scripts/build.sh [version]
set -euo pipefail

VERSION="${1:-dev}"
LDFLAGS="-s -w -X main.version=${VERSION}"

mkdir -p bin
CGO_ENABLED=0 go build -ldflags="$LDFLAGS" -o bin/profilepulse ./cmd/server
echo "✓ binary built at bin/profilepulse"
