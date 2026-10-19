package app_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// startLines is the text of the setup assistant's first dialog.
var startLines = []string{
	"Your notes will be kept in a private GitHub repository and reach your other computers on their own.",
	"You need a GitHub account. Misoku signs in with gh, GitHub's command-line tool, which keeps the access: Misoku never sees or stores it.",
	"In the repository your notes are not encrypted, so it stays private.",
}

// startDialog opens the setup assistant's first dialog on a window the size
// of the demo video's terminal (130x37), where its long lines wrap.
func startDialog(t *testing.T) string {
	t.Helper()
	vault := t.TempDir()
	writeOutside(t, vault, "a.md", "hello")
	h := window(t, vault, filepath.Join(t.TempDir(), "config.json"))
	h.send(tea.WindowSizeMsg{Width: 130, Height: 37})
	h.send(core.SyncSetupMsg{})
	v := ansi.Strip(h.m.View()) // h.view expects the default window size
	if !strings.Contains(v, "Sync this vault") {
		t.Fatalf("the assistant must open with what will happen:\n%s", v)
	}
	return v
}

// A dialog wraps a long line between words: every word of the text is
// whole on one row ("their", not "thei" and "r").
func TestDialogWrapsAtWordBoundaries(t *testing.T) {
	t.Parallel()
	v := startDialog(t)
	onRows := map[string]bool{}
	for _, row := range dialogRows(v) {
		for _, w := range strings.Fields(row) {
			onRows[w] = true
		}
	}
	for _, w := range strings.Fields(strings.Join(startLines, " ")) {
		if !onRows[w] {
			t.Errorf("the word %q is split across rows:\n%s", w, v)
		}
	}
}

// The rows a long line wraps onto start where its first row starts: the
// text reads as one paragraph, not as an indented list.
func TestDialogWrappedRowsAreNotIndented(t *testing.T) {
	t.Parallel()
	v := startDialog(t)
	var indents []int
	in := false
	for _, row := range dialogRows(v) {
		text := strings.TrimSpace(row)
		switch {
		case strings.HasPrefix(text, "Your notes"):
			in = true
		case strings.Contains(text, "Cancel"):
			in = false
		}
		if in && text != "" {
			indents = append(indents, len(row)-len(strings.TrimLeft(row, " ")))
		}
	}
	if len(indents) <= len(startLines) {
		t.Fatalf("the long lines should wrap on a 130-column window:\n%s", v)
	}
	for _, n := range indents {
		if n != indents[0] {
			t.Fatalf("every row of the text must start at column %d, got %v:\n%s", indents[0], indents, v)
		}
	}
}

// A single word longer than a row (a deep path) is still broken so the
// dialog fits the window, and reads whole once joined: `:sync status`
// naming a conflict copy with a long name, on an 80-column window.
func TestDialogBreaksAWordLongerThanTheRow(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	long := "sub/" + strings.Repeat("x", 100) + " (conflict desk 2026-10-14 1530).md"
	writeOutside(t, vault, long, "theirs")
	h := window(t, vault, syncConfig(t, vault))
	h.send(tea.WindowSizeMsg{Width: 80, Height: 40})

	askSyncStatus(h)
	v := ansi.Strip(h.m.View()) // h.view expects the default window size
	if !strings.Contains(noSpace(dialogText(v)), noSpace(long)) {
		t.Errorf(":sync status must show %q:\n%s", long, v)
	}
	for _, line := range strings.Split(v, "\n") {
		if w := ansi.StringWidth(line); w > 80 {
			t.Fatalf("a row %d columns wide does not fit the 80-column window:\n%s", w, v)
		}
	}
}
