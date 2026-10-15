package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

const removeButton = "Remove the lock and sync"

// leftoverLock leaves .git/index.lock in vault, as a git that stopped
// half-way does, and returns its path.
func leftoverLock(t *testing.T, vault string) string {
	t.Helper()
	lock := filepath.Join(vault, ".git", "index.lock")
	if err := os.WriteFile(lock, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return lock
}

func lockThere(t *testing.T, lock string) bool {
	t.Helper()
	_, err := os.Lstat(lock)
	return err == nil
}

// runLong runs a command that may take a while (removal, then a sync) and
// returns its message.
func runLong(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("no command was returned")
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		return msg
	case <-time.After(60 * time.Second):
		t.Fatal("the command did not finish")
	}
	return nil
}

// chooseRemove picks "Remove the lock and sync" the deliberate way: move
// to the button, then Enter. It returns the command Enter gave.
func chooseRemove(h *harness) tea.Cmd {
	h.t.Helper()
	h.update(tea.KeyMsg{Type: tea.KeyLeft})
	return h.update(tea.KeyMsg{Type: tea.KeyEnter})
}

// S13: a sync stopped by a leftover lock never opens a dialog on its own
// (the user may be typing in a note): only the status bar says so, and
// `:sync status`, opened on purpose, offers to remove it.
func TestSyncLeftoverLockShowsOnlyInTheStatusBar(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	lock := leftoverLock(t, vault)
	writeOutside(t, vault, "new.md", "fresh")
	h := window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	if v := h.view(); strings.Contains(v, removeButton) {
		t.Fatalf("no dialog may open after a sync:\n%s", v)
	}
	wantBar(t, h, "sync paused")
	wantBar(t, h, ":sync status")
	v := openSyncStatus(h)
	for _, want := range []string{"git lock file", "may belong to a git command still running", "only if no git command is running", removeButton, "Cancel"} {
		if !strings.Contains(v, want) {
			t.Fatalf(":sync status must show %q:\n%s", want, v)
		}
	}
	if !lockThere(t, lock) || onRemote(t, remote, "new.md") {
		t.Fatal("nothing happens before the user confirms")
	}
}

// Confirming removes exactly that lock and syncs right away.
func TestSyncConfirmedLockRemovalSyncs(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	lock := leftoverLock(t, vault)
	writeOutside(t, vault, "new.md", "fresh")
	h := window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	openSyncStatus(h)

	removed := runLong(t, chooseRemove(h))
	if lockThere(t, lock) {
		t.Fatal("confirming must remove the lock")
	}
	h.finish(h.update(removed))
	if !onRemote(t, remote, "new.md") {
		t.Fatalf("the sync must run right after:\n%s", h.view())
	}
	wantBar(t, h, "synced")
}

// In `:sync status`, Cancel (or Enter alone, which starts on Cancel)
// touches nothing.
func TestSyncLockRemovalCancelledTouchesNothing(t *testing.T) {
	t.Parallel()
	for _, key := range []tea.KeyMsg{{Type: tea.KeyEsc}, {Type: tea.KeyEnter}} {
		remote, vault := syncedVault(t)
		lock := leftoverLock(t, vault)
		writeOutside(t, vault, "new.md", "fresh")
		h := window(t, vault, syncConfig(t, vault))
		h.finish(h.update(app.SyncNowMsg{}))
		openSyncStatus(h)
		if cmd := h.update(key); cmd != nil {
			runLong(t, cmd)
		}
		if !lockThere(t, lock) || onRemote(t, remote, "new.md") {
			t.Fatalf("%v must touch nothing", key)
		}
		if strings.Contains(h.view(), removeButton) {
			t.Fatalf("%v closes the dialog", key)
		}
	}
}

// A lock that changed since the dialog showed it belongs to a running git.
func TestSyncLockChangedSinceShownIsNotRemoved(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	lock := leftoverLock(t, vault)
	writeOutside(t, vault, "new.md", "fresh")
	h := window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	openSyncStatus(h)
	if err := os.WriteFile(lock, []byte("a git started meanwhile"), 0o644); err != nil {
		t.Fatal(err)
	}
	if next := h.update(runLong(t, chooseRemove(h))); next != nil {
		t.Fatal("no sync may start past a lock that was not removed")
	}
	if !lockThere(t, lock) || onRemote(t, remote, "new.md") {
		t.Fatal("a changed lock must stay")
	}
	wantBar(t, h, "Lock not removed")
}

// `:sync status` offers the removal after a sync stopped on the lock.
func TestSyncStatusOffersToRemoveTheLock(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	lock := leftoverLock(t, vault)
	writeOutside(t, vault, "new.md", "fresh")
	h := window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	if v := openSyncStatus(h); !strings.Contains(v, removeButton) || !strings.Contains(v, "Remote:") {
		t.Fatalf(":sync status must offer to remove the lock:\n%s", v)
	}
	if cmd := h.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}); cmd != nil {
		runLong(t, cmd)
	}
	if !lockThere(t, lock) || !strings.Contains(h.view(), removeButton) {
		t.Fatal("a letter never chooses Remove in :sync status either")
	}
	h.finish(h.update(runLong(t, chooseRemove(h))))
	if lockThere(t, lock) || !onRemote(t, remote, "new.md") {
		t.Fatal("confirming from :sync status removes the lock and syncs")
	}
}

// D2: a folder that does not sync never gets the offer, and runs no git.
func TestSyncOffNeverOffersLockRemoval(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	lock := leftoverLock(t, vault)
	h := window(t, vault, syncConfig(t, t.TempDir()))
	if cmd := h.update(app.SyncNowMsg{}); cmd != nil {
		t.Fatal("sync is off")
	}
	if v := openSyncStatus(h); strings.Contains(v, removeButton) {
		t.Fatalf("no offer when sync is off:\n%s", v)
	}
	if !lockThere(t, lock) {
		t.Fatal("the lock must stay")
	}
}

// the dialog pops up by itself when a background sync (on open,
// or sync_now) finishes. A user typing in a note at that moment sends the
// next keystroke to the dialog; a word starting with "r" removes the lock
// without the dialog ever being read.
func TestSyncLockDialogIgnoresTypingMeantForANote(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	lock := leftoverLock(t, vault)
	h := window(t, vault, syncConfig(t, vault))
	h.keys("1")                       // open doc.md
	h.keys("i")                       // insert mode
	cmd := h.update(app.SyncNowMsg{}) // a sync is running in the background
	h.keys("note ")
	h.finish(cmd) // ...and it finishes while the user types
	h.keys("hello really")
	h.key(tea.KeyEnter)
	if !lockThere(t, lock) {
		t.Fatal("a keystroke meant for the note removed the lock without the dialog being read")
	}
}

// the lock was removed and taken again by a new git (same size,
// new time) after the dialog appeared: confirming must not remove it.
func TestSyncLockReplacedWithTheSameSizeIsNotRemoved(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	lock := leftoverLock(t, vault)
	writeOutside(t, vault, "new.md", "fresh")
	h := window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	openSyncStatus(h)
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(lock, nil, 0o644); err != nil { // same (empty) size
		t.Fatal(err)
	}
	h.update(runLong(t, chooseRemove(h)))
	if !lockThere(t, lock) || onRemote(t, remote, "new.md") {
		t.Fatal("a lock taken again after the dialog appeared must stay")
	}
}

// Enter, Esc, q, c, space never remove.
func TestSyncLockDialogKeysThatNeverRemove(t *testing.T) {
	t.Parallel()
	for _, k := range []tea.KeyMsg{{Type: tea.KeyEnter}, {Type: tea.KeyEsc}, {Type: tea.KeySpace}, {Type: tea.KeyRunes, Runes: []rune{'q'}}, {Type: tea.KeyRunes, Runes: []rune{'c'}}} {
		_, vault := syncedVault(t)
		lock := leftoverLock(t, vault)
		h := window(t, vault, syncConfig(t, vault))
		h.finish(h.update(app.SyncNowMsg{}))
		openSyncStatus(h)
		if !strings.Contains(h.view(), removeButton) {
			t.Fatal("no dialog")
		}
		if cmd := h.update(k); cmd != nil {
			runLong(t, cmd)
		}
		if !lockThere(t, lock) {
			t.Fatalf("%q removed the lock", k.String())
		}
	}
}

// Realistic note-writing typed while a sync stops on a leftover lock: no
// dialog takes the keys, and none of them removes the lock.
func TestSyncNoteWritingNeverRemovesTheLock(t *testing.T) {
	t.Parallel()
	runes := func(s string) []tea.Msg {
		var out []tea.Msg
		for _, r := range s {
			out = append(out, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
		return out
	}
	key := func(k tea.KeyType) []tea.Msg { return []tea.Msg{tea.KeyMsg{Type: k}} }
	cat := func(parts ...[]tea.Msg) []tea.Msg {
		var out []tea.Msg
		for _, p := range parts {
			out = append(out, p...)
		}
		return out
	}
	for name, seq := range map[string][]tea.Msg{
		"words":             runes("really remove this right now"),
		"word-then-enter":   cat(runes("really"), key(tea.KeyEnter)),
		"indent-item-enter": cat(key(tea.KeyTab), runes("- sub item"), key(tea.KeyEnter)),
		"down-then-enter":   cat(key(tea.KeyDown), runes("next line"), key(tea.KeyEnter)),
		"right-arrow-enter": cat(key(tea.KeyRight), key(tea.KeyEnter)),
		"burst-hjkl-space":  cat(runes("hjkl lkjh rrr qqq"), key(tea.KeySpace), key(tea.KeyEnter)),
		"two-tabs-enter":    cat(key(tea.KeyTab), key(tea.KeyTab), key(tea.KeyEnter)),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, vault := syncedVault(t)
			lock := leftoverLock(t, vault)
			h := window(t, vault, syncConfig(t, vault))
			h.keys("1")
			h.keys("i")
			cmd := h.update(app.SyncNowMsg{})
			h.keys("note ")
			h.finish(cmd)
			h.send(seq...)
			if !lockThere(t, lock) {
				t.Fatal("note-writing keys removed the lock")
			}
			if v := h.view(); strings.Contains(v, removeButton) {
				t.Fatalf("no dialog may take the keys meant for the note:\n%s", v)
			}
		})
	}
}

// Other dialogs keep their first-letter keys.
func TestDeleteDialogStillTakesItsLetter(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "Only Page", "x"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("e")
	h.keys("d")
	if v := h.view(); !strings.Contains(v, "Delete Only Page.md?") {
		t.Fatalf("expected delete confirmation:\n%s", v)
	}
	h.keys("d")
	if len(store.List()) != 0 {
		t.Fatal("the letter d must still choose Delete")
	}
}
