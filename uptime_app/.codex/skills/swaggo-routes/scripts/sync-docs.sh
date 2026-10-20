#!/usr/bin/env bash
set -euo pipefail

# This skill lives at <application>/.codex/skills/swaggo-routes/.
app_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../../.." && pwd)"

if [[ ! -f "$app_root/backend/go.mod" || ! -f "$app_root/Makefile" ]]; then
  echo "Expected uptime_app with backend/go.mod and Makefile at $app_root" >&2
  exit 1
fi

# Separate invocations preserve order even when MAKEFLAGS enables parallel jobs.
make -C "$app_root" swagger
make -C "$app_root" redoc
cd "$app_root/backend"
go test ./internal/httpapi
