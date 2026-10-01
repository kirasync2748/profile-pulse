#!/usr/bin/env bash
# Run the development server with live Go source (go run).
set -euo pipefail

export ADDR="${ADDR:-:3000}"
echo "→ starting ProfilePulse on ${ADDR} ..."
exec go run ./cmd/server
