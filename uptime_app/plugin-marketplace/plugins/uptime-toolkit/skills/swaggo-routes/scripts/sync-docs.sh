#!/usr/bin/env bash
set -euo pipefail

# Run from the application root or pass --root /path/to/uptime_app.
app_root="$PWD"
case "${1:-}" in
  --help|-h) echo "Usage: bash sync-docs.sh [--root /path/to/uptime_app]"; exit 0 ;;
  --root)
    if [[ $# -ne 2 || ! -d "$2" ]]; then echo "Expected --root /path/to/uptime_app" >&2; exit 2; fi
    app_root="$(cd -- "$2" && pwd)"
    ;;
  '') ;;
  *) echo "Usage: bash sync-docs.sh [--root /path/to/uptime_app]" >&2; exit 2 ;;
esac

if [[ ! -f "$app_root/backend/go.mod" || ! -f "$app_root/Makefile" ]]; then
  echo "Expected uptime_app with backend/go.mod and Makefile at $app_root" >&2
  exit 1
fi

# Separate invocations preserve order even when MAKEFLAGS enables parallel jobs.
make -C "$app_root" swagger
make -C "$app_root" redoc
cd "$app_root/backend"
go test ./internal/httpapi
