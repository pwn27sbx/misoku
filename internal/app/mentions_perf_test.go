package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// RED test for 0.25.0 mention rescan: saving the active note must not trigger a
// full-vault mention rescan, and an external edit to one other note must rescan
// only that changed note.
func TestSaveActiveNoteDoesNotRescanVaultForMentions(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range [][2]string{
		{"Target.md", "# Target\n\ninitial body"},
		{"Other1.md", "mentions Target in plain text"},
		{"Other2.md", "nothing related here"},
	} {
		if err := os.WriteFile(filepath.Join(dir, tc[0]), []byte(tc[1]), 0o644); err != nil {
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
	m.openInCurrentBuffer("Target.md")

	// Initial index + mention scan.
	if cmd := m.startLinkIndex(); cmd != nil {
		m.applyLinkIndex(cmd().(linkIndexedMsg))
	}
	if cmd := m.startMentionScan(); cmd != nil {
		m.applyMentions(cmd().(mentionsScannedMsg))
	}
	if m.mentions.shown != "Target.md" || len(m.mentions.rows) != 1 {
		t.Fatalf("initial mentions = shown:%q rows:%+v, want Target.md with 1 row", m.mentions.shown, m.mentions.rows)
	}
	if len(m.mentions.rows) != 1 || m.mentions.rows[0].ID != "Other1.md" {
		t.Fatalf("initial mentions = %+v, want [Other1.md]", m.mentions.rows)
	}

	// Edit active note and save: another note's mentions of Target cannot have changed.
	cur := m.activeBuffer()
	m.content.SetBuffer(core.Page{ID: cur.id, Title: cur.title, Content: "# Target\n\nupdated body"}, false)
	// Advance mtime slightly so LinkIndex sees Target.md changed.
	m.writeBuffer(cur, nil)
	at := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(filepath.Join(dir, "Target.md"), at, at)

	if cmd := m.startLinkIndex(); cmd != nil {
		next, _ = m.Update(cmd())
		m = next.(Model)
	}
	if m.mentions.wanted {
		t.Fatal("saving active note scheduled a mention rescan; expected 0 mention rescans")
	}
	if cmd := m.startMentionScan(); cmd != nil {
		t.Fatal("saving active note started a mention scan command; expected nil")
	}
}

func TestExternalEditRescansOnlyChangedNote(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range [][2]string{
		{"Target.md", "# Target\n\nbody"},
		{"Other1.md", "mentions Target once"},
		{"Other2.md", "no mention yet"},
	} {
		if err := os.WriteFile(filepath.Join(dir, tc[0]), []byte(tc[1]), 0o644); err != nil {
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
	m.openInCurrentBuffer("Target.md")

	m.refreshGraph()
	if cmd := m.startLinkIndex(); cmd != nil {
		m.applyLinkIndex(cmd().(linkIndexedMsg))
	}
	if cmd := m.startMentionScan(); cmd != nil {
		m.applyMentions(cmd().(mentionsScannedMsg))
	}
	if m.mentions.shown != "Target.md" || len(m.mentions.rows) != 1 {
		t.Fatalf("initial mentions = shown:%q rows:%+v, want Target.md with 1 row", m.mentions.shown, m.mentions.rows)
	}

	// Externally edit Other2.md to add a mention of Target.
	other2 := filepath.Join(dir, "Other2.md")
	if err := os.WriteFile(other2, []byte("now mentions Target too"), 0o644); err != nil {
		t.Fatal(err)
	}
	at := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(other2, at, at)

	m.refreshGraph()
	if cmd := m.startLinkIndex(); cmd != nil {
		m.applyLinkIndex(cmd().(linkIndexedMsg))
	}
	if !m.mentions.wanted || len(m.mentions.onlyIDs) != 1 || m.mentions.onlyIDs[0] != "Other2.md" {
		t.Fatalf("expected targeted mention rescan of [Other2.md], got wanted=%v onlyIDs=%v", m.mentions.wanted, m.mentions.onlyIDs)
	}
	if cmd := m.startMentionScan(); cmd != nil {
		next, _ = m.Update(cmd())
		m = next.(Model)
	}
	if len(m.mentions.rows) != 2 {
		t.Fatalf("expected 2 mention rows after incremental scan, got %+v", m.mentions.rows)
	}
}
