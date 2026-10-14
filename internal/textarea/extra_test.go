package textarea_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/textarea"

	tea "github.com/charmbracelet/bubbletea"
)

func key(t *testing.T, m textarea.Model, s string) textarea.Model {
	t.Helper()
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
	_ = cmd
	return m
}

func keyName(t *testing.T, m textarea.Model, name string, kp tea.KeyType) textarea.Model {
	t.Helper()
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: kp})
	_ = cmd
	return m
}

func seeded() textarea.Model {
	m := textarea.New()
	m.SetValue("alpha beta gamma\nsecond line\nthird line")
	m.SetWidth(40)
	m.SetHeight(5)
	m.GotoTop()
	return m
}

func TestCursorPosition(t *testing.T) {
	m := seeded()
	l, c := m.CursorPosition()
	if l != 1 || c != 1 {
		t.Errorf("CursorPosition = (%d,%d), want (1,1)", l, c)
	}
}

func TestScrollByAndYOffset(t *testing.T) {
	m := seeded()
	m.ScrollBy(1)
	if m.YOffset() != 1 {
		t.Errorf("YOffset = %d, want 1", m.YOffset())
	}
	m.ScrollBy(-5)
	if m.YOffset() != 0 {
		t.Errorf("YOffset = %d, want 0", m.YOffset())
	}
}

func TestMoveCursorByAndGotoLine(t *testing.T) {
	m := seeded()
	m.MoveCursorBy(2)
	if m.Line() != 2 {
		t.Errorf("Line = %d, want 2", m.Line())
	}
	m.MoveCursorBy(-1)
	if m.Line() != 1 {
		t.Errorf("Line = %d, want 1", m.Line())
	}
	m.GotoLine(3)
	if m.Line() != 2 {
		t.Errorf("Line = %d, want 2", m.Line())
	}
	m.GotoLine(99)
	if m.Line() != 2 {
		t.Errorf("clamped Line = %d, want 2", m.Line())
	}
}

func TestCharWordMovement(t *testing.T) {
	m := seeded()
	m.SetRowCol(0, 0)
	m.CharRight()
	if r, c := m.RowCol(); c != 1 {
		t.Errorf("col = %d, want 1 (row %d)", c, r)
	}
	m.CharLeft()
	if _, c := m.RowCol(); c != 0 {
		t.Errorf("col = %d, want 0", c)
	}
	m.WordForward()
	if _, c := m.RowCol(); c == 0 {
		t.Error("WordForward should move the cursor")
	}
	m.WordBackward()
	if _, c := m.RowCol(); c != 0 {
		t.Errorf("WordBackward col = %d, want 0", c)
	}
}

func TestRowColAndLineBeforeCursor(t *testing.T) {
	m := seeded()
	m.SetRowCol(0, 5)
	if r, c := m.RowCol(); r != 0 || c != 5 {
		t.Errorf("RowCol = %d,%d", r, c)
	}
	if got := m.LineBeforeCursor(); got != "alpha" {
		t.Errorf("LineBeforeCursor = %q", got)
	}
}

func TestCurrentLineDeleteCharForward(t *testing.T) {
	m := seeded()
	if m.CurrentLine() != "alpha beta gamma" {
		t.Errorf("CurrentLine = %q", m.CurrentLine())
	}
	m.SetRowCol(0, 0)
	m.DeleteCharForward()
	if m.CurrentLine() != "lpha beta gamma" {
		t.Errorf("after x: %q", m.CurrentLine())
	}
}

func TestDeleteLine(t *testing.T) {
	m := seeded()
	m.GotoLine(2)
	m.DeleteLine()
	if m.Value() != "alpha beta gamma\nthird line" {
		t.Errorf("Value = %q", m.Value())
	}
	m.DeleteLine()
	m.DeleteLine()
	if m.Value() != "" && m.LineCount() != 1 {
		t.Errorf("Value = %q", m.Value())
	}
}

func TestOpenLineBelowAndAbove(t *testing.T) {
	m := seeded()
	m.GotoLine(1)
	m.OpenLineBelow()
	if m.LineCount() != 4 || m.Line() != 1 {
		t.Errorf("lines=%d line=%d", m.LineCount(), m.Line())
	}
	m.OpenLineAbove()
	if m.LineCount() != 5 || m.Line() != 1 {
		t.Errorf("lines=%d line=%d", m.LineCount(), m.Line())
	}
}

func TestDeleteBeforeAndIndentOutdent(t *testing.T) {
	m := seeded()
	m.SetRowCol(0, 5)
	m.DeleteBefore(3)
	if m.CurrentLine() != "al beta gamma" {
		t.Errorf("after DeleteBefore: %q", m.CurrentLine())
	}
	m.IndentLine(2)
	if !strings.HasPrefix(m.CurrentLine(), "  ") {
		t.Errorf("after IndentLine: %q", m.CurrentLine())
	}
	m.OutdentLine(2)
	if strings.HasPrefix(m.CurrentLine(), " ") {
		t.Errorf("after OutdentLine: %q", m.CurrentLine())
	}
}

func TestCursorScreen(t *testing.T) {
	m := seeded()
	m.SetRowCol(0, 0)
	m.EnsureVisible()
	x, y, ok := m.CursorScreen()
	if !ok || y != 0 {
		t.Errorf("CursorScreen = (%d,%d,%v)", x, y, ok)
	}
}

func TestClickAt(t *testing.T) {
	m := seeded()
	m.ClickAt(7, 1)
	if m.Line() != 1 {
		t.Errorf("after ClickAt line=%d, want 1", m.Line())
	}
}

func TestSelectionLifecycle(t *testing.T) {
	m := seeded()
	m.SetRowCol(0, 0)
	m.StartSelection()
	if !m.HasSelection() {
		t.Fatal("expected selection")
	}
	m.CharRight()
	m.CharRight()
	got := m.SelectedText()
	if got != "alp" {
		t.Errorf("SelectedText = %q", got)
	}
	m.ClearSelection()
	if m.HasSelection() || m.SelectedText() != "" {
		t.Error("expected cleared selection")
	}
}

func TestSetAnchorAndDeleteSelection(t *testing.T) {
	m := seeded()
	m.SetAnchor(0, 0)
	m.SetRowCol(0, 4)
	m.DeleteSelection()
	if got := m.CurrentLine(); got != " beta gamma" {
		t.Errorf("after DeleteSelection: %q", got)
	}
}

func TestLinewiseSelectionAndDelete(t *testing.T) {
	m := seeded()
	m.SelectLinewise = true
	m.GotoLine(1)
	m.StartSelection()
	m.GotoLine(2)
	got := m.SelectedText()
	if !strings.Contains(got, "alpha") || !strings.Contains(got, "second") {
		t.Errorf("linewise SelectedText = %q", got)
	}
	m.DeleteSelection()
	if m.LineCount() != 1 {
		t.Errorf("LineCount = %d, want 1", m.LineCount())
	}
}

func TestEnsureVisibleLongDoc(t *testing.T) {
	m := textarea.New()
	var b strings.Builder
	for i := 0; i < 50; i++ {
		b.WriteString("line\n")
	}
	m.SetValue(b.String())
	m.SetHeight(5)
	m.GotoBottom()
	m.EnsureVisible()
	if m.YOffset() == 0 {
		t.Error("expected scrolled view")
	}
}

func TestKeyDrivenEditing(t *testing.T) {
	m := textarea.New()
	m.SetValue("foo bar baz")
	m.SetWidth(40)
	m.SetHeight(5)
	m.Focus()
	m.GotoLine(1)
	m.CursorEnd()

	// alt+backspace deletes word backward
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace, Alt: true})
	if got := m.Value(); got != "foo bar " {
		t.Errorf("after alt+backspace: %q", got)
	}
	// ctrl+u deletes before cursor
	m.SetRowCol(0, 4)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	if got := m.Value(); got != "bar " {
		t.Errorf("after ctrl+u: %q", got)
	}
	// ctrl+k deletes after cursor
	m.SetRowCol(0, 1)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	if got := m.Value(); got != "b" {
		t.Errorf("after ctrl+k: %q", got)
	}
}

func TestWordCaseKeys(t *testing.T) {
	m := textarea.New()
	m.SetValue("hello world")
	m.SetWidth(40)
	m.SetHeight(5)
	m.Focus()
	m.GotoTop()
	m.SetRowCol(0, 0)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u"), Alt: true})
	if !strings.Contains(m.Value(), "HELLO") {
		t.Errorf("after alt+u: %q", m.Value())
	}
	m2 := textarea.New()
	m2.SetValue("HELLO WORLD")
	m2.SetWidth(40)
	m2.SetHeight(5)
	m2.Focus()
	m2.GotoTop()
	m2, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l"), Alt: true})
	if !strings.Contains(m2.Value(), "hello") {
		t.Errorf("after alt+l: %q", m2.Value())
	}
}

func TestLengthAndCursorEnd(t *testing.T) {
	m := seeded()
	if m.Length() != len([]rune(m.Value())) {
		t.Errorf("Length = %d", m.Length())
	}
	m.GotoTop()
	m.CursorEnd()
	if got := m.LineBeforeCursor(); got != "alpha beta gamma" {
		t.Errorf("LineBeforeCursor = %q", got)
	}
}

func TestMergeLinesViaEnterAndBackspace(t *testing.T) {
	m := textarea.New()
	m.SetValue("ab\ncd")
	m.SetWidth(40)
	m.SetHeight(5)
	m.Focus()
	m.GotoLine(2)
	m.SetCursor(0)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.Value() != "abcd" {
		t.Errorf("after backspace merge line: %q", m.Value())
	}
	m.SetRowCol(0, 2)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.Value() != "ab\ncd" {
		t.Errorf("after enter split: %q", m.Value())
	}
}

func TestInsertRuneExported(t *testing.T) {
	m := textarea.New()
	m.SetValue("ac")
	m.SetCursor(1)
	m.InsertRune('b')
	if m.Value() != "abc" {
		t.Errorf("after InsertRune: %q", m.Value())
	}
}

func TestViewWithSelectionUsesPaint(t *testing.T) {
	m := seeded()
	m.SetRowCol(0, 0)
	m.StartSelection()
	m.SetRowCol(0, 5)
	v := m.View()
	if !strings.Contains(v, "alpha") {
		t.Error("view should still contain text")
	}
}

func TestTransposeAndDeleteKeys(t *testing.T) {
	m := textarea.New()
	m.SetValue("abc")
	m.SetWidth(40)
	m.SetHeight(5)
	m.Focus()
	m.GotoTop()
	m.SetRowCol(0, 2)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	if m.Value() != "acb" {
		t.Errorf("after ctrl+t: %q", m.Value())
	}
	m.SetRowCol(0, 1)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	if m.Value() != "ab" {
		t.Errorf("after delete: %q", m.Value())
	}
}
