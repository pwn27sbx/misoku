package app_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"

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
			h.finish(h.update(app.SyncNowMsg{}))
			wantLine(t, h, "pending · "+c.want+" retries")
			if strings.Contains(bottomRow(h), "space s") {
				t.Fatalf("the default key is not this user's:\n%s", bottomRow(h))
			}

			detached := historyStops[3]
			_, _, stopped := stoppedWindow(t, detached, opt)
			wantLine(t, stopped, detached.bar+" · "+c.want)

			_, locked := syncedVault(t)
			leftoverLock(t, locked)
			lh := wideWindow(t, locked, syncConfig(t, locked), opt)
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

// S7: the same guard as `:sync status`: keys typed meanwhile keep the offer away.
func TestSyncKeyOffersNoRemovalWhileTyping(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	lock := leftoverLock(t, vault)
	h := wideWindow(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	atNote(h)
	pending := pressSync(h)
	h.keys("i- item")
	h.finish(pending)
	if strings.Contains(h.screen(), removeButton) {
		t.Fatalf("the offer must not open under typed keys:\n%s", h.screen())
	}
	if !lockThere(t, lock) {
		t.Fatal("typed keys must not remove the lock")
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
