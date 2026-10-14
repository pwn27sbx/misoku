package app_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

func explorerVisible(v string) bool { return strings.Contains(v, "Find note") }

// openNote opens the only note and leaves the editor in NORMAL mode.
func openNote(t *testing.T, opts ...app.Option) *harness {
	t.Helper()
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "hello"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, opts...)
	h.keys("1")
	h.key(tea.KeyEsc)
	return h
}

func TestToggleExplorerCanMoveToALeaderKey(t *testing.T) {
	h := openNote(t, app.WithKeys(map[string][]string{"toggle_explorer": {"space e"}}))
	// Same as Ctrl+B by default: the first press focuses the explorer, the
	// next one hides it, the one after shows it again.
	h.keys(" e")
	if v := h.view(); !explorerVisible(v) || !strings.Contains(v, "EXPLORER") {
		t.Fatalf("space e should focus the explorer:\n%s", v)
	}
	h.keys(" e")
	if explorerVisible(h.view()) {
		t.Fatalf("space e again should hide it:\n%s", h.view())
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlB})
	if explorerVisible(h.view()) {
		t.Fatalf("ctrl+b was replaced and must do nothing now:\n%s", h.view())
	}
	h.keys(" e")
	if !explorerVisible(h.view()) {
		t.Fatalf("space e should bring it back:\n%s", h.view())
	}
}

func TestLeaderShortcutsDoNotFireWhileTyping(t *testing.T) {
	h := openNote(t, app.WithKeys(map[string][]string{"toggle_explorer": {"space e"}}))
	h.keys("A e")
	v := h.view()
	if !explorerVisible(v) || !strings.Contains(v, "hello e") {
		t.Fatalf("in INSERT mode space e is text:\n%s", v)
	}
}

func TestCtrlShortcutsStillWorkWhileTyping(t *testing.T) {
	h := openNote(t, app.WithKeys(map[string][]string{"toggle_explorer": {"ctrl+e"}}))
	h.keys("A")
	h.send(tea.KeyMsg{Type: tea.KeyCtrlE})
	if v := h.view(); !strings.Contains(v, "EXPLORER") || strings.Contains(v, "INSERT") {
		t.Fatalf("ctrl+e should focus the explorer even from INSERT mode:\n%s", v)
	}
}

func TestAnUnboundActionDoesNothing(t *testing.T) {
	h := openNote(t, app.WithKeys(map[string][]string{"close_tab": {}}))
	h.keys(" x")
	if tabs := strings.Split(h.view(), "\n")[0]; !strings.Contains(tabs, "doc.md") {
		t.Fatalf("close_tab is unbound, the tab must stay: %q", tabs)
	}
}

func TestExplorerShortcutsCanBeChanged(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "x"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, app.WithKeys(map[string][]string{"explorer.delete": {"X"}}))
	h.keys("e")
	h.keys("d")
	if strings.Contains(h.view(), "Delete doc.md?") {
		t.Fatalf("d no longer deletes:\n%s", h.view())
	}
	h.keys("X")
	if !strings.Contains(h.view(), "Delete doc.md?") {
		t.Fatalf("X should ask to delete:\n%s", h.view())
	}
}

func TestHelpShowsTheActiveShortcuts(t *testing.T) {
	h := openNote(t, app.WithKeys(map[string][]string{"toggle_explorer": {"space e"}}))
	h.keys("?")
	v := h.view()
	line := ""
	for _, l := range strings.Split(v, "\n") {
		if strings.Contains(l, "Toggle explorer") {
			line = l
		}
	}
	if !strings.Contains(line, "SPC e") || strings.Contains(line, "Ctrl+B") {
		t.Fatalf("help should list SPC e for Toggle explorer, got %q", line)
	}
}

func TestConfigMistakesShowInTheStatusBar(t *testing.T) {
	store := newTestStore(t)
	h := newHarness(t, store, app.WithKeys(map[string][]string{"toggle_explorr": {"ctrl+e"}}))
	// Shown when Misoku opens; like Vim messages, the next key clears it.
	if v := h.view(); !strings.Contains(v, `unknown action "toggle_explorr"`) {
		t.Fatalf("a typo in the config should be reported on startup:\n%s", v)
	}
}

func TestUIConfigReachesThePreview(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "# Title\n\n- [-] doing\n"); err != nil {
		t.Fatal(err)
	}
	ui := config.UI{HeadingIcons: []string{"① "}}
	ui.Checkbox.Todo = "󰥔"
	h := newHarness(t, store, app.WithUI(ui))
	h.keys("1")
	v := h.view()
	if !strings.Contains(v, "① Title") || !strings.Contains(v, "󰥔 doing") {
		t.Fatalf("the preview should use the configured glyphs:\n%s", v)
	}
	// Levels not given keep their default icon.
	h.keys("o## Sub")
	h.key(tea.KeyEsc)
	if v := h.view(); !strings.Contains(v, "󰉬 Sub") {
		t.Fatalf("level 2 keeps its default icon:\n%s", v)
	}
}

func TestProjectKeyOpensTheFolderInTheExplorer(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "suelta", "x"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("Alpha", "primera", "y"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("Trabajo", "cliente", "z"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	// Three recent notes take 1–3; the projects Alpha and Trabajo are 4 and 5.
	h.keys("5")
	v := h.view()
	if !strings.Contains(v, "EXPLORER") || !strings.Contains(v, "cliente") || strings.Contains(v, "primera") {
		t.Fatalf("the explorer should open with Trabajo (not Alpha) expanded:\n%s", v)
	}
	h.key(tea.KeyEnter) // on the folder: collapses it, proving it is the selection
	if strings.Contains(h.view(), "cliente") {
		t.Fatalf("Trabajo should be the selected row:\n%s", h.view())
	}
}
