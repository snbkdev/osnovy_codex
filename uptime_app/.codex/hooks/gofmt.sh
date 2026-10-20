set -euo pipefail

cat >/dev/null

command -v gofmt >/dev/null 2>&1 || {echo '{}'; exit 0;}

while IFS= read -r file; do
    [ -f "$file" ] && gofmt -w "$file"
done < <(git diff --name-only --diff-filter=ACM 2>/dev/null | grep '\.go$' || true)


echo '{}'