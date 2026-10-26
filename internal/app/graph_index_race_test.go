package app

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// asyncRuntime runs every command in its own goroutine, like Bubble Tea,
// and lets the test deliver finished messages in any order.
type asyncRuntime struct {
	mu   sync.Mutex
	done []tea.Msg
	wg   sync.WaitGroup
}

func (r *asyncRuntime) run(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ch := make(chan tea.Msg, 1)
		go func() { ch <- cmd() }()
		select {
		case msg := <-ch:
			if b, ok := msg.(tea.BatchMsg); ok {
				for _, c := range b {
					r.run(c)
				}
				return
			}
			r.mu.Lock()
			r.done = append(r.done, msg)
			r.mu.Unlock()
		case <-time.After(200 * time.Millisecond): // timers, blinks
		}
	}()
}

func (r *asyncRuntime) take(rng *rand.Rand) (tea.Msg, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.done) == 0 {
		return nil, false
	}
	i := rng.Intn(len(r.done))
	msg := r.done[i]
	r.done = append(r.done[:i], r.done[i+1:]...)
	return msg, true
}

func keyMsgs(s string) []tea.Msg {
	var out []tea.Msg
	for _, c := range s {
		out = append(out, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{c}})
	}
	return out
}

// Rapid saves, outside edits, scans and typing while index builds run and
// land out of order: no data race (run with -race), and once everything
// settles the index equals a fresh synchronous read of the disk.
func TestLinkIndexSettlesUnderOutOfOrderBuilds(t *testing.T) {
	for seed := int64(1); seed <= 3; seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			rng := rand.New(rand.NewSource(seed))
			store, err := core.NewStore(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			const n = 30
			for i := 0; i < n; i++ {
				if _, err := store.SaveAs("", fmt.Sprintf("N%02d", i), fmt.Sprintf("[[N%02d]]", (i+1)%n)); err != nil {
					t.Fatal(err)
				}
			}
			m := New(store)
			rt := &asyncRuntime{}
			send := func(msg tea.Msg) {
				next, cmd := m.Update(msg)
				m = next.(Model)
				rt.run(cmd)
			}
			send(tea.WindowSizeMsg{Width: 120, Height: 40})
			for _, k := range append(append([]tea.Msg{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\\'}}}, keyMsgs("n03")...), tea.KeyMsg{Type: tea.KeyEnter}, tea.KeyMsg{Type: tea.KeyEsc}) {
				send(k)
			}
			stamp := time.Now()
			for step := 0; step < 250; step++ {
				switch rng.Intn(9) {
				case 0: // outside edit
					stamp = stamp.Add(time.Minute)
					id := fmt.Sprintf("N%02d.md", rng.Intn(n))
					abs := filepath.Join(store.Root(), id)
					_ = os.WriteFile(abs, []byte(fmt.Sprintf("[[N%02d]] [[N%02d]]", rng.Intn(n), rng.Intn(n))), 0o644)
					_ = os.Chtimes(abs, stamp, stamp)
				case 1: // workspace scan
					send(workspaceScannedMsg{treeGen: m.treeGen, pages: store.List()})
				case 2: // typing in the open note
					send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
					for _, k := range keyMsgs(fmt.Sprintf(" [x](N%02d.md)", rng.Intn(n))) {
						send(k)
					}
					send(tea.KeyMsg{Type: tea.KeyEsc})
				case 3: // save
					for _, k := range keyMsgs(":w") {
						send(k)
					}
					send(tea.KeyMsg{Type: tea.KeyEnter})
				case 4: // toggle the graph
					for _, k := range keyMsgs(":graph") {
						send(k)
					}
					send(tea.KeyMsg{Type: tea.KeyEnter})
					send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
					send(tea.KeyMsg{Type: tea.KeyEnter})
				case 5, 6: // a save elsewhere (applyTree, buffers) asks for a build
					m.refreshGraph()
					send(tea.KeyMsg{Type: tea.KeyEsc})
				default: // deliver a finished command, any order
					if msg, ok := rt.take(rng); ok {
						send(msg)
					}
				}
				_ = m.View()
			}
			// Settle: save, deliver everything, then scan until no build is asked for.
			send(tea.KeyMsg{Type: tea.KeyEsc})
			for _, k := range keyMsgs(":w") {
				send(k)
			}
			send(tea.KeyMsg{Type: tea.KeyEnter})
			for round := 0; ; round++ {
				rt.wg.Wait()
				for {
					msg, ok := rt.take(rng)
					if !ok {
						break
					}
					if _, ok := msg.(linkIndexedMsg); ok {
						send(msg)
					}
				}
				rt.wg.Wait()
				if m.index.gen == m.index.applied && !m.linkIndexOutOfDate(store.List()) {
					break
				}
				if round > 5 {
					t.Fatalf("index never settles: gen %d applied %d", m.index.gen, m.index.applied)
				}
				send(workspaceScannedMsg{treeGen: m.treeGen, pages: store.List()})
			}
			t.Logf("builds started %d", m.index.gen)
			notes, edges := m.linkGraph()
			wantNotes, wantEdges := store.LinkGraph()
			if !reflect.DeepEqual(notes, wantNotes) || !reflect.DeepEqual(edges, wantEdges) {
				t.Fatalf("settled graph differs from disk:\n%v\n%v", edges, wantEdges)
			}
			for _, p := range wantNotes {
				if got, want := m.backlinks(p.ID), store.Backlinks(p.ID); !reflect.DeepEqual(got, want) {
					t.Fatalf("backlinks of %s = %v, want %v", p.ID, got, want)
				}
			}
		})
	}
}

// An outside edit made after a running build listed the notes, while a scan
// declines to ask for another build, is picked up by the next scan.
func TestScanCatchesAnEditMadeDuringABuild(t *testing.T) {
	store, _ := core.NewStore(t.TempDir())
	for _, n := range [][2]string{{"A", "to [[B]]"}, {"B", "b"}, {"C", "c"}} {
		if _, err := store.SaveAs("", n[0], n[1]); err != nil {
			t.Fatal(err)
		}
	}
	m := New(store)
	m.refreshGraph()
	build := m.startLinkIndex()
	result := build() // listed and read before the edit
	abs := filepath.Join(store.Root(), "C.md")
	_ = os.WriteFile(abs, []byte("to [[B]]"), 0o644)
	later := time.Now().Add(time.Hour)
	_ = os.Chtimes(abs, later, later)
	scan := func() { m.applyWorkspaceScan(workspaceScannedMsg{treeGen: m.treeGen, pages: store.List()}) }
	scan()
	if m.index.wanted {
		t.Fatal("unexpected: scan asked for a build while one runs")
	}
	next, cmd := m.Update(result)
	m = next.(Model)
	if cmd != nil {
		t.Fatal("no build expected right after the result")
	}
	scan()
	if !m.index.wanted {
		t.Fatal("the edit made during the build is lost: the next scan asks for nothing")
	}
	next, cmd = m.Update(nil) // any message starts the build asked for
	m = next.(Model)
	if cmd == nil {
		t.Fatal("the build asked for did not start")
	}
	r, ok := cmd().(linkIndexedMsg)
	if !ok {
		t.Fatal("expected the build's result")
	}
	m.applyLinkIndex(r)
	if got := m.backlinks("B.md"); len(got) != 2 {
		t.Fatalf("backlinks of B = %v, want A and C", got)
	}
}
