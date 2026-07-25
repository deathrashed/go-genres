#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
NAME="genres-discogs"
mkdir -p "discogs/bin"
echo "  Building discogs …"
go build -o "discogs/bin/$NAME" "./discogs"
