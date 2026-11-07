#!/bin/sh
# Offline checks for scripts/require-green-ci.sh: a fake gh prints the run
# list the GitHub API would return, and every case asserts the exit status.
set -eu

here=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
gate="$here/require-green-ci.sh"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

cat > "$tmp/gh" <<'FAKE'
#!/bin/sh
printf '{"workflow_runs": [%s]}\n' "$FAKE_RUNS"
FAKE
chmod +x "$tmp/gh"

run_case() {
	name=$1 expect=$2 runs=$3
	if PATH="$tmp:$PATH" FAKE_RUNS="$runs" sh "$gate" owner/repo abc123 \
		>"$tmp/out" 2>"$tmp/err"; then
		got=0
	else
		got=1
	fi
	if [ "$got" -ne "$expect" ]; then
		echo "FAIL: $name: exit $got, want $expect"
		cat "$tmp/err"
		exit 1
	fi
	echo "ok: $name"
}

run_case "green run" 0 \
	'{"head_branch":"main","status":"completed","conclusion":"success"}'
run_case "red run" 1 \
	'{"head_branch":"main","status":"completed","conclusion":"failure"}'
run_case "run still going" 1 \
	'{"head_branch":"main","status":"in_progress","conclusion":null}'
run_case "no runs at all" 1 ''
run_case "only a pull request run" 1 \
	'{"head_branch":"feature/x","status":"completed","conclusion":"success"}'
run_case "latest re-run is green" 0 \
	'{"head_branch":"main","status":"completed","conclusion":"success"},{"head_branch":"main","status":"completed","conclusion":"failure"}'
run_case "latest re-run is still going" 1 \
	'{"head_branch":"main","status":"in_progress","conclusion":null},{"head_branch":"main","status":"completed","conclusion":"success"}'

echo "release gate checks passed"
