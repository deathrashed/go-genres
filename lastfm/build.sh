#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
NAME="genres-lastfm"
mkdir -p "lastfm/bin"
echo "  Building lastfm …"
go build -o "lastfm/bin/$NAME" "./lastfm"
