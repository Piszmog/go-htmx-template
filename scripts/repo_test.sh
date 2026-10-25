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

grep -q '^permissions:' .github/workflows/ci.yml ||
	fail "ci.yml must declare top-level token permissions"

grep -q '^  test:' .github/workflows/release.yml ||
	fail "release.yml must have a test job"

grep -q 'needs: \[calculate-version, test\]' .github/workflows/release.yml ||
	fail "release tag job must depend on the test job"

grep -q '^\*\.sqlite3\*$' .dockerignore ||
	fail ".dockerignore must exclude SQLite sidecar files"

! grep -q 'go get -u all' README.md ||
	fail "README.md should not recommend go get -u all"

grep -qi 'per-instance' README.md ||
	fail "README.md should document that rate limiting is per-instance"

! grep -q 'TODO' e2e/testdata/seed.sql ||
	fail "seed.sql should not contain a TODO placeholder"

pass "repository hygiene checks"
