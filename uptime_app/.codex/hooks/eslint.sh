#!/usr/bin/env bash
set -euo pipefail

hook_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
root=$(cd -- "$hook_dir/../.." && pwd -P)
source "$hook_dir/hook-paths.sh"

payload=$(cat)
# Unknown input still triggers lint, matching the previous hook behavior.
if ! event=$(printf '%s' "$payload" | hook_read_paths) || [[ -z "$event" ]]; then
    event='{"cwd":"","paths":[]}'
fi
cwd=$(printf '%s' "$event" | jq -r '.cwd')
cwd=${cwd:-$root}
run_lint=false
[[ $(printf '%s' "$event" | jq '.paths | length') != 0 ]] || run_lint=true
while IFS= read -r -d '' path; do
    [[ "$path" = /* ]] || path="$cwd/$path"
    absolute=$(hook_canonical "$path")
    case "$absolute" in
        "$root/frontend"|"$root/frontend/"*) run_lint=true ;;
    esac
done < <(printf '%s' "$event" | jq -j '.paths[] | ., "\u0000"')

if [[ "$run_lint" = true ]]; then
    if ! (cd -- "$root/frontend" && npm run lint) >&2; then
        exit 2
    fi
fi
printf '{}\n'
