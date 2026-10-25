#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

fail() {
	echo "FAIL: $*" >&2
	exit 1
}

pass() {
	echo "ok - $*"
}

# job_block prints the YAML block for a top-level job in a workflow file.
# Matching on the block instead of a literal line keeps these checks working
# when keys are reordered or reindented.
job_block() { # job, file
	awk -v job="  $1:" '
		$0 == job { found = 1; next }
		found && /^  [A-Za-z0-9_-]+:/ { exit }
		found { print }
	' "$2"
}

# needs_contains asserts that a job depends on another job. It accepts both the
# inline (`needs: [a, b]`) and block (`needs:\n  - a`) YAML forms.
needs_contains() { # file, job, dependency
	job_block "$2" "$1" |
		awk '
			/^    needs:/ { wanted = 1; buf = $0; next }
			wanted && /^      - / { buf = buf " " $0; next }
			wanted { wanted = 0 }
			END { print buf }
		' |
		tr -d '[],' |
		grep -qw -- "$3"
}

grep -q '^permissions:' .github/workflows/ci.yml ||
	fail "ci.yml must declare top-level token permissions"

[ -n "$(job_block test .github/workflows/release.yml)" ] ||
	fail "release.yml must have a test job"

[ -n "$(job_block lint .github/workflows/release.yml)" ] ||
	fail "release.yml must have a lint job"

needs_contains .github/workflows/release.yml tag test ||
	fail "release tag job must depend on the test job"

needs_contains .github/workflows/release.yml tag lint ||
	fail "release tag job must depend on the lint job"

needs_contains .github/workflows/release.yml publish release ||
	fail "publish must depend on the release job"

grep -q '^\*\.sqlite3\*$' .dockerignore ||
	fail ".dockerignore must exclude SQLite sidecar files"

! grep -q 'go get -u all' README.md ||
	fail "README.md should not recommend go get -u all"

grep -qi 'per-instance' README.md ||
	fail "README.md should document that rate limiting is per-instance"

! grep -q 'TODO' e2e/testdata/seed.sql ||
	fail "seed.sql should not contain a TODO placeholder"

pass "repository hygiene checks"
