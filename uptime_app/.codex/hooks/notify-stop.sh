#!/usr/bin/env bash
set -euo pipefail

# Consume the hook payload; notification text does not include tool output.
cat >/dev/null

if [[ "$(uname -s)" = Darwin ]]; then
    if ! osascript -e 'display notification "Работа завершена в uptime_app." with title "Codex"'; then
        printf 'Could not send the macOS completion notification.\n' >&2
    fi
fi

printf '{}\n'
