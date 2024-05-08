#!/bin/sh

old_version=$(grep 'github.com/a-h/templ' go.mod | awk '{print $2}')

go install github.com/a-h/templ/cmd/templ@latest

go get -u github.com/a-h/templ
go mod tidy

new_version=$(grep 'github.com/a-h/templ' go.mod | awk '{print $2}')

sed -i '' -e "s/${old_version}/${new_version}/g" "Dockerfile"

for file in ".github/workflows"/*; do
	if [ -f "$file" ]; then
		sed -i '' -e "s/${old_version}/${new_version}/g" "$file"
	fi
done
