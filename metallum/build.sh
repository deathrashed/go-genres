#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
NAME="genres-metallum"
mkdir -p "metallum/bin"
echo "  Building metallum …"
go build -o "metallum/bin/$NAME" "./metallum"
