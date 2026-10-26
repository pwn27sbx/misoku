package app_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/keymap"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// paletteTitle is in the palette's first row while it is open.
const paletteTitle = "MISOKU KEYMAPS"

// tallPalette opens the palette from the home screen of a terminal tall
// enough to list every entry at once, and returns its text.
func tallPalette(t *testing.T, opts ...app.Option) string {
	t.Helper()
	h := newHarnessSize(t, 160, 400, newTestStore(t), opts...)
	h.keys("?")
	v := ansi.Strip(h.m.View())
	if !strings.Contains(v, paletteTitle) {
		t.Fatalf("? should open the palette:\n%s", v)
	}
	return v
}

// paletteRow returns the palette row holding both key and title, or "".
func paletteRow(v, key, title string) string {
	for _, l := range strings.Split(v, "\n") {
		if i := strings.Index(l, key); i >= 0 && strings.Contains(l[i+len(key):], title) {
			return l
		}
	}
	return ""
}

func TestPaletteListsEveryActionWithItsEffectiveKeys(t *testing.T) {
	user := map[string][]string{"toggle_explorer": {"space e"}, "focus_explorer": {"ctrl+e"}}
	v := tallPalette(t, app.WithKeys(user))
	km, _ := keymap.New(user)
	for _, a := range keymap.Actions {
		if paletteRow(v, km.Label(a.Action), a.Help) == "" {
			t.Errorf("%s: no row with %q and %q", a.Action, km.Label(a.Action), a.Help)
		}
	}
	if l := helpLine(v, "Toggle explorer"); !strings.Contains(l, "SPC e") || strings.Contains(l, "Ctrl+B") {
		t.Errorf("toggle_explorer moved to space e in the config, got %q", l)
	}
}

func TestPaletteListsEveryCommandOfTheCommandLine(t *testing.T) {
	v := tallPalette(t)
	for _, c := range content.Commands {
		if paletteRow(v, c.Usage, c.Help) == "" {
			t.Errorf("no row with %q and %q", c.Usage, c.Help)
		}
	}
}

func TestPaletteListsTheFixedKeys(t *testing.T) {
	v := tallPalette(t)
	for _, k := range [][2]string{
		// editor
		{"u", "Undo"},
		{"Ctrl+R", "Redo"},
		{"i a A I", "Insert mode"},
		{"gd / gx", "Follow the link under the cursor"},
		// preview
		{"< / >", "Calendar month"},
		{"zM / zR", "Fold / unfold all"},
		// graph
		{"h j k l", "Move from note to note"},
		{"Enter", "Open the selected note"},
		{"+ / -", "Zoom in / out"},
		{"0", "Centre the graph"},
		{"Esc", "Back"},
		// global graph
		{"h j k l", "Move to the nearest note"},
		{"n / N", "Next / previous link"},
		{"Enter", "Open the note, or start a missing one"},
		{"0", "Centre on the selection"},
		{"Esc", "Back to the previous selection"},
	} {
		if paletteRow(v, k[0], k[1]) == "" {
			t.Errorf("no row with %q and %q", k[0], k[1])
		}
	}
}

// Everything the hand-written help listed before the palette is still there.
func TestPaletteCoversTheOldHelp(t *testing.T) {
	v := tallPalette(t)
	for _, k := range [][2]string{
		{"SPC Shift+P", "Export note as PDF"}, {":export [file]", "Export to a file"}, {"misoku export", "From the shell"},
		{"i a A I", "Insert mode"}, {"Tab/S-Tab", "Indent / outdent (insert)"}, {"o / O", "New line below / above"},
		{"h j k l", "Move"}, {"w / b", "Next / previous word"}, {"0 / $", "Line start / end"},
		{"gg / G", "Top / bottom"}, {"Ctrl+D/U", "Half page down / up"}, {"x / dd", "Delete char / line"},
		{"yy / p", "Copy line / paste"}, {"v / V", "Select, then y copy, d cut"}, {"/ (insert)", "Block menu"},
		{"[[ (insert)", "Link to a note"}, {"gd / gx", "Follow the link under the cursor"},
		{":w", "Save"}, {":w name", "Save as name.md"}, {":wq / :x", "Save and quit"}, {":q / :q!", "Quit / discard"},
		{":bd", "Close tab"}, {":enew", "New note"}, {":colo", "Switch theme"}, {":sync", "Sync the vault with git"},
		{":sync status", "Sync state, conflict copies"}, {":sync setup", "Connect the vault to GitHub"},
		{":addrow", "Table row"}, {":addcol", "Table column"},
		{"Click", "Tabs, tree, cursor"}, {"Wheel", "Scroll any pane"},
		{"< / >", "Calendar month"}, {"T", "Calendar"}, {"zM / zR", "Fold / unfold all"},
	} {
		if paletteRow(v, k[0], k[1]) == "" {
			t.Errorf("no row with %q and %q", k[0], k[1])
		}
	}
}

// openPalette opens a note and the palette over it, on the usual terminal.
func openPalette(t *testing.T, opts ...app.Option) *harness {
	t.Helper()
	h := openNote(t, opts...)
	h.keys("?")
	if !strings.Contains(h.view(), paletteTitle) {
		t.Fatalf("? should open the palette:\n%s", h.view())
	}
	return h
}

// typeQuery types s into the open palette's search, one key at a time.
func typeQuery(h *harness, s string) {
	h.t.Helper()
	for _, r := range s {
		if r == ' ' {
			h.send(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
			continue
		}
		h.keys(string(r))
	}
}

func TestPaletteExplainsTheSelectedEntryAtTheBottom(t *testing.T) {
	h := openPalette(t)
	typeQuery(h, "reading mode")
	if got := noSpace(dialogText(h.view())); !strings.Contains(got, noSpace(keymap.LongText(keymap.ReadingMode))) {
		t.Fatalf("the bottom should explain reading mode with its Long text:\n%s", h.view())
	}
	for range 30 {
		h.key(tea.KeyBackspace)
	}
	typeQuery(h, ":settings")
	var long string
	for _, c := range content.Commands {
		if c.Usage == ":settings" {
			long = c.Long
		}
	}
	if got := noSpace(dialogText(h.view())); long == "" || !strings.Contains(got, noSpace(long)) {
		t.Fatalf("the bottom should explain :settings with the catalog's Long text %q:\n%s", long, h.view())
	}
}

func TestPaletteScrollsInsteadOfCuttingRows(t *testing.T) {
	h := newHarnessSize(t, 80, 24, newTestStore(t))
	h.keys("?")
	v := ansi.Strip(h.m.View())
	if strings.Contains(v, "enlarge the terminal") || strings.Contains(dialogText(v), "…") {
		t.Fatalf("no row may be cut on a small terminal:\n%s", v)
	}
	if !strings.Contains(v, "Export note as PDF") || strings.Contains(v, "Scroll any pane") {
		t.Fatalf("a small terminal shows the top of the list first:\n%s", v)
	}
	for range 200 {
		h.key(tea.KeyDown)
	}
	v = ansi.Strip(h.m.View())
	if !strings.Contains(v, "Scroll any pane") || strings.Contains(v, "Export note as PDF") {
		t.Fatalf("Down should scroll the list to its last entries:\n%s", v)
	}
	if lines := strings.Split(h.m.View(), "\n"); len(lines) != 24 {
		t.Fatalf("the frame must stay 24 rows, got %d", len(lines))
	}
}

func TestPaletteFilterIsFuzzy(t *testing.T) {
	h := openPalette(t)
	typeQuery(h, "exp")
	v := h.view()
	for _, want := range []string{"Export note as PDF", ":export [file]", "Toggle explorer", "Focus explorer"} {
		if !strings.Contains(v, want) {
			t.Errorf("exp should list %q:\n%s", want, v)
		}
	}
	for range 3 {
		h.key(tea.KeyBackspace)
	}
	typeQuery(h, "tglxpl")
	v = h.view()
	if !strings.Contains(v, "Toggle explorer") {
		t.Errorf("a subsequence (tglxpl) should match Toggle explorer:\n%s", v)
	}
	if strings.Contains(v, "Save and quit") || strings.Contains(v, "Export note as PDF") {
		t.Errorf("tglxpl should not match unrelated entries:\n%s", v)
	}
}

func TestPaletteEnterRunsARunnableEntry(t *testing.T) {
	h := openPalette(t)
	typeQuery(h, "reading mode")
	h.key(tea.KeyEnter)
	if v := h.view(); strings.Contains(v, paletteTitle) || mode(v) != "READING" {
		t.Fatalf("Enter on Reading mode should close the palette and run it:\n%s", v)
	}

	h = openPalette(t)
	typeQuery(h, ":settings")
	h.key(tea.KeyEnter)
	if v := h.view(); strings.Contains(v, paletteTitle) || !strings.Contains(v, "Accordion folders") {
		t.Fatalf("Enter on :settings should open the settings menu:\n%s", v)
	}
}

func TestPaletteEnterOnlyExplainsAKeyThatIsNotACommand(t *testing.T) {
	h := openPalette(t)
	typeQuery(h, "redo")
	if v := h.view(); !strings.Contains(v, "use the key") || strings.Contains(v, "Enter run") {
		t.Fatalf("the footer should say Redo is a key, not a command:\n%s", v)
	}
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, paletteTitle) || !strings.Contains(v, "Redo") {
		t.Fatalf("Enter on a key only explains it: the palette stays open:\n%s", v)
	}
	for range 4 {
		h.key(tea.KeyBackspace)
	}
	typeQuery(h, "reading mode")
	if v := h.view(); !strings.Contains(v, "Enter run") {
		t.Fatalf("the footer should offer Enter on a runnable entry:\n%s", v)
	}
}

func TestPaletteKeysTypeIntoTheSearchAndOnlyEscCloses(t *testing.T) {
	h := openPalette(t)
	typeQuery(h, "?q ")
	v := h.view()
	if !strings.Contains(v, paletteTitle) {
		t.Fatalf("? q and space must type into the search, not close:\n%s", v)
	}
	if !strings.Contains(v, "?q") {
		t.Fatalf("the search should hold %q:\n%s", "?q ", v)
	}
	h.click(1, 1)
	if !strings.Contains(h.view(), paletteTitle) {
		t.Fatalf("a click does not close the palette:\n%s", h.view())
	}
	h.key(tea.KeyEsc)
	if v := h.view(); strings.Contains(v, paletteTitle) {
		t.Fatalf("Esc closes the palette:\n%s", v)
	}
	if v := h.view(); mode(v) != "NORMAL" {
		t.Fatalf("the typed q must not have quit or changed the editor:\n%s", v)
	}
}

// The old help cut a long key against its description ("SPC fFind note").
// In the palette a key is never cut and never touches its title.
func TestPaletteKeysNeverRunIntoTheirTitle(t *testing.T) {
	glued := regexp.MustCompile(`\S(Find note|Toggle explorer|This help)`)
	for _, size := range [][2]int{{60, 20}, {80, 24}, {140, 40}} {
		h := newHarnessSize(t, size[0], size[1], newTestStore(t))
		h.keys("?")
		typeQuery(h, "find note")
		v := ansi.Strip(h.m.View())
		if !strings.Contains(v, `Ctrl+P / \ / SPC f`) {
			t.Errorf("%v: the whole find_note key should show:\n%s", size, v)
		}
		if glued.MatchString(v) {
			t.Errorf("%v: a key runs into its title:\n%s", size, v)
		}
	}
}

// A key too long for the key column gets a line of its own, whole.
func TestPaletteGivesALongKeyItsOwnLine(t *testing.T) {
	user := map[string][]string{"find_note": {"ctrl+p", "\\", "space f", "ctrl+g", "alt+f", "space o"}}
	h := newHarnessSize(t, 60, 20, newTestStore(t), app.WithKeys(user))
	h.keys("?")
	typeQuery(h, "find note")
	v := ansi.Strip(h.m.View())
	if !strings.Contains(v, `Ctrl+P / \ / SPC f / Ctrl+G / Alt+F / SPC o`) {
		t.Fatalf("the whole configured key should show:\n%s", v)
	}
	if regexp.MustCompile(`\S(Find note)`).MatchString(v) {
		t.Fatalf("the long key runs into its title:\n%s", v)
	}
}

// ? opens the palette wherever it worked before: the editor, the explorer
// and the home screen.
func TestQuestionMarkOpensThePaletteEverywhere(t *testing.T) {
	h := openNote(t)
	h.keys("?")
	if !strings.Contains(h.view(), paletteTitle) {
		t.Fatalf("? from the editor:\n%s", h.view())
	}
	h = openNote(t)
	h.keys(" e")
	h.keys("?")
	if !strings.Contains(h.view(), paletteTitle) {
		t.Fatalf("? from the explorer:\n%s", h.view())
	}
	h = newHarness(t, newTestStore(t))
	h.keys("?")
	if !strings.Contains(h.view(), paletteTitle) {
		t.Fatalf("? from home:\n%s", h.view())
	}
}

// homePalette opens the palette from the home screen, with doc.md open in a
// tab the home screen hides when hidden is set.
func homePalette(t *testing.T, hidden bool) *harness {
	t.Helper()
	var h *harness
	if hidden {
		h = openNote(t)
		h.keys(" d")
	} else {
		h = newHarness(t, newTestStore(t))
	}
	h.keys("?")
	if v := h.view(); !strings.Contains(v, paletteTitle) || !strings.Contains(v, "HOME") {
		t.Fatalf("? should open the palette over home:\n%s", v)
	}
	return h
}

// From home, Enter does what the key would do there: the actions home
// withholds (they need an open note) are only explained.
func TestPaletteFromHomeOnlyRunsWhatTheHomeKeysRun(t *testing.T) {
	for _, q := range []string{"reading mode", "next pane", "toggle preview", "close tab", "save (new"} {
		h := homePalette(t, true)
		typeQuery(h, q)
		if v := h.view(); !strings.Contains(v, "Needs an open note") || strings.Contains(v, "Enter run") {
			t.Errorf("%q from home: the footer should say it needs an open note:\n%s", q, v)
		}
		h.key(tea.KeyEnter)
		v := h.view()
		if !strings.Contains(v, paletteTitle) || strings.Contains(v, "No file") {
			t.Errorf("%q from home must not run:\n%s", q, v)
		}
		h.key(tea.KeyEsc)
		if v := h.view(); !strings.Contains(v, "HOME") {
			t.Errorf("%q from home left the home screen:\n%s", q, v)
		}
		h.key(tea.KeyEsc) // home's Esc goes back to the open note
		if v := h.view(); !strings.Contains(strings.Split(v, "\n")[0], "doc.md") {
			t.Errorf("%q from home closed the hidden tab:\n%s", q, v)
		}
	}
	// What home runs, the palette runs from home too.
	h := homePalette(t, false)
	typeQuery(h, "themes")
	if v := h.view(); !strings.Contains(v, "Enter run") {
		t.Fatalf("Themes runs from home:\n%s", v)
	}
	h.key(tea.KeyEnter)
	if v := h.view(); strings.Contains(v, paletteTitle) || !strings.Contains(v, "THEMES") && !strings.Contains(v, "Themes") {
		t.Fatalf("Enter on Themes from home should open the theme picker:\n%s", v)
	}
}

// From home, only the commands that need no note run; the rest are
// explained with the same footer.
func TestPaletteFromHomeOnlyRunsCommandsThatNeedNoNote(t *testing.T) {
	for _, q := range []string{":wq", ":w", ":bd", ":export"} {
		h := homePalette(t, true)
		typeQuery(h, q)
		if v := h.view(); !strings.Contains(v, "Needs an open note") {
			t.Errorf("%s from home: the footer should say it needs an open note:\n%s", q, v)
		}
		h.key(tea.KeyEnter)
		if v := h.view(); h.quit || !strings.Contains(v, paletteTitle) || strings.Contains(v, "E32") {
			t.Errorf("%s from home must not run:\n%s", q, v)
		}
	}
	h := homePalette(t, false)
	typeQuery(h, ":settings")
	h.key(tea.KeyEnter)
	if v := h.view(); strings.Contains(v, paletteTitle) || !strings.Contains(v, "Accordion folders") {
		t.Fatalf(":settings runs from home:\n%s", v)
	}
	h = homePalette(t, false)
	typeQuery(h, ":q / :q!")
	h.key(tea.KeyEnter)
	if !h.quit {
		t.Fatalf(":q runs from home, as q does:\n%s", h.view())
	}
}
