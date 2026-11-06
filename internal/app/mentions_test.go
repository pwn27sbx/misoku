package app_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// mentionNotes is a workspace around the note "Diseño": Linked links to it
// and names it, Ref only names it (twice, once in another case), Zed names
// nothing. The explorer lists them in that order and has the focus.
func mentionNotes(t *testing.T) *harness {
	t.Helper()
	return relinkNotes(t,
		"Diseño", "# Diseño\n\nbody",
		"Linked", "see [[Diseño]] and Diseño",
		"Ref", "hablando de diseño y Diseño",
		"Zed", "nothing here",
	)
}

// previewSection returns the preview text from the "Unlinked mentions"
// heading on, or "" when there is none.
func (h *harness) previewSection() string {
	h.t.Helper()
	v := h.view()
	i := strings.Index(v, "Unlinked mentions")
	if i < 0 {
		return ""
	}
	return v[i:]
}

// touchLater rewrites a note on disk as another program would, with a
// modification time the workspace check can't miss.
func touchLater(t *testing.T, root, rel, text string) {
	t.Helper()
	writeOutside(t, root, rel, text)
	at := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(root, filepath.FromSlash(rel)), at, at); err != nil {
		t.Fatal(err)
	}
}

// S8: the section lists the notes that name this one without linking it;
// the note itself, notes already linking it and templates are left out.
func TestUnlinkedMentionsListTheRightNotes(t *testing.T) {
	store := newTestStore(t)
	for _, n := range [][3]string{
		{"", "Diseño", "# Diseño\n\nDiseño body"},
		{"", "Linked", "see [[Diseño]] and Diseño"},
		{"", "Ref", "hablando de diseño y Diseño"},
		{"Plantillas", "Tpl", "Diseño template"},
	} {
		if _, err := store.SaveAs(n[0], n[1], n[2]); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\\'}})
	h.keys("diseño")
	h.key(tea.KeyEnter)
	sec := h.previewSection()
	if !strings.Contains(sec, "Unlinked mentions in 1 note") || !strings.Contains(sec, "Ref · 2 mentions") {
		t.Fatalf("expected Ref under Unlinked mentions:\n%s", h.view())
	}
	for _, other := range []string{"Linked", "Tpl", "Diseño ·"} {
		if strings.Contains(sec, other) {
			t.Fatalf("%q must not be listed:\n%s", other, sec)
		}
	}
	if !strings.Contains(h.view(), "Linked from 1 note") {
		t.Fatalf("the backlinks stay:\n%s", h.view())
	}
}

// S8: the link button asks first, then rewrites every mention in that note
// on disk; the note moves from the mentions to the backlinks.
func TestLinkingMentionsRewritesTheNote(t *testing.T) {
	h := mentionNotes(t)
	h.openRow(0)
	h.clickPreview("link ]")
	if v := h.view(); !strings.Contains(v, "Link 2 mentions in Ref?") {
		t.Fatalf("expected the confirm dialog:\n%s", v)
	}
	if got := h.disk("Ref.md"); got != "hablando de diseño y Diseño" {
		t.Fatalf("nothing is written before the answer, Ref.md = %q", got)
	}
	h.keys("y")
	if got := h.disk("Ref.md"); got != "hablando de [[Diseño|diseño]] y [[Diseño]]" {
		t.Fatalf("Ref.md = %q", got)
	}
	v := h.view()
	if !strings.Contains(v, "Linked from 2 notes") || h.previewSection() != "" {
		t.Fatalf("Ref must move from the mentions to the backlinks:\n%s", v)
	}
	if !strings.Contains(v, "Linked 2 mentions in Ref") {
		t.Fatalf("expected the status:\n%s", v)
	}
}

// S8: No leaves the note as written.
func TestLinkingMentionsNoKeepsTheNote(t *testing.T) {
	h := mentionNotes(t)
	h.openRow(0)
	h.clickPreview("link ]")
	h.keys("n")
	if got := h.disk("Ref.md"); got != "hablando de diseño y Diseño" {
		t.Fatalf("Ref.md = %q", got)
	}
}

// S8: a row's name opens the note.
func TestUnlinkedMentionRowOpensTheNote(t *testing.T) {
	h := mentionNotes(t)
	h.openRow(0)
	h.clickPreview("Ref ·")
	if got := h.editor(); got != "hablando de diseño y Diseño" {
		t.Fatalf("expected Ref open, editor holds %q", got)
	}
}

// S8: a note changed on disk since the scan is left as it is, and said so.
func TestLinkingMentionsSkipsANoteChangedSinceTheScan(t *testing.T) {
	h := mentionNotes(t)
	h.openRow(0)
	h.clickPreview("link ]")
	writeOutside(t, h.store.Root(), "Ref.md", "changed: Diseño")
	h.keys("y")
	if got := h.disk("Ref.md"); got != "changed: Diseño" {
		t.Fatalf("a changed note must be left alone, Ref.md = %q", got)
	}
	if v := h.view(); !strings.Contains(v, "Ref.md changed since") {
		t.Fatalf("expected the skip to be reported:\n%s", v)
	}
}

// S8: an open tab of the note with unsaved edits is refused, the file and
// the tab untouched.
func TestLinkingMentionsRefusesADirtyTab(t *testing.T) {
	h := mentionNotes(t)
	h.openRow(2) // Ref
	h.keys("A mine")
	h.key(tea.KeyEsc)
	h.keys(" e")
	h.openRow(0) // Diseño, active
	h.clickPreview("link ]")
	if v := h.view(); strings.Contains(v, "Link 2 mentions") || !strings.Contains(v, "Ref.md has unsaved changes") {
		t.Fatalf("expected a refusal and no dialog:\n%s", v)
	}
	if got := h.disk("Ref.md"); got != "hablando de diseño y Diseño" {
		t.Fatalf("Ref.md = %q", got)
	}
	h.keys("[")
	if got := h.editor(); got != "hablando de diseño y Diseño mine" {
		t.Fatalf("the dirty tab keeps its edit, editor holds %q", got)
	}
}

// S8: an open clean tab of the note shows the linked text, and stays clean.
func TestLinkingMentionsReloadsACleanTab(t *testing.T) {
	h := mentionNotes(t)
	h.openRow(2) // Ref
	h.keys(" e")
	h.openRow(0) // Diseño, active
	h.clickPreview("link ]")
	h.keys("y")
	h.keys("[")
	if got := h.editor(); got != "hablando de [[Diseño|diseño]] y [[Diseño]]" {
		t.Fatalf("the clean tab must show the rewrite, editor holds %q", got)
	}
	if h.tabDirty("Ref.md") {
		t.Fatalf("the reloaded tab stays clean: %q", h.tabline())
	}
}

// sendHolding is send, except that messages whose type is named hold are
// kept back and returned instead of handled.
func (h *harness) sendHolding(hold string, msg tea.Msg) []tea.Msg {
	h.t.Helper()
	var held []tea.Msg
	var cmd tea.Cmd
	h.m, cmd = h.m.Update(msg)
	queue := []tea.Cmd{cmd}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		switch out := runCmd(c).(type) {
		case nil:
		case tea.BatchMsg:
			queue = append(queue, out...)
		default:
			name := fmt.Sprintf("%T", out)
			if strings.Contains(name, "cursor.") || strings.Contains(name, "Blink") {
				continue
			}
			if name == hold {
				held = append(held, out)
				continue
			}
			var next tea.Cmd
			h.m, next = h.m.Update(out)
			queue = append(queue, next)
		}
	}
	return held
}

// S8: a scan that a newer one replaced is dropped when it arrives late.
func TestUnlinkedMentionsDropAnOlderScan(t *testing.T) {
	h := mentionNotes(t)
	h.openRow(2) // Ref
	h.keys(" e")
	h.openRow(0) // Diseño, active
	h.keys("[")  // Ref
	older := h.sendHolding("app.mentionsScannedMsg", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	if len(older) != 1 {
		t.Fatalf("showing Diseño again must scan once, held %d", len(older))
	}
	touchLater(t, h.store.Root(), "Ref.md", "no mention any more")
	newer := h.sendHolding("app.mentionsScannedMsg", app.CheckWorkspaceMsg{})
	if len(newer) != 1 {
		t.Fatalf("a change on disk must scan again, held %d", len(newer))
	}
	h.send(newer[0], older[0])
	if sec := h.previewSection(); sec != "" {
		t.Fatalf("the older scan must be dropped:\n%s", h.view())
	}
}
