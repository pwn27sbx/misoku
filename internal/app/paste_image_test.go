package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// pastedPNG writes the file a clipboard read would hand over and returns the
// message that carries it.
func pastedPNG(t *testing.T) core.ClipboardImageMsg {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "pasted-2026-11-01_10-00-00.png")
	if err := os.WriteFile(path, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	return core.ClipboardImageMsg{Path: path, MIME: "image/png", Ext: ".png", Bytes: 3}
}

// openWithClipboard opens the only note in insert mode, with the clipboard
// holding the picture msg.
func openWithClipboard(t *testing.T, msg core.ClipboardImageMsg) *harness {
	t.Helper()
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "hola\n"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, app.WithClipboardImage(func(context.Context) (core.ClipboardImageMsg, bool, error) {
		return msg, true, nil
	}))
	h.keys("1")
	h.key(tea.KeyEsc)
	h.keys("i") // insert mode
	return h
}

// A pasted picture is copied next to the note and linked on a line of its own,
// which is what makes the preview draw it instead of an icon.
func TestPastingAPictureSavesItNextToTheNoteAndLinksIt(t *testing.T) {
	msg := pastedPNG(t)
	h := openWithClipboard(t, msg)

	h.send(tea.KeyMsg{Type: tea.KeyCtrlV})
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS}) // write the note so the file holds it

	note, err := h.store.Get("doc.md")
	if err != nil {
		t.Fatal(err)
	}
	link := "assets/pasted-2026-11-01_10-00-00.png"
	if !strings.Contains(note.Content, "]("+link+")") {
		t.Fatalf("the note does not link %q:\n%q", link, note.Content)
	}
	for _, line := range strings.Split(note.Content, "\n") {
		if !strings.Contains(line, link) {
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(line), "![") || strings.TrimSpace(line) != line ||
			!strings.HasSuffix(line, "]("+link+")") {
			t.Fatalf("the link is not alone on its line: %q", line)
		}
	}
	if data, err := os.ReadFile(filepath.Join(h.store.Root(), link)); err != nil || string(data) != "png" {
		t.Fatalf("the picture was not copied: %q, %v", data, err)
	}
	if _, err := os.Stat(msg.Path); !os.IsNotExist(err) {
		t.Fatalf("the temporary file was not removed: %v", err)
	}
}

// The "/" menu entry does the same as the key.
func TestPasteImageFromTheSlashMenu(t *testing.T) {
	msg := pastedPNG(t)
	h := openWithClipboard(t, msg)
	h.key(tea.KeyEsc) // back to normal, as the menu is opened from there

	h.send(core.SlashActionMsg{Action: "media:paste"})
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})

	note, err := h.store.Get("doc.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note.Content, "assets/pasted-2026-11-01_10-00-00.png") {
		t.Fatalf("the menu entry did not paste the picture:\n%q", note.Content)
	}
}

// Without an open note nothing is written and the status line says so.
func TestPastingWithoutANoteSaysToOpenOne(t *testing.T) {
	msg := pastedPNG(t)
	h := newHarness(t, newTestStore(t), app.WithClipboardImage(func(context.Context) (core.ClipboardImageMsg, bool, error) {
		return msg, true, nil
	}))

	h.send(core.ClipboardImageMsg{Path: msg.Path, Ext: msg.Ext})

	if v := h.view(); !strings.Contains(v, "Open a note first") {
		t.Fatalf("expected the status line to ask for a note:\n%s", v)
	}
	if _, err := os.Stat(msg.Path); err != nil {
		t.Fatalf("the temporary file should still exist for a later attempt: %v", err)
	}
}
