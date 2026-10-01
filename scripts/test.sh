#!/usr/bin/env bash
# Run all quality gates: format check, vet, and tests with the race detector.
set -euo pipefail

echo "→ gofmt check"
fmt="$(gofmt -l .)"
if [ -n "$fmt" ]; then
  echo "gofmt: the following files need formatting:"
  echo "$fmt"
  exit 1
fi

echo "→ go vet"
go vet ./...

echo "→ go test (race)"
go test ./... -race -count=1

echo "✓ all checks passed"
