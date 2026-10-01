#!/bin/sh
# dev.sh — local development runner with Go live-reload (no external deps).
# Watches .go files and rebuilds/restarts the server.
set -e
cd /app

go mod download
BIN=/tmp/profilepulse-main

build() {
  go build -o "$BIN" ./cmd/server
}

echo "[dev] building..."
build
echo "[dev] starting server..."
"$BIN" &
PID=$!

while true; do
  sleep 2
  CHANGED=$(find . -path ./.git -prune -o -path ./tmp -prune -o -name '*.go' -newer "$BIN" -print 2>/dev/null | head -1)
  if [ -n "$CHANGED" ]; then
    echo "[dev] Go change detected, rebuilding..."
    if build; then
      kill "$PID" 2>/dev/null
      wait "$PID" 2>/dev/null
      "$BIN" &
      PID=$!
      echo "[dev] restarted"
    else
      echo "[dev] build failed, keeping previous binary running"
    fi
  fi
done
