package app_test

// Regression tests from the recheck of F2.12: saving over an outside change
// keeps both versions, and the sync on opening reports config warnings.

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// fixedClock always says 9:05 on 2026-10-16, so every copy made in a test
// gets the same base name and the " 2", " 3"... suffixes are exercised.
func fixedClock() time.Time { return time.Date(2026, 10, 16, 9, 5, 0, 0, time.Local) }

// The disk keeps changing (another editor) while the user keeps editing
// and saving: each outside version lands in its own copy, the note holds
// every edit of the user's, and a save with no new outside change makes no
// copy.
func TestEachOutsideChangeSavedOverGetsItsOwnCopy(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "base"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, app.WithClock(fixedClock))
	h.keys("1")
	want := "base"
	var theirs []string
	for i := 0; i < 6; i++ {
		word := fmt.Sprintf(" m%d", i)
		h.keys("A" + word)
		h.key(tea.KeyEsc)
		want += word
		other := fmt.Sprintf("outside %d", i)
		writeOutside(t, store.Root(), "doc.md", other)
		theirs = append(theirs, other)
		if i%2 == 0 {
			h.send(app.CheckWorkspaceMsg{}) // half are noticed by the poll first
		}
		h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
		if got := readNote(t, store.Root(), "doc.md"); got != want {
			t.Fatalf("save %d: the note must hold the user's text, got %q want %q", i, got, want)
		}
		if !strings.Contains(statusBar(h), "Both kept") {
			t.Fatalf("save %d: the bar must say both were kept:\n%s", i, statusBar(h))
		}
		// Saving again with nothing new outside makes no copy.
		h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	}
	got := copiesOf(t, store.Root(), "doc")
	sort.Strings(got)
	sort.Strings(theirs)
	if strings.Join(got, "|") != strings.Join(theirs, "|") {
		t.Fatalf("one copy per outside version, nothing lost:\n got %q\nwant %q", got, theirs)
	}
}

// A copy name already taken (a sync's copy, or one made a minute ago) is
// never overwritten: the next free suffix is used.
func TestKeepingBothNeverOverwritesAnExistingCopy(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "base"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, app.WithClock(fixedClock))
	h.keys("1")
	taken := "doc (conflict 2026-10-16 0905).md"
	writeOutside(t, store.Root(), taken, "older copy")
	writeOutside(t, store.Root(), "doc (conflict 2026-10-16 0905) 2.md", "older copy 2")
	h.keys("A mine")
	h.key(tea.KeyEsc)
	writeOutside(t, store.Root(), "doc.md", "theirs")
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if got := readNote(t, store.Root(), taken); got != "older copy" {
		t.Fatalf("an existing copy must not be overwritten, got %q", got)
	}
	if got := readNote(t, store.Root(), "doc (conflict 2026-10-16 0905) 2.md"); got != "older copy 2" {
		t.Fatalf("an existing copy must not be overwritten, got %q", got)
	}
	if got := readNote(t, store.Root(), "doc (conflict 2026-10-16 0905) 3.md"); got != "theirs" {
		t.Fatalf("the outside version must go to the next free name, got %q", got)
	}
	if got := readNote(t, store.Root(), "doc.md"); got != "base mine" {
		t.Fatalf("got %q", got)
	}
}

// When the copy cannot be written (read-only folder), nothing is written:
// the outside version stays in the note, the user's text stays in the
// editor, the bar says why, and Save All on quit does not quit.
func TestKeepingBothWritesNothingWhenTheCopyCannotBeWritten(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("folder permissions differ on Windows")
	}
	store := newTestStore(t)
	if _, err := store.SaveAs("sub", "doc", "base"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, app.WithClock(fixedClock))
	h.keys("1")
	h.keys("A mine")
	h.key(tea.KeyEsc)
	writeOutside(t, store.Root(), "sub/doc.md", "theirs")
	dir := filepath.Join(store.Root(), "sub")
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if got := readNote(t, store.Root(), "sub/doc.md"); got != "theirs" {
		t.Fatalf("nothing may be written, got %q", got)
	}
	if got := copiesOf(t, dir, "doc"); len(got) != 0 {
		t.Fatalf("no copy, got %q", got)
	}
	if bar := statusBar(h); !strings.Contains(bar, "Not saved") {
		t.Fatalf("the bar must say it was not saved:\n%s", bar)
	}
	if !strings.Contains(h.view(), "base mine") {
		t.Fatalf("the user's text must stay in the editor:\n%s", h.view())
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlC})
	h.keys("s") // Save All
	if h.quit {
		t.Fatal("Save All must not quit when a note could not be saved")
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if got := readNote(t, store.Root(), "sub/doc.md"); got != "base mine" {
		t.Fatalf("once writable, saving works, got %q", got)
	}
	if got := copiesOf(t, dir, "doc"); len(got) != 1 || got[0] != "theirs" {
		t.Fatalf("and keeps the outside version, got %q", got)
	}
}

// Save All on quit with three notes changed outside, one in a folder:
// all are kept twice and Misoku quits.
func TestSaveAllKeepsBothVersionsOfSeveralOutsideChanges(t *testing.T) {
	store := newTestStore(t)
	for _, n := range []string{"alpha", "beta", "gamma"} {
		if _, err := store.SaveAs("", n, "base "+n); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store, app.WithClock(fixedClock))
	h.keys("e")
	h.key(tea.KeyEnter)
	h.keys("A mine")
	h.key(tea.KeyEsc)
	for i := 0; i < 2; i++ {
		h.keys(" e")
		h.keys("j")
		h.key(tea.KeyEnter)
		h.keys("A mine")
		h.key(tea.KeyEsc)
	}
	for _, n := range []string{"alpha", "beta", "gamma"} {
		writeOutside(t, store.Root(), n+".md", "theirs "+n)
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlC})
	h.keys("s")
	if !h.quit {
		t.Fatalf("Save All keeps both versions and quits:\n%s", h.view())
	}
	edited := 0
	for _, n := range []string{"alpha", "beta", "gamma"} {
		note := readNote(t, store.Root(), n+".md")
		copies := copiesOf(t, store.Root(), n)
		switch {
		case note == "base "+n+" mine" && len(copies) == 1 && copies[0] == "theirs "+n:
			edited++
		case note == "theirs "+n && len(copies) == 0:
			// not opened by the setup's navigation: untouched
		default:
			t.Fatalf("%s: note %q, copies %q", n, note, copies)
		}
	}
	if edited < 2 {
		t.Fatalf("setup: want at least 2 edited notes, got %d", edited)
	}
}

// B1: an unusable interval stays reported after the sync on opening.
func TestOpeningSyncTellsTheConfigWarningOnce(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	h := window(t, vault, intervalConfig(t, vault, "30s"), app.WithClock(newTestClock().Now))
	h.finish(h.update(app.SyncNowMsg{}))
	if v := noSpace(statusBar(h)); !strings.Contains(v, "Synced") || !strings.Contains(v, "sync:vault") {
		t.Fatalf("the opening sync must carry the warning:\n%s", statusBar(h))
	}
	// Said once: the next sync asked for does not repeat it.
	h.finish(h.update(app.SyncNowMsg{}))
	if strings.Contains(statusBar(h), "sync: vault") {
		t.Fatalf("the warning is told once:\n%s", statusBar(h))
	}
}
