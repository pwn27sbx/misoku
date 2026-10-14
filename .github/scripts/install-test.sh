#!/bin/sh
# Serves the snapshot release in ./dist and runs install.sh against it.
set -eu
port=8765
log=$(mktemp)

# Bind to loopback only: listening on every interface can be blocked by the
# macOS firewall on CI runners.
python3 -m http.server "$port" --bind 127.0.0.1 --directory dist >"$log" 2>&1 &
server=$!
trap 'kill "$server" 2>/dev/null || true; rm -f "$log"' EXIT

# Wait until the server answers instead of guessing with a fixed sleep.
up=0
for _ in $(seq 1 30); do
	if python3 -c "import urllib.request,sys; urllib.request.urlopen('http://127.0.0.1:$port/checksums.txt', timeout=2)" 2>/dev/null; then
		up=1
		break
	fi
	sleep 1
done
if [ "$up" -ne 1 ]; then
	echo "test server did not start:" >&2
	cat "$log" >&2
	exit 1
fi

MISOKU_BASE_URL="http://127.0.0.1:$port" \
	MISOKU_INSTALL_DIR="$PWD/bin" sh ./install.sh
./bin/misoku --version
./bin/misoku --list-themes | grep -q onedark
echo "install.sh OK"
