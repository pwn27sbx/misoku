package main

import (
	"bytes"
	"encoding/json"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/rich"
	"github.com/pwn27sbx/misoku/internal/theme"
)

func TestHeadlessPrintsOnePayloadWithTiles(t *testing.T) {
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := "# Title\n\nA paragraph that fills the page.\n\n```go\nx := 1\n```\n"
	if _, err := store.SaveAs("", "Head", body); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := runHeadless(store, "Head.md", 60, 10, "onedark", "test", &out); code != 0 {
		t.Fatalf("runHeadless exit %d", code)
	}
	var p headlessPayload
	if err := json.Unmarshal(out.Bytes(), &p); err != nil {
		t.Fatalf("payload is not one JSON object: %v\n%s", err, out.String())
	}
	if p.Protocol != headlessProtocol || p.Note != "Head.md" || p.Cols != 60 || len(p.Tiles) == 0 {
		t.Fatalf("payload = %+v", p)
	}
	if note, err := store.Get("Head.md"); err != nil || p.Note != note.ID {
		t.Fatalf("payload note %q is not the store's %q (%v)", p.Note, note.ID, err)
	}
	rows := 0
	for i, tile := range p.Tiles {
		if tile.Row != i*10 {
			t.Fatalf("tile %d starts at row %d, want %d", i, tile.Row, i*10)
		}
		rows += tile.Rows
	}
	if rows < p.PageRows {
		t.Fatalf("tiles cover %d rows of %d", rows, p.PageRows)
	}
	for _, tile := range p.Tiles {
		data, err := os.ReadFile(tile.Path)
		if err != nil {
			t.Fatalf("tile %s: %v", tile.Path, err)
		}
		if _, err := png.Decode(bytes.NewReader(data)); err != nil {
			t.Fatalf("tile %s is not a PNG: %v", tile.Path, err)
		}
		if tile.Width != p.WidthPx {
			t.Fatalf("tile width %d, want %d", tile.Width, p.WidthPx)
		}
	}
}

// Headless mode lays the page out through the same constructor as the app, so
// a note with maths reaches the tiles as drawn boxes, not as source text.
func TestHeadlessDrawsMath(t *testing.T) {
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := "# Maths\n\nInline $E=mc^2$ and a block:\n\n$$\n\\frac{a}{b}\n$$\n"
	if _, err := store.SaveAs("", "Math", body); err != nil {
		t.Fatal(err)
	}
	page, _, _, id, err := headlessPage(store, "Math.md", 60, theme.DefaultTheme())
	if err != nil {
		t.Fatal(err)
	}
	if id != "Math.md" {
		t.Fatalf("headless page id = %q, want the store's Math.md", id)
	}
	boxes := 0
	for _, it := range page.Items {
		if it.Kind == rich.ItemBox {
			boxes++
		}
	}
	if boxes != 2 {
		t.Fatalf("maths boxes on the headless page = %d, want 2", boxes)
	}
	for _, it := range page.Items {
		if it.Kind == rich.ItemText && strings.Contains(it.Text, "frac") {
			t.Fatalf("the maths source was drawn as text: %q", it.Text)
		}
	}
}

// Zero columns or rows take the documented defaults before the page is laid
// out, so the payload and the tile pixels agree.
func TestHeadlessDefaultsBeforeLayout(t *testing.T) {
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("", "Head", "A paragraph.\n"); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := runHeadless(store, "Head.md", 0, 0, "onedark", "test", &out); code != 0 {
		t.Fatalf("runHeadless exit %d", code)
	}
	var p headlessPayload
	if err := json.Unmarshal(out.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Cols != 100 || p.Rows != 40 {
		t.Fatalf("payload cols/rows = %d/%d, want the 100x40 defaults", p.Cols, p.Rows)
	}
	for _, tile := range p.Tiles {
		if tile.Width != p.WidthPx {
			t.Fatalf("tile width %d, want %d", tile.Width, p.WidthPx)
		}
	}
}
