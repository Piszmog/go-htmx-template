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

grep -q '^  workflow_run:' .github/workflows/release.yml ||
	fail "release.yml must run after the CI workflow"

grep -qF 'workflows: [CI]' .github/workflows/release.yml ||
	fail "release.yml must be triggered by the CI workflow"

grep -qF 'branches: [main]' .github/workflows/release.yml ||
	fail "release.yml must only run for main"

grep -qF "github.event.workflow_run.conclusion == 'success'" .github/workflows/release.yml ||
	fail "release.yml must only run after CI succeeds"

grep -qF "github.event.workflow_run.event == 'push'" .github/workflows/release.yml ||
	fail "release.yml must only run for push events"

[ -n "$(job_block version .github/workflows/release.yml)" ] ||
	fail "release.yml must have a version job"

[ -n "$(job_block tag .github/workflows/release.yml)" ] ||
	fail "release.yml must have a tag job"

needs_contains .github/workflows/release.yml release version ||
	fail "release job must depend on the version job"

needs_contains .github/workflows/release.yml release tag ||
	fail "release job must depend on the tag job"

needs_contains .github/workflows/release.yml publish release ||
	fail "publish must depend on the release job"

for job in version tag release publish; do
	job_block "$job" .github/workflows/release.yml |
		grep -qF 'ref: ${{ env.SHA }}' ||
		fail "$job job must check out the tested commit"
done

grep -q '^name: CI$' .github/workflows/ci.yml ||
	fail "ci.yml must stay named CI so release.yml can trigger"

grep -qF 'group: ci-${{ github.event_name' .github/workflows/ci.yml ||
	fail "ci.yml must group push runs per commit"

needs_contains .github/workflows/release.yml tag version ||
	fail "tag job must depend on the version job"

needs_contains .github/workflows/release.yml publish tag ||
	fail "publish must depend on the tag job"

grep -q '^  replace_existing_artifacts: true$' .goreleaser.yaml ||
	fail "GoReleaser must replace existing artifacts on re-run"

grep -q '^\*\.sqlite3\*$' .dockerignore ||
	fail ".dockerignore must exclude SQLite sidecar files"

! grep -q 'go get -u all' README.md ||
	fail "README.md should not recommend go get -u all"

grep -qi 'per-instance' README.md ||
	fail "README.md should document that rate limiting is per-instance"

! grep -q 'TODO' e2e/testdata/seed.sql ||
	fail "seed.sql should not contain a TODO placeholder"

pass "repository hygiene checks"
