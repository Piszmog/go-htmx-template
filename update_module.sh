#!/bin/sh

if [ "$#" -ne 1 ]; then
	echo "Usage: $0 <new module name>"
	exit 1
fi

new_module=$1

find . -type d -name .git -prune -o -type f -print0 | while IFS= read -r -d '' file; do
	sed -e "s|go-htmx-template|${new_module}|g" "$file" > "$file.tmp" && mv "$file.tmp" "$file"
done
