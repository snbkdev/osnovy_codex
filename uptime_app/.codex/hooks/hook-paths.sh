#!/usr/bin/env bash
# Shared JSON input parsing and path resolution for file-edit hooks.
hook_read_paths() {
    jq -c '
        def paths:
            if type == "object" then
                ([.file_path, .path, .destination, .new_path][] | select(type == "string")),
                (.edits[]? | paths)
            else empty end;
        . as $event | .tool_input as $input |
        {cwd: ($event.cwd // ""), paths: [
            ($input | paths),
            (($input | if type == "string" then . else (.patch // .input // "") end)
                | split("\n")[]
                | capture("^\\*\\*\\* (?:Add File|Update File|Delete File|Move to): (?<path>.+)$")
                | .path)
        ]}'
}

hook_canonical() {
    local path="$1" parent name
    if [[ -d "$path" ]]; then
        (cd -- "$path" && pwd -P)
    elif [[ -e "$path" || -L "$path" ]]; then
        if [[ -L "$path" ]]; then
            name=$(readlink "$path") || return
            [[ "$name" = /* ]] || name="$(dirname "$path")/$name"
            hook_canonical "$name"
        else
            parent=$(hook_canonical "$(dirname "$path")") || return
            printf '%s/%s\n' "${parent%/}" "$(basename "$path")"
        fi
    else
        parent=$(hook_canonical "$(dirname "$path")") || return
        name=$(basename "$path")
        case "$name" in
            .) printf '%s\n' "$parent" ;;
            ..) dirname "$parent" ;;
            *) printf '%s/%s\n' "${parent%/}" "$name" ;;
        esac
    fi
}
