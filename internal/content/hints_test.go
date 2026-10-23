package content_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/keymap"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

func TestEmptyStateHintShowsTheEffectiveKeys(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(100, 20)
	if v := c.View(); !strings.Contains(v, "Ctrl+N  new note") || !strings.Contains(v, "Ctrl+P  search") || strings.Contains(v, "/  search") {
		t.Fatalf("default hint should name Ctrl+N and Ctrl+P (/ is the block menu):\n%s", v)
	}

	km, _ := keymap.New(map[string][]string{"new_note": {"ctrl+o"}, "find_note": {"space f"}})
	c.SetKeys(km)
	v := c.View()
	if !strings.Contains(v, "Ctrl+O  new note") || !strings.Contains(v, "SPC f  search") || strings.Contains(v, "Ctrl+P") {
		t.Fatalf("hint should follow the configured keys:\n%s", v)
	}
}

func TestCtrlBDoesNotPageUpTheEditor(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 10)
	var b strings.Builder
	for i := 1; i <= 100; i++ {
		fmt.Fprintf(&b, "row %d\n", i)
	}
	c.SetPage(core.Page{ID: "a.md", Content: b.String()})
	c.GotoLine(50)
	start, _ := c.CursorPosition()

	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
	if line, _ := c.CursorPosition(); line != start {
		t.Fatalf("ctrl+b belongs to toggle_explorer and must not move the cursor: %d → %d", start, line)
	}
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	if line, _ := c.CursorPosition(); line >= start {
		t.Fatalf("pgup should still page up: %d → %d", start, line)
	}
}
