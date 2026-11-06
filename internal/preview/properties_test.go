package preview_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"
)

func compileProps(t *testing.T, md string, width int) ([]string, []preview.Hit, []preview.Block) {
	t.Helper()
	out, hits, blocks := preview.CompileBlocks(md, theme.DefaultTheme(), preview.DefaultLook(), width, "", preview.CalendarView{})
	return strings.Split(ansi.Strip(out), "\n"), hits, blocks
}

func TestPropertiesTableRows(t *testing.T) {
	md := "---\ntitle: Plan\nicon: ☁️\nCOVER:\ntags: [go, tui]\naliases:\n  - a\n  - b\nstatus: active\n---\n# Body\n"
	rows, hits, _ := compileProps(t, md, 40)
	// The icon comes first, then the table, then the body.
	if rows[0] != " ☁️" {
		t.Fatalf("the icon should stay first, got %q", rows[0])
	}
	start := -1
	for i, r := range rows {
		if strings.HasPrefix(r, "┌") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("no properties table:\n%s", strings.Join(rows, "\n"))
	}
	got := rows[start : start+6]
	for i, r := range got {
		if w := ansi.StringWidth(r); w != ansi.StringWidth(got[0]) {
			t.Errorf("row %d is %d cells wide, want %d: %q", i, w, ansi.StringWidth(got[0]), r)
		}
	}
	for i, cell := range [][2]string{{"title", "Plan"}, {"tags", "go, tui"}, {"aliases", "a, b"}, {"status", "active"}} {
		r := got[i+1]
		parts := strings.Split(strings.Trim(r, "│"), "│")
		if len(parts) != 2 || strings.TrimSpace(parts[0]) != cell[0] || strings.TrimSpace(parts[1]) != cell[1] {
			t.Errorf("row %d = %q, want %s | %s", i, r, cell[0], cell[1])
		}
	}
	for _, r := range got {
		if strings.Contains(strings.ToLower(r), "cover") || strings.Contains(r, "icon") {
			t.Errorf("cover/icon must not be listed: %q", r)
		}
	}

	// Only scalar values are clickable, with the field's key and line.
	var props []preview.Hit
	for _, h := range hits {
		if strings.HasPrefix(h.Kind, "props:") {
			props = append(props, h)
		}
	}
	if len(props) != 2 {
		t.Fatalf("want hits for title and status only, got %+v", props)
	}
	if props[0].Arg != "title" || props[0].Line != 1 || props[0].Row != start+1 {
		t.Errorf("title hit = %+v", props[0])
	}
	if props[1].Arg != "status" || props[1].Line != 8 || props[1].Row != start+4 {
		t.Errorf("status hit = %+v", props[1])
	}
	// The hit covers the value cell, not the key cell.
	keyEnd := strings.Index(got[1], "│ Plan")
	if props[0].X0 != ansi.StringWidth(got[1][:keyEnd])+1 || props[0].X1 != ansi.StringWidth(got[1])-1 {
		t.Errorf("title hit spans [%d,%d) on %q", props[0].X0, props[0].X1, got[1])
	}
}

func TestPropertiesTableFitsThePane(t *testing.T) {
	md := "---\ndescription: " + strings.Repeat("long words ", 10) + "\n---\nx\n"
	rows, _, _ := compileProps(t, md, 30)
	for _, r := range rows {
		if w := ansi.StringWidth(r); w > 30 {
			t.Errorf("row is %d cells wide, pane is 30: %q", w, r)
		}
	}
	if !strings.Contains(strings.Join(rows, "\n"), "…") {
		t.Errorf("a long value should be truncated:\n%s", strings.Join(rows, "\n"))
	}
}

func TestNoPropertiesTableWithoutOtherKeys(t *testing.T) {
	for _, md := range []string{
		"# Body\ntext\n",
		"---\nicon: ☁️\n---\n# Body\n",
		"---\n---\n# Body\n",
		"---\ntitle: open\n# not closed\n",
	} {
		rows, hits, _ := compileProps(t, md, 40)
		if strings.Contains(strings.Join(rows, "\n"), "┌") {
			t.Errorf("%q should draw no table:\n%s", md, strings.Join(rows, "\n"))
		}
		for _, h := range hits {
			if strings.HasPrefix(h.Kind, "props:") {
				t.Errorf("%q should have no property hits: %+v", md, h)
			}
		}
	}
}

func TestPropertiesKeepBlockLines(t *testing.T) {
	md := "---\ntitle: Plan\nstatus: active\n---\n# Heading\n\nParagraph text\n"
	rows, _, blocks := compileProps(t, md, 40)
	found := false
	for _, b := range blocks {
		if b.Line == 6 {
			found = true
			if !strings.Contains(rows[b.Row], "Paragraph text") {
				t.Errorf("block of line 6 starts on row %d = %q", b.Row, rows[b.Row])
			}
		}
	}
	if !found {
		t.Fatalf("no block for the paragraph on line 6: %+v", blocks)
	}
}
