#!/bin/sh
# require-green-ci.sh OWNER/REPO COMMIT
#
# Release gate: exits 0 only when the CI workflow run for COMMIT on main
# completed successfully. A tagged commit whose CI is red, still running or
# missing must not publish binaries. The GitHub CLI needs a token with
# actions: read (GH_TOKEN in the Release workflow).
set -eu

repo=${1:?usage: require-green-ci.sh <owner/repo> <commit-sha>}
sha=${2:?usage: require-green-ci.sh <owner/repo> <commit-sha>}

# The tagged commit comes from a push to main, which is what starts CI; a tag
# does not. The API lists newest first, so the first run on main is the latest
# one (a re-run wins over the run it replaced).
runs=$(gh api \
	"repos/$repo/actions/workflows/ci.yml/runs?head_sha=$sha&event=push&per_page=100")

latest=$(printf '%s' "$runs" | python3 -c '
import json, sys
runs = [r for r in json.load(sys.stdin).get("workflow_runs", [])
        if r.get("head_branch") == "main"]
if not runs:
    print("none")
else:
    print(runs[0].get("status", "unknown"), runs[0].get("conclusion") or "pending")
')

case "$latest" in
"completed success")
	echo "CI for $sha is green"
	;;
"none")
	echo "no CI run on main for $sha: push the commit and let CI pass before tagging" >&2
	exit 1
	;;
*)
	echo "CI for $sha is $latest; wait for a green run before tagging" >&2
	exit 1
	;;
esac
