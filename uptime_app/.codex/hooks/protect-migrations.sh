#!/usr/bin/env bash
set -euo pipefail

block() {
    printf '%s\n' "$1" >&2
    exit 2
}
trap 'block "Migration protection failed; operation denied."' ERR
hook_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
root=$(cd -- "$hook_dir/../.." && pwd -P)
source "$hook_dir/hook-paths.sh"

event=$(hook_read_paths)
[[ -n "$event" ]] || block "Cannot read hook input; operation denied."
[[ $(printf '%s' "$event" | jq '.paths | length') != 0 ]] ||
    block "Cannot determine edited paths; operation denied."
cwd=$(printf '%s' "$event" | jq -r '.cwd')
cwd=${cwd:-$root}
git_root=$(git -C "$root" rev-parse --show-toplevel)
git_root=$(hook_canonical "$git_root")
migrations="$root/backend/migrations"
committed=$(git -C "$git_root" ls-tree -r --name-only HEAD -- "${migrations#"$git_root/"}")

while IFS= read -r -d '' path; do
    [[ "$path" = /* ]] || path="$cwd/$path"
    absolute=$(hook_canonical "$path")
    while IFS= read -r file; do
        [[ -n "$file" ]] || continue
        if [[ "$git_root/$file" = "$absolute" || "$git_root/$file" = "$absolute/"* ]]; then
            block "Editing or deleting committed migrations is forbidden: $path. Create a new migration instead."
        fi
    done <<< "$committed"
done < <(printf '%s' "$event" | jq -j '.paths[] | ., "\u0000"')
printf '{}\n'
