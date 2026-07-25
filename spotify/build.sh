#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
NAME="genres-spotify"
mkdir -p "spotify/bin"
echo "  Building spotify …"
go build -o "spotify/bin/$NAME" "./spotify"
