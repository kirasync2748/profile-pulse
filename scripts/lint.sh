#!/usr/bin/env bash
# Run golangci-lint if it is installed.
set -euo pipefail

if ! command -v golangci-lint &>/dev/null; then
  echo "golangci-lint not installed; skipping"
  echo "  install: https://golangci-lint.run/usage/install/"
  exit 0
fi

echo "→ golangci-lint"
golangci-lint run ./...
echo "✓ lint passed"
