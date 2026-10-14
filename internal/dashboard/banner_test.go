package dashboard_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/pwn27sbx/misoku/internal/dashboard"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// The banner is MISOKU in the figlet "Bloody" font, the one the LazyVim
// dashboard header uses.
var misokuBanner = []string{
	` ███▄ ▄███▓ ██▓  ██████  ▒█████   ██ ▄█▀ █    ██ `,
	`▓██▒▀█▀ ██▒▓██▒▒██    ▒ ▒██▒  ██▒ ██▄█▒  ██  ▓██▒`,
	`▓██    ▓██░▒██▒░ ▓██▄   ▒██░  ██▒▓███▄░ ▓██  ▒██░`,
	`▒██    ▒██ ░██░  ▒   ██▒▒██   ██░▓██ █▄ ▓▓█  ░██░`,
	`▒██▒   ░██▒░██░▒██████▒▒░ ████▓▒░▒██▒ █▄▒▒█████▓ `,
	`░ ▒░   ░  ░░▓  ▒ ▒▓▒ ▒ ░░ ▒░▒░▒░ ▒ ▒▒ ▓▒░▒▓▒ ▒ ▒ `,
	`░  ░      ░ ▒ ░░ ░▒  ░ ░  ░ ▒ ▒░ ░ ░▒ ▒░░░▒░ ░ ░ `,
	`░      ░    ▒ ░░  ░  ░  ░ ░ ░ ▒  ░ ░░ ░  ░░░ ░ ░ `,
	`       ░    ░        ░      ░ ░  ░  ░      ░     `,
}

func TestHomeShowsTheMisokuBanner(t *testing.T) {
	db := dashboard.New(theme.DefaultTheme())
	db.SetSize(100, 40)
	view := ansi.Strip(db.View())

	// Every row centred the same way: the letters only line up if each row
	// starts at the same column.
	col := -1
	for i, row := range misokuBanner {
		trimmed := strings.TrimRight(row, " ")
		line := ""
		for _, l := range strings.Split(view, "\n") {
			if strings.Contains(l, trimmed) {
				line = l
				break
			}
		}
		if line == "" {
			t.Fatalf("banner row %d missing from the home screen:\n%s", i, view)
		}
		at := len([]rune(line[:strings.Index(line, trimmed)]))
		if col == -1 {
			col = at
		} else if at != col {
			t.Fatalf("banner row %d starts at column %d, row 0 at %d: the letters are misaligned", i, at, col)
		}
	}
	if strings.Contains(view, "████████╗") {
		t.Fatalf("the old TSUZURI banner is still there:\n%s", view)
	}
	if strings.Contains(view, "綴り") {
		t.Fatalf("the tagline still has Tsuzuri's name:\n%s", view)
	}
	if !strings.Contains(view, "notes that live in plain Markdown files") {
		t.Fatalf("the tagline should stay:\n%s", view)
	}
}
