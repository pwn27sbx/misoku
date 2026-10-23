package keymap_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/keymap"
)

func TestDefaultsAreTodaysShortcuts(t *testing.T) {
	m, warnings := keymap.New(nil)
	if len(warnings) != 0 {
		t.Fatalf("defaults must not warn: %v", warnings)
	}
	cases := map[string][]string{
		keymap.ToggleExplorer: {"ctrl+b"},
		keymap.Save:           {"ctrl+s", "space w"},
		keymap.CloseTab:       {"space x"},
		keymap.ExplorerDelete: {"d", "x", "delete"},
		keymap.ExportPDF:      {"space P"},
		keymap.ReadingMode:    {"space r"},
	}
	for action, want := range cases {
		if got := m.Keys(action); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %v, want %v", action, got, want)
		}
	}
}

func TestUserKeysReplaceTheDefaultsOfThatAction(t *testing.T) {
	m, warnings := keymap.New(map[string][]string{
		"toggle_explorer": {"Space e", "Ctrl+E"},
		"focus_explorer":  {},
		"close_tab":       {},
	})
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if got := m.Keys(keymap.ToggleExplorer); !reflect.DeepEqual(got, []string{"space e", "ctrl+e"}) {
		t.Fatalf("toggle_explorer = %v", got)
	}
	if got := m.Keys(keymap.CloseTab); len(got) != 0 {
		t.Fatalf("an empty list unbinds the action, got %v", got)
	}
	if got := m.Keys(keymap.Save); !reflect.DeepEqual(got, []string{"ctrl+s", "space w"}) {
		t.Fatalf("actions not in the config keep their defaults, got %v", got)
	}
}

func TestMatching(t *testing.T) {
	m, _ := keymap.New(map[string][]string{"toggle_explorer": {"space e", "ctrl+e"}})
	if !m.Direct(keymap.ToggleExplorer, "ctrl+e") {
		t.Error("ctrl+e should match directly")
	}
	if m.Direct(keymap.ToggleExplorer, "e") {
		t.Error("plain e is only the second half of the leader sequence")
	}
	if !m.Leader(keymap.ToggleExplorer, "e") {
		t.Error("space e should match after the leader")
	}
	if m.Leader(keymap.Save, "e") {
		t.Error("save is not on space e")
	}
	if !m.IsLeader(" ") {
		t.Error("space is the leader")
	}
}

func TestConfigMistakesAreReported(t *testing.T) {
	m, warnings := keymap.New(map[string][]string{
		"toggle_explorr": {"ctrl+e"},
		"save":           {"ctrl+x ctrl+s"},
	})
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, `unknown action "toggle_explorr"`) {
		t.Errorf("missing unknown-action warning:\n%s", joined)
	}
	if !strings.Contains(joined, `"ctrl+x ctrl+s"`) {
		t.Errorf("missing warning for a sequence that does not start with space:\n%s", joined)
	}
	if got := m.Keys(keymap.Save); !reflect.DeepEqual(got, []string{"ctrl+s", "space w"}) {
		t.Errorf("an action whose keys were all rejected keeps its defaults, got %v", got)
	}
}

func TestLabelsForTheHelpScreen(t *testing.T) {
	m, _ := keymap.New(map[string][]string{"toggle_explorer": {"space e", "ctrl+e"}})
	if got := m.Label(keymap.ToggleExplorer); got != "SPC e / Ctrl+E" {
		t.Fatalf("label = %q", got)
	}
	if got := m.Label(keymap.FindNote); got != "Ctrl+P / \\ / SPC f" {
		t.Fatalf("label = %q", got)
	}
}

func TestSharedKeysAreReported(t *testing.T) {
	_, warnings := keymap.New(map[string][]string{"toggle_explorer": {"space e"}})
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, `"space e"`) || !strings.Contains(joined, "toggle_explorer") || !strings.Contains(joined, "focus_explorer") {
		t.Fatalf("a key on two actions should be reported, got:\n%s", joined)
	}
	_, warnings = keymap.New(map[string][]string{"toggle_explorer": {"space e"}, "focus_explorer": {}})
	if len(warnings) != 0 {
		t.Fatalf("no clash once focus_explorer is unbound, got %v", warnings)
	}
	// explorer.find runs the same finder as find_note, so it may reuse its keys.
	_, warnings = keymap.New(map[string][]string{"explorer.find": {"ctrl+p"}})
	if len(warnings) != 0 {
		t.Fatalf("explorer.find may share find_note's keys, got %v", warnings)
	}
}

func TestShiftIsSpelledOutForCapitalsAfterTheLeader(t *testing.T) {
	m, _ := keymap.New(nil)
	if got := m.Label(keymap.ExportPDF); got != "SPC Shift+P" {
		t.Errorf("Label(ExportPDF) = %q, want %q", got, "SPC Shift+P")
	}
	if got := m.Label(keymap.ExplorerBottom); got != "G / End" {
		t.Errorf("a capital on its own keeps its letter: Label(ExplorerBottom) = %q", got)
	}
	if got := m.Label(keymap.Help); got != "? / SPC h / SPC ?" {
		t.Errorf("only capitals get Shift+: Label(Help) = %q", got)
	}
	if got := m.Label(keymap.PrevTab); got != "[ / SPC S-Tab" {
		t.Errorf("Label(PrevTab) = %q", got)
	}
}

func TestExplorerKeyShadowedByAGeneralKeyIsReported(t *testing.T) {
	_, warnings := keymap.New(map[string][]string{"explorer.collapse_all": {"tab"}})
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, `"tab"`) || !strings.Contains(joined, "explorer.collapse_all") || !strings.Contains(joined, "next_pane") {
		t.Fatalf("an explorer key that a general action takes first should be reported, got:\n%s", joined)
	}
}

func TestExplorerKeyShadowedByTheLeaderIsReported(t *testing.T) {
	for _, k := range []string{"space", "space r"} {
		_, warnings := keymap.New(map[string][]string{"explorer.rename": {k}})
		joined := strings.Join(warnings, "\n")
		if !strings.Contains(joined, fmt.Sprintf("%q", k)) || !strings.Contains(joined, "explorer.rename") || !strings.Contains(joined, "leader") {
			t.Errorf("%q starts the leader, so it never reaches the explorer; got:\n%s", k, joined)
		}
	}
}

func TestAKeySharedByTheSameActionIsNotReported(t *testing.T) {
	// find_note and explorer.find open the same finder: the general key
	// running first does exactly what the explorer key would.
	_, warnings := keymap.New(map[string][]string{"find_note": {"ctrl+g"}, "explorer.find": {"ctrl+g", "f"}})
	if len(warnings) != 0 {
		t.Fatalf("a key shared by find_note and explorer.find is fine, got %v", warnings)
	}
}

func TestToggleFoldDefaultsLeaveSpaceAndTabToTheApp(t *testing.T) {
	m, _ := keymap.New(nil)
	if got := m.Keys(keymap.ExplorerToggleFold); !reflect.DeepEqual(got, []string{"z"}) {
		t.Fatalf("space is the leader and tab is next_pane, so fold keeps only z; got %v", got)
	}
}

func TestHintRendersTheFirstKey(t *testing.T) {
	m, _ := keymap.New(map[string][]string{"find_note": {"space f", "ctrl+p"}, "close_tab": {}})
	if got := m.Hint(keymap.FindNote); got != "SPC f" {
		t.Errorf("Hint(FindNote) = %q, want %q", got, "SPC f")
	}
	if got := m.Hint(keymap.ExportPDF); got != "SPC Shift+P" {
		t.Errorf("Hint(ExportPDF) = %q, want the help screen's spelling", got)
	}
	if got := m.Hint(keymap.CloseTab); got != "" {
		t.Errorf("an unbound action has no hint, got %q", got)
	}
}
