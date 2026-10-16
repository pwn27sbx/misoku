package app_test

// Regression tests from the independent check of automatic sync (F2.12):
// timed sync and the commit after a save, driven through the public Update
// loop with an injected clock.

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// saveMore appends text to the open note (doc.md) and saves it.
func saveMore(h *harness, text string) {
	h.t.Helper()
	h.keys("A" + text)
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
}

// quitPush runs what Misoku runs once its screen closes and returns what
// it printed.
func quitPush(h *harness) string {
	h.t.Helper()
	var out bytes.Buffer
	h.m.(app.Model).SyncOnQuit(context.Background(), &out, app.QuitSyncLimit)
	return out.String()
}

// Typing and saving all the time while timed syncs and save commits fire,
// some of them finishing after more saves, and another device pushing
// other notes: every save reaches origin, every note from the other device
// arrives, and no git runs beside another one in this window.
func TestContinuousTypingDuringAutomaticSyncNeverLosesASave(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	clock := newTestClock()
	h := window(t, vault, intervalConfig(t, vault, "1m"), app.WithClock(clock.Now))
	h.finish(h.update(app.SyncNowMsg{}))
	h.keys("1")
	h.key(tea.KeyEsc)
	rng := rand.New(rand.NewSource(212))
	want := "base"
	var desk []string
	for i := 0; i < 40; i++ {
		word := fmt.Sprintf(" w%d", i)
		saveMore(h, word)
		want += word
		if i%7 == 3 {
			name := fmt.Sprintf("desk%d.md", i)
			pushFromDesk(t, remote, name, "from desk")
			desk = append(desk, name)
		}
		clock.add(time.Duration(rng.Intn(5000)) * time.Millisecond)
		if i%9 == 8 {
			clock.add(time.Minute)
		}
		// Half of the runs end only after one more save: the save lands
		// while the sync or the commit is still running.
		running := runAll(t, h.update(app.CheckWorkspaceMsg{}))
		if len(running) > 1 {
			t.Fatalf("round %d: two git runs started at once", i)
		}
		if len(running) == 1 && i%2 == 0 {
			word := fmt.Sprintf(" r%d", i)
			saveMore(h, word)
			want += word
			if cmd := h.update(app.CheckWorkspaceMsg{}); cmd != nil {
				for _, msg := range runAll(t, cmd) {
					t.Fatalf("round %d: a tick started git while a run was going: %T", i, msg)
				}
			}
		}
		for _, msg := range running {
			h.send(msg)
		}
	}
	// Nothing saved may wait only on disk: the last commit holds it all.
	if got := gitIn(t, vault, "show", "HEAD:doc.md"); got != want {
		t.Fatalf("saves not committed:\n got %q\nwant %q", got, want)
	}
	// The other device may have pushed after the last sync: one more sync
	// joins it, then quitting has nothing left.
	h.finish(h.update(app.SyncNowMsg{}))
	if out := quitPush(h); !strings.Contains(out, "Nothing new to push.") {
		t.Fatalf("quit push: %s", out)
	}
	if got := gitIn(t, remote, "show", "main:doc.md"); got != want {
		t.Fatalf("origin lost saves:\n got %q\nwant %q", got, want)
	}
	for _, name := range desk {
		if !onRemote(t, remote, name) {
			t.Fatalf("%s from the other device is gone from origin", name)
		}
	}
	if st := gitIn(t, vault, "status", "--porcelain"); st != "" {
		t.Fatalf("the vault is not clean after quitting:\n%s", st)
	}
}

// A save made while the commit after a save runs is committed next.
func TestSaveDuringALocalCommitIsCommittedNext(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	clock := newTestClock()
	h := window(t, vault, syncConfig(t, vault), app.WithClock(clock.Now))
	openAndSave(h, " a")
	clock.add(app.SaveCommitDelay)
	running := runAll(t, h.update(app.CheckWorkspaceMsg{}))
	if len(running) != 1 {
		t.Fatalf("want the local commit to run, got %d", len(running))
	}
	saveMore(h, " b")
	for _, msg := range running {
		h.send(msg)
	}
	if got := gitIn(t, vault, "show", "HEAD:doc.md"); got != "base a" {
		t.Fatalf("the first commit holds the first save, got %q", got)
	}
	clock.add(app.SaveCommitDelay)
	if n := poll(h); n != 1 {
		t.Fatalf("the save made during the commit is committed next (%d runs)", n)
	}
	if got := gitIn(t, vault, "show", "HEAD:doc.md"); got != "base a b" {
		t.Fatalf("got %q", got)
	}
}

// Quitting right after a save (before its commit is due) pushes it, and
// quitting while the commit after a save runs waits for it and pushes the
// saves made meanwhile too.
func TestQuitRightAfterASavePushesIt(t *testing.T) {
	t.Parallel()
	t.Run("before the commit", func(t *testing.T) {
		t.Parallel()
		remote, vault := syncedVault(t)
		clock := newTestClock()
		h := window(t, vault, syncConfig(t, vault), app.WithClock(clock.Now))
		h.finish(h.update(app.SyncNowMsg{})) // the sync on opening takes the lock
		openAndSave(h, " last")
		if out := quitPush(h); !strings.Contains(out, "Changes pushed.") {
			t.Fatalf("quit: %s", out)
		}
		if got := gitIn(t, remote, "show", "main:doc.md"); got != "base last" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("during the commit", func(t *testing.T) {
		t.Parallel()
		remote, vault := syncedVault(t)
		clock := newTestClock()
		h := window(t, vault, syncConfig(t, vault), app.WithClock(clock.Now))
		openAndSave(h, " a")
		clock.add(app.SaveCommitDelay)
		cmd := h.update(app.CheckWorkspaceMsg{})
		var local tea.Cmd
		for _, c := range []tea.Cmd{cmd} {
			msg := c()
			if b, ok := msg.(tea.BatchMsg); ok {
				for _, c := range b {
					if c != nil && !isTimer(c) {
						local = c
					}
				}
			}
		}
		if local == nil {
			t.Fatal("want the local commit command")
		}
		saveMore(h, " b") // the run has not started yet
		go local()
		if out := quitPush(h); !strings.Contains(out, "Changes pushed.") {
			t.Fatalf("quit: %s", out)
		}
		if got := gitIn(t, remote, "show", "main:doc.md"); got != "base a b" {
			t.Fatalf("got %q", got)
		}
	})
}

// Several syncs asked for during a save commit make one sync after it.
func TestSyncsAskedDuringASaveCommitRunOnce(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	clock := newTestClock()
	h := window(t, vault, syncConfig(t, vault), app.WithClock(clock.Now))
	openAndSave(h, " mine")
	clock.add(app.SaveCommitDelay)
	committed := runAll(t, h.update(app.CheckWorkspaceMsg{}))
	for i := 0; i < 3; i++ {
		if cmd := h.update(app.SyncNowMsg{}); cmd != nil {
			t.Fatal("no git may start while the commit runs")
		}
		h.send(core.SyncMsg{})
	}
	queued := h.update(committed[0])
	if queued == nil {
		t.Fatal("the queued sync must start")
	}
	msgs := runAll(t, queued)
	if len(msgs) != 1 {
		t.Fatalf("one queued sync, got %d", len(msgs))
	}
	if next := h.update(msgs[0]); next != nil {
		t.Fatal("the queued sync runs only once")
	}
	if got := gitIn(t, remote, "show", "main:doc.md"); got != "base mine" {
		t.Fatalf("got %q", got)
	}
	if n := poll(h); n != 0 {
		t.Fatalf("nothing more runs (%d)", n)
	}
}

// "off" stops timed sync only: saves are still committed, never pushed.
func TestIntervalOffStillCommitsSaves(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	pushed := remoteHead(t, remote)
	clock := newTestClock()
	h := window(t, vault, intervalConfig(t, vault, "OFF"), app.WithClock(clock.Now))
	openAndSave(h, " mine")
	clock.add(app.SaveCommitDelay)
	if n := poll(h); n != 1 {
		t.Fatalf("the save is committed (%d)", n)
	}
	if got := gitIn(t, vault, "show", "HEAD:doc.md"); got != "base mine" {
		t.Fatalf("got %q", got)
	}
	clock.add(48 * time.Hour)
	if n := poll(h); n != 0 || remoteHead(t, remote) != pushed {
		t.Fatalf("nothing is pushed on its own (%d)", n)
	}
}

// An unusable interval is shown in the status bar when the window opens.
func TestUnusableIntervalWarnsInTheBarOnOpening(t *testing.T) {
	t.Parallel()
	for _, bad := range []string{"30s", "0", "garbage"} {
		t.Run(bad, func(t *testing.T) {
			t.Parallel()
			_, vault := syncedVault(t)
			h := window(t, vault, intervalConfig(t, vault, bad), app.WithClock(newTestClock().Now))
			// The bar may cut the warning after the vault's path, but it
			// must start telling it.
			if bar := statusBar(h); !strings.Contains(bar, "sync: vault") {
				t.Fatalf("the bar must report the interval:\n%s", bar)
			}
		})
	}
}

// S11: once the window holding the lock closes, the other one takes over
// on its next save commit and then on its timer.
func TestSecondWindowTakesOverAutomaticSyncAfterTheFirstCloses(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	cfg := syncConfig(t, vault)
	firstClock, secondClock := newTestClock(), newTestClock()
	first := window(t, vault, cfg, app.WithClock(firstClock.Now))
	first.finish(first.update(app.SyncNowMsg{}))
	second := window(t, vault, cfg, app.WithClock(secondClock.Now))
	before := gitIn(t, vault, "rev-parse", "HEAD")
	openAndSave(second, " two")
	secondClock.add(app.SaveCommitDelay)
	poll(second)
	if gitIn(t, vault, "rev-parse", "HEAD") != before {
		t.Fatal("the second window must not commit while the first holds the lock")
	}
	first.m.(app.Model).ReleaseSyncLock()
	saveMore(second, " three")
	secondClock.add(app.SaveCommitDelay)
	if n := poll(second); n != 1 {
		t.Fatalf("want the save commit (%d)", n)
	}
	if got := gitIn(t, vault, "show", "HEAD:doc.md"); got != "base two three" {
		t.Fatalf("the second window takes over its save commits, got %q", got)
	}
	secondClock.add(5 * time.Minute)
	if n := poll(second); n != 1 {
		t.Fatalf("want the timed sync (%d)", n)
	}
	if got := gitIn(t, remote, "show", "main:doc.md"); got != "base two three" {
		t.Fatalf("the second window pushes once it holds the lock, got %q", got)
	}
}

// A window that takes the lock over through the commit after a save no
// longer says another window syncs: it holds the lock now.
func TestTakeoverBySaveCommitUpdatesTheIndicator(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	cfg := intervalConfig(t, vault, "off")
	first := window(t, vault, cfg, app.WithClock(newTestClock().Now))
	first.finish(first.update(app.SyncNowMsg{}))
	clock := newTestClock()
	second := window(t, vault, cfg, app.WithClock(clock.Now))
	second.finish(second.update(app.SyncNowMsg{}))
	if bar := statusBar(second); !strings.Contains(bar, "other window") {
		t.Fatalf("precondition: the second window says another one syncs:\n%s", bar)
	}
	first.m.(app.Model).ReleaseSyncLock()
	openAndSave(second, " mine")
	clock.add(app.SaveCommitDelay)
	if n := poll(second); n != 1 {
		t.Fatalf("want the save commit (%d)", n)
	}
	if bar := statusBar(second); strings.Contains(bar, "other window") {
		t.Fatalf("the window holds the lock now but its bar says:\n%s", bar)
	}
}

// A timed sync brings another device's edit of the note being edited (not
// saved yet). Saving keeps both versions without asking: the user's in the
// note, the other device's in a conflict copy next to it, and the bar says
// where. A key typed right after saving can choose nothing.
func TestSavingOverAnEditFromAnotherDeviceKeepsBothVersions(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	clock := newTestClock()
	h := window(t, vault, syncConfig(t, vault), app.WithClock(clock.Now))
	h.finish(h.update(app.SyncNowMsg{}))
	h.keys("1")
	h.keys("A mine")
	h.key(tea.KeyEsc)
	pushFromDesk(t, remote, "doc.md", "base desk")
	clock.add(5 * time.Minute)
	if n := poll(h); n != 1 {
		t.Fatalf("want the timed sync (%d)", n)
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	copyName := "doc (conflict " + clock.Now().Format("2006-01-02 1504") + ").md"
	if v := strings.Join(strings.Fields(h.view()), ""); !strings.Contains(v, strings.Join(strings.Fields(copyName), "")) {
		t.Fatalf("the bar must name the copy:\n%s", h.view())
	}
	h.keys(" ") // the user keeps typing
	if got := readNote(t, vault, "doc.md"); got != "base mine" {
		t.Fatalf("the note must hold the user's version, got %q", got)
	}
	if got := readNote(t, vault, copyName); got != "base desk" {
		t.Fatalf("the other device's version must be kept in %q, got %q", copyName, got)
	}
}

// An unusable interval is still reported on the status line once the sync
// on opening, which Init starts before the first frame, has finished.
func TestUnusableIntervalStaysInTheBarAfterTheSyncOnOpening(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	h := window(t, vault, intervalConfig(t, vault, "30s"), app.WithClock(newTestClock().Now))
	h.finish(h.update(app.SyncNowMsg{})) // what Init starts in the real app
	if bar := statusBar(h); !strings.Contains(bar, "sync: vault") && !strings.Contains(bar, "interval") {
		t.Fatalf("the unusable interval is no longer reported once the window has opened:\n%s", bar)
	}
}

// A timed sync that starts between confirming "Remove the lock and sync"
// and the removal's answer does not swallow the sync the user asked for:
// it runs once the timed one ends.
func TestLockRemovalSyncWaitsForATimedSyncInsteadOfBeingDropped(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	lock := leftoverLock(t, vault)
	clock := newTestClock()
	h := window(t, vault, syncConfig(t, vault), app.WithClock(clock.Now))
	h.finish(h.update(app.SyncNowMsg{}))
	openSyncStatus(h)
	removed := runLong(t, chooseRemove(h))
	if lockThere(t, lock) {
		t.Fatal("confirming must remove the lock")
	}
	clock.add(5 * time.Minute)
	timed := runAll(t, h.update(app.CheckWorkspaceMsg{}))
	if len(timed) != 1 {
		t.Fatalf("want the timed sync to be running, got %d", len(timed))
	}
	if cmd := h.update(removed); cmd != nil {
		t.Fatal("no second git may start while the timed sync runs")
	}
	if bar := statusBar(h); strings.Contains(bar, "Already syncing") {
		t.Fatalf("the sync asked for must wait, not be dropped:\n%s", bar)
	}
	queued := h.update(timed[0])
	if queued == nil {
		t.Fatal("the sync asked for must start once the timed sync ends")
	}
	h.finish(queued)
	wantBar(t, h, "synced")
}
