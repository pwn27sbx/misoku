package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

const noPreviousNote = "No previous note in this tab"

var ctrlO = tea.KeyMsg{Type: tea.KeyCtrlO}

// chainNotes is a workspace where Alpha links to Bravo and Bravo to Charlie,
// each link on line 5.
func chainNotes(t *testing.T) *core.Store {
	t.Helper()
	store := newTestStore(t)
	for _, n := range [][2]string{
		{"Alpha", "# Alpha\n\nintro\n\nto [[Bravo]]"},
		{"Bravo", "# Bravo\n\nintro\n\nto [[Charlie]]"},
		{"Charlie", "# Charlie\n\nlast line"},
		{"Delta", "# Delta\n\nother"},
		{"Echo", "# Echo\n\n[markdown target](Bravo.md)"},
	} {
		if _, err := store.SaveAs("", n[0], n[1]); err != nil {
			t.Fatal(err)
		}
	}
	return store
}

// find opens the note matching query from the finder, in the active tab.
func (h *harness) find(query string) {
	h.t.Helper()
	h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\\'}})
	h.keys(query)
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
}

// openChain opens Alpha in NORMAL mode.
func openChain(t *testing.T, opts ...app.Option) *harness {
	t.Helper()
	h := newHarness(t, chainNotes(t), opts...)
	h.find("alpha")
	if v := h.view(); !strings.Contains(v, "to [[Bravo]]") {
		t.Fatalf("Alpha.md not open:\n%s", v)
	}
	return h
}

// onLink puts the cursor on the [[link]] of line 5.
func (h *harness) onLink() {
	h.t.Helper()
	h.keys("gg" + "jjjj" + "0lllll")
}

// follow presses gd on the [[link]] of line 5.
func (h *harness) follow() {
	h.t.Helper()
	h.onLink()
	h.keys("gd")
}

// wantTabs fails unless every name in open has a tab and no name in closed.
func (h *harness) wantTabs(open, closed []string) {
	h.t.Helper()
	row := h.tabline()
	for _, name := range open {
		if !strings.Contains(row, name) {
			h.t.Fatalf("expected a tab %q in %q", name, row)
		}
	}
	for _, name := range closed {
		if strings.Contains(row, name) {
			h.t.Fatalf("expected no tab %q in %q", name, row)
		}
	}
}

// wantShown fails unless the frame contains text.
func (h *harness) wantShown(text string) {
	h.t.Helper()
	if v := h.view(); !strings.Contains(v, text) {
		h.t.Fatalf("expected %q on screen:\n%s", text, v)
	}
}

// S1: Enter on a [[link]] opens the note in a new tab.
func TestEnterOpensTheLinkedNoteInANewTab(t *testing.T) {
	h := openChain(t)
	h.onLink()
	h.key(tea.KeyEnter)
	h.wantTabs([]string{"Alpha", "Bravo"}, nil)
	h.wantShown("to [[Charlie]]")
}

// S1: a click on a [[link]] or a Markdown link to a note opens it in a new tab.
func TestClickOpensTheLinkedNoteInANewTab(t *testing.T) {
	tests := []struct{ name, from, click, fromTab, opened string }{
		{"wiki link", "alpha", "Bravo", "Alpha", "to [[Charlie]]"},
		{"markdown link", "echo", "markdown target", "Echo", "to [[Charlie]]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, chainNotes(t))
			h.find(tt.from)
			h.clickPreview(tt.click)
			h.wantTabs([]string{tt.fromTab, "Bravo"}, nil)
			h.wantShown(tt.opened)
		})
	}
}

// S1: following a link to a note that is already open focuses its tab.
func TestFollowingALinkToAnOpenNoteFocusesItsTab(t *testing.T) {
	follow := map[string]func(h *harness){
		"enter": func(h *harness) { h.onLink(); h.key(tea.KeyEnter) },
		"click": func(h *harness) { h.clickPreview("Bravo") },
	}
	for name, press := range follow {
		t.Run(name, func(t *testing.T) {
			h := openChain(t)
			h.onLink()
			h.key(tea.KeyEnter)
			h.keys("[") // back to Alpha's tab
			h.wantShown("to [[Bravo]]")
			press(h)
			h.wantShown("to [[Charlie]]")
			if n := strings.Count(h.tabline(), "Bravo"); n != 1 {
				t.Fatalf("Bravo should keep its one tab, got %d in %q", n, h.tabline())
			}
		})
	}
}

// S1: a link to a heading still jumps there once its note is open.
func TestEnterOnAHeadingLinkJumpsAfterOpening(t *testing.T) {
	h := openStart(t, linkedNotes(t))
	h.key(tea.KeyEsc)
	h.keys("gg0" + strings.Repeat("l", 28)) // onto "[[AWS Notes#EC2 Compute|compute]]"
	h.key(tea.KeyEnter)
	h.wantTabs([]string{"Start", "AWS Notes"}, nil)
	h.wantShown("Ln 63")
}

// S1: a click on a link leaves the focus on the preview.
func TestClickOnALinkKeepsThePreviewFocused(t *testing.T) {
	h := openChain(t)
	h.clickPreview("Bravo")
	h.keys("ix") // would type into the note if the editor had the focus
	if h.tabDirty("Bravo.md") {
		t.Fatalf("the editor took the focus:\n%s", h.view())
	}
}

// S2: gd replaces the note in the same tab, on either kind of link.
func TestGdOpensTheNoteInTheSameTab(t *testing.T) {
	tests := []struct{ name, from, fromTab, move string }{
		{"wiki link", "alpha", "Alpha", "gg" + "jjjj" + "0lllll"},
		{"markdown link", "echo", "Echo", "gg" + "jj" + "0l"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, chainNotes(t))
			h.find(tt.from)
			h.keys(tt.move)
			h.keys("gd")
			h.wantTabs([]string{"Bravo"}, []string{tt.fromTab})
			h.wantShown("to [[Charlie]]")
		})
	}
}

// S2: gd never replaces a tab with unsaved changes.
func TestGdOnAnUnsavedTabOpensNextToIt(t *testing.T) {
	h := openChain(t)
	h.keys("Gox")
	h.key(tea.KeyEsc)
	h.follow()
	h.wantTabs([]string{"Alpha", "Bravo"}, nil)
	h.wantShown("has unsaved changes")
}

// S2: gd on a missing note starts it.
func TestGdOnAMissingNoteStartsADraft(t *testing.T) {
	h := openStart(t, linkedNotes(t))
	h.key(tea.KeyEsc)
	h.keys("Gkk0" + strings.Repeat("l", 13)) // onto "[[Fresh Idea]]"
	h.keys("gd")
	h.wantTabs([]string{"Fresh Idea"}, nil)
	h.wantShown("save to create it")
}

// S3: ctrl+o returns to the previous note, at the line its cursor was on.
func TestCtrlOReturnsToThePreviousNoteAtItsLine(t *testing.T) {
	h := openChain(t)
	h.follow()
	h.wantShown("to [[Charlie]]")
	h.send(ctrlO)
	h.wantTabs([]string{"Alpha"}, []string{"Bravo"})
	h.wantShown("to [[Bravo]]")
	h.wantShown("Ln 5")
}

// S3: each press walks one note further back.
func TestCtrlOWalksFurtherBack(t *testing.T) {
	h := openChain(t)
	h.follow()
	h.follow()
	h.wantShown("# Charlie")
	h.send(ctrlO)
	h.wantShown("to [[Charlie]]")
	h.send(ctrlO)
	h.wantShown("to [[Bravo]]")
}

// S3: with nothing to go back to, the status says so and nothing changes.
func TestCtrlOWithNothingToGoBackToSaysSo(t *testing.T) {
	h := openChain(t)
	h.send(ctrlO)
	h.wantShown(noPreviousNote)
	h.wantTabs([]string{"Alpha"}, nil)
	h.wantShown("to [[Bravo]]")
}

// S3, S4: each tab walks back its own notes.
func TestCtrlOIsPerTab(t *testing.T) {
	h := openChain(t)
	h.follow() // Alpha's tab now shows Bravo and can go back
	h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\\'}})
	h.keys("delta")
	h.send(tea.KeyMsg{Type: tea.KeyCtrlT})
	h.key(tea.KeyEsc)
	h.send(ctrlO)
	h.wantShown(noPreviousNote)
	h.wantShown("# Delta")
	h.keys("[")
	h.send(ctrlO)
	h.wantShown("to [[Bravo]]")
}

// S4: going back records nothing, so the walk ends.
func TestCtrlODoesNotRecordItself(t *testing.T) {
	h := openChain(t)
	h.follow()
	h.send(ctrlO)
	h.send(ctrlO)
	h.wantShown(noPreviousNote)
	h.wantShown("to [[Bravo]]")
}

// S4: the finder's Enter and :e record the note they replace.
func TestReplacingTheNoteFromTheFinderOrTheCommandLineRecords(t *testing.T) {
	replace := map[string]func(h *harness){
		"finder": func(h *harness) { h.find("delta") },
		"ex":     func(h *harness) { h.ex("e Delta") },
	}
	for name, open := range replace {
		t.Run(name, func(t *testing.T) {
			h := openChain(t)
			open(h)
			h.wantTabs([]string{"Delta"}, []string{"Alpha"})
			h.send(ctrlO)
			h.wantTabs([]string{"Alpha"}, []string{"Delta"})
		})
	}
}

// S4: a tab with unsaved changes is never replaced by going back.
func TestCtrlOOnAnUnsavedTabOpensTheNoteNextToIt(t *testing.T) {
	h := openChain(t)
	h.follow()
	h.keys("Gox")
	h.key(tea.KeyEsc)
	h.send(ctrlO)
	h.wantTabs([]string{"Alpha", "Bravo"}, nil)
	h.wantShown("has unsaved changes")
	if !h.tabDirty("Bravo.md") {
		t.Fatalf("Bravo should keep its edits:\n%s", h.view())
	}
}

// S4: an entry follows its note through a rename.
func TestCtrlOFollowsARenamedNote(t *testing.T) {
	h := openChain(t)
	h.follow()
	h.keys(" e") // the explorer
	h.renameRow(0, "Alpha", "Zulu")
	h.send(ctrlO)
	h.wantTabs([]string{"Zulu"}, []string{"Bravo"})
	h.wantShown("to [[Bravo]]")
}

// S4: an entry whose note is gone is skipped.
func TestCtrlOSkipsADeletedNote(t *testing.T) {
	h := openChain(t)
	h.follow()
	h.follow()
	h.wantShown("# Charlie")
	if err := os.Remove(filepath.Join(h.store.Root(), "Bravo.md")); err != nil {
		t.Fatal(err)
	}
	h.send(ctrlO)
	h.wantTabs([]string{"Alpha"}, []string{"Charlie"})
	h.wantShown("to [[Bravo]]")
}

// S3: inside INSERT ctrl+o does not go back.
func TestCtrlOInsideInsertDoesNotGoBack(t *testing.T) {
	h := openChain(t)
	h.follow()
	h.keys("i")
	h.send(ctrlO)
	h.wantTabs([]string{"Bravo"}, []string{"Alpha"})
	h.wantShown("to [[Charlie]]")
}

// S3: go_back is configurable like the other actions.
func TestGoBackCanBeRebound(t *testing.T) {
	h := openChain(t, app.WithKeys(map[string][]string{"go_back": {"ctrl+y"}}))
	h.follow()
	h.send(ctrlO)
	h.wantShown("to [[Charlie]]")
	h.send(tea.KeyMsg{Type: tea.KeyCtrlY})
	h.wantShown("to [[Bravo]]")
}

// S5: the finder's Enter still shows the note in the same tab.
func TestFinderEnterStillReplacesTheTab(t *testing.T) {
	h := openChain(t)
	h.find("delta")
	h.wantTabs([]string{"Delta"}, []string{"Alpha"})
}
