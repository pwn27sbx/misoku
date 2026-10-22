package app_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// testClock is a clock a test moves by hand: timed sync and the commit
// after a save are decided by the app's clock, never by real waiting.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func newTestClock() *testClock {
	return &testClock{now: time.Date(2026, 10, 16, 9, 0, 0, 0, time.Local)}
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}

func (c *testClock) add(d time.Duration) { c.set(c.Now().Add(d)) }

// intervalConfig lists vault in sync.vaults with the given interval.
func intervalConfig(t *testing.T, vault, interval string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := config.Config{Sync: config.Sync{Vaults: map[string]config.SyncVault{vault: {Device: "laptop", Interval: interval}}}}
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	return path
}

// runAll runs cmd and every command it batches, except timers, to the end
// (git may be slow on a loaded runner), and returns their messages.
func runAll(t testing.TB, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	var msgs []tea.Msg
	queue := []tea.Cmd{cmd}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == nil || isTimer(c) {
			continue
		}
		done := make(chan tea.Msg, 1)
		go func() { done <- c() }()
		var msg tea.Msg
		select {
		case msg = <-done:
		case <-time.After(60 * time.Second):
			t.Fatal("a command did not finish")
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			queue = append(queue, batch...)
			continue
		}
		if msg != nil {
			msgs = append(msgs, msg)
		}
	}
	return msgs
}

// poll delivers one tick of the workspace poll, which timed sync and the
// commit after a save ride on, runs the git work it starts as the runtime
// would in the background, and feeds the outcome back. It returns how many
// commands that work took: 0 means no git was started.
func poll(h *harness) int {
	h.t.Helper()
	msgs := runAll(h.t, h.update(app.CheckWorkspaceMsg{}))
	for _, msg := range msgs {
		h.send(msg)
	}
	return len(msgs)
}

// openAndSave opens doc.md, appends text to it and saves it.
func openAndSave(h *harness, text string) {
	h.t.Helper()
	h.keys("1")
	h.keys("A" + text)
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
}

func commitCount(t *testing.T, vault string) int {
	t.Helper()
	n, err := strconv.Atoi(gitIn(t, vault, "rev-list", "--count", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// v0.6.0: a listed vault syncs every interval, counted from the last sync.
func TestTimedSyncRunsEveryInterval(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	clock := newTestClock()
	start := clock.Now()
	h := window(t, vault, intervalConfig(t, vault, "2m"), app.WithClock(clock.Now))
	h.finish(h.update(app.SyncNowMsg{})) // the sync on opening
	writeOutside(t, vault, "new.md", "fresh")

	clock.set(start.Add(2*time.Minute - time.Second))
	if n := poll(h); n != 0 || onRemote(t, remote, "new.md") {
		t.Fatalf("no sync before the interval (%d commands ran)", n)
	}
	clock.set(start.Add(2 * time.Minute))
	if n := poll(h); n != 1 || !onRemote(t, remote, "new.md") {
		t.Fatalf("a sync every interval must push the new note (%d commands ran)", n)
	}

	writeOutside(t, vault, "later.md", "later")
	clock.set(start.Add(4*time.Minute - time.Second))
	if n := poll(h); n != 0 {
		t.Fatalf("the interval counts from the last sync (%d commands ran)", n)
	}
	clock.set(start.Add(4 * time.Minute))
	if n := poll(h); n != 1 || !onRemote(t, remote, "later.md") {
		t.Fatalf("the next interval syncs again (%d commands ran)", n)
	}
}

// With no interval set, timed sync runs every 5 minutes.
func TestTimedSyncDefaultsToFiveMinutes(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	clock := newTestClock()
	start := clock.Now()
	h := window(t, vault, syncConfig(t, vault), app.WithClock(clock.Now))
	writeOutside(t, vault, "new.md", "fresh")
	clock.set(start.Add(5*time.Minute - time.Second))
	if n := poll(h); n != 0 {
		t.Fatalf("no timed sync before 5 minutes (%d commands ran)", n)
	}
	clock.set(start.Add(5 * time.Minute))
	if n := poll(h); n != 1 || !onRemote(t, remote, "new.md") {
		t.Fatalf("the default interval is 5 minutes (%d commands ran)", n)
	}
}

// An interval Misoku cannot use is reported and the default is used.
func TestTimedSyncWithAnUnusableIntervalWarnsAndUsesTheDefault(t *testing.T) {
	t.Parallel()
	for _, bad := range []string{"10s", "soon", "-5m"} {
		t.Run(bad, func(t *testing.T) {
			t.Parallel()
			remote, vault := syncedVault(t)
			clock := newTestClock()
			start := clock.Now()
			h := window(t, vault, intervalConfig(t, vault, bad), app.WithClock(clock.Now))
			// The bar may cut the long warning; `:sync status` shows it all,
			// wrapped over dialog rows that dialogText joins again.
			h.send(core.SyncStatusMsg{})
			if v := noSpace(dialogText(h.view())); !strings.Contains(v, `interval"`+bad+`"`) {
				t.Fatalf("the unusable interval must be reported:\n%s", h.view())
			}
			h.key(tea.KeyEnter)
			writeOutside(t, vault, "new.md", "fresh")
			clock.set(start.Add(time.Minute))
			if n := poll(h); n != 0 {
				t.Fatalf("%q must not sync before the default interval (%d commands ran)", bad, n)
			}
			clock.set(start.Add(config.DefaultSyncInterval))
			if n := poll(h); n != 1 || !onRemote(t, remote, "new.md") {
				t.Fatalf("%q must sync at the default interval (%d commands ran)", bad, n)
			}
		})
	}
}

// "off" turns timed sync off: nothing runs on its own, a sync asked for
// still runs, and `:sync status` says timed sync is off.
func TestTimedSyncOffNeverSyncsOnItsOwn(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	clock := newTestClock()
	h := window(t, vault, intervalConfig(t, vault, "off"), app.WithClock(clock.Now))
	writeOutside(t, vault, "new.md", "fresh")
	clock.add(24 * time.Hour)
	if n := poll(h); n != 0 || onRemote(t, remote, "new.md") {
		t.Fatalf("timed sync is off (%d commands ran)", n)
	}
	h.finish(h.update(app.SyncNowMsg{}))
	if !onRemote(t, remote, "new.md") {
		t.Fatal("a sync asked for still runs")
	}
	h.send(core.SyncStatusMsg{})
	if v := noSpace(dialogText(h.view())); !strings.Contains(v, "Timedsync:off") {
		t.Fatalf("`:sync status` must say timed sync is off:\n%s", h.view())
	}
}

// A tick while a sync runs starts no second one; the timed sync then runs
// on the next tick after it.
func TestTimedSyncNeverStartsWhileASyncRuns(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	clock := newTestClock()
	h := window(t, vault, syncConfig(t, vault), app.WithClock(clock.Now))
	running := h.update(app.SyncNowMsg{})
	clock.add(10 * time.Minute)
	if n := poll(h); n != 0 {
		t.Fatalf("a tick while a sync runs must start nothing (%d commands ran)", n)
	}
	h.finish(running)
	writeOutside(t, vault, "new.md", "fresh")
	if n := poll(h); n != 0 {
		t.Fatalf("the interval counts from the sync that just ran (%d commands ran)", n)
	}
	clock.add(5 * time.Minute)
	if n := poll(h); n != 1 || !onRemote(t, remote, "new.md") {
		t.Fatalf("the next timed sync runs (%d commands ran)", n)
	}
}

// S11: a window that does not hold the vault's lock never commits or
// pushes on a timer or after a save; the window that holds it does.
func TestOnlyTheWindowHoldingTheLockSyncsOnItsOwn(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	cfg := syncConfig(t, vault)
	firstClock, secondClock := newTestClock(), newTestClock()
	first := window(t, vault, cfg, app.WithClock(firstClock.Now))
	first.finish(first.update(app.SyncNowMsg{}))

	second := window(t, vault, cfg, app.WithClock(secondClock.Now))
	before := gitIn(t, vault, "rev-parse", "HEAD")
	openAndSave(second, " second")
	secondClock.add(10 * time.Minute)
	poll(second)
	secondClock.add(10 * time.Minute)
	poll(second)
	if gitIn(t, vault, "rev-parse", "HEAD") != before || remoteHead(t, remote) != before {
		t.Fatal("the window without the lock must not commit or push")
	}

	firstClock.add(5 * time.Minute)
	if n := poll(first); n != 1 {
		t.Fatalf("the window holding the lock syncs on its timer (%d commands ran)", n)
	}
	if got := gitIn(t, remote, "show", "main:doc.md"); got != "base second" {
		t.Fatalf("the save from the other window is pushed by the one holding the lock, got %q", got)
	}
}

// remoteHead is the commit origin's main points at.
func remoteHead(t *testing.T, remote string) string {
	t.Helper()
	return gitIn(t, remote, "rev-parse", "main")
}

// S3: a save is committed locally once saves pause, with no network: what
// another device pushed is not fetched and nothing is pushed.
func TestSaveIsCommittedLocallyWithoutTheNetwork(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	pushFromDesk(t, remote, "desk.md", "from desk")
	pushed := remoteHead(t, remote)
	tracking := gitIn(t, vault, "rev-parse", "refs/remotes/origin/main")
	commits := commitCount(t, vault)
	clock := newTestClock()
	h := window(t, vault, syncConfig(t, vault), app.WithClock(clock.Now))

	openAndSave(h, " mine")
	if n := poll(h); n != 0 {
		t.Fatalf("the commit waits for saves to pause (%d commands ran)", n)
	}
	clock.add(app.SaveCommitDelay)
	if n := poll(h); n != 1 {
		t.Fatalf("a pause after a save commits it (%d commands ran)", n)
	}
	if got := commitCount(t, vault); got != commits+1 {
		t.Fatalf("want exactly one new commit, got %d", got-commits)
	}
	if got := gitIn(t, vault, "show", "HEAD:doc.md"); got != "base mine" {
		t.Fatalf("the commit must hold the saved note, got %q", got)
	}
	if gitIn(t, vault, "rev-parse", "refs/remotes/origin/main") != tracking {
		t.Fatal("a save must not fetch")
	}
	if remoteHead(t, remote) != pushed {
		t.Fatal("a save must not push")
	}
	if _, err := os.Stat(filepath.Join(vault, "desk.md")); err == nil {
		t.Fatal("a save must not bring the other device's notes")
	}
	if n := poll(h); n != 0 {
		t.Fatalf("once committed, nothing more runs (%d commands ran)", n)
	}
	if bar := statusBar(h); strings.Contains(bar, "Synced") || strings.Contains(bar, "Sync ") || strings.Contains(bar, "synced") {
		t.Fatalf("a commit after a save is quiet and is not a sync:\n%s", bar)
	}
}

// A timed sync speaks on the status line only of a new problem: the same
// problem again leaves the line to the user, and the indicator keeps
// showing it.
func TestTimedSyncSpeaksOnlyOfNewProblems(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	gitIn(t, vault, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))
	clock := newTestClock()
	h := window(t, vault, syncConfig(t, vault), app.WithClock(clock.Now))
	writeOutside(t, vault, "new.md", "fresh")
	clock.add(5 * time.Minute)
	if n := poll(h); n != 1 {
		t.Fatalf("want the timed sync to run (%d commands ran)", n)
	}
	first := statusBar(h)
	if !strings.Contains(first, "Sync") {
		t.Fatalf("a new problem is told on the status line:\n%s", first)
	}
	h.send(core.StatusMsg{Text: "typing along"})
	clock.add(5 * time.Minute)
	if n := poll(h); n != 1 {
		t.Fatalf("want the timed sync to run again (%d commands ran)", n)
	}
	bar := statusBar(h)
	if !strings.Contains(bar, "typing along") {
		t.Fatalf("the same problem again leaves the status line alone:\n%s", bar)
	}
	if !strings.Contains(bar, "󰓦") {
		t.Fatalf("the indicator still shows the state:\n%s", bar)
	}
}

// Saves in quick succession make one commit, holding the last one.
func TestQuickSavesMakeOneCommit(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	commits := commitCount(t, vault)
	clock := newTestClock()
	h := window(t, vault, syncConfig(t, vault), app.WithClock(clock.Now))
	openAndSave(h, " one")
	for _, more := range []string{" two", " three"} {
		clock.add(app.SaveCommitDelay - time.Second)
		if n := poll(h); n != 0 {
			t.Fatalf("no commit while saves keep coming (%d commands ran)", n)
		}
		h.keys("A" + more)
		h.key(tea.KeyEsc)
		h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	}
	clock.add(app.SaveCommitDelay)
	if n := poll(h); n != 1 {
		t.Fatalf("one commit once saves pause (%d commands ran)", n)
	}
	if got := commitCount(t, vault); got != commits+1 {
		t.Fatalf("quick saves make one commit, got %d", got-commits)
	}
	if got := gitIn(t, vault, "show", "HEAD:doc.md"); got != "base one two three" {
		t.Fatalf("the commit holds the last save, got %q", got)
	}
}

// A folder that is not listed, or MISOKU_SYNC=0, runs no git after a save
// or on a timer.
func TestSaveAndTimerRunNoGitWhenSyncIsOff(t *testing.T) {
	for _, c := range []struct {
		name   string
		listed bool
		env    string
	}{
		{"not listed", false, ""},
		{"MISOKU_SYNC=0", true, "0"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, vault := syncedVault(t)
			t.Setenv("MISOKU_SYNC", c.env)
			listed := t.TempDir()
			if c.listed {
				listed = vault
			}
			before := gitIn(t, vault, "rev-parse", "HEAD")
			clock := newTestClock()
			h := window(t, vault, syncConfig(t, listed), app.WithClock(clock.Now))
			openAndSave(h, " mine")
			clock.add(time.Hour)
			if n := poll(h); n != 0 {
				t.Fatalf("no git may run (%d commands ran)", n)
			}
			if gitIn(t, vault, "rev-parse", "HEAD") != before {
				t.Fatal("nothing may be committed")
			}
			if _, err := os.Stat(filepath.Join(vault, ".git", "misoku-sync.lock")); err == nil {
				t.Fatal("no lock may be taken")
			}
		})
	}
}

// S12: a save made while a sync runs, after that sync committed, is not
// lost: it is committed once the sync ends.
func TestSaveDuringASyncIsCommittedAfterIt(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	clock := newTestClock()
	h := window(t, vault, syncConfig(t, vault), app.WithClock(clock.Now))
	h.keys("1")
	h.key(tea.KeyEsc)
	// The sync runs to the end on disk, but the app hears of it only
	// after the save: the save lands while the sync is still running.
	done := runAll(t, h.update(app.SyncNowMsg{}))
	h.keys("A mine")
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	for _, msg := range done {
		h.send(msg)
	}
	clock.add(app.SaveCommitDelay)
	if n := poll(h); n != 1 {
		t.Fatalf("the save must be committed after the sync (%d commands ran)", n)
	}
	if got := gitIn(t, vault, "show", "HEAD:doc.md"); got != "base mine" {
		t.Fatalf("the save made during the sync must be committed, got %q", got)
	}
	if onRemote(t, remote, "doc.md") && gitIn(t, remote, "show", "main:doc.md") != "base" {
		t.Fatal("the commit after a save is not pushed")
	}
}

// A sync asked for while the commit after a save runs is not dropped: it
// runs once that commit ends.
func TestSyncAskedDuringASaveCommitRunsAfterIt(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	clock := newTestClock()
	h := window(t, vault, syncConfig(t, vault), app.WithClock(clock.Now))
	openAndSave(h, " mine")
	clock.add(app.SaveCommitDelay)
	committed := runAll(t, h.update(app.CheckWorkspaceMsg{}))
	if len(committed) != 1 {
		t.Fatalf("want the commit to run, got %d commands", len(committed))
	}
	if cmd := h.update(app.SyncNowMsg{}); cmd != nil {
		t.Fatal("no second git may start while the commit runs")
	}
	queued := h.update(committed[0])
	if queued == nil {
		t.Fatal("the sync asked for must start once the commit ends")
	}
	h.finish(queued)
	if got := gitIn(t, remote, "show", "main:doc.md"); got != "base mine" {
		t.Fatalf("the queued sync must push the save, got %q", got)
	}
}
