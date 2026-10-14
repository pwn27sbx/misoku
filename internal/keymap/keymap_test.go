package keymap_test

import (
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
	// Explorer keys only apply inside the explorer: they may reuse general keys.
	_, warnings = keymap.New(map[string][]string{"explorer.find": {"ctrl+p"}})
	if len(warnings) != 0 {
		t.Fatalf("explorer and general actions do not clash, got %v", warnings)
	}
}
