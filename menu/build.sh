#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
mkdir -p menu/bin
echo "  Building menu …"
go build -o "menu/bin/genres" "./menu"
