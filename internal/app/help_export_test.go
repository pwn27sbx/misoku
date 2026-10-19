package app_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/keymap"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// helpText renders the keymap help on a large terminal, without colours.
func helpText(t *testing.T, user map[string][]string) string {
	t.Helper()
	km, warnings := keymap.New(user)
	if len(warnings) > 0 {
		t.Fatalf("keymap warnings: %v", warnings)
	}
	return ansi.Strip(app.RenderKeymapModal(theme.DefaultTheme(), km, 200, 100))
}

// helpLine returns the help line that contains want, or "".
func helpLine(help, want string) string {
	for _, l := range strings.Split(help, "\n") {
		if strings.Contains(l, want) {
			return l
		}
	}
	return ""
}

func TestHelpShowsExportGroupFirst(t *testing.T) {
	help := helpText(t, nil)
	exp, gen := strings.Index(help, "EXPORT"), strings.Index(help, "GENERAL")
	if exp < 0 || gen < 0 || exp > gen {
		t.Fatalf("EXPORT (%d) must come before GENERAL (%d):\n%s", exp, gen, help)
	}
	for _, want := range [][2]string{
		{"SPC Shift+P", "Export note as PDF"},
		{":export [file]", "Export to a file"},
		{"misoku export", "From the shell"},
	} {
		if l := helpLine(help, want[1]); !strings.Contains(l, want[0]) {
			t.Errorf("export group: want %q next to %q, got line %q", want[0], want[1], l)
		}
	}
}

func TestHelpListsExportOnlyOnce(t *testing.T) {
	help := helpText(t, nil)
	if n := strings.Count(help, "Export note as PDF"); n != 1 {
		t.Errorf("the export key must be listed once (only in EXPORT), got %d", n)
	}
	if n := strings.Count(help, ":export"); n != 1 {
		t.Errorf(":export must be listed once (only in EXPORT), got %d", n)
	}
}

func TestHelpExportGroupShowsTheConfiguredKey(t *testing.T) {
	help := helpText(t, map[string][]string{"export_pdf": {"ctrl+e"}})
	l := helpLine(help, "Export note as PDF")
	if !strings.Contains(l, "Ctrl+E") || strings.Contains(l, "Shift+P") {
		t.Fatalf("export line should show the user's key, got %q", l)
	}
}

func TestHelpExportGroupWithoutKeysStillShowsTheCommand(t *testing.T) {
	help := helpText(t, map[string][]string{"export_pdf": {}})
	if !strings.Contains(help, "EXPORT") || !strings.Contains(help, ":export [file]") || !strings.Contains(help, "misoku export") {
		t.Fatalf("with no keys the EXPORT group still lists the command and the CLI:\n%s", help)
	}
	if strings.Contains(help, "Export note as PDF") {
		t.Errorf("an unbound action has no key line:\n%s", help)
	}
}
