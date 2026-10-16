#!/bin/sh
# Misoku installer for macOS and Linux (any distribution, x86-64 or arm64).
#
#   curl -fsSL https://raw.githubusercontent.com/pwn27sbx/misoku/main/install.sh | sh
#
# It installs misoku and gh, GitHub's command-line tool (Misoku signs in to
# GitHub with it), side by side:
#   macOS  /usr/local/bin (asks for your password when it must create or
#          write that folder)
#   Linux  ~/.local/bin (adding it to your PATH when it is not there)
# A misoku already on your PATH is updated where it is. A gh that this
# installer did not put there (Homebrew's, your distribution's) is never
# touched: Misoku uses it.
#
# Environment overrides:
#   MISOKU_VERSION      release tag to install (default: latest), e.g. v0.1.0
#   MISOKU_INSTALL_DIR  where to put misoku and gh
#   MISOKU_BASE_URL     where to download Misoku (default: GitHub releases)
#   MISOKU_GH_BASE_URL  where to download gh's archive (default: gh's GitHub
#                        release); tests point it at a local folder (file://)
#   MISOKU_GH_SHA256    the SHA-256 that gh's archive must have instead of the
#                        one written below; honoured only when
#                        MISOKU_GH_BASE_URL is a file:// URL (offline tests),
#                        so a real download is always checked against the
#                        pinned hash
#   MISOKU_TTY          the terminal the git question is answered from
#                        (default: /dev/tty), for tests
#   MISOKU_TEST_MAC_DIR the Mac's default folder (default: /usr/local/bin),
#                        for tests
set -eu

REPO="pwn27sbx/misoku"

# gh is pinned: its version and the SHA-256 of each archive are written here
# rather than read from the release's checksums.txt, which would only catch
# a damaged download, not a tampered release. The hashes are those of
# https://github.com/cli/cli/releases/download/v2.102.0/gh_2.102.0_checksums.txt
# (the archives also carry GitHub build attestations). To update: change the
# version and the four hashes in a pull request.
GH_VERSION=2.102.0
GH_SHA256_MACOS_ARM64=da922c20d1792e5b2cbf375593d7a658acf034c12c84e007e71c76ef959c337e
GH_SHA256_MACOS_AMD64=b245f24eb2bf5f75b426b4c26da3651a107f8d5b6f4fddfbfccc5679041378b3
GH_SHA256_LINUX_AMD64=bb766f710eef8ede859c18578c72c327597cd4c8a85b06001b1f3843c6019386
GH_SHA256_LINUX_ARM64=7862c86c72f43df3a2d93ddde6f473285b4e2af61b494849846827e513ef6484

# GH_MARK, next to a gh this installer put there, holds its version: only
# that gh is ever updated or replaced.
GH_MARK=.misoku-gh-version

# The line added to the shell's startup file starts with this, so a second
# run finds it instead of adding another.
PATH_NOTE="# added by the Misoku installer"

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

sha256() {
	if have sha256sum; then
		sha256sum "$1" | cut -d ' ' -f 1
	else
		shasum -a 256 "$1" | cut -d ' ' -f 1
	fi
}

# check FILE WANT: stop unless FILE's SHA-256 is WANT.
check() {
	[ -n "$2" ] || die "no checksum for $(basename "$1"): stopping"
	[ "$(sha256 "$1")" = "$2" ] || die "checksum mismatch for $(basename "$1"): the download is damaged or not the official one; nothing was installed"
}

# writable DIR: DIR, or the folder it would be created in, can be written
# without sudo.
writable() {
	w=$1
	while [ ! -d "$w" ]; do w=$(dirname "$w"); done
	[ -w "$w" ]
}

# same_dir A B: A and B are the same folder.
same_dir() {
	[ -d "$1" ] && [ -d "$2" ] && [ "$(cd "$1" && pwd -P)" = "$(cd "$2" && pwd -P)" ]
}

# rc_file: the startup file of the user's shell.
rc_file() {
	case "${SHELL:-}" in
		*/zsh) printf '%s\n' "${ZDOTDIR:-$HOME}/.zshrc" ;;
		*/bash)
			if [ "$os" = Darwin ]; then printf '%s\n' "$HOME/.bash_profile"; else printf '%s\n' "$HOME/.bashrc"; fi
			;;
		*/fish) printf '%s\n' "${XDG_CONFIG_HOME:-$HOME/.config}/fish/config.fish" ;;
		*) printf '%s\n' "$HOME/.profile" ;;
	esac
}

# The whole install is one function, called on the last line: piped from
# curl, a shell such as dash reads the script in blocks, and nothing runs
# until all of it has arrived, so a cut download runs nothing and no
# command that reads stdin can eat the rest of the script.
main() {
	# The checksums are not optional: without a tool to compute them nothing is
	# downloaded.
	have sha256sum || have shasum || die "need sha256sum or shasum to check the downloads; nothing was installed"

	os=$(uname -s)
	case "$os" in
		Darwin)
			archive=misoku-macos.tar.gz # universal: Intel and Apple Silicon
			# uname -m says x86_64 under Rosetta; this key is 1 on Apple
			# Silicon either way, and missing on Intel.
			if [ "$(sysctl -n hw.optional.arm64 2>/dev/null || true)" = 1 ]; then
				gh_arch=arm64 gh_sum=$GH_SHA256_MACOS_ARM64
			else
				gh_arch=amd64 gh_sum=$GH_SHA256_MACOS_AMD64
			fi
			gh_archive="gh_${GH_VERSION}_macOS_${gh_arch}.zip"
			have unzip || die "need unzip to unpack gh"
			;;
		Linux)
			arch=$(uname -m)
			case "$arch" in
				x86_64 | amd64) archive=misoku-linux.tar.gz gh_arch=amd64 gh_sum=$GH_SHA256_LINUX_AMD64 ;;
				aarch64 | arm64) archive=misoku-linux-arm64.tar.gz gh_arch=arm64 gh_sum=$GH_SHA256_LINUX_ARM64 ;;
				*) die "prebuilt Linux binaries are x86-64 and arm64 only (this is $arch); install with: go install github.com/$REPO/cmd/misoku@latest" ;;
			esac
			gh_archive="gh_${GH_VERSION}_linux_${gh_arch}.tar.gz"
			;;
		*) die "unsupported OS: $os (on Windows use install.ps1)" ;;
	esac
	gh_dir="${gh_archive%.zip}"
	gh_dir="${gh_dir%.tar.gz}"

	version=${MISOKU_VERSION:-latest}
	if [ -n "${MISOKU_BASE_URL:-}" ]; then
		base=$MISOKU_BASE_URL
	elif [ "$version" = latest ]; then
		base="https://github.com/$REPO/releases/latest/download"
	else
		base="https://github.com/$REPO/releases/download/$version"
	fi
	gh_base=${MISOKU_GH_BASE_URL:-https://github.com/cli/cli/releases/download/v$GH_VERSION}
	# Pinning is the point: a command someone pastes must not be able to swap
	# the hash for a download from the network.
	case "$gh_base" in
		file://*) gh_sum=${MISOKU_GH_SHA256:-$gh_sum} ;;
	esac

	# Where to install. A misoku already on the PATH is updated in its folder,
	# unless it is a link a package manager keeps (Homebrew, Nix): that one is
	# theirs to update.
	dir=${MISOKU_INSTALL_DIR:-}
	linked=""
	if [ -z "$dir" ] && old=$(command -v misoku 2>/dev/null) && [ -f "$old" ]; then
		if [ -L "$old" ]; then linked=$old; else dir=$(dirname "$old"); fi
	fi
	if [ -z "$dir" ]; then
		dir="$HOME/.local/bin"
		if [ "$os" = Darwin ]; then
			mac_dir=${MISOKU_TEST_MAC_DIR:-/usr/local/bin}
			if writable "$mac_dir"; then
				dir=$mac_dir
			elif have sudo; then
				# sudo -v asks for the password once (on the terminal, even
				# piped from curl); an account that is not an administrator,
				# or no terminal, gets ~/.local/bin instead.
				say "Installing to $mac_dir needs your password."
				if sudo -v </dev/null; then
					dir=$mac_dir
				else
					say "Could not use sudo: installing to $dir instead."
				fi
			fi
		fi
	fi
	# A link in the install folder itself (stow, a dev build) is replaced by
	# the new copy: no need to say it was left.
	if [ -n "$linked" ] && same_dir "$(dirname "$linked")" "$dir"; then linked=""; fi

	# sudo is used only when the folder cannot be created or written otherwise.
	sudo=""
	writable "$dir" || sudo=sudo
	if [ -n "$sudo" ]; then
		have sudo || die "cannot write to $dir; set MISOKU_INSTALL_DIR to a folder you can write"
		[ "$dir" = "${mac_dir:-}" ] || say "Installing to $dir needs your password."
	fi

	# What to do with gh: install it when there is none, update the one this
	# installer put here, and leave any other alone.
	gh_do=install
	gh_other=$(command -v gh 2>/dev/null || true)
	if [ -e "$dir/gh" ] || [ -L "$dir/gh" ]; then
		if [ ! -f "$dir/$GH_MARK" ]; then
			gh_do=keep gh_other="$dir/gh"
		elif [ "$(cat "$dir/$GH_MARK")" = "$GH_VERSION" ]; then
			gh_do=current
		else
			gh_do=update
		fi
	elif [ -n "$gh_other" ]; then
		gh_do=keep
	fi

	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT
	trap 'echo; die "cancelled"' INT TERM

	# Everything is downloaded and checked before anything is installed.
	say "Downloading Misoku ($version) for $os..."
	fetch "$base/$archive" "$tmp/$archive" || die "download failed: $base/$archive"
	fetch "$base/checksums.txt" "$tmp/checksums.txt" quiet || die "download failed: $base/checksums.txt (needed to check the download)"
	check "$tmp/$archive" "$(grep " $archive\$" "$tmp/checksums.txt" | cut -d ' ' -f 1)"
	tar -xzf "$tmp/$archive" -C "$tmp" misoku || die "could not unpack $archive"
	chmod +x "$tmp/misoku"

	case "$gh_do" in
		install | update)
			say "Also installing gh $GH_VERSION, GitHub's command-line tool: Misoku signs in to GitHub with it."
			fetch "$gh_base/$gh_archive" "$tmp/$gh_archive" || die "download failed: $gh_base/$gh_archive"
			check "$tmp/$gh_archive" "$gh_sum"
			mkdir "$tmp/gh"
			if [ "$os" = Darwin ]; then
				unzip -q "$tmp/$gh_archive" "$gh_dir/bin/gh" -d "$tmp/gh" || die "could not unpack $gh_archive"
			else
				tar -xzf "$tmp/$gh_archive" -C "$tmp/gh" "$gh_dir/bin/gh" || die "could not unpack $gh_archive"
			fi
			chmod +x "$tmp/gh/$gh_dir/bin/gh"
			;;
	esac

	$sudo mkdir -p "$dir"
	$sudo mv "$tmp/misoku" "$dir/misoku"
	say "Installed $("$dir/misoku" --version </dev/null) to $dir/misoku"
	if [ -n "$linked" ]; then
		say "Note: the misoku at $linked (a link your package manager keeps) was not updated and may run first; the new copy is at $dir/misoku."
	fi

	case "$gh_do" in
		install | update)
			# The mark goes first, so a gh cut short halfway is still known as
			# this installer's and replaced next time.
			printf 'pending\n' >"$tmp/mark"
			$sudo mv "$tmp/mark" "$dir/$GH_MARK"
			$sudo mv "$tmp/gh/$gh_dir/bin/gh" "$dir/gh"
			printf '%s\n' "$GH_VERSION" >"$tmp/mark"
			$sudo mv "$tmp/mark" "$dir/$GH_MARK"
			say "Installed gh $GH_VERSION to $dir/gh"
			;;
		current) say "gh $GH_VERSION is already in $dir" ;;
		keep) say "Using your gh at $gh_other (left as it is)" ;;
	esac

	# git is needed only to sync; its absence never stops the install.
	git_hint="Misoku works without it; it is needed only to sync your notes between computers."
	if [ "$os" = Darwin ]; then
		# /usr/bin/git is a stub until Apple's command line tools are installed.
		g=$(command -v git 2>/dev/null || true)
		if ! xcode-select -p >/dev/null 2>&1 && { [ -z "$g" ] || [ "$g" = /usr/bin/git ]; }; then
			xcode-select --install </dev/null >/dev/null 2>&1 || true
			say "git is not installed: Apple's installer for it opened; finish it whenever you like. $git_hint"
		fi
	elif ! have git; then
		tty=${MISOKU_TTY:-/dev/tty}
		pm=""
		for p in apt-get dnf pacman zypper apk; do
			if have "$p"; then pm=$p; break; fi
		done
		asudo=""
		[ "$(id -u)" = 0 ] || asudo=sudo
		answer=n
		# The question needs a person at a terminal: piped from curl, the
		# answer comes from the terminal, not from stdin.
		if [ -n "$pm" ] && { [ -z "$asudo" ] || have sudo; } && (: <"$tty") 2>/dev/null; then
			printf 'Install git? (needs your password) [Y/n] ' >&2
			read -r answer <"$tty" || answer=n
			[ -n "$answer" ] || answer=y
		fi
		case "$answer" in
			y | Y | yes | Yes)
				ok=1
				case "$pm" in
					apt-get) $asudo apt-get update </dev/null && $asudo apt-get install -y git </dev/null || ok=0 ;;
					dnf) $asudo dnf install -y git </dev/null || ok=0 ;;
					pacman) $asudo pacman -S --noconfirm git </dev/null || ok=0 ;;
					zypper) $asudo zypper -n install git </dev/null || ok=0 ;;
					apk) $asudo apk add git </dev/null || ok=0 ;;
				esac
				[ "$ok" = 1 ] || say "git could not be installed. $git_hint"
				;;
			*) say "git is not installed. $git_hint Install it with your package manager." ;;
		esac
	fi


	case ":$PATH:" in
		*":$dir:"*) ;;
		*)
			shown=$dir
			case "$dir" in "$HOME"/*) shown="\$HOME${dir#"$HOME"}" ;; esac
			rc=$(rc_file)
			case "$rc" in
				*/config.fish) line="set -gx PATH \"$shown\" \$PATH $PATH_NOTE" ;;
				*) line="export PATH=\"$shown:\$PATH\" $PATH_NOTE" ;;
			esac
			if ! grep -qxF "$line" "$rc" 2>/dev/null; then
				mkdir -p "$(dirname "$rc")"
				# Start on a line of its own when the file does not end in one.
				if [ -s "$rc" ] && [ "$(tail -c 1 "$rc")" != "" ]; then printf '\n' >>"$rc"; fi
				printf '%s\n' "$line" >>"$rc"
				say "Added $dir to your PATH in $rc"
			fi
			say "Close this window and open a new Terminal, then run 'misoku'."
			exit 0
			;;
	esac
	say "Run 'misoku' to open your notes (a Nerd Font is recommended for icons)."
}

main "$@"
