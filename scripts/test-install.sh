#!/bin/sh
# Tests install.sh without a network: a fake Misoku release and fake gh
# archives in a temporary folder, served as file:// URLs (MISOKU_BASE_URL,
# MISOKU_GH_BASE_URL), and a PATH of only the tools install.sh needs plus
# fakes of uname, sysctl, sudo, xcode-select and apt-get. Each case runs in
# its own fake HOME, so the real one is never touched.
#
#   sh scripts/test-install.sh
#
# Needs curl (for file:// URLs), tar, gzip, zip or python3, and shasum or
# sha256sum. Runs on macOS and Linux; both systems are simulated on either.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
installer="$root/install.sh"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

ghv=$(sed -n 's/^GH_VERSION=//p' "$installer")
[ -n "$ghv" ] || { echo "GH_VERSION not found in install.sh" >&2; exit 1; }

sum() {
	if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | cut -d ' ' -f 1; else shasum -a 256 "$1" | cut -d ' ' -f 1; fi
}

# script FILE TEXT: an executable that prints TEXT.
script() {
	mkdir -p "$(dirname "$1")"
	printf '#!/bin/sh\necho "%s"\n' "$2" >"$1"
	chmod +x "$1"
}

# The fake Misoku release: each archive's misoku says which one it is.
rel="$work/release"
mkdir -p "$rel"
for a in macos linux linux-arm64; do
	d="$work/build-$a"
	script "$d/misoku" "misoku new-$a"
	tar -czf "$rel/misoku-$a.tar.gz" -C "$d" misoku
	printf '%s  %s\n' "$(sum "$rel/misoku-$a.tar.gz")" "misoku-$a.tar.gz" >>"$rel/checksums.txt"
done

# The fake gh archives, laid out like the real ones.
ghrel="$work/gh-release"
mkdir -p "$ghrel"
for a in macOS_arm64 macOS_amd64 linux_amd64 linux_arm64; do
	d="$work/gh-build/gh_${ghv}_$a"
	script "$d/bin/gh" "gh version $ghv ($a)"
	case "$a" in
		macOS_*)
			if command -v zip >/dev/null 2>&1; then
				(cd "$work/gh-build" && zip -qr "$ghrel/gh_${ghv}_$a.zip" "gh_${ghv}_$a")
			else
				(cd "$work/gh-build" && python3 -c 'import sys,zipfile,os
z=zipfile.ZipFile(sys.argv[1],"w")
for r,_,fs in os.walk(sys.argv[2]):
    for f in fs:
        p=os.path.join(r,f); i=zipfile.ZipInfo(p); i.external_attr=0o755<<16; z.writestr(i,open(p,"rb").read())
z.close()' "$ghrel/gh_${ghv}_$a.zip" "gh_${ghv}_$a")
			fi
			;;
		*) tar -czf "$ghrel/gh_${ghv}_$a.tar.gz" -C "$work/gh-build" "gh_${ghv}_$a" ;;
	esac
done

# tools BIN [no-sha]: a PATH folder with the real tools install.sh needs
# and the fakes. The fakes read FAKE_OS, FAKE_ARCH, FAKE_ARM64 and
# FAKE_CLT, and log to $LOG.
tools() {
	b=$1
	mkdir -p "$b"
	for t in sh cat grep cut dirname basename mkdir mv chmod rm mktemp tar gzip unzip curl tail id sed; do
		p=$(command -v "$t") || { echo "missing tool: $t" >&2; exit 1; }
		ln -s "$p" "$b/$t"
	done
	if [ "${2:-}" != no-sha ]; then
		for t in shasum sha256sum; do
			if p=$(command -v "$t"); then ln -s "$p" "$b/$t"; fi
		done
	fi
	cat >"$b/uname" <<'EOF'
#!/bin/sh
case "$1" in -s) echo "$FAKE_OS" ;; -m) echo "$FAKE_ARCH" ;; *) echo "$FAKE_OS" ;; esac
EOF
	cat >"$b/sysctl" <<'EOF'
#!/bin/sh
[ -n "${FAKE_ARM64:-}" ] || exit 1
echo "$FAKE_ARM64"
EOF
	cat >"$b/sudo" <<'EOF'
#!/bin/sh
echo "sudo $*" >>"$LOG"
if [ "$1" = -v ]; then [ -z "${FAKE_SUDO_FAIL:-}" ]; exit; fi
exec "$@"
EOF
	cat >"$b/xcode-select" <<'EOF'
#!/bin/sh
echo "xcode-select $*" >>"$LOG"
[ "$1" = -p ] && [ -z "${FAKE_CLT:-}" ] && exit 2
exit 0
EOF
	cat >"$b/apt-get" <<'EOF'
#!/bin/sh
echo "apt-get $*" >>"$LOG"
EOF
	chmod +x "$b/uname" "$b/sysctl" "$b/sudo" "$b/xcode-select" "$b/apt-get"
}
tools "$work/bin"
tools "$work/bin-nosha" no-sha

# A curl that serves https://mirror.test/ from the fake gh release, so a
# gh URL that is not file:// can be tested offline; other URLs go to the
# real curl.
mkdir -p "$work/mirror"
{
	echo '#!/bin/sh'
	echo "ghrel='$ghrel' real='$(command -v curl)'"
	cat <<'CURL'
url="" out=""
for a in "$@"; do
	case "$a" in https://mirror.test/*) url=$a ;; esac
	[ "$prev" = -o ] && out=$a
	prev=$a
done
[ -n "$url" ] || exec "$real" "$@"
cat "$ghrel/${url#https://mirror.test/}" >"$out"
CURL
} >"$work/mirror/curl"
chmod +x "$work/mirror/curl"

fails=0
pass() { echo "ok   $1"; }
fail() {
	echo "FAIL $1" >&2
	sed 's/^/     | /' "$out" >&2
	fails=$((fails + 1))
}
# expect NAME CONDITION...: CONDITION is a command that must succeed.
expect() {
	n=$1
	shift
	if "$@"; then pass "$n"; else fail "$n"; fi
}
count() { grep -cxF "$2" "$1" 2>/dev/null || true; }

# run CASE [VAR=value...]: runs install.sh as a Linux x86-64 user with
# bash, in a fresh HOME (unless HOME is given), with gh's hash set to the
# fake archive's. Leaves its output in $out and its exit code in $code.
case_home() { echo "$work/home-$1"; }
run() {
	c=$1
	shift
	h=$(case_home "$c")
	mkdir -p "$h"
	LOG="$work/log-$c"
	: >"$LOG"
	out="$work/out-$c"
	set +e
	/usr/bin/env -i HOME="$h" PATH="$work/bin" SHELL=/bin/bash LOG="$LOG" \
		FAKE_OS=Linux FAKE_ARCH=x86_64 MISOKU_TTY="$work/no-tty" \
		MISOKU_BASE_URL="file://$rel" MISOKU_GH_BASE_URL="file://$ghrel" \
		MISOKU_GH_SHA256="$(sum "$ghrel/gh_${ghv}_linux_amd64.tar.gz")" \
		"$@" /bin/sh "$installer" >"$out" 2>&1
	code=$?
	set -e
}
says() { grep -qF "$1" "$out"; }

# 1. Fresh Linux install: misoku and gh in ~/.local/bin, gh marked as ours,
# one PATH line in .bashrc, and the one extra step said.
run linux
h=$(case_home linux)
bin="$h/.local/bin"
line="export PATH=\"\$HOME/.local/bin:\$PATH\" # added by the Misoku installer"
expect "linux: exits 0" test "$code" = 0
expect "linux: misoku installed" test "$("$bin/misoku")" = "misoku new-linux"
expect "linux: gh installed next to it" test "$("$bin/gh")" = "gh version $ghv (linux_amd64)"
expect "linux: gh marked with its version" test "$(cat "$bin/.misoku-gh-version")" = "$ghv"
expect "linux: one PATH line in .bashrc" test "$(count "$h/.bashrc" "$line")" = 1
expect "linux: says to open a new Terminal" says "Close this window and open a new Terminal"
expect "linux: says gh is installed (one line)" says "Also installing gh $ghv"
expect "linux: no git and no terminal: goes on and says so" says "git is not installed."
expect "linux: no sudo" test ! -s "$LOG"

# 2. The same install again: the PATH line is not added twice and gh is
# already current.
run linux HOME="$h"
expect "rerun: exits 0" test "$code" = 0
expect "rerun: still one PATH line" test "$(count "$h/.bashrc" "$line")" = 1
expect "rerun: gh already current" says "gh $ghv is already in $bin"

# 3. Our older gh is updated.
printf 'old\n' >"$bin/gh"
printf '2.0.0\n' >"$bin/.misoku-gh-version"
run linux HOME="$h"
expect "update: exits 0" test "$code" = 0
expect "update: our gh updated" test "$("$bin/gh")" = "gh version $ghv (linux_amd64)"
expect "update: mark updated" test "$(cat "$bin/.misoku-gh-version")" = "$ghv"

# 4. A gh we did not install, in the install folder, is left as it is.
h=$(case_home theirs)
mkdir -p "$h/.local/bin"
printf 'theirs\n' >"$h/.local/bin/gh"
run theirs
expect "their gh in our folder: exits 0" test "$code" = 0
expect "their gh in our folder: untouched" test "$(cat "$h/.local/bin/gh")" = theirs
expect "their gh in our folder: not marked" test ! -e "$h/.local/bin/.misoku-gh-version"
expect "their gh in our folder: says so" says "Using your gh at $h/.local/bin/gh"

# 5. A gh elsewhere on the PATH (Homebrew's, apt's) is used, not shadowed.
script "$work/brew/gh" "brew gh"
run brew PATH="$work/bin:$work/brew"
h=$(case_home brew)
expect "gh on PATH: exits 0" test "$code" = 0
expect "gh on PATH: none installed next to misoku" test ! -e "$h/.local/bin/gh"
expect "gh on PATH: untouched" test "$("$work/brew/gh")" = "brew gh"

# 6. A misoku already on the PATH is updated where it is.
script "$work/old/misoku" "misoku old"
run oldmisoku PATH="$work/bin:$work/old"
h=$(case_home oldmisoku)
expect "existing misoku: exits 0" test "$code" = 0
expect "existing misoku: updated in place" test "$("$work/old/misoku")" = "misoku new-linux"
expect "existing misoku: gh next to it" test -x "$work/old/gh"
expect "existing misoku: nothing in ~/.local/bin" test ! -e "$h/.local/bin/misoku"
expect "existing misoku: folder on PATH, no PATH line" test ! -e "$h/.bashrc"

# 6b. A misoku on the PATH that is a package manager's link is not
# updated; one line names it and says where the new copy is.
mkdir -p "$work/cellar" "$work/brewbin"
script "$work/cellar/misoku" "misoku brew"
ln -s "$work/cellar/misoku" "$work/brewbin/misoku"
run linked PATH="$work/bin:$work/brewbin"
h=$(case_home linked)
expect "linked misoku: exits 0" test "$code" = 0
expect "linked misoku: left as it is" test "$("$work/brewbin/misoku")" = "misoku brew"
expect "linked misoku: new copy in ~/.local/bin" test "$("$h/.local/bin/misoku")" = "misoku new-linux"
expect "linked misoku: one line names both" says "Note: the misoku at $work/brewbin/misoku (a link your package manager keeps) was not updated and may run first; the new copy is at $h/.local/bin/misoku."

# 7. A bad Misoku checksum stops before anything is installed.
cp -R "$rel" "$work/bad-release"
printf '0000  misoku-linux.tar.gz\n' >"$work/bad-release/checksums.txt"
run badmisoku MISOKU_BASE_URL="file://$work/bad-release"
h=$(case_home badmisoku)
expect "bad misoku checksum: fails" test "$code" != 0
expect "bad misoku checksum: says so" says "checksum mismatch for misoku-linux.tar.gz"
expect "bad misoku checksum: nothing installed" test ! -e "$h/.local/bin/misoku"

# 7a. A checksums.txt that does not list the archive stops too.
printf '%s  misoku-macos.tar.gz\n' "$(sum "$rel/misoku-macos.tar.gz")" >"$work/bad-release/checksums.txt"
run unlisted MISOKU_BASE_URL="file://$work/bad-release"
expect "unlisted archive: fails" test "$code" != 0
expect "unlisted archive: says so" says "no checksum for misoku-linux.tar.gz"
expect "unlisted archive: nothing installed" test ! -e "$(case_home unlisted)/.local/bin/misoku"

# 7b. No checksums.txt at all stops too (it used to be skipped).
rm "$work/bad-release/checksums.txt"
run nosums MISOKU_BASE_URL="file://$work/bad-release"
expect "no checksums.txt: fails" test "$code" != 0
expect "no checksums.txt: nothing installed" test ! -e "$(case_home nosums)/.local/bin/misoku"

# 8. A gh archive that is not the one pinned in install.sh stops the install.
run badgh MISOKU_GH_SHA256=
h=$(case_home badgh)
expect "bad gh checksum: fails" test "$code" != 0
expect "bad gh checksum: says so" says "checksum mismatch for gh_${ghv}_linux_amd64.tar.gz"
expect "bad gh checksum: nothing installed" test ! -e "$h/.local/bin/misoku" -a ! -e "$h/.local/bin/gh"

# 8b. MISOKU_GH_SHA256 is ignored when gh comes from the network: the
# pinned hash still applies, and the fake archive stops the install.
run override PATH="$work/mirror:$work/bin" MISOKU_GH_BASE_URL="https://mirror.test"
h=$(case_home override)
expect "hash override over the network: fails" test "$code" != 0
expect "hash override over the network: pinned hash used" says "checksum mismatch for gh_${ghv}_linux_amd64.tar.gz"
expect "hash override over the network: nothing installed" test ! -e "$h/.local/bin/misoku" -a ! -e "$h/.local/bin/gh"

# 9. Without shasum or sha256sum nothing is downloaded.
run nosha PATH="$work/bin-nosha"
h=$(case_home nosha)
expect "no sha tool: fails" test "$code" != 0
expect "no sha tool: says so" says "need sha256sum or shasum"
expect "no sha tool: nothing installed" test ! -e "$h/.local/bin/misoku"

# 10. Linux arm64 gets its own archives.
run arm FAKE_ARCH=aarch64 MISOKU_GH_SHA256="$(sum "$ghrel/gh_${ghv}_linux_arm64.tar.gz")"
h=$(case_home arm)
expect "linux arm64: exits 0" test "$code" = 0
expect "linux arm64: misoku-linux-arm64" test "$("$h/.local/bin/misoku")" = "misoku new-linux-arm64"
expect "linux arm64: gh linux_arm64" test "$("$h/.local/bin/gh")" = "gh version $ghv (linux_arm64)"

# 11. fish gets a fish line, once.
run fish SHELL=/usr/bin/fish
h=$(case_home fish)
fline="set -gx PATH \"\$HOME/.local/bin\" \$PATH # added by the Misoku installer"
run fish HOME="$h" SHELL=/usr/bin/fish
expect "fish: one line in config.fish" test "$(count "$h/.config/fish/config.fish" "$fline")" = 1
expect "fish: no .bashrc" test ! -e "$h/.bashrc"

# 12. Linux without git, at a terminal: asks, and installs git with the
# distribution's package manager.
printf 'y\n' >"$work/tty-yes"
run gitask MISOKU_TTY="$work/tty-yes"
expect "git question: exits 0" test "$code" = 0
expect "git question: asked" says "Install git? (needs your password)"
expect "git question: apt-get install with sudo" grep -qx "sudo apt-get install -y git" "$LOG"
printf 'n\n' >"$work/tty-no"
run gitno MISOKU_TTY="$work/tty-no"
expect "git declined: exits 0, nothing run" test "$code" = 0 -a ! -s "$LOG"

# 13. Mac on Apple Silicon (sysctl, even under Rosetta): arm64 gh, the
# zsh PATH line, and Apple's git installer started without waiting.
run mac FAKE_OS=Darwin FAKE_ARCH=x86_64 FAKE_ARM64=1 SHELL=/bin/zsh \
	MISOKU_INSTALL_DIR="$(case_home mac)/bin" \
	MISOKU_GH_SHA256="$(sum "$ghrel/gh_${ghv}_macOS_arm64.zip")"
h=$(case_home mac)
expect "mac arm64: exits 0" test "$code" = 0
expect "mac arm64: universal misoku" test "$("$h/bin/misoku")" = "misoku new-macos"
expect "mac arm64: gh macOS_arm64 despite uname x86_64" test "$("$h/bin/gh")" = "gh version $ghv (macOS_arm64)"
expect "mac arm64: zsh PATH line" grep -qF "export PATH=\"\$HOME/bin:\$PATH\"" "$h/.zshrc"
expect "mac: Apple's git installer started" grep -qx "xcode-select --install" "$LOG"
expect "mac: says git is coming" says "Apple's installer for it opened"

# 14. Mac on Intel: amd64 gh; with Apple's tools present, no installer.
run macintel FAKE_OS=Darwin FAKE_ARCH=x86_64 FAKE_CLT=1 SHELL=/bin/zsh \
	MISOKU_INSTALL_DIR="$(case_home macintel)/bin" \
	MISOKU_GH_SHA256="$(sum "$ghrel/gh_${ghv}_macOS_amd64.zip")"
h=$(case_home macintel)
expect "mac intel: exits 0" test "$code" = 0
expect "mac intel: gh macOS_amd64" test "$("$h/bin/gh")" = "gh version $ghv (macOS_amd64)"
expect "mac intel: no git installer" test "$(grep -c -- --install "$LOG" || true)" = 0


# ---- Extra cases (independent verifier) -------------------------------

# 15. A HOME with spaces: installs, the PATH line is written once and
# quoted, and a second run adds nothing.
h="$work/home with space"
mkdir -p "$h"
run spaces HOME="$h"
h="$work/home with space"
expect "spaces: exits 0" test "$code" = 0
expect "spaces: misoku installed" test "$("$h/.local/bin/misoku")" = "misoku new-linux"
expect "spaces: gh installed" test "$("$h/.local/bin/gh")" = "gh version $ghv (linux_amd64)"
expect "spaces: one PATH line" test "$(count "$h/.bashrc" "$line")" = 1
run spaces HOME="$h"
h="$work/home with space"
expect "spaces rerun: exits 0" test "$code" = 0
expect "spaces rerun: still one PATH line" test "$(count "$h/.bashrc" "$line")" = 1

# 16. The temporary folder is removed after a success and after a failure.
mkdir -p "$work/tmp-ok" "$work/tmp-fail"
run tmpok TMPDIR="$work/tmp-ok"
expect "tmp: success leaves nothing" test -z "$(ls -A "$work/tmp-ok")"
run tmpfail TMPDIR="$work/tmp-fail" MISOKU_GH_SHA256=
expect "tmp: failure exits non-zero" test "$code" != 0
expect "tmp: failure leaves nothing" test -z "$(ls -A "$work/tmp-fail")"

# 17. A tampered misoku archive under the published checksums.txt stops.
cp -R "$rel" "$work/tampered"
script "$work/evil/misoku" "evil"
tar -czf "$work/tampered/misoku-linux.tar.gz" -C "$work/evil" misoku
run tampered MISOKU_BASE_URL="file://$work/tampered"
h=$(case_home tampered)
expect "tampered misoku: fails" test "$code" != 0
expect "tampered misoku: says mismatch" says "checksum mismatch for misoku-linux.tar.gz"
expect "tampered misoku: nothing installed" test ! -e "$h/.local/bin/misoku" -a ! -e "$h/.local/bin/gh"

# 18. A hash override with a matching tampered gh still fails over https.
run override2 PATH="$work/mirror:$work/bin" MISOKU_GH_BASE_URL="https://mirror.test" \
	MISOKU_GH_SHA256="$(sum "$ghrel/gh_${ghv}_linux_amd64.tar.gz")"
expect "override matching tampered gh over https: fails" test "$code" != 0
expect "override matching tampered gh over https: nothing installed" test ! -e "$(case_home override2)/.local/bin/gh"

# 19. A gh link in our folder that we did not put there (Homebrew on an
# Intel Mac links /usr/local/bin/gh) is left alone, dangling or not.
h=$(case_home ghlink)
mkdir -p "$h/.local/bin"
script "$work/cellar-gh/gh" "cellar gh"
ln -s "$work/cellar-gh/gh" "$h/.local/bin/gh"
run ghlink
expect "gh link: exits 0" test "$code" = 0
expect "gh link: still a link to theirs" test "$(readlink "$h/.local/bin/gh")" = "$work/cellar-gh/gh"
expect "gh link: target untouched" test "$("$work/cellar-gh/gh")" = "cellar gh"
expect "gh link: not marked" test ! -e "$h/.local/bin/.misoku-gh-version"
h=$(case_home ghdangling)
mkdir -p "$h/.local/bin"
ln -s "$work/nowhere/gh" "$h/.local/bin/gh"
run ghdangling
expect "dangling gh link: exits 0" test "$code" = 0
expect "dangling gh link: left as it is" test "$(readlink "$h/.local/bin/gh")" = "$work/nowhere/gh"

# 20. sudo runs only mkdir and mv, never a downloaded program.
tools "$work/bin-sudo"
cat >"$work/bin-sudo/sudo" <<'EOF'
#!/bin/sh
echo "sudo $*" >>"$LOG"
chmod u+w "$LOCKED"
"$@"
r=$?
chmod u-w "$LOCKED"
exit $r
EOF
chmod +x "$work/bin-sudo/sudo"
mkdir -p "$work/locked"
chmod u-w "$work/locked"
run sudoscope PATH="$work/bin-sudo" LOCKED="$work/locked" MISOKU_INSTALL_DIR="$work/locked/bin"
chmod u+w "$work/locked"
expect "sudo: exits 0" test "$code" = 0
expect "sudo: says it needs the password" says "needs your password"
expect "sudo: misoku installed" test "$("$work/locked/bin/misoku")" = "misoku new-linux"
expect "sudo: only mkdir and mv" test -z "$(grep -v -e '^sudo mkdir -p ' -e '^sudo mv ' "$LOG")"
expect "sudo: was used" grep -q '^sudo mv ' "$LOG"

# 21. Startup files: zsh with ZDOTDIR, bash on a Mac, plain sh, a file
# without a final newline, and a startup file that is a link (dotfiles).
run zdot SHELL=/bin/zsh ZDOTDIR="$work/zdot"
expect "zsh ZDOTDIR: line in \$ZDOTDIR/.zshrc" test "$(count "$work/zdot/.zshrc" "$line")" = 1
run macbash FAKE_OS=Darwin FAKE_CLT=1 SHELL=/bin/bash MISOKU_INSTALL_DIR="$(case_home macbash)/.local/bin" \
	MISOKU_GH_SHA256="$(sum "$ghrel/gh_${ghv}_macOS_amd64.zip")"
expect "mac bash: line in .bash_profile" test "$(count "$(case_home macbash)/.bash_profile" "$line")" = 1
run plainsh SHELL=/bin/sh
expect "sh: line in .profile" test "$(count "$(case_home plainsh)/.profile" "$line")" = 1
h=$(case_home nonl)
mkdir -p "$h"
printf 'alias x=y' >"$h/.bashrc"
run nonl
expect "no final newline: old line intact" test "$(count "$h/.bashrc" "alias x=y")" = 1
expect "no final newline: PATH line on its own" test "$(count "$h/.bashrc" "$line")" = 1
h=$(case_home rclink)
mkdir -p "$h" "$work/dotfiles"
printf '# mine\n' >"$work/dotfiles/bashrc"
ln -s "$work/dotfiles/bashrc" "$h/.bashrc"
run rclink
expect "linked .bashrc: still a link" test -L "$h/.bashrc"
expect "linked .bashrc: line added to the target" test "$(count "$work/dotfiles/bashrc" "$line")" = 1

# 22. Piped like `curl | sh`, with no terminal at all: does not hang,
# does not ask, and finishes.
h=$(case_home piped)
mkdir -p "$h"
LOG="$work/log-piped"
: >"$LOG"
out="$work/out-piped"
set +e
/usr/bin/env -i HOME="$h" PATH="$work/bin" SHELL=/bin/bash LOG="$LOG" \
	FAKE_OS=Linux FAKE_ARCH=x86_64 \
	MISOKU_BASE_URL="file://$rel" MISOKU_GH_BASE_URL="file://$ghrel" \
	MISOKU_GH_SHA256="$(sum "$ghrel/gh_${ghv}_linux_amd64.tar.gz")" \
	/usr/bin/perl -e 'alarm 60; exec @ARGV' /bin/sh -c "cat '$installer' | sh" >"$out" 2>&1 </dev/null
code=$?
set -e
expect "piped, no terminal: exits 0 (no hang)" test "$code" = 0
expect "piped, no terminal: not asked" test -z "$(grep -F 'Install git?' "$out")"
expect "piped, no terminal: finished" says "Close this window and open a new Terminal"

# 25. Piped like `curl | sh` at a real terminal (a pty from script(1)):
# the git question is read from /dev/tty, not from the piped script, and
# the rest of the script still runs.
if [ "$(/usr/bin/uname -s)" = Darwin ] && [ -x /usr/bin/script ]; then
	h=$(case_home pty)
	mkdir -p "$h"
	LOG="$work/log-pty"
	: >"$LOG"
	out="$work/out-pty"
	set +e
	{ /usr/bin/perl -e 'sleep 4'; printf 'y\n'; /usr/bin/perl -e 'sleep 4'; } | /usr/bin/env -i HOME="$h" PATH="$work/bin" SHELL=/bin/bash LOG="$LOG" \
		FAKE_OS=Linux FAKE_ARCH=x86_64 \
		MISOKU_BASE_URL="file://$rel" MISOKU_GH_BASE_URL="file://$ghrel" \
		MISOKU_GH_SHA256="$(sum "$ghrel/gh_${ghv}_linux_amd64.tar.gz")" \
		/usr/bin/perl -e 'alarm 60; exec @ARGV' /usr/bin/script -q /dev/null /bin/sh -c "cat '$installer' | sh" >"$out" 2>&1
	code=$?
	set -e
	expect "pty piped: exits 0" test "$code" = 0
	expect "pty piped: asked" says "Install git? (needs your password)"
	expect "pty piped: answer from the terminal installs git" grep -qx "sudo apt-get install -y git" "$LOG"
	expect "pty piped: finished" says "Close this window and open a new Terminal"
fi
# 23. A package manager that reads stdin while the script is piped in
# must not eat the rest of the script (dash reads its input in blocks).
if command -v dash >/dev/null 2>&1; then
	tools "$work/bin-greedy"
	cat >"$work/bin-greedy/apt-get" <<'EOF'
#!/bin/sh
echo "apt-get $*" >>"$LOG"
cat >/dev/null
EOF
	chmod +x "$work/bin-greedy/apt-get"
	ln -sf "$(command -v dash)" "$work/bin-greedy/dash"
	h=$(case_home greedy)
	mkdir -p "$h"
	LOG="$work/log-greedy"
	: >"$LOG"
	out="$work/out-greedy"
	set +e
	/usr/bin/env -i HOME="$h" PATH="$work/bin-greedy" SHELL=/bin/bash LOG="$LOG" \
		FAKE_OS=Linux FAKE_ARCH=x86_64 MISOKU_TTY="$work/tty-yes" \
		MISOKU_BASE_URL="file://$rel" MISOKU_GH_BASE_URL="file://$ghrel" \
		MISOKU_GH_SHA256="$(sum "$ghrel/gh_${ghv}_linux_amd64.tar.gz")" \
		/bin/sh -c "cat '$installer' | dash" >"$out" 2>&1
	code=$?
	set -e
	expect "dash piped, apt-get reads stdin: git installed" grep -qx "apt-get install -y git" "$LOG"
	expect "dash piped, apt-get reads stdin: rest of script ran" says "Close this window and open a new Terminal"
fi

# 24. A misoku link in the default folder (stow, a dev build): the note
# says it was not updated, so it must still be the link.
h=$(case_home homelink)
mkdir -p "$h/.local/bin" "$work/devbuild"
script "$work/devbuild/misoku" "misoku dev"
ln -s "$work/devbuild/misoku" "$h/.local/bin/misoku"
run homelink PATH="$work/bin:$h/.local/bin"
expect "link in default folder: exits 0" test "$code" = 0
if says "was not updated"; then
	expect "link in default folder: note true (link kept)" test -L "$h/.local/bin/misoku"
fi
expect "link in default folder: replaced by the new copy" test ! -L "$h/.local/bin/misoku" -a "$("$h/.local/bin/misoku")" = "misoku new-linux"
expect "link in default folder: its target untouched" test "$("$work/devbuild/misoku")" = "misoku dev"

# 26. A Mac account that cannot sudo (not an administrator, or no
# terminal): /usr/local/bin is not writable and sudo -v fails, so misoku
# goes to ~/.local/bin with the PATH line and a new Terminal.
mkdir -p "$work/maclocked"
chmod u-w "$work/maclocked"
run macnoadmin FAKE_OS=Darwin FAKE_ARM64=1 FAKE_CLT=1 FAKE_SUDO_FAIL=1 SHELL=/bin/zsh \
	MISOKU_TEST_MAC_DIR="$work/maclocked/bin" \
	MISOKU_GH_SHA256="$(sum "$ghrel/gh_${ghv}_macOS_arm64.zip")"
chmod u+w "$work/maclocked"
h=$(case_home macnoadmin)
expect "mac no admin: exits 0" test "$code" = 0
expect "mac no admin: sudo tried once" test "$(grep -c '^sudo' "$LOG")" = 1 -a "$(grep -cx 'sudo -v' "$LOG")" = 1
expect "mac no admin: misoku in ~/.local/bin" test "$("$h/.local/bin/misoku")" = "misoku new-macos"
expect "mac no admin: gh in ~/.local/bin" test "$("$h/.local/bin/gh")" = "gh version $ghv (macOS_arm64)"
expect "mac no admin: nothing in the Mac folder" test ! -e "$work/maclocked/bin"
expect "mac no admin: zsh PATH line" test "$(count "$h/.zshrc" "$line")" = 1
expect "mac no admin: open a new Terminal" says "Close this window and open a new Terminal"

# 27. A Mac admin: sudo -v works, so misoku goes to the Mac folder with sudo.
tools "$work/bin-macsudo"
cat >"$work/bin-macsudo/sudo" <<'EOF'
#!/bin/sh
echo "sudo $*" >>"$LOG"
[ "$1" = -v ] && exit 0
chmod u+w "$LOCKED"
"$@"
r=$?
chmod u-w "$LOCKED"
exit $r
EOF
chmod +x "$work/bin-macsudo/sudo"
mkdir -p "$work/macadmin"
chmod u-w "$work/macadmin"
run macadmin PATH="$work/bin-macsudo" LOCKED="$work/macadmin" FAKE_OS=Darwin FAKE_ARM64=1 FAKE_CLT=1 SHELL=/bin/zsh \
	MISOKU_TEST_MAC_DIR="$work/macadmin/bin" \
	MISOKU_GH_SHA256="$(sum "$ghrel/gh_${ghv}_macOS_arm64.zip")"
chmod u+w "$work/macadmin"
expect "mac admin: exits 0" test "$code" = 0
expect "mac admin: password asked once" test "$(grep -c 'needs your password' "$out")" = 1
expect "mac admin: misoku in the Mac folder" test "$("$work/macadmin/bin/misoku")" = "misoku new-macos"
expect "mac admin: sudo -v first" test "$(head -n 1 "$LOG")" = "sudo -v"

if [ "$fails" -ne 0 ]; then
	echo "$fails install check(s) failed" >&2
	exit 1
fi
echo "install.sh checks OK"
