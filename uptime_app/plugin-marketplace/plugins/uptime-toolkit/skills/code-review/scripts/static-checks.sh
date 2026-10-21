#!/usr/bin/env bash
set -uo pipefail

usage() {
  echo "Usage: bash static-checks.sh [--root /path/to/uptime_app]"
}

app_root="$PWD"
case "${1:-}" in
  --help|-h) usage; exit 0 ;;
  --root)
    if [[ $# -ne 2 || ! -d "$2" ]]; then usage >&2; exit 2; fi
    app_root="$(cd -- "$2" && pwd)"
    ;;
  '') ;;
  *) usage >&2; exit 2 ;;
esac
if [[ ! -f "$app_root/backend/go.mod" || ! -f "$app_root/frontend/package.json" ]]; then
  echo "Expected backend/go.mod and frontend/package.json at $app_root" >&2
  exit 2
fi

failed=0
run_check() {
  local label="$1"
  shift
  printf '\n== %s ==\n' "$label"
  if ( "$@" ); then
    printf 'PASS: %s\n' "$label"
  else
    local code=$?
    printf 'FAIL: %s (exit %s)\n' "$label" "$code" >&2
    failed=1
  fi
}
check_gofmt() {
  local files
  files="$(gofmt -l "$app_root/backend")" || return
  if [[ -n "$files" ]]; then
    printf '%s\n' "$files"
    echo "Go files above need gofmt; no files were modified." >&2
    return 1
  fi
}
check_vet() {
  cd -- "$app_root/backend" || return
  go vet ./...
}
check_frontend() {
  cd -- "$app_root/frontend" || return
  if [[ ! -d node_modules ]]; then
    echo "Frontend dependencies missing; npm ci is required before this check." >&2
    return 2
  fi
  npm run "$1"
}

printf 'Checking working tree at %s\n' "$app_root"
run_check 'Go formatting (read-only)' check_gofmt
run_check 'Go vet' check_vet
run_check 'Next.js ESLint' check_frontend lint
run_check 'TypeScript' check_frontend typecheck
printf '\nStatic checks finished; failures=%s\n' "$failed"
exit "$failed"
