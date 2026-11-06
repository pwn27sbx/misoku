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
		keymap.ToggleExplorer:      {"ctrl+b"},
		keymap.Save:                {"ctrl+s", "space w"},
		keymap.CloseTab:            {"space x"},
		keymap.ExplorerDelete:      {"d", "x", "delete"},
		keymap.ExportPDF:           {"space P"},
		keymap.ReadingView:         {"space p"},
		keymap.EditorView:          {"space r"},
		keymap.Graph:               {"space g"},
		keymap.GlobalGraph:         {"space G"},
		keymap.ExplorerFileManager: {"M"},
		keymap.ExplorerEditFolder:  {"m"},
		keymap.ExplorerPreview:     {"p"},
		keymap.ExplorerNewSubNote:  {"s"},
		keymap.ExplorerInsert:      {"i", "I", "a", "A", "o", "O"},
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
	if got := m.Label(keymap.FindNote); got != "Ctrl+P / \\ / SPC f / SPC o f / SPC o g" {
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

// Every general action sits in one of the leader guide's groups, so a
// leader key the user gives it always shows up in the guide; explorer
// actions are not leader actions and have no group.
func TestEveryGeneralActionHasALeaderGuideGroup(t *testing.T) {
	groups := map[string]bool{}
	for _, g := range keymap.Groups {
		groups[g] = true
	}
	if !reflect.DeepEqual(keymap.Groups, []string{keymap.GroupNote, keymap.GroupView, keymap.GroupGo}) {
		t.Fatalf("groups: got %v, want note, view, go", keymap.Groups)
	}
	for _, a := range keymap.Actions {
		switch a.Section {
		case "general":
			if !groups[a.Group] {
				t.Errorf("%s: group %q is not a leader guide group", a.Action, a.Group)
			}
		default:
			if a.Group != "" {
				t.Errorf("%s: explorer action has group %q", a.Action, a.Group)
			}
		}
	}
	want := map[string]string{
		keymap.NewNote:       keymap.GroupNote,
		keymap.ReadingView:   keymap.GroupView,
		keymap.FocusExplorer: keymap.GroupGo,
	}
	for _, a := range keymap.Actions {
		if g, ok := want[a.Action]; ok && a.Group != g {
			t.Errorf("%s: group %q, want %q", a.Action, a.Group, g)
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

// LeaderLabel shows only what follows Space, from the effective keymap.
func TestLeaderLabelListsOnlyTheKeysAfterSpace(t *testing.T) {
	m, _ := keymap.New(nil)
	cases := map[string]string{
		keymap.Help:           "h / ?",
		keymap.ExportPDF:      "Shift+P",
		keymap.PrevTab:        "S-Tab",
		keymap.NextTab:        "Tab",
		keymap.ToggleExplorer: "",
	}
	for action, want := range cases {
		if got := m.LeaderLabel(action); got != want {
			t.Errorf("%s: got %q, want %q", action, got, want)
		}
	}
	m, _ = keymap.New(map[string][]string{"toggle_explorer": {"Space e", "ctrl+e"}, "focus_explorer": {}})
	if got := m.LeaderLabel(keymap.ToggleExplorer); got != "e" {
		t.Errorf("configured toggle_explorer: got %q, want %q", got, "e")
	}
	if got := m.LeaderLabel(keymap.FocusExplorer); got != "" {
		t.Errorf("unbound focus_explorer: got %q, want empty", got)
	}
}

// The command palette explains the selected action at its bottom with Long:
// every action has one, and it says more than the one-line Help.
func TestEveryActionHasALongExplanation(t *testing.T) {
	for _, a := range keymap.Actions {
		if a.Long == "" || a.Long == a.Help {
			t.Errorf("%s: Long %q must explain more than Help %q", a.Action, a.Long, a.Help)
		}
		if got := keymap.LongText(a.Action); got != a.Long {
			t.Errorf("LongText(%s) = %q, want %q", a.Action, got, a.Long)
		}
	}
	if got := keymap.LongText("nope"); got != "" {
		t.Errorf("an unknown action has no explanation, got %q", got)
	}
}

// A leader sequence may take two keys after Space ("space o c"), like the
// LazyVim companion's <leader>oc; the two-key ones work as before.
func TestThreeKeyLeaderSequences(t *testing.T) {
	m, warnings := keymap.New(map[string][]string{"toggle_checkbox": {"Space o c"}})
	if len(warnings) != 0 {
		t.Fatalf("a three-key sequence must be accepted, got %v", warnings)
	}
	if got := m.Keys(keymap.ToggleCheckbox); !reflect.DeepEqual(got, []string{"space o c"}) {
		t.Fatalf("toggle_checkbox: got %v", got)
	}
	if !m.LeaderSeq(keymap.ToggleCheckbox, []string{"o", "c"}) {
		t.Error("o c should complete space o c")
	}
	if m.LeaderSeq(keymap.ToggleCheckbox, []string{"o"}) {
		t.Error("o alone does not complete space o c")
	}
	if !m.LeaderPrefix([]string{keymap.ToggleCheckbox}, []string{"o"}) {
		t.Error("o starts space o c")
	}
	if m.LeaderPrefix([]string{keymap.ToggleCheckbox}, []string{"o", "c"}) {
		t.Error("o c is complete, not the start of a longer sequence")
	}
	if m.LeaderPrefix([]string{keymap.Graph}, []string{"o"}) {
		t.Error("only the listed actions count")
	}
	if !m.Leader(keymap.Graph, "g") || !m.LeaderSeq(keymap.Graph, []string{"g"}) {
		t.Error("the two-key space g still matches")
	}
	if got := m.Label(keymap.RemoveCheckbox); got != "SPC o Shift+C" {
		t.Errorf("Label(RemoveCheckbox) = %q", got)
	}
	_, warnings = keymap.New(map[string][]string{"save": {"space a b c"}})
	if joined := strings.Join(warnings, "\n"); !strings.Contains(joined, `"space a b c"`) {
		t.Errorf("four keys are still rejected:\n%s", joined)
	}
}

func TestTheMisokuGroupDefaults(t *testing.T) {
	m, warnings := keymap.New(nil)
	if len(warnings) != 0 {
		t.Fatalf("defaults must not warn: %v", warnings)
	}
	cases := map[string][]string{
		keymap.ToggleCheckbox: {"space o c"},
		keymap.RemoveCheckbox: {"space o C"},
		keymap.FindNote:       {"ctrl+p", "\\", "space f", "space o f", "space o g"},
		keymap.NewNote:        {"ctrl+n", "space n", "space o n"},
		keymap.PasteImage:     {"space i", "space o p"},
		keymap.DailyNote:      {"space o d"},
		keymap.Home:           {"space d"},
	}
	for action, want := range cases {
		if got := m.Keys(action); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %v, want %v", action, got, want)
		}
	}
	// The guide's first page shows only what follows Space directly, and o
	// as the misoku group; after o, the group's own keys.
	if got := m.LeaderLabel(keymap.FindNote); got != "f" {
		t.Errorf("LeaderLabel(FindNote) = %q, want f", got)
	}
	if got := m.LeaderLabelAfter(keymap.FindNote, []string{"o"}); got != "f / g" {
		t.Errorf("LeaderLabelAfter(FindNote, o) = %q", got)
	}
	if got := m.LeaderLabelAfter(keymap.RemoveCheckbox, []string{"o"}); got != "Shift+C" {
		t.Errorf("LeaderLabelAfter(RemoveCheckbox, o) = %q", got)
	}
	all := []string{keymap.NewNote, keymap.FindNote, keymap.ToggleCheckbox, keymap.Graph}
	if got := m.LeaderSubgroups(all, nil); !reflect.DeepEqual(got, []keymap.Subgroup{{Key: "o", Name: "misoku"}}) {
		t.Errorf("LeaderSubgroups = %v, want one o misoku", got)
	}
	if got := m.LeaderSubgroups(all, []string{"o"}); len(got) != 0 {
		t.Errorf("nothing nests under o, got %v", got)
	}
}

// A sequence that is the start of another can never wait for the rest.
func TestAPrefixOfAnotherSequenceIsReported(t *testing.T) {
	_, warnings := keymap.New(map[string][]string{"graph": {"space o"}})
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, `"space o"`) || !strings.Contains(joined, "graph") || !strings.Contains(joined, "toggle_checkbox") {
		t.Fatalf("space o against space o c should be reported, got:\n%s", joined)
	}
}

// The LazyVim companion's ]o and [o live under space o: Space, o, then ]
// or [, and the default map reports no clash for them.
func TestLinkJumpDefaults(t *testing.T) {
	m, warnings := keymap.New(nil)
	if len(warnings) != 0 {
		t.Fatalf("the defaults should not clash, got %v", warnings)
	}
	for action, last := range map[string]string{keymap.NextLink: "]", keymap.PrevLink: "["} {
		if got := m.Keys(action); !reflect.DeepEqual(got, []string{"space o " + last}) {
			t.Errorf("%s: got %v", action, got)
		}
		if !m.LeaderSeq(action, []string{"o", last}) {
			t.Errorf("o %s should complete %s", last, action)
		}
	}
}

// S4: config files written before the views were swapped keep working:
// reading_mode binds the reading view and toggle_preview the editor view,
// without a warning; the new name wins when both are given.
func TestOldViewActionNamesMapToTheViews(t *testing.T) {
	km, warnings := keymap.New(map[string][]string{"reading_mode": {"space v"}, "toggle_preview": {"space z"}})
	if len(warnings) != 0 {
		t.Fatalf("old names must not warn: %v", warnings)
	}
	if got := km.Keys(keymap.ReadingView); !reflect.DeepEqual(got, []string{"space v"}) {
		t.Errorf("reading_mode should bind reading_view, got %v", got)
	}
	if got := km.Keys(keymap.EditorView); !reflect.DeepEqual(got, []string{"space z"}) {
		t.Errorf("toggle_preview should bind editor_view, got %v", got)
	}
	km, _ = keymap.New(map[string][]string{"reading_mode": {"space v"}, "reading_view": {"space y"}})
	if got := km.Keys(keymap.ReadingView); !reflect.DeepEqual(got, []string{"space y"}) {
		t.Errorf("reading_view should win over reading_mode, got %v", got)
	}
}
