package main

import (
	"bytes"
	"encoding/json"
	"image/png"
	"os"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
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
