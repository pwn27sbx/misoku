package app

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// Opening a note while the link index is still being built must not read
// every note on the update loop: that froze the open on a big vault (23 s on
// a 5k-note one). The backlinks show when the build lands, unchanged.
func TestOpenNoteDoesNotScanTheWholeVault(t *testing.T) {
	const notes = 1000
	dir := t.TempDir()
	generateSyntheticVault(t, dir, notes)
	store, err := core.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := New(store)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 48})
	m = next.(Model)
	m.reloadTree()
	if len(m.notes) != notes {
		t.Fatalf("vault has %d notes, want %d", len(m.notes), notes)
	}
	target := m.notes[0].ID

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	m.openInCurrentBuffer(target)
	runtime.ReadMemStats(&after)
	if alloc := after.TotalAlloc - before.TotalAlloc; alloc > 64<<20 {
		t.Fatalf("opening a note allocated %d MB with no index yet: it read the whole vault on the update loop", alloc>>20)
	}
	if got := m.preview.Backlinks(); len(got) != 0 {
		t.Fatalf("backlinks showed before the index was built: %v", backlinkIDs(got))
	}

	// The build the open asked for shows the same backlinks the disk has.
	cmd := m.startLinkIndex()
	if cmd == nil {
		t.Fatal("the open did not ask for a link index build")
	}
	deliver(&m, cmd())
	want := backlinkIDs(store.Backlinks(target))
	if len(want) == 0 {
		t.Fatal("setup: the target note has no backlinks")
	}
	if got := backlinkIDs(m.preview.Backlinks()); !reflect.DeepEqual(got, want) {
		t.Fatalf("backlinks after the build = %v, want %v", got, want)
	}
}
