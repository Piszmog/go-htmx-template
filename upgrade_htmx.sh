#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
	echo "Usage: $0 <new htmx version>"
	exit 1
fi

old_version=""
new_version=$1
# npm versions are bare semver ("4.0.0"), while release tags are "v4.0.0".
clean_version="${new_version#v}"

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

if [ "$old_version" = "$new_version" ]; then
	echo "Error: htmx ${new_version} is already vendored"
	exit 1
fi

new_file="./internal/dist/assets/js/htmx@${new_version}.min.js"
# htmx no longer publishes htmx.min.js as a GitHub release asset (v4 ships
# htmx-<version>-dist.zip), so pull the file from npm via jsdelivr instead.
if ! curl -sfL -o "$new_file" "https://cdn.jsdelivr.net/npm/htmx.org@${clean_version}/dist/htmx.min.js"; then
	echo "Error: failed to download htmx ${new_version}"
	rm -f "$new_file"
	exit 1
fi

if [ "$(wc -c < "$new_file")" -lt 10240 ] || ! grep -q 'var htmx' "$new_file"; then
	echo "Error: downloaded htmx ${new_version} does not look like the htmx bundle"
	rm -f "$new_file"
	exit 1
fi

if ! grep -q "$clean_version" "$new_file"; then
	echo "Error: downloaded htmx bundle does not report version ${clean_version}"
	rm -f "$new_file"
	exit 1
fi

sed -e "s|htmx@${old_version}\.min\.js|htmx@${new_version}.min.js|g" "./internal/components/core/html.templ" > "./internal/components/core/html.templ.tmp" \
	&& mv "./internal/components/core/html.templ.tmp" "./internal/components/core/html.templ"

sed -e "s|htmx@${old_version}\.min\.js|htmx@${new_version}.min.js|g" "./README.md" > "./README.md.tmp" \
	&& mv "./README.md.tmp" "./README.md"

rm "./internal/dist/assets/js/htmx@${old_version}.min.js"
