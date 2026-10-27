#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
	echo "Usage: $0 <new module name>"
	exit 1
fi

new_module=$1

# Escape characters that are special to the sed replacement below.
escaped_module=$(printf '%s' "$new_module" | sed -e 's/[\\&|]/\\&/g')

# Only process known text file types to avoid corrupting binaries. Exclude the
# generated asset bundle (large, no module references) and this script itself:
# rewriting the running script can corrupt bash's read buffer and drops its
# executable bit.
find . \
	-type d \( -name .git -o -name tmp -o -name goreleaser-dist -o -path './internal/dist/assets' \) -prune -o \
	-type f ! -name update_module.sh \( -name '*.go' -o -name '*.mod' -o -name '*.sum' -o -name '*.templ' \
	-o -name '*.sql' -o -name '*.sh' -o -name '*.md' -o -name '*.yml' -o -name '*.yaml' \
	-o -name '*.json' -o -name '*.toml' -o -name '*.css' -o -name '*.js' -o -name '*.html' \
	-o -name '*.txt' -o -name 'Dockerfile' -o -name 'justfile' \) \
	-print0 | while IFS= read -r -d '' file; do
	# cp -p preserves modes; redirecting into the copy keeps them.
	cp -p "$file" "$file.tmp"
	sed -e "s|go-htmx-template|${escaped_module}|g" "$file" > "$file.tmp"
	mv "$file.tmp" "$file"
done
