package app_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// pressSync presses the default sync key at a note and returns its command.
func pressSync(h *harness) tea.Cmd {
	h.t.Helper()
	h.send(tea.KeyMsg{Type: tea.KeySpace})
	return h.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
}

// atNote leaves the start screen for doc.md, in normal mode.
func atNote(h *harness) {
	h.t.Helper()
	h.keys("1")
	h.key(tea.KeyEsc)
}

// wideWindow is a window that shows whole lines in the bar and in dialogs.
func wideWindow(t *testing.T, vault, cfg string, opts ...app.Option) *harness {
	t.Helper()
	h := window(t, vault, cfg, opts...)
	h.send(tea.WindowSizeMsg{Width: 1000, Height: termH})
	return h
}

func bottomRow(h *harness) string {
	h.t.Helper()
	lines := strings.Split(h.screen(), "\n")
	return lines[len(lines)-1]
}

func wantLine(t *testing.T, h *harness, want string) {
	t.Helper()
	if got := bottomRow(h); !strings.Contains(got, want) {
		t.Fatalf("the status bar must show %q:\n%s", want, got)
	}
}

// wantNoDialog fails when a dialog of the sync key is open.
func wantNoDialog(t *testing.T, h *harness) {
	t.Helper()
	v := h.screen()
	for _, button := range []string{"Check again", removeButton, "Sign in"} {
		if strings.Contains(v, button) {
			t.Fatalf("no dialog may be open (%q):\n%s", button, v)
		}
	}
}

// repoState is everything a dialog that "changes nothing" must leave alone.
func repoState(t *testing.T, vault string) string {
	t.Helper()
	return gitIn(t, vault, "rev-parse", "HEAD") + "\n" +
		gitIn(t, vault, "status", "--porcelain", "--ignored") + "\n" +
		gitIn(t, vault, "for-each-ref")
}

// onlyStopButtons is a row of the box holding these two buttons and no other.
var onlyStopButtons = regexp.MustCompile(`│\s+Check again\s+Close\s+│`)

var gitWord = regexp.MustCompile(`(?i)\bgit\b`)

// wantNoCommand fails when text names a git or Misoku command.
func wantNoCommand(t *testing.T, text string) {
	t.Helper()
	if gitWord.MatchString(text) {
		t.Errorf("no git in plain words:\n%s", text)
	}
	for _, w := range []string{"`", "misoku ", ":sync", " --", "git-", "gh auth"} {
		if strings.Contains(text, w) {
			t.Errorf("no command (%q):\n%s", w, text)
		}
	}
}

// historyStop is a stop the sync key only explains. fix is what the user does
// about it outside Misoku.
type historyStop struct {
	name string
	make func(t *testing.T, remote, vault string)
	fix  func(t *testing.T, remote, vault string)
	bar  string
	why  string
	// detail is what `:sync status` keeps for someone who knows git.
	detail string
}

var historyStops = []historyStop{
	{
		name: "rewritten",
		make: func(t *testing.T, remote, _ string) {
			desk := clone(t, remote)
			gitIn(t, desk, "commit", "-q", "--amend", "-m", "rewritten")
			gitIn(t, desk, "push", "-q", "-f", "origin", "main")
		},
		fix: func(t *testing.T, _, vault string) {
			gitIn(t, vault, "fetch", "-q", "origin")
			gitIn(t, vault, "reset", "-q", "--hard", "origin/main")
		},
		bar:    "this vault and origin no longer match",
		why:    "was rewritten",
		detail: "git fetch origin",
	},
	{
		name: "unrelated",
		make: func(t *testing.T, _, vault string) {
			parent := t.TempDir()
			gitIn(t, parent, "init", "-q", "--bare", "other.git")
			other := filepath.Join(parent, "other.git")
			desk := clone(t, other)
			writeOutside(t, desk, "README.md", "another history")
			gitIn(t, desk, "add", "-A")
			gitIn(t, desk, "commit", "-q", "-m", "other")
			gitIn(t, desk, "push", "-q", "origin", "HEAD:main")
			gitIn(t, vault, "remote", "set-url", "origin", other)
			gitIn(t, vault, "update-ref", "-d", "refs/remotes/origin/main")
		},
		fix: func(t *testing.T, remote, vault string) {
			gitIn(t, vault, "remote", "set-url", "origin", remote)
			gitIn(t, vault, "update-ref", "refs/remotes/origin/main", "HEAD")
		},
		bar:    "this vault and origin share no history",
		why:    "share no history",
		detail: "misoku sync init",
	},
	{
		name: "merge in progress",
		make: func(t *testing.T, _, vault string) {
			head := gitIn(t, vault, "rev-parse", "HEAD")
			writeOutside(t, vault, filepath.Join(".git", "MERGE_HEAD"), head+"\n")
		},
		fix: func(t *testing.T, _, vault string) {
			if err := os.Remove(filepath.Join(vault, ".git", "MERGE_HEAD")); err != nil {
				t.Fatal(err)
			}
		},
		bar:    "an unfinished merge or rebase is waiting in this vault",
		why:    "not finished",
		detail: "git merge --abort",
	},
	{
		name: "detached head",
		make: func(t *testing.T, _, vault string) {
			gitIn(t, vault, "checkout", "-q", "--detach")
		},
		fix: func(t *testing.T, _, vault string) {
			gitIn(t, vault, "checkout", "-q", "main")
		},
		bar:    "this vault is not on a branch",
		why:    "not on a branch",
		detail: "git switch main",
	},
}

// stoppedWindow is a window whose first sync hit c.
func stoppedWindow(t *testing.T, c historyStop, opts ...app.Option) (remote, vault string, h *harness) {
	t.Helper()
	remote, vault = syncedVault(t)
	c.make(t, remote, vault)
	h = wideWindow(t, vault, syncConfig(t, vault), opts...)
	h.finish(h.update(app.SyncNowMsg{}))
	return remote, vault, h
}

// S9: plain words in the bar, and the key explains without changing anything.
func TestSyncKeyExplainsAStopAndChangesNothing(t *testing.T) {
	t.Parallel()
	for _, c := range historyStops {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, vault, h := stoppedWindow(t, c)
			wantLine(t, h, "Sync paused: "+c.bar+" · space s")
			wantLine(t, h, "sync paused · space s")
			wantNoCommand(t, bottomRow(h))
			wantNoDialog(t, h)
			before := repoState(t, vault)
			atNote(h)
			if cmd := pressSync(h); cmd != nil {
				t.Fatal("opening the explanation must run nothing")
			}
			box := dialogText(h.screen())
			for _, want := range []string{c.why, "Nothing was changed", "Check again", "Close"} {
				if !strings.Contains(box, want) {
					t.Fatalf("the dialog must say %q:\n%s", want, box)
				}
			}
			wantNoCommand(t, box)
			// No button joins histories, aborts a merge or switches branches.
			if !onlyStopButtons.MatchString(h.screen()) {
				t.Errorf("the only buttons are Check again and Close:\n%s", h.screen())
			}
			// No letter chooses: a second press of the key is not a choice.
			h.keys("csqj")
			if !strings.Contains(h.screen(), "Check again") {
				t.Fatal("a letter must not close or choose")
			}
			if cmd := h.update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil { // Close, the default
				t.Fatal("Close must run nothing")
			}
			wantNoDialog(t, h)
			if after := repoState(t, vault); after != before {
				t.Fatalf("Close must change nothing:\n%s\nwas:\n%s", after, before)
			}
		})
	}
}

// S9: Check again is a normal sync: a problem the user fixed clears.
func TestCheckAgainClearsAFixedProblem(t *testing.T) {
	t.Parallel()
	for _, c := range historyStops {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			remote, vault, h := stoppedWindow(t, c)
			atNote(h)
			pressSync(h)
			c.fix(t, remote, vault)
			h.key(tea.KeyLeft)
			h.finish(h.update(tea.KeyMsg{Type: tea.KeyEnter}))
			wantNoDialog(t, h)
			wantLine(t, h, "Synced")
			writeOutside(t, vault, "new.md", "fresh")
			h.finish(pressSync(h)) // the key syncs again, with no dialog
			wantNoDialog(t, h)
			if !onRemote(t, remote, "new.md") {
				t.Fatal("once fixed, the key must sync")
			}
		})
	}
}

// S9: the same stop says so again, and the dialog does not reopen on its own.
func TestCheckAgainRepeatsARealStop(t *testing.T) {
	t.Parallel()
	for _, c := range historyStops {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, vault, h := stoppedWindow(t, c)
			before := repoState(t, vault)
			atNote(h)
			pressSync(h)
			h.key(tea.KeyLeft)
			h.finish(h.update(tea.KeyMsg{Type: tea.KeyEnter}))
			wantNoDialog(t, h)
			wantLine(t, h, "Sync paused: "+c.bar+" · space s")
			if after := repoState(t, vault); after != before {
				t.Fatalf("Check again must change nothing here:\n%s\nwas:\n%s", after, before)
			}
			if cmd := pressSync(h); cmd != nil {
				t.Fatal("the key explains again")
			}
			if !strings.Contains(h.screen(), "Check again") {
				t.Fatalf("the key must open the explanation again:\n%s", h.screen())
			}
		})
	}
}

// `:sync status` keeps the technical detail, git commands included.
func TestSyncStatusKeepsTheTechnicalDetail(t *testing.T) {
	t.Parallel()
	for _, c := range historyStops {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, _, h := stoppedWindow(t, c)
			askSyncStatus(h)
			if box := noSpace(dialogText(h.screen())); !strings.Contains(box, noSpace(c.detail)) {
				t.Fatalf(":sync status must keep %q:\n%s", c.detail, box)
			}
		})
	}
}

// S5: the sync on opening and timed syncs hitting any stop open no dialog,
// also once the stop is known.
func TestNoStopOpensADialogOnItsOwn(t *testing.T) {
	t.Parallel()
	cases := map[string]func(t *testing.T, remote, vault string){
		"lock": func(t *testing.T, _, vault string) { leftoverLock(t, vault) },
		"offline": func(t *testing.T, remote, _ string) {
			if err := os.Rename(remote, remote+".gone"); err != nil {
				t.Fatal(err)
			}
		},
	}
	for _, c := range historyStops {
		cases[c.name] = c.make
	}
	for name, stop := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			remote, vault := syncedVault(t)
			stop(t, remote, vault)
			clock := newTestClock()
			h := wideWindow(t, vault, intervalConfig(t, vault, "2m"), app.WithClock(clock.Now))
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
		})
	}
}

// S5: a key pressed during the commit after a save starts a sync when the
// commit ends: by then the user may be typing, so it opens nothing.
func TestSyncQueuedDuringACommitOpensNoDialog(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	clock := newTestClock()
	h := wideWindow(t, vault, intervalConfig(t, vault, "0"), app.WithClock(clock.Now))
	gitIn(t, vault, "checkout", "-q", "--detach")
	h.finish(h.update(app.SyncNowMsg{}))
	openAndSave(h, "more")
	clock.add(10 * time.Second)
	commit := h.update(app.CheckWorkspaceMsg{})
	if cmd := pressSync(h); cmd != nil {
		t.Fatal("a key during the commit waits for it")
	}
	for _, msg := range runAll(t, commit) {
		h.send(msg)
	}
	wantNoDialog(t, h)
	wantLine(t, h, "not on a branch")
}

// S6: the bar and the indicator say the key retries, and it does.
func TestSyncKeyRetriesWhileOffline(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	if err := os.Rename(remote, remote+".gone"); err != nil {
		t.Fatal(err)
	}
	writeOutside(t, vault, "new.md", "fresh")
	h := wideWindow(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	wantLine(t, h, "Sync: offline, 1 pending · space s retries")
	wantLine(t, h, "offline · 1 pending · space s")
	atNote(h)
	h.finish(pressSync(h)) // still offline: tries, and says so again
	wantNoDialog(t, h)
	wantLine(t, h, "Sync: offline, 1 pending · space s retries")
	if err := os.Rename(remote+".gone", remote); err != nil {
		t.Fatal(err)
	}
	h.key(tea.KeyEsc)
	h.finish(pressSync(h))
	wantNoDialog(t, h)
	if !onRemote(t, remote, "new.md") {
		t.Fatal("the key must retry the sync")
	}
	wantLine(t, h, "Synced")
}

// S6: the bar names the key the user actually has.
func TestStoppedBarNamesTheConfiguredKey(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		keys []string
		want string
	}{
		{"rebound", []string{"ctrl+y"}, "Ctrl+Y"},
		{"rebound behind the leader", []string{"space y"}, "space y"},
		{"unbound", []string{}, ":sync"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			remote, vault := syncedVault(t)
			if err := os.Rename(remote, remote+".gone"); err != nil {
				t.Fatal(err)
			}
			opt := app.WithKeys(map[string][]string{"sync_now": c.keys})
			h := wideWindow(t, vault, syncConfig(t, vault), opt)
			atNote(h) // the fallback `:sync` works only with a note open
			h.finish(h.update(app.SyncNowMsg{}))
			wantLine(t, h, "pending · "+c.want+" retries")
			if strings.Contains(bottomRow(h), "space s") {
				t.Fatalf("the default key is not this user's:\n%s", bottomRow(h))
			}

			detached := historyStops[3]
			dremote, dvault := syncedVault(t)
			detached.make(t, dremote, dvault)
			stopped := wideWindow(t, dvault, syncConfig(t, dvault), opt)
			atNote(stopped)
			stopped.finish(stopped.update(app.SyncNowMsg{}))
			wantLine(t, stopped, detached.bar+" · "+c.want)

			_, locked := syncedVault(t)
			leftoverLock(t, locked)
			lh := wideWindow(t, locked, syncConfig(t, locked), opt)
			atNote(lh)
			lh.finish(lh.update(app.SyncNowMsg{}))
			wantLine(t, lh, "lock left · "+c.want)
		})
	}
}

// S7: the key opens the existing dialog: default Cancel, no letter removes.
func TestSyncKeyOpensTheLockDialog(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	lock := leftoverLock(t, vault)
	writeOutside(t, vault, "new.md", "fresh")
	h := wideWindow(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	wantLine(t, h, "(.git/index.lock) · space s")
	wantLine(t, h, "lock left · space s")
	if strings.Contains(bottomRow(h), ":sync status") {
		t.Fatalf("the bar names the key now:\n%s", bottomRow(h))
	}
	atNote(h)
	h.finish(pressSync(h))
	if !strings.Contains(h.screen(), removeButton) {
		t.Fatalf("the key must open the dialog:\n%s", h.screen())
	}
	h.keys("rcs")
	if !lockThere(t, lock) || !strings.Contains(h.screen(), removeButton) {
		t.Fatal("no letter may choose")
	}
	if cmd := h.update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil { // Cancel, the default
		runLong(t, cmd)
	}
	wantNoDialog(t, h)
	if !lockThere(t, lock) || onRemote(t, remote, "new.md") {
		t.Fatal("Cancel must touch nothing")
	}
	h.finish(pressSync(h))
	h.finish(tea.Cmd(func() tea.Msg { return runLong(t, chooseRemove(h)) }))
	if lockThere(t, lock) || !onRemote(t, remote, "new.md") {
		t.Fatalf("confirming removes the lock and syncs:\n%s", h.screen())
	}
}

// S5, S7: keys typed while the status is read are the note's: no dialog of any
// kind opens under them, the keys keep going to the note, and the bar says how
// to open it.
func TestSyncKeyOpensNoDialogWhileTyping(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	lock := leftoverLock(t, vault)
	h := wideWindow(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	atNote(h)
	pending := pressSync(h)
	h.keys("A typed")
	h.finish(pending)
	if v := h.screen(); strings.Contains(v, "╭") {
		t.Fatalf("no dialog may open under typed keys:\n%s", v)
	}
	wantLine(t, h, "space s opens it")
	h.keys(" more")
	if got := h.editor(); !strings.Contains(got, "base typed more") {
		t.Fatalf("the keys must reach the note, got %q:\n%s", got, h.screen())
	}
	if !lockThere(t, lock) {
		t.Fatal("typed keys must not remove the lock")
	}
	h.key(tea.KeyEsc)
	h.key(tea.KeyEsc)
	h.finish(pressSync(h))
	if !strings.Contains(h.screen(), removeButton) {
		t.Fatalf("the key opens it once nothing was typed:\n%s", h.screen())
	}
}

// With no key pressed meanwhile the read opens the dialog, as `:sync status`
// does; typing there still gets its status with no offer.
func TestSyncStatusTypedBeforeTheReadStillShows(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	leftoverLock(t, vault)
	h := wideWindow(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	atNote(h)
	h.keys(":sync status")
	h.update(tea.KeyMsg{Type: tea.KeyEnter})
	pending := h.update(core.SyncStatusMsg{})
	h.keys("i- item")
	h.finish(pending)
	if v := noSpace(dialogText(h.screen())); !strings.Contains(v, ":syncstatusagain") {
		t.Fatalf(":sync status keeps showing its detail:\n%s", h.screen())
	}
}

// stopRows is what a real bar shows with a note open, at each width.
func wantRows(t *testing.T, h *harness, rows map[int]string) {
	t.Helper()
	for w, want := range rows {
		if got := barAt(h, w); !strings.Contains(got, want) {
			t.Errorf("at %d columns the bar must show %q:\n%s", w, want, got)
		}
	}
}

// The bar gives a stop's message the room the arrival notice gets: the parts go
// first, and the compact form keeps the key.
func TestStopMessagesShowAtNormalWidths(t *testing.T) {
	t.Parallel()
	const compact = "Sync paused · space s"
	long := func(bar string) string { return "Sync paused: " + bar + " · space s" }
	for _, c := range []struct {
		name string
		make func(t *testing.T, remote, vault string)
		rows map[int]string
	}{
		{"offline", func(t *testing.T, remote, _ string) {
			if err := os.Rename(remote, remote+".gone"); err != nil {
				t.Fatal(err)
			}
		}, map[int]string{
			80:  "Sync offline · space s retries  Esc",
			100: "Sync: offline, 0 pending · space s retries",
			120: "Sync: offline, 0 pending · space s retries",
			140: "Sync: offline, 0 pending · space s retries",
		}},
		{"lock", func(t *testing.T, _, vault string) { leftoverLock(t, vault) }, map[int]string{
			80:  compact,
			100: compact,
			120: "Sync paused: " + "a git lock file is in the way (.git/index.lock) · space s",
			140: "Sync paused: " + "a git lock file is in the way (.git/index.lock) · space s",
		}},
		{historyStops[0].name, historyStops[0].make, map[int]string{80: compact, 100: compact, 120: long(historyStops[0].bar), 140: long(historyStops[0].bar)}},
		{historyStops[1].name, historyStops[1].make, map[int]string{80: compact, 100: compact, 120: long(historyStops[1].bar), 140: long(historyStops[1].bar)}},
		{historyStops[2].name, historyStops[2].make, map[int]string{80: compact, 100: compact, 120: long(historyStops[2].bar), 140: long(historyStops[2].bar)}},
		{historyStops[3].name, historyStops[3].make, map[int]string{80: compact, 100: long(historyStops[3].bar), 120: long(historyStops[3].bar), 140: long(historyStops[3].bar)}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			remote, vault := syncedVault(t)
			c.make(t, remote, vault)
			h := wideWindow(t, vault, syncConfig(t, vault))
			atNote(h)
			h.finish(h.update(app.SyncNowMsg{}))
			wantRows(t, h, c.rows)
			// The parts come back once the message is dismissed.
			h.key(tea.KeyEsc)
			if got := barAt(h, 140); !strings.Contains(got, "Ln 1, Col 1") || !strings.Contains(got, "Markdown") {
				t.Errorf("the parts must come back:\n%s", got)
			}
		})
	}
}

// A stop with notes that arrived keeps both in the compact form too.
func TestCompactStopKeepsTheArrivals(t *testing.T) {
	t.Parallel()
	remote, vault, h := arrivalWindow(t)
	pushNotes(t, remote, "nota.md")
	gitIn(t, vault, "remote", "set-url", "--push", "origin", filepath.Join(t.TempDir(), "gone.git"))
	writeOutside(t, vault, "mine.md", "mine")
	h.finish(h.update(app.SyncNowMsg{}))
	wantRows(t, h, map[int]string{
		60:  "Sync offline · space s retries",
		80:  "Sync offline · space s retries · 1 new",
		100: "Sync offline · space s retries · 1 new note: nota.md",
		140: "Sync: offline, 2 pending · space s retries · 1 new note from another computer: nota.md",
	})
}

// A stop's line that also says something else draws as any other message.
func TestStopSharingTheLineKeepsTheParts(t *testing.T) {
	t.Parallel()
	remote, vault, h := arrivalWindow(t)
	pushNotes(t, remote, "doc.md")
	gitIn(t, vault, "remote", "set-url", "--push", "origin", filepath.Join(t.TempDir(), "gone.git"))
	writeOutside(t, vault, "mine.md", "mine")
	h.finish(h.update(app.SyncNowMsg{}))
	got := barAt(h, 120)
	for _, want := range []string{"Markdown", "Ln 1, Col 1", "Sync: offline"} {
		if !strings.Contains(got, want) {
			t.Errorf("the bar draws as for any other message (%q):\n%s", want, got)
		}
	}
}

// Any other stop draws as before: the parts stay.
func TestOtherStopTakesNoRoom(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	gitIn(t, vault, "remote", "remove", "origin")
	h := wideWindow(t, vault, syncConfig(t, vault))
	atNote(h)
	h.finish(h.update(app.SyncNowMsg{}))
	if got := barAt(h, 120); !strings.Contains(got, "Markdown") || !strings.Contains(got, "Ln 1, Col 1") {
		t.Errorf("this message takes no room from the parts:\n%s", got)
	}
}

// A lock that went away since the sync stopped needs no dialog: the key syncs.
func TestSyncKeySyncsWhenTheLockIsGone(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	lock := leftoverLock(t, vault)
	writeOutside(t, vault, "new.md", "fresh")
	h := wideWindow(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	atNote(h)
	h.finish(pressSync(h))
	wantNoDialog(t, h)
	if !onRemote(t, remote, "new.md") {
		t.Fatal("with the lock gone the key must sync")
	}
}

// A lock that changed is not removed; the key then looks again, instead of
// offering forever a file that is no longer there as shown.
func TestSyncKeyLooksAgainAfterTheLockChanged(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	lock := leftoverLock(t, vault)
	writeOutside(t, vault, "new.md", "fresh")
	h := wideWindow(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	atNote(h)
	h.finish(pressSync(h))
	if err := os.WriteFile(lock, []byte("a git started meanwhile"), 0o644); err != nil {
		t.Fatal(err)
	}
	if next := h.update(runLong(t, chooseRemove(h))); next != nil {
		t.Fatal("no sync may start past a lock that was not removed")
	}
	wantLine(t, h, "Lock not removed")
	if !lockThere(t, lock) {
		t.Fatal("a changed lock must stay")
	}
	h.key(tea.KeyEsc)
	h.finish(pressSync(h)) // a sync, which stops on the lock as it is now
	wantNoDialog(t, h)
	h.finish(pressSync(h))
	h.finish(tea.Cmd(func() tea.Msg { return runLong(t, chooseRemove(h)) }))
	if lockThere(t, lock) || !onRemote(t, remote, "new.md") {
		t.Fatalf("the fresh offer must remove it and sync:\n%s", h.screen())
	}
}

// Any other stop keeps today's text, and the key still just syncs.
func TestOtherStopsKeepTheirTextAndSync(t *testing.T) {
	for _, c := range []struct {
		name string
		make func(t *testing.T) string // returns the vault
		want string
	}{
		{"no remote", func(t *testing.T) string {
			_, vault := syncedVault(t)
			gitIn(t, vault, "remote", "remove", "origin")
			return vault
		}, "Sync paused: the repository has no remote named origin; add one with `git remote add origin <url>`"},
		{"not a repository", func(t *testing.T) string {
			return t.TempDir()
		}, "Sync paused: this vault is not a git repository"},
		{"not the repository's top folder", func(t *testing.T) string {
			_, vault := syncedVault(t)
			sub := filepath.Join(vault, "sub")
			writeOutside(t, sub, "a.md", "a")
			return sub
		}, "Sync paused: this vault is inside the git repository at"},
		{"no identity", func(t *testing.T) string {
			_, vault := syncedVault(t)
			empty := filepath.Join(t.TempDir(), "gitconfig")
			writeOutside(t, filepath.Dir(empty), "gitconfig", "")
			t.Setenv("GIT_CONFIG_GLOBAL", empty)
			return vault
		}, "Sync paused: git does not know who you are"},
	} {
		t.Run(c.name, func(t *testing.T) {
			vault := c.make(t)
			h := wideWindow(t, vault, syncConfig(t, vault))
			h.finish(h.update(app.SyncNowMsg{}))
			wantLine(t, h, c.want)
			if strings.Contains(bottomRow(h), "space s") {
				t.Fatalf("this stop's text does not change:\n%s", bottomRow(h))
			}
			h.finish(h.update(app.SyncNowMsg{}))
			wantNoDialog(t, h)
			wantLine(t, h, c.want)
		})
	}
}

// T1's notice stays: notes that arrived with a stop keep both.
func TestArrivalsAndAStopKeepBoth(t *testing.T) {
	t.Parallel()
	remote, vault, h := arrivalWindow(t)
	pushNotes(t, remote, "nota.md")
	gitIn(t, vault, "remote", "set-url", "--push", "origin", filepath.Join(t.TempDir(), "gone.git"))
	writeOutside(t, vault, "mine.md", "mine")
	h.send(tea.WindowSizeMsg{Width: 1000, Height: termH})
	h.finish(h.update(app.SyncNowMsg{}))
	wantLine(t, h, "pending · space s retries · 1 new note from another computer: nota.md")
}

// A local commit says nothing about origin: the stop it did not hit stays, and
// the key still explains it.
func TestLocalCommitKeepsTheStop(t *testing.T) {
	t.Parallel()
	c := historyStops[0]
	remote, vault := syncedVault(t)
	c.make(t, remote, vault)
	clock := newTestClock()
	h := wideWindow(t, vault, intervalConfig(t, vault, "0"), app.WithClock(clock.Now))
	h.finish(h.update(app.SyncNowMsg{}))
	openAndSave(h, "more")
	clock.add(10 * time.Second)
	if n := poll(h); n != 1 {
		t.Fatalf("the commit after the save must run (%d commands ran)", n)
	}
	wantLine(t, h, "sync paused · space s")
	if cmd := pressSync(h); cmd != nil || !strings.Contains(h.screen(), "Check again") {
		t.Fatalf("the key must still explain:\n%s", h.screen())
	}
}

// Another open dialog (it may be asking about unsaved notes) is left alone: the
// explanation goes to the bar.
func TestStopDialogLeavesAnotherDialogAlone(t *testing.T) {
	t.Parallel()
	_, _, h := stoppedWindow(t, historyStops[3])
	askSyncStatus(h)
	if cmd := h.update(app.SyncNowMsg{}); cmd != nil {
		t.Fatal("nothing may run under another dialog")
	}
	v := h.screen()
	if !strings.Contains(v, "Sync status") || strings.Contains(v, "Check again") {
		t.Fatalf("the open dialog must stay:\n%s", v)
	}
}

// While a sync runs the key says so: what stopped the last one may be over.
func TestSyncKeyWhileSyncingOpensNothing(t *testing.T) {
	t.Parallel()
	_, _, h := stoppedWindow(t, historyStops[3])
	atNote(h)
	pressSync(h)
	h.key(tea.KeyLeft)
	running := h.update(tea.KeyMsg{Type: tea.KeyEnter}) // Check again
	if cmd := pressSync(h); cmd != nil {
		t.Fatal("one sync at a time")
	}
	wantNoDialog(t, h)
	wantLine(t, h, "Already syncing")
	h.finish(running)
}

// longNamedNote moves the vault's only note to a name longer than the bar shows,
// in a folder, so the left side of the bar is as wide as it gets.
func longNamedNote(t *testing.T, vault string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(vault, "a-folder-with-a-rather-long-name"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, vault, "mv", "doc.md", "a-folder-with-a-rather-long-name/a-note-with-a-name-longer-than-the-bar-shows.md")
	gitIn(t, vault, "commit", "-q", "-m", "long name")
	gitIn(t, vault, "push", "-q", "origin", "main")
}

// narrowStops is every stop made by the state of the vault alone, with the
// words its message keeps down to the shortest form.
var narrowStops = []struct {
	name    string
	make    func(t *testing.T, remote, vault string)
	compact string // %s is the key
	tight   string
}{
	{"offline", func(t *testing.T, remote, _ string) {
		if err := os.Rename(remote, remote+".gone"); err != nil {
			t.Fatal(err)
		}
	}, "Sync offline · %s retries", "Sync offline · %s"},
	{"lock", func(t *testing.T, _, vault string) { leftoverLock(t, vault) }, "Sync paused · %s", "Sync paused · %s"},
	{historyStops[0].name, historyStops[0].make, "Sync paused · %s", "Sync paused · %s"},
	{historyStops[1].name, historyStops[1].make, "Sync paused · %s", "Sync paused · %s"},
	{historyStops[2].name, historyStops[2].make, "Sync paused · %s", "Sync paused · %s"},
	{historyStops[3].name, historyStops[3].make, "Sync paused · %s", "Sync paused · %s"},
}

// From the app's minimum width up, a stop's message keeps the key: the parts go
// first, then the folder, then the words, and the note's name last.
func TestStopKeepsTheKeyFromTheMinimumWidth(t *testing.T) {
	t.Parallel()
	const custom = "Ctrl+Alt+Shift+Pgdown"
	customKey := app.WithKeys(map[string][]string{"sync_now": {"ctrl+alt+shift+pgdown"}})
	for _, stop := range narrowStops {
		for _, c := range []struct {
			name   string
			long   bool // the note has a long name, in a folder
			dirty  bool
			home   bool
			custom bool
			at60   string // %s is the key
		}{
			{name: "short name", at60: stop.compact},
			{name: "long name", long: true, at60: stop.tight},
			{name: "long name with unsaved edits", long: true, dirty: true, at60: stop.tight},
			{name: "long name and a long custom key", long: true, custom: true, at60: stop.tight},
			{name: "start screen", home: true, at60: "· %s"},
		} {
			t.Run(stop.name+"/"+c.name, func(t *testing.T) {
				t.Parallel()
				remote, vault := syncedVault(t)
				if c.long {
					longNamedNote(t, vault)
				}
				stop.make(t, remote, vault)
				key, opts := "space s", []app.Option{}
				if c.custom {
					key, opts = custom, []app.Option{customKey}
				}
				h := wideWindow(t, vault, syncConfig(t, vault), opts...)
				if !c.home {
					atNote(h)
				}
				if c.dirty {
					h.keys("Aedit")
					h.key(tea.KeyEsc)
				}
				h.finish(h.update(app.SyncNowMsg{}))
				if got, want := barAt(h, 60), strings.ReplaceAll(c.at60, "%s", key); !strings.Contains(got, want) {
					t.Errorf("at 60 columns the bar must show %q:\n%s", want, got)
				}
				for w := 60; w <= 140; w++ {
					got := barAt(h, w)
					if !strings.Contains(got, "· "+key) {
						t.Errorf("at %d columns the bar must keep the key %q:\n%s", w, key, got)
					}
				}
			})
		}
	}
}

// What gives way is only what the message needs: with room for the message the
// position stays, and at the minimum width the note's name is cut, not dropped.
func TestStopGivesUpOnlyWhatItNeeds(t *testing.T) {
	t.Parallel()
	offline := narrowStops[0]
	remote, vault := syncedVault(t)
	offline.make(t, remote, vault)
	h := wideWindow(t, vault, syncConfig(t, vault))
	atNote(h)
	h.finish(h.update(app.SyncNowMsg{}))
	if got := barAt(h, 80); !strings.Contains(got, "Sync offline · space s retries") || !strings.Contains(got, "Ln 1, Col 1") {
		t.Errorf("at 80 columns the message and the position both fit:\n%s", got)
	}
	got := barAt(h, 60)
	if !strings.Contains(got, "Sync offline · space s retries") || !strings.Contains(got, "doc.md") {
		t.Errorf("at 60 columns the message and the note's name both fit:\n%s", got)
	}

	remote, vault = syncedVault(t)
	longNamedNote(t, vault)
	offline.make(t, remote, vault)
	h = wideWindow(t, vault, syncConfig(t, vault))
	atNote(h)
	h.finish(h.update(app.SyncNowMsg{}))
	got = barAt(h, 60)
	if !strings.Contains(got, "a-note-with") || strings.Contains(got, "a-folder") {
		t.Errorf("at 60 columns the folder goes first and the name is only cut:\n%s", got)
	}
}

// On the start screen `:` opens no command line: with no key to name, the bar
// says what happened and suggests nothing.
func TestStartScreenWithNoSyncKeySuggestsNoCommand(t *testing.T) {
	t.Parallel()
	unbound := app.WithKeys(map[string][]string{"sync_now": {}})
	for _, c := range []struct {
		name string
		make func(t *testing.T, remote, vault string)
		want string
	}{
		{"offline", narrowStops[0].make, "Sync: offline, 0 pending"},
		{"lock", narrowStops[1].make, "Sync paused: a git lock file is in the way (.git/index.lock)"},
		{"detached", historyStops[3].make, "Sync paused: " + historyStops[3].bar},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			remote, vault := syncedVault(t)
			c.make(t, remote, vault)
			h := wideWindow(t, vault, syncConfig(t, vault), unbound)
			h.finish(h.update(app.SyncNowMsg{}))
			wantLine(t, h, c.want)
			if got := bottomRow(h); strings.Contains(got, ":sync") {
				t.Errorf("no command on the start screen:\n%s", got)
			}
			if got := barAt(h, 80); strings.Contains(got, ":sync") {
				t.Errorf("no command on the start screen, in the compact form either:\n%s", got)
			}
			h.key(tea.KeyEsc)
			if got := barAt(h, 200); strings.Contains(got, ":sync") {
				t.Errorf("no command in the indicator either:\n%s", got)
			}
		})
	}
}

// stateRow is the State row of the `:sync status` dialog, up to the box's border.
var stateRow = regexp.MustCompile(`State:[^│]*`)

// The State row says the state; the key belongs to the bar's indicator.
func TestSyncStatusStateRowNamesNoKey(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		make  func(t *testing.T, remote, vault string)
		state string
	}{
		{"offline", narrowStops[0].make, "State: offline · 0 pending"},
		{"lock", narrowStops[1].make, "State: sync paused, lock left"},
		{"detached", historyStops[3].make, "State: sync paused"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			remote, vault := syncedVault(t)
			c.make(t, remote, vault)
			h := wideWindow(t, vault, syncConfig(t, vault))
			h.finish(h.update(app.SyncNowMsg{}))
			if got := bottomRow(h); !strings.Contains(got, "· space s") {
				t.Fatalf("setup: the bar's indicator names the key:\n%s", got)
			}
			h.key(tea.KeyEsc)
			askSyncStatus(h)
			row := strings.TrimSpace(stateRow.FindString(h.screen()))
			if row != c.state {
				t.Errorf("the State row:\n got %q\nwant %q", row, c.state)
			}
		})
	}
}

// A successful local commit shows a stop in this vault ended: the indicator
// stops naming it, the key syncs instead of explaining, and the note goes out.
func TestLocalCommitClearsAStopItDisproves(t *testing.T) {
	t.Parallel()
	for _, c := range []historyStop{historyStops[2], historyStops[3]} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			remote, vault := syncedVault(t)
			c.make(t, remote, vault)
			clock := newTestClock()
			h := wideWindow(t, vault, intervalConfig(t, vault, "0"), app.WithClock(clock.Now))
			h.finish(h.update(app.SyncNowMsg{}))
			wantLine(t, h, "sync paused · space s")
			c.fix(t, remote, vault)
			openAndSave(h, "more")
			clock.add(10 * time.Second)
			if n := poll(h); n != 1 {
				t.Fatalf("the commit after the save must run (%d commands ran)", n)
			}
			if got := bottomRow(h); strings.Contains(got, "sync paused") || strings.Contains(got, "· space s") {
				t.Errorf("the stop ended, so the bar must stop naming it:\n%s", got)
			}
			h.finish(pressSync(h))
			wantNoDialog(t, h)
			wantLine(t, h, "Synced")
			if pushed, head := gitIn(t, remote, "rev-parse", "main"), gitIn(t, vault, "rev-parse", "HEAD"); pushed != head {
				t.Errorf("the key must sync: origin at %s, vault at %s", pushed, head)
			}
		})
	}
}

// A local run that did not succeed disproves nothing: the key still explains.
func TestLocalRunThatFailsKeepsTheStop(t *testing.T) {
	t.Parallel()
	c := historyStops[3]
	remote, vault := syncedVault(t)
	c.make(t, remote, vault)
	clock := newTestClock()
	h := wideWindow(t, vault, intervalConfig(t, vault, "0"), app.WithClock(clock.Now))
	h.finish(h.update(app.SyncNowMsg{}))
	c.fix(t, remote, vault)
	gitIn(t, vault, "remote", "remove", "origin") // now it pauses for another reason
	openAndSave(h, "more")
	clock.add(10 * time.Second)
	if n := poll(h); n != 1 {
		t.Fatalf("the commit after the save must run (%d commands ran)", n)
	}
	if cmd := pressSync(h); cmd != nil || !strings.Contains(h.screen(), "Check again") {
		t.Fatalf("the key must still explain:\n%s", h.screen())
	}
}
