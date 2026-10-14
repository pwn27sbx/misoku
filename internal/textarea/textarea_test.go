package textarea_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/textarea"
)

func TestNewModelDefaults(t *testing.T) {
	m := textarea.New()
	if m.Value() != "" {
		t.Errorf("expected empty value, got %q", m.Value())
	}
	if m.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", m.LineCount())
	}
	if m.Focused() {
		t.Error("expected new model to be unfocused")
	}
}

func TestSetValueAndLineCount(t *testing.T) {
	m := textarea.New()
	m.SetValue("one\ntwo\nthree")
	if got := m.Value(); got != "one\ntwo\nthree" {
		t.Errorf("Value() = %q", got)
	}
	if m.LineCount() != 3 {
		t.Errorf("LineCount() = %d, want 3", m.LineCount())
	}
	m.Reset()
	if m.Value() != "" || m.LineCount() != 1 {
		t.Errorf("after Reset: value %q lines %d", m.Value(), m.LineCount())
	}
}

func TestInsertString(t *testing.T) {
	m := textarea.New()
	m.SetValue("ac")
	m.SetCursor(1)
	m.InsertString("b")
	if got := m.Value(); got != "abc" {
		t.Errorf("Value() = %q, want %q", got, "abc")
	}
}

func TestCursorMovement(t *testing.T) {
	m := textarea.New()
	m.SetValue("top\nmiddle\nbottom")
	m.GotoTop()
	m.CursorDown()
	if m.Line() != 1 {
		t.Errorf("Line() = %d, want 1", m.Line())
	}
	m.CursorUp()
	if m.Line() != 0 {
		t.Errorf("Line() = %d, want 0", m.Line())
	}
	m.CursorStart()
	if info := m.LineInfo(); info.CharOffset != 0 {
		t.Errorf("CharOffset = %d, want 0", info.CharOffset)
	}
}

func TestFocusBlur(t *testing.T) {
	m := textarea.New()
	m.Focus()
	if !m.Focused() {
		t.Error("expected focused after Focus()")
	}
	m.Blur()
	if m.Focused() {
		t.Error("expected unfocused after Blur()")
	}
}

func TestViewContainsContent(t *testing.T) {
	m := textarea.New()
	m.SetValue("hello world")
	m.SetWidth(40)
	m.SetHeight(5)
	v := m.View()
	if !strings.Contains(v, "hello world") {
		t.Errorf("expected view to contain buffer text, got:\n%s", v)
	}
}

func TestGotoTopViaCursorMovement(t *testing.T) {
	m := textarea.New()
	m.SetValue("a\nb\nc")
	m.GotoBottom()
	m.CursorUp()
	m.CursorUp()
	if m.Line() != 0 {
		t.Errorf("Line() = %d, want 0", m.Line())
	}
}
