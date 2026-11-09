package app_test

import (
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// editStore is a vault with a note in it, a folder note, and the note the
// editor starts on.
func editStore(t *testing.T) *harness {
	t.Helper()
	store := newTestStore(t)
	for _, n := range [][3]string{
		{"", "Start", "start"},
		{"", "Existing", "the note"},
		{"Projects", "AWS", "cloud"},
	} {
		if _, err := store.SaveAs(n[0], n[1], n[2]); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("\\")
	h.keys("start")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	return h
}

// S3: :e name opens an existing note, resolved as a [[link]] would be.
func TestEditCommandOpensTheNote(t *testing.T) {
	h := editStore(t)

	h.keys(":e Existing")
	h.key(tea.KeyEnter)
	if got := h.editor(); got != "the note" {
		t.Fatalf(":e Existing should open it, editor holds %q", got)
	}

	h.keys(":e Projects/AWS")
	h.key(tea.KeyEnter)
	if got := h.editor(); got != "cloud" {
		t.Fatalf(":e Projects/AWS should resolve by path, editor holds %q", got)
	}
}

// S3: :e a name that does not exist starts it as following a link does.
func TestEditCommandStartsAMissingNote(t *testing.T) {
	h := editStore(t)

	h.keys(":e New one")
	h.key(tea.KeyEnter)
	if got := h.editor(); got != "" {
		t.Fatalf("a new note starts empty, editor holds %q", got)
	}
	if got, want := app.StatusText(h.m), `New note "New one": save to create it`; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
}

// S3: bare :e still opens an empty tab.
func TestEditCommandAloneIsUnchanged(t *testing.T) {
	h := editStore(t)
	tabs := len(app.OpenTabs(h.m))

	h.keys(":e")
	h.key(tea.KeyEnter)
	if got := len(app.OpenTabs(h.m)); got != tabs+1 {
		t.Fatalf("bare :e should open a new tab, %d -> %d tabs", tabs, got)
	}
	if got := h.editor(); got != "" {
		t.Fatalf("a new tab starts empty, editor holds %q", got)
	}
}
