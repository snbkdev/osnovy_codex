#!/usr/bin/env bash
set -e

# Resolve relative env-file paths from backend/, just as make dev does.
cd "$(dirname "$0")/../backend"
set -a
env_file="${BACKEND_ENV_FILE:-./.env}"
if [ -f "$env_file" ]; then
  . "$env_file"
elif [ -n "${BACKEND_ENV_FILE:-}" ]; then
  echo "Backend environment file not found: $env_file" >&2
  exit 1
fi
set +a
exec "$@"
