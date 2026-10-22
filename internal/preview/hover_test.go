package preview_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"
)

func TestHoveringFollowsTheMouse(t *testing.T) {
	p := preview.New(theme.DefaultTheme())
	p.SetSize(80, 20)
	p.SetPage(core.Page{ID: "p1", Content: "First\n\nSecond"})
	if p.Hovering() {
		t.Fatal("a new preview should not be hovered")
	}
	p, _ = p.Update(tea.MouseMsg{X: 10, Y: 0, Action: tea.MouseActionMotion})
	if !p.Hovering() {
		t.Fatal("motion over a block should hover it")
	}
	p.ClearHover()
	if p.Hovering() {
		t.Fatal("ClearHover should end the hover")
	}
}
