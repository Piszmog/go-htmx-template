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

	cp "$repo_root/update_module.sh" "$repo_root/migrate.sh" "$repo_root/upgrade_htmx.sh" "$dir/"
	cat >"$dir/sample.go" <<'EOF'
package sample

import "go-htmx-template/internal/db"
EOF
	chmod 755 "$dir/migrate.sh" "$dir/upgrade_htmx.sh"

	(cd "$dir" && ./update_module.sh "github.com/me/app" >/dev/null)

	[ -x "$dir/migrate.sh" ] || fail "update_module.sh removed the executable bit from migrate.sh"
	[ -x "$dir/upgrade_htmx.sh" ] || fail "update_module.sh removed the executable bit from upgrade_htmx.sh"
	grep -q 'github.com/me/app/internal/db' "$dir/sample.go" || fail "update_module.sh did not rewrite sample.go"
	grep -q 'go-htmx-template' "$dir/update_module.sh" || fail "update_module.sh rewrote itself"

	pass "update_module.sh preserves modes and skips itself"
}

# migrate.sh must normalize relative SQLite paths for both the modernc
# (sqlite) and mattn (sqlite3) drivers. A fake go binary captures the database
# URL without downloading or compiling the migrate CLI.
test_migrate_sqlite_paths() {
	local bin="$work/bin"
	local dir="$work/mig"
	mkdir -p "$bin" "$dir"

	cat >"$bin/go" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${GO_SHIM_LOG:?}"
EOF
	chmod 755 "$bin/go"

	local mig_dir
	mig_dir="$(cd "$dir" && pwd)"

	for protocol in sqlite sqlite3; do
		local log="$work/go-$protocol.log"
		: >"$log"
		(
			cd "$dir"
			GO_SHIM_LOG="$log" PATH="$bin:$PATH" "$repo_root/migrate.sh" \
				-p "$protocol" -u ./db.sqlite3 >/dev/null
		)

		grep -q "${protocol}://${mig_dir}/db.sqlite3" "$log" ||
			fail "migrate.sh did not normalize the $protocol relative path: $(cat "$log")"
	done

	pass "migrate.sh normalizes relative sqlite and sqlite3 paths"
}

test_update_module
test_migrate_sqlite_paths
