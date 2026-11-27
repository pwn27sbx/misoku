package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// RED test for #11: opening the finder must not read note bodies synchronously
// inside Update, and full-text hits must arrive asynchronously while stale
// generations are dropped.
func TestOpenFinderDoesNotReadNoteBodiesInUpdate(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"Alpha.md", "Beta.md", "Gamma.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("# "+name+"\nneedle text inside "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	store, err := core.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := New(store)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m = next.(Model)

	cmd := m.openFinder()
	if m.finder == nil {
		t.Fatal("finder did not open")
	}
	if len(m.finder.lines) != 0 {
		t.Fatalf("openFinder() read %d note bodies synchronously inside Update; expected 0", len(m.finder.lines))
	}
	if !m.finder.loading {
		t.Fatal("expected finder.loading to be true until async load completes")
	}
	if cmd == nil {
		t.Fatal("expected openFinder() to return an async load command")
	}
}

func TestFinderLoadsTextAsynchronouslyAndDropsStaleGen(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Alpha.md"), []byte("# Alpha\nunique_needle_xyz\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := core.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := New(store)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m = next.(Model)

	// Open finder gen 1, then reopen gen 2 before gen 1 finishes.
	_ = m.openFinder()
	gen1 := m.finder.gen
	cmd2 := m.openFinder()
	gen2 := m.finder.gen
	if gen2 <= gen1 {
		t.Fatalf("expected monotonic finderGen: gen1=%d gen2=%d", gen1, gen2)
	}

	// Deliver stale gen1 message: must be ignored.
	next, _ = m.Update(finderLinesLoadedMsg{
		gen:   gen1,
		lines: map[string][]string{"Alpha.md": {"stale"}},
	})
	m = next.(Model)
	if len(m.finder.lines) != 0 || !m.finder.loading {
		t.Fatalf("stale finderLinesLoadedMsg (gen %d) was applied to gen %d", gen1, gen2)
	}

	// Type a full-text query before async load lands.
	m.finder.input.SetValue("unique_needle_xyz")
	m.finder.refilter()
	if len(m.finder.matches) != 0 {
		t.Fatalf("expected 0 text hits before async load, got %d", len(m.finder.matches))
	}

	// Execute only the async loader from cmd2 (skipping cursor.Blink timers).
	if msg := cmd2(); msg != nil {
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batch {
				if c == nil {
					continue
				}
				if loaded, ok := c().(finderLinesLoadedMsg); ok {
					next, _ = m.Update(loaded)
					m = next.(Model)
				}
			}
		} else if loaded, ok := msg.(finderLinesLoadedMsg); ok {
			next, _ = m.Update(loaded)
			m = next.(Model)
		}
	}
	if m.finder.loading {
		t.Fatal("expected finder.loading to be false after gen2 load")
	}
	if len(m.finder.matches) == 0 || m.finder.matches[0].line != 2 {
		t.Fatalf("expected full-text hit on line 2 after async load, got %+v", m.finder.matches)
	}
}
