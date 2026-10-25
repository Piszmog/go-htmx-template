#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

fail() {
	echo "FAIL: $*" >&2
	exit 1
}

pass() {
	echo "ok - $*"
}

# update_module.sh must rewrite the module in tracked text files without
# changing file modes or rewriting itself.
test_update_module() {
	local dir="$work/update"
	mkdir -p "$dir"

	cp "$repo_root/update_module.sh" "$repo_root/upgrade_htmx.sh" "$dir/"
	cat >"$dir/sample.go" <<'EOF'
package sample

import "go-htmx-template/internal/db"
EOF
	chmod 755 "$dir/upgrade_htmx.sh"

	(cd "$dir" && ./update_module.sh "github.com/me/app" >/dev/null)

	[ -x "$dir/upgrade_htmx.sh" ] || fail "update_module.sh removed the executable bit from upgrade_htmx.sh"
	grep -q 'github.com/me/app/internal/db' "$dir/sample.go" || fail "update_module.sh did not rewrite sample.go"
	grep -q 'go-htmx-template' "$dir/update_module.sh" || fail "update_module.sh rewrote itself"

	pass "update_module.sh preserves modes and skips itself"
}

test_update_module
