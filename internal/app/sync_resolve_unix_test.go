//go:build !windows

// The fake git and gh are shell scripts: these tests run on macOS and Linux.
// They change PATH, so none runs in parallel. No real gh or browser runs.

package app_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// deniedGit fails origin's network commands as git does when the access
// expired, until the fake gh gave it access; then origin reaches $FAKE_REMOTE.
const deniedGit = `#!/bin/sh
d="$FAKE_GH_DIR"
for a in "$@"; do
	case "$a" in fetch|push|ls-remote)
		[ -f "$d/access" ] && exec "$REAL_GIT" -c "url.$FAKE_REMOTE.insteadOf=$FAKE_ORIGIN" "$@"
		echo "fatal: unable to access '$FAKE_ORIGIN': could not read Username: terminal prompts disabled" >&2
		exit 128 ;;
	esac
done
exec "$REAL_GIT" "$@"
`

// signInFakeGH logs its calls. nologin: the sign-in does not finish. nosetup:
// git gets no access.
const signInFakeGH = `#!/bin/sh
d="$FAKE_GH_DIR"
printf '%s\n' "$*" >> "$d/calls"
case "$1 $2" in
"auth status") [ -f "$d/loggedin" ] && exit 0; echo "not logged in" >&2; exit 1 ;;
"auth login") [ -f "$d/nologin" ] || : > "$d/loggedin" ;;
"auth setup-git") [ -f "$d/nosetup" ] && { echo "gh refused" >&2; exit 1; }; : > "$d/access" ;;
*) exit 2 ;;
esac
`

const githubOrigin = "https://github.com/ana/notes.git"

type deniedVault struct {
	remote, vault, fake, cfg string
	outside                  []string
	exec                     app.Option
}

// newDeniedVault is a synced vault whose origin refuses git. PATH holds only
// the fake git and, with gh, the fake gh.
func newDeniedVault(t *testing.T, origin string, gh bool) *deniedVault {
	t.Helper()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not installed")
	}
	remote, vault := syncedVault(t)
	d := &deniedVault{remote: remote, vault: vault, fake: t.TempDir()}
	bin := filepath.Join(d.fake, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(deniedGit), 0o755); err != nil {
		t.Fatal(err)
	}
	if gh {
		if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(signInFakeGH), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("FAKE_GH_DIR", d.fake)
	t.Setenv("REAL_GIT", realGit)
	t.Setenv("FAKE_REMOTE", remote)
	t.Setenv("FAKE_ORIGIN", origin)
	t.Setenv("PATH", bin)
	gitIn(t, vault, "remote", "set-url", "origin", origin)
	d.cfg = syncConfig(t, vault)
	d.exec = app.WithExec(func(c *exec.Cmd, done tea.ExecCallback) tea.Cmd {
		d.outside = append(d.outside, strings.Join(c.Args[1:], " "))
		return func() tea.Msg { return done(c.Run()) }
	})
	return d
}

func (d *deniedVault) window(t *testing.T) *harness {
	t.Helper()
	h := wideWindow(t, d.vault, d.cfg, d.exec)
	h.finish(h.update(app.SyncNowMsg{}))
	return h
}

func (d *deniedVault) calls(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(d.fake, "calls"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(data)
}

func (d *deniedVault) flag(t *testing.T, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(d.fake, name), nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

const signInLogin = "auth login --web --git-protocol https --hostname github.com"

// S8: the key offers the assistant's sign-in; after it, sync runs.
func TestSyncKeySignsInToGitHubThenSyncs(t *testing.T) {
	d := newDeniedVault(t, githubOrigin, true)
	writeOutside(t, d.vault, "new.md", "fresh")
	h := d.window(t)
	wantLine(t, h, "GitHub did not let this computer in · space s signs in")
	wantLine(t, h, "sync error · space s")
	wantNoDialog(t, h)
	if strings.Contains(bottomRow(h), "gh auth") {
		t.Fatalf("the bar names the key, not gh's commands:\n%s", bottomRow(h))
	}
	atNote(h)
	if cmd := pressSync(h); cmd != nil {
		t.Fatal("opening the offer must run nothing")
	}
	if box := dialogText(h.screen()); !strings.Contains(box, "Sign in") || !strings.Contains(box, "Cancel") || !strings.Contains(box, "one-time code") {
		t.Fatalf("the offer must be clear:\n%s", box)
	}
	if got := d.calls(t); got != "" || len(d.outside) != 0 {
		t.Fatalf("nothing runs before the choice: %q %q", got, d.outside)
	}
	h.key(tea.KeyLeft)
	h.finish(h.update(tea.KeyMsg{Type: tea.KeyEnter}))
	if len(d.outside) != 1 || d.outside[0] != signInLogin {
		t.Fatalf("the sign-in must run outside the screen, once: %q", d.outside)
	}
	want := signInLogin + "\nauth status --hostname github.com\nauth setup-git --hostname github.com\n"
	if got := d.calls(t); got != want {
		t.Fatalf("gh ran:\n%s\nwant:\n%s", got, want)
	}
	if !onRemote(t, d.remote, "new.md") {
		t.Fatalf("after signing in, sync runs:\n%s", h.screen())
	}
	wantLine(t, h, "Synced")
	wantNoDialog(t, h)
}

// S8: Cancel runs nothing and leaves the state as it was.
func TestSignInCancelChangesNothing(t *testing.T) {
	d := newDeniedVault(t, githubOrigin, true)
	h := d.window(t)
	before := repoState(t, d.vault)
	barBefore := bottomRow(h)
	atNote(h)
	pressSync(h)
	h.keys("sq") // a second press of the key is not a choice
	if len(d.outside) != 0 || !strings.Contains(h.screen(), "Cancel") {
		t.Fatalf("a letter must not choose: %q\n%s", d.outside, h.screen())
	}
	if cmd := h.update(tea.KeyMsg{Type: tea.KeyEsc}); cmd != nil {
		t.Fatal("Cancel must run nothing")
	}
	wantNoDialog(t, h)
	if got := d.calls(t); got != "" || len(d.outside) != 0 {
		t.Fatalf("Cancel must run nothing: %q %q", got, d.outside)
	}
	if after := repoState(t, d.vault); after != before {
		t.Fatalf("Cancel must change nothing:\n%s\nwas:\n%s", after, before)
	}
	if !strings.Contains(barBefore, "sync error") || !strings.Contains(bottomRow(h), "sync error · space s") {
		t.Fatalf("the indicator stays:\n%s", bottomRow(h))
	}
	if cmd := pressSync(h); cmd != nil || !strings.Contains(h.screen(), "one-time code") {
		t.Fatalf("the key offers it again:\n%s", h.screen())
	}
}

// S8: a sign-in that did not finish says so, gives git nothing and syncs nothing.
func TestSignInThatDidNotFinishStaysStopped(t *testing.T) {
	for _, c := range []struct {
		flag, want, calls string
	}{
		{"nologin", "the sign-in did not finish", signInLogin + "\nauth status --hostname github.com\n"},
		{"nosetup", "gh refused", signInLogin + "\nauth status --hostname github.com\nauth setup-git --hostname github.com\n"},
	} {
		t.Run(c.flag, func(t *testing.T) {
			d := newDeniedVault(t, githubOrigin, true)
			d.flag(t, c.flag)
			writeOutside(t, d.vault, "new.md", "fresh")
			h := d.window(t)
			atNote(h)
			pressSync(h)
			h.key(tea.KeyLeft)
			h.finish(h.update(tea.KeyMsg{Type: tea.KeyEnter}))
			wantLine(t, h, c.want)
			wantNoDialog(t, h)
			if got := d.calls(t); got != c.calls {
				t.Fatalf("gh ran:\n%s\nwant:\n%s", got, c.calls)
			}
			if onRemote(t, d.remote, "new.md") {
				t.Fatal("nothing may sync")
			}
			h.key(tea.KeyEsc)
			if cmd := pressSync(h); cmd != nil || !strings.Contains(h.screen(), "one-time code") {
				t.Fatalf("the key offers it again:\n%s", h.screen())
			}
		})
	}
}

// S8: no sign-in where it could not help: it explains, and Check again syncs.
func TestAccessStopWithNoSignInOnlyExplains(t *testing.T) {
	for _, c := range []struct {
		name, origin string
		gh           bool
	}{
		{"no gh", githubOrigin, false},
		{"another host", "https://gitlab.example/ana/notes.git", true},
		{"a lookalike host", "https://github.com.evil.example/ana/notes.git", true},
		{"a lookalike path", "https://evil.example/github.com/x", true},
		{"ssh to GitHub", "git@github.com:ana/notes.git", true},
		{"http to GitHub", "http://github.com/ana/notes.git", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := newDeniedVault(t, c.origin, c.gh)
			h := d.window(t)
			wantLine(t, h, "Sync error: could not sign in to origin")
			// The message itself names the key, not only the indicator.
			wantLine(t, h, " · space s  Esc dismiss")
			if strings.Contains(bottomRow(h), "signs in") {
				t.Fatalf("no sign-in is on offer:\n%s", bottomRow(h))
			}
			atNote(h)
			before := repoState(t, d.vault)
			if cmd := pressSync(h); cmd != nil {
				t.Fatal("explaining must run nothing")
			}
			box := dialogText(h.screen())
			if strings.Contains(box, "Sign in") || !strings.Contains(noSpace(box), noSpace("could not sign in to origin")) || !strings.Contains(box, "Check again") {
				t.Fatalf("it must explain, with no sign-in:\n%s", box)
			}
			h.key(tea.KeyLeft)
			h.finish(h.update(tea.KeyMsg{Type: tea.KeyEnter}))
			wantNoDialog(t, h)
			wantLine(t, h, "Sync error: could not sign in to origin")
			if len(d.outside) != 0 || d.calls(t) != "" {
				t.Fatalf("gh must not run: %q %q", d.outside, d.calls(t))
			}
			if after := repoState(t, d.vault); after != before {
				t.Fatalf("nothing may change:\n%s\nwas:\n%s", after, before)
			}
		})
	}
}

// Credentials in origin's URL appear in no message, and gh's access would not
// replace them: no sign-in.
func TestEmbeddedCredentialsNeverShow(t *testing.T) {
	const secret = "s3cret"
	d := newDeniedVault(t, "https://tok:"+secret+"@github.com/ana/notes.git", true)
	h := d.window(t)
	screens := []string{h.screen()}
	atNote(h)
	pressSync(h)
	box := h.screen()
	screens = append(screens, box)
	if strings.Contains(box, "Sign in") {
		t.Fatalf("no sign-in over the URL's own credentials:\n%s", box)
	}
	h.key(tea.KeyEsc)
	screens = append(screens, messagesView(h))
	askSyncStatus(h)
	screens = append(screens, h.screen())
	for _, s := range screens {
		if strings.Contains(s, secret) || strings.Contains(s, "tok:") {
			t.Fatalf("credentials on screen:\n%s", s)
		}
	}
}

// S5: an automatic sync that is denied opens nothing, not even the offer.
func TestDeniedAutomaticSyncOpensNoDialog(t *testing.T) {
	d := newDeniedVault(t, githubOrigin, true)
	clock := newTestClock()
	h := wideWindow(t, d.vault, intervalConfig(t, d.vault, "2m"), d.exec, app.WithClock(clock.Now))
	if !initAsksForSync(h) {
		t.Fatal("opening must sync")
	}
	h.finish(h.update(app.SyncNowMsg{}))
	wantNoDialog(t, h)
	for range 2 {
		clock.add(2 * time.Minute)
		if n := poll(h); n != 1 {
			t.Fatalf("a timed sync must run (%d commands ran)", n)
		}
		wantNoDialog(t, h)
	}
	if d.calls(t) != "" || len(d.outside) != 0 {
		t.Fatalf("gh must not run on its own: %q %q", d.calls(t), d.outside)
	}
}

// Stops that keep today's behaviour and need a program out of the way.
func TestGitProblemsKeepTheirTextAndSync(t *testing.T) {
	for _, c := range []struct {
		name, script, want string
	}{
		{"git missing", "", "Sync paused: git did not run"},
		{"git too old", "#!/bin/sh\necho git version 2.0.0\n", "Sync paused: git 2.0 is too old"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, vault := syncedVault(t)
			cfg := syncConfig(t, vault)
			bin := t.TempDir()
			if c.script != "" {
				if err := os.WriteFile(filepath.Join(bin, "git"), []byte(c.script), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", bin)
			h := wideWindow(t, vault, cfg)
			h.finish(h.update(app.SyncNowMsg{}))
			wantLine(t, h, c.want)
			if strings.Contains(bottomRow(h), "space s") {
				t.Fatalf("this stop's text does not change:\n%s", bottomRow(h))
			}
			h.finish(h.update(app.SyncNowMsg{}))
			wantNoDialog(t, h)
		})
	}
}

// Enter alone launches nothing: the sign-in starts on Cancel, as the lock's
// dialog does, and takes an arrow or Tab first.
func TestEnterAloneDoesNotSignIn(t *testing.T) {
	d := newDeniedVault(t, githubOrigin, true)
	h := d.window(t)
	atNote(h)
	pressSync(h)
	if cmd := h.update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
		t.Fatal("Enter on Cancel must run nothing")
	}
	wantNoDialog(t, h)
	if len(d.outside) != 0 || d.calls(t) != "" {
		t.Fatalf("Enter alone must launch nothing: %q %q", d.outside, d.calls(t))
	}
	pressSync(h)
	h.key(tea.KeyTab)
	h.finish(h.update(tea.KeyMsg{Type: tea.KeyEnter}))
	if len(d.outside) != 1 {
		t.Fatalf("Tab, then Enter, chooses Sign in: %q", d.outside)
	}
}

// The access stop takes the bar's room as the others do.
func TestAccessStopsShowAtNormalWidths(t *testing.T) {
	for _, c := range []struct {
		name, origin string
		rows         map[int]string
	}{
		{"sign-in offered", githubOrigin, map[int]string{
			80:  "Sync paused · space s signs in",
			100: "Sync paused · space s signs in",
			120: "Sync paused: GitHub did not let this computer in · space s signs in",
			140: "Sync paused: GitHub did not let this computer in · space s signs in",
		}},
		{"no sign-in", "https://gitlab.example/ana/notes.git", map[int]string{
			80:  "Sync error · space s",
			100: "Sync error · space s",
			120: "Sync error · space s",
			140: "Sync error · space s",
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := newDeniedVault(t, c.origin, true)
			h := wideWindow(t, d.vault, d.cfg, d.exec)
			atNote(h)
			h.finish(h.update(app.SyncNowMsg{}))
			for w, want := range c.rows {
				if got := barAt(h, w); !strings.Contains(got, want) {
					t.Errorf("at %d columns the bar must show %q:\n%s", w, want, got)
				}
			}
		})
	}
}

// Fix for names git can carry and a terminal must not obey: a line break, or
// escape sequences. A placeholder shows wherever arrivals are shown.
func TestArrivedNamesWithControlCharactersAreSafe(t *testing.T) {
	remote, _, h := arrivalWindow(t)
	const clear, clip = "\x1b[2J", "\x1b]52;c;QUJD\x07"
	names := []string{"two\nlines.md", "wipe" + clear + ".md", "clip" + clip + ".md", "del\x7f.md"}
	pushNotes(t, remote, names...)
	h.finish(h.update(app.SyncNowMsg{}))
	seen := map[string]string{
		"bar":     barAt(h, 200),
		"compact": barAt(h, 80),
	}
	h.send(tea.WindowSizeMsg{Width: 200, Height: termH})
	if rows := strings.Count(h.screen(), "\n") + 1; rows != termH {
		t.Errorf("a name must not add a row: %d rows in a frame of %d", rows, termH)
	}
	seen["messages"] = messagesView(h)
	askSyncStatus(h)
	seen["sync status"] = h.screen()
	for where, v := range seen {
		for _, raw := range []string{clear, clip, "\x1b]52", "two\nlines"} {
			if strings.Contains(v, raw) {
				t.Errorf("%s must not carry %q:\n%q", where, raw, v)
			}
		}
	}
	for _, where := range []string{"messages", "sync status"} {
		if !strings.Contains(seen[where], "two\uFFFDlines.md") || !strings.Contains(seen[where], "del\uFFFD.md") {
			t.Errorf("%s must show the placeholder:\n%s", where, seen[where])
		}
	}
	if !strings.Contains(seen["bar"], "\uFFFD") || strings.Contains(seen["bar"], "\x7f") {
		t.Errorf("the bar must show the placeholder:\n%s", seen["bar"])
	}
}

// The access stops keep the key from the app's minimum width up too.
func TestAccessStopsKeepTheKeyFromTheMinimumWidth(t *testing.T) {
	for _, c := range []struct {
		name, origin string
		long         bool
		at60         string
	}{
		{"sign-in offered", githubOrigin, false, "Sync paused · space s signs in"},
		{"sign-in offered, long name", githubOrigin, true, "Sync paused · space s"},
		{"no sign-in", "https://gitlab.example/ana/notes.git", false, "Sync error · space s"},
		{"no sign-in, long name", "https://gitlab.example/ana/notes.git", true, "Sync error · space s"},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := newDeniedVault(t, c.origin, true)
			if c.long {
				writeOutside(t, d.vault, "a-note-with-a-name-longer-than-the-bar-shows.md", "long")
				if err := os.Remove(filepath.Join(d.vault, "doc.md")); err != nil {
					t.Fatal(err)
				}
			}
			h := wideWindow(t, d.vault, d.cfg, d.exec)
			atNote(h)
			h.finish(h.update(app.SyncNowMsg{}))
			if got := barAt(h, 60); !strings.Contains(got, c.at60) {
				t.Errorf("at 60 columns the bar must show %q:\n%s", c.at60, got)
			}
			for w := 60; w <= 140; w++ {
				if got := barAt(h, w); !strings.Contains(got, "· space s") {
					t.Errorf("at %d columns the bar must keep the key:\n%s", w, got)
				}
			}
		})
	}
}
