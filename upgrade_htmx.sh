#!/bin/sh

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

curl -sL -o "./internal/dist/assets/js/htmx@${new_version}.min.js" "https://github.com/bigskysoftware/htmx/releases/download/${new_version}/htmx.min.js"

sed -i '' -e "s/${old_version}/${new_version}/g" "./internal/components/core/html.templ"

rm "./internal/dist/assets/js/htmx@${old_version}.min.js"
