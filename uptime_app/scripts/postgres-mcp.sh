#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../backend"
env_file="${BACKEND_ENV_FILE:-./.env}"
if [[ ! -f "$env_file" ]]; then
    printf 'Backend environment file not found: %s\n' "$env_file" >&2
    exit 1
fi
source "$env_file"

# Keep credentials in the existing local env file, outside MCP configuration.
export DATABASE_URI="${DATABASE_URL:?DATABASE_URL must be set in backend/.env}"
# A container reaches the published local PostgreSQL port through the Docker host.
DATABASE_URI="${DATABASE_URI/@127.0.0.1:/@host.docker.internal:}"
DATABASE_URI="${DATABASE_URI/@localhost:/@host.docker.internal:}"
exec docker run --rm -i -e DATABASE_URI crystaldba/postgres-mcp:latest --access-mode=restricted
