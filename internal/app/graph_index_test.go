package app

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// indexStore is a workspace where A links to B, and C to nothing.
func indexStore(t *testing.T) *core.Store {
	t.Helper()
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range [][2]string{{"A", "to [[B]]"}, {"B", "b"}, {"C", "c"}} {
		if _, err := store.SaveAs("", n[0], n[1]); err != nil {
			t.Fatal(err)
		}
	}
	return store
}

// rewrite changes a note on disk as another program would, with a later
// modification time than any before it.
func rewrite(t *testing.T, store *core.Store, id, content string, hours int) {
	t.Helper()
	abs := filepath.Join(store.Root(), filepath.FromSlash(id))
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	at := time.Now().Add(time.Duration(hours) * time.Hour)
	if err := os.Chtimes(abs, at, at); err != nil {
		t.Fatal(err)
	}
}

// deliver feeds msg to the model like the runtime and returns the command
// Update gave back.
func deliver(m *Model, msg tea.Msg) tea.Cmd {
	next, cmd := m.Update(msg)
	*m = next.(Model)
	return cmd
}

// startIndex asks for a build and returns its command, without running it.
func startIndex(t *testing.T, m *Model) tea.Cmd {
	t.Helper()
	m.refreshGraph()
	cmd := m.startLinkIndex()
	if cmd == nil {
		t.Fatal("a build was asked for but none started")
	}
	return cmd
}

func backlinkIDs(bl []core.Backlink) []string {
	var ids []string
	for _, b := range bl {
		ids = append(ids, b.ID)
	}
	return ids
}

// A build that a newer one replaced is dropped when it arrives first, and
// the newer one is applied when it arrives.
func TestLinkIndexDropsResultOfAnOlderBuild(t *testing.T) {
	store := indexStore(t)
	m := New(store)
	first := startIndex(t, &m)()
	rewrite(t, store, "C.md", "to [[B]]", 1)
	second := startIndex(t, &m)

	deliver(&m, first)
	if m.index.idx != nil {
		t.Fatal("the result of a build a newer one replaced should be dropped")
	}
	deliver(&m, second())
	if got, want := backlinkIDs(m.backlinks("B.md")), []string{"A.md", "C.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("backlinks of B = %v, want %v", got, want)
	}
}

// Two builds that finish out of order: the newer is kept.
func TestLinkIndexKeepsNewerResultWhenOlderArrivesLate(t *testing.T) {
	store := indexStore(t)
	m := New(store)
	older := startIndex(t, &m)
	olderMsg := older() // reads A → B only
	rewrite(t, store, "C.md", "to [[B]]", 1)
	newerMsg := startIndex(t, &m)()

	deliver(&m, newerMsg)
	deliver(&m, olderMsg)
	if got, want := backlinkIDs(m.backlinks("B.md")), []string{"A.md", "C.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("backlinks of B = %v, want %v: the older result replaced the newer", got, want)
	}
	if notes, edges := m.linkGraph(); len(notes) != 3 || len(edges) != 2 {
		t.Fatalf("graph = %v %v, want 3 notes and 2 links", notes, edges)
	}
}

// Before the first build arrives, backlinks and the graph are read from
// the disk as before.
func TestLinkIndexFallsBackToTheDiskBeforeTheFirstBuild(t *testing.T) {
	store := indexStore(t)
	m := New(store)
	if m.index.idx != nil {
		t.Fatal("no build should have arrived yet")
	}
	if got := backlinkIDs(m.backlinks("B.md")); !reflect.DeepEqual(got, []string{"A.md"}) {
		t.Fatalf("backlinks of B = %v, want [A.md]", got)
	}
	notes, edges := m.linkGraph()
	wantNotes, wantEdges := store.LinkGraph()
	if !reflect.DeepEqual(notes, wantNotes) || !reflect.DeepEqual(edges, wantEdges) {
		t.Fatalf("graph = %v %v, want %v %v", notes, edges, wantNotes, wantEdges)
	}
}

// Once built, backlinks come from the index: moving through notes doesn't
// read every note again. A change on disk shows after the next build.
func TestBacklinksComeFromTheIndex(t *testing.T) {
	store := indexStore(t)
	m := New(store)
	deliver(&m, startIndex(t, &m)())
	rewrite(t, store, "C.md", "to [[B]]", 1)
	if got := backlinkIDs(m.backlinks("B.md")); !reflect.DeepEqual(got, []string{"A.md"}) {
		t.Fatalf("backlinks of B = %v: they should come from the index, not the disk", got)
	}
	deliver(&m, startIndex(t, &m)())
	if got := backlinkIDs(m.backlinks("B.md")); !reflect.DeepEqual(got, []string{"A.md", "C.md"}) {
		t.Fatalf("backlinks of B after a build = %v", got)
	}
}

// A workspace check asks for a build when a note's modification time
// changed, but not while one is still running: that one's result is
// compared on the next check, so a slow build is never replaced over and
// over before it lands.
func TestScanAsksForABuildWhenANoteChanged(t *testing.T) {
	store := indexStore(t)
	m := New(store)
	deliver(&m, startIndex(t, &m)())

	scan := func() workspaceScannedMsg { return workspaceScannedMsg{treeGen: m.treeGen, pages: store.List()} }
	m.applyWorkspaceScan(scan())
	if m.index.wanted {
		t.Fatal("nothing changed: no build should be asked for")
	}
	rewrite(t, store, "C.md", "to [[B]]", 1)
	pending := startIndex(t, &m) // a build already running
	m.applyWorkspaceScan(scan())
	if m.index.wanted {
		t.Fatal("no new build while one is running")
	}
	deliver(&m, pending())
	rewrite(t, store, "C.md", "c", 2)
	m.applyWorkspaceScan(scan())
	if !m.index.wanted {
		t.Fatal("a note changed on disk: a build should be asked for")
	}
}

// A build runs on its own copy of the index: reading the one in use while
// it runs is safe (run with -race).
func TestLinkIndexBuildDoesNotShareTheIndexInUse(t *testing.T) {
	store := indexStore(t)
	m := New(store)
	deliver(&m, startIndex(t, &m)())
	rewrite(t, store, "C.md", "to [[B]]", 1)
	rewrite(t, store, "A.md", "to [[C]]", 1)
	cmd := startIndex(t, &m)

	var wg sync.WaitGroup
	var msg tea.Msg
	wg.Add(1)
	go func() { defer wg.Done(); msg = cmd() }()
	for i := 0; i < 50; i++ {
		_ = m.backlinks("B.md")
		_, _ = m.linkGraph()
	}
	wg.Wait()
	if got := backlinkIDs(m.backlinks("B.md")); !reflect.DeepEqual(got, []string{"A.md"}) {
		t.Fatalf("the index in use changed while the build ran: backlinks of B = %v", got)
	}
	deliver(&m, msg)
	if got := backlinkIDs(m.backlinks("B.md")); !reflect.DeepEqual(got, []string{"C.md"}) {
		t.Fatalf("backlinks of B after the build = %v, want [C.md]", got)
	}
}

// A note that could not be read is read again on the next workspace check,
// though its modification time did not change.
func TestScanRereadsANoteThatFailedToRead(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs file permissions that stop the owner reading")
	}
	store := indexStore(t)
	abs := filepath.Join(store.Root(), "A.md")
	if err := os.Chmod(abs, 0o000); err != nil {
		t.Fatal(err)
	}
	m := New(store)
	deliver(&m, startIndex(t, &m)())
	if got := backlinkIDs(m.backlinks("B.md")); got != nil {
		t.Fatalf("backlinks of B = %v while A can't be read", got)
	}
	if err := os.Chmod(abs, 0o644); err != nil {
		t.Fatal(err)
	}
	m.applyWorkspaceScan(workspaceScannedMsg{treeGen: m.treeGen, pages: store.List()})
	if !m.index.wanted {
		t.Fatal("a note that failed to read should be read again on the next check")
	}
	deliver(&m, m.startLinkIndex()())
	if got := backlinkIDs(m.backlinks("B.md")); !reflect.DeepEqual(got, []string{"A.md"}) {
		t.Fatalf("backlinks of B = %v, want [A.md]", got)
	}
}
