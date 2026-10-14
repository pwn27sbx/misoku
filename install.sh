#!/bin/sh
# Misoku installer for macOS and Linux (any distribution, x86-64).
#
#   curl -fsSL https://raw.githubusercontent.com/pwn27sbx/misoku/main/install.sh | sh
#
# Environment overrides:
#   MISOKU_VERSION      release tag to install (default: latest), e.g. v0.1.0
#   MISOKU_INSTALL_DIR  where to put the binary (default: /usr/local/bin if
#                        writable or sudo is available, else ~/.local/bin)
#   MISOKU_BASE_URL     download location (default: GitHub releases)
set -eu

REPO="pwn27sbx/misoku"

say() { printf '%s\n' "$*"; }
die() { printf 'misoku install: %s\n' "$*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

# fetch URL OUTFILE [quiet]. GitHub's download host resolves to several
# addresses; when one of them doesn't answer, a short connect timeout makes
# curl move on to the next instead of hanging for the OS default (75s on
# macOS). A progress bar shows when a person is watching.
fetch() {
	if have curl; then
		progress=-s
		if [ -t 2 ] && [ -z "${3:-}" ]; then progress=-#; fi
		curl -fSL $progress --connect-timeout 5 --retry 3 --retry-delay 1 "$1" -o "$2"
	elif have wget; then
		wget -q --timeout=5 --tries=3 -O "$2" "$1"
	else
		die "need curl or wget"
	fi
}

os=$(uname -s)
arch=$(uname -m)
case "$os" in
	Darwin) archive=misoku-macos.tar.gz ;; # universal: Intel and Apple Silicon
	Linux)
		case "$arch" in
			x86_64 | amd64) archive=misoku-linux.tar.gz ;;
			*) die "prebuilt Linux binaries are x86-64 only (this is $arch); install with: go install github.com/$REPO/cmd/misoku@latest" ;;
		esac
		;;
	*) die "unsupported OS: $os (on Windows use install.ps1)" ;;
esac

version=${MISOKU_VERSION:-latest}
if [ -n "${MISOKU_BASE_URL:-}" ]; then
	base=$MISOKU_BASE_URL
elif [ "$version" = latest ]; then
	base="https://github.com/$REPO/releases/latest/download"
else
	base="https://github.com/$REPO/releases/download/$version"
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
trap 'echo; die "cancelled"' INT TERM

say "Downloading Misoku ($version) for $os..."
fetch "$base/$archive" "$tmp/$archive" || die "download failed: $base/$archive"

if fetch "$base/checksums.txt" "$tmp/checksums.txt" quiet 2>/dev/null; then
	want=$(grep " $archive\$" "$tmp/checksums.txt" | cut -d ' ' -f 1)
	if have sha256sum; then
		got=$(sha256sum "$tmp/$archive" | cut -d ' ' -f 1)
	elif have shasum; then
		got=$(shasum -a 256 "$tmp/$archive" | cut -d ' ' -f 1)
	else
		got=$want
	fi
	[ -z "$want" ] || [ "$want" = "$got" ] || die "checksum mismatch for $archive"
fi

tar -xzf "$tmp/$archive" -C "$tmp" misoku || die "could not unpack $archive"
chmod +x "$tmp/misoku"

dir=${MISOKU_INSTALL_DIR:-}
sudo=""
if [ -z "$dir" ]; then
	if [ -w /usr/local/bin ]; then
		dir=/usr/local/bin
	elif have sudo && [ -d /usr/local/bin ]; then
		dir=/usr/local/bin
		sudo=sudo
	else
		dir="$HOME/.local/bin"
	fi
fi
mkdir -p "$dir" 2>/dev/null || $sudo mkdir -p "$dir"
if [ -w "$dir" ]; then sudo=""; fi
$sudo mv "$tmp/misoku" "$dir/misoku"

say "Installed $("$dir/misoku" --version) to $dir/misoku"
case ":$PATH:" in
	*":$dir:"*) ;;
	*) say "Add $dir to your PATH, e.g.:  echo 'export PATH=\"$dir:\$PATH\"' >> ~/.profile" ;;
esac
say "Run 'misoku' in a folder of notes (a Nerd Font is recommended for icons)."
