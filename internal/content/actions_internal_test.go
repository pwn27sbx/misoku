package content

import (
	"testing"

	"github.com/pwn27sbx/misoku/internal/theme"
)

// V1: the editor records the actions the tutorial listens to only while the
// app asked for them, and a drain returns each one once.
func TestActionsAreRecordedOnlyWhenAsked(t *testing.T) {
	m := New(theme.DefaultTheme())
	m.reportAction("motion", "hjkl")
	if got := m.TakeActions(); got != nil {
		t.Fatalf("an editor that was not asked to report recorded %v", got)
	}

	m.SetReportActions(true)
	m.reportAction("motion", "hjkl")
	m.reportAction("yank", "line")
	got := m.TakeActions()
	if len(got) != 2 || got[0].Name != "motion" || got[0].Value != "hjkl" || got[1].Name != "yank" {
		t.Fatalf("TakeActions = %v, want the motion and the yank", got)
	}
	if again := m.TakeActions(); again != nil {
		t.Fatalf("a second drain returned %v, want nothing", again)
	}
	m.SetReportActions(false)
	m.reportAction("motion", "words")
	if got := m.TakeActions(); got != nil {
		t.Fatalf("turning the reports off still recorded %v", got)
	}
}

// V1: a copy of the editor made by Update shares the log with the model the
// app holds, so a drain after the copy sees what the copy recorded.
func TestActionsSurviveACopy(t *testing.T) {
	m := New(theme.DefaultTheme())
	m.SetReportActions(true)
	copied := m
	copied.reportAction("paste", "")
	if got := m.TakeActions(); len(got) != 1 || got[0].Name != "paste" {
		t.Fatalf("the original drained %v, want the copy's paste", got)
	}
}
