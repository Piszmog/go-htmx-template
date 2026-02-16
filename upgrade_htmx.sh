#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
	echo "Usage: $0 <new htmx version>"
	exit 1
fi

old_version=""
new_version=$1

for filename in "./internal/dist/assets/js"/*; do
	if [[ "$filename" == "./internal/dist/assets/js/htmx"* ]]; then
		old_version=$(echo "$filename" | awk -F'@' '{gsub(/\.min\.js/, "", $2); print $2}')
		break
	fi
done

if [ -z "$old_version" ]; then
	echo "Error: could not find existing htmx version in ./internal/dist/assets/js/"
	exit 1
fi

if ! curl -sfL -o "./internal/dist/assets/js/htmx@${new_version}.min.js" "https://github.com/bigskysoftware/htmx/releases/download/${new_version}/htmx.min.js"; then
	echo "Error: failed to download htmx ${new_version}"
	exit 1
fi

sed -e "s|htmx@${old_version}\.min\.js|htmx@${new_version}.min.js|g" "./internal/components/core/html.templ" > "./internal/components/core/html.templ.tmp" \
	&& mv "./internal/components/core/html.templ.tmp" "./internal/components/core/html.templ"

rm "./internal/dist/assets/js/htmx@${old_version}.min.js"
