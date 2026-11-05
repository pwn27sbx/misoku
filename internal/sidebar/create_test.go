package sidebar_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/sidebar"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

func newEntrySidebar() sidebar.Model {
	sb := sidebar.New(theme.DefaultTheme())
	sb.SetSize(30, 20)
	sb.SetPages([]core.Page{
		{ID: "FLM", Title: "FLM", IsFolder: true},
		{ID: "FLM/x.md", Title: "x", ParentID: "FLM"},
		{ID: "zeta.md", Title: "zeta"},
	})
	sb.SetFocused(true)
	return sb
}

func typeRunes(sb sidebar.Model, s string) sidebar.Model {
	for _, r := range s {
		sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return sb
}

func TestNewEntryLineOpensUnderTheContextFolder(t *testing.T) {
	sb := typeRunes(newEntrySidebar(), "n")
	if !sb.IsBusy() {
		t.Fatal("n should open the name line and capture keys")
	}
	sb = typeRunes(sb, "test/")
	rows := strings.Split(sb.View(), "\n")
	flm, line := -1, -1
	for i, r := range rows {
		if strings.Contains(r, "FLM") {
			flm = i
		}
		if strings.Contains(r, "test/") {
			line = i
		}
	}
	if flm < 0 || line != flm+1 {
		t.Fatalf("the name line belongs right under FLM:\n%s", sb.View())
	}

	sb, cmd := sb.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if sb.IsBusy() || cmd == nil {
		t.Fatal("Enter should close the line and ask to create")
	}
	req, ok := cmd().(sidebar.CreateRequestMsg)
	if !ok || req.ParentID != "FLM" || req.Name != "test/" {
		t.Fatalf("got %#v", req)
	}
}

func TestNewEntryLineEscCancelsAndEmptyEnterStartsADraft(t *testing.T) {
	sb := typeRunes(newEntrySidebar(), "nabc")
	sb, cmd := sb.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if sb.IsBusy() || cmd != nil {
		t.Fatal("Esc should close the line and do nothing")
	}

	sb = typeRunes(sb, "n")
	_, cmd = sb.Update(tea.KeyMsg{Type: tea.KeyEnter})
	created, ok := cmd().(core.PageCreatedMsg)
	if !ok || created.Page.ParentID != "FLM" {
		t.Fatalf("an empty name starts a draft in the folder, as n did: %#v", cmd())
	}
}
