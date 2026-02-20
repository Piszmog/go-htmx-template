#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
	echo "Usage: $0 <new module name>"
	exit 1
fi

new_module=$1

# Only process known text file types to avoid corrupting binaries.
find . -type d -name .git -prune -o \
	-type f \( -name '*.go' -o -name '*.mod' -o -name '*.sum' -o -name '*.templ' \
	-o -name '*.sql' -o -name '*.sh' -o -name '*.md' -o -name '*.yml' -o -name '*.yaml' \
	-o -name '*.json' -o -name '*.toml' -o -name '*.css' -o -name '*.js' -o -name '*.html' \
	-o -name '*.txt' -o -name 'Dockerfile' -o -name 'Makefile' \) \
	-print0 | while IFS= read -r -d '' file; do
	sed -e "s|go-htmx-template|${new_module}|g" "$file" > "$file.tmp" && mv "$file.tmp" "$file"
done
