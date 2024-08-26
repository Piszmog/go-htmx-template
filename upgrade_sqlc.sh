#!/bin/sh

url=$(curl -s 'https://downloads.sqlc.dev/')

new_version=$(echo "$url" | sed -e 's/<li>/<li>\n/g' | tr -d '\n' | sed -e 's/<li>/\n<li>/g' | sed -e 's/<\/li>/<\/li>\n/g' | grep -o '<li>[^<]*<a[^>]*>[^<]*</a>[^<]*</li>' | sed -e 's/<[^>]*>//g' | tail -n 1 | awk '{$1=$1;print}')

old_version=$(grep 'sqlc-version' .github/workflows/ci.yml | awk '{print $2}' | tr -d "'" | head -n 1)

for file in ".github/workflows"/*; do
	if [ -f "$file" ]; then
		sed -i '' -e "s/${old_version}/${new_version}/g" "$file"
	fi
done
