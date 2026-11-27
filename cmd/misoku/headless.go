package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/rich"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// Headless mode renders a note's page and writes its tiles for a host that
// owns the screen and the input (a Claude Code pane, a preview window), the
// way mddog's --headless does for its plugin. It prints one JSON line; the
// host draws the tiles and deletes them.
const headlessProtocol = 1

// headlessPayload is the one line headless mode prints.
type headlessPayload struct {
	Protocol int            `json:"protocol"`
	Version  string         `json:"version"`
	Note     string         `json:"note"`
	Theme    string         `json:"theme"`
	Cols     int            `json:"cols"`
	Rows     int            `json:"rows"`
	PageRows int            `json:"page_rows"`
	WidthPx  int            `json:"width_px"`
	Font     string         `json:"font,omitempty"`
	Tiles    []headlessTile `json:"tiles"`
}

type headlessTile struct {
	Row    int    `json:"row"`
	Cols   int    `json:"cols"`
	Rows   int    `json:"rows"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Path   string `json:"path"`
}

// runHeadless lays the note out at cols cells, writes one PNG per screenful
// and prints the payload. It returns the process exit code.
func runHeadless(store *core.Store, noteID string, cols, rows int, themeName, version string, out io.Writer) int {
	note, err := store.Get(noteID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "misoku --headless: %v\n", err)
		return 1
	}
	th, ok := theme.Get(themeName)
	if !ok {
		th = theme.DefaultTheme()
	}
	fonts, err := rich.LoadFonts(rich.FontConfig{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "misoku --headless: %v\n", err)
		return 1
	}
	pal := rich.PaletteFromTheme(th)
	doc := rich.Parse([]byte(note.Content))
	dir := filepath.Join(store.Root(), filepath.FromSlash(pathDir(note.ID)))
	page := rich.Layout(doc, rich.LayoutOptions{
		Width: cols, Fonts: fonts, Palette: pal,
		Images: rich.ImageOptions{BaseDir: dir, Root: store.Root()},
	})
	if rows <= 0 {
		rows = 40
	}
	base := filepath.Join(os.TempDir(), "misoku-headless")
	if err := os.MkdirAll(base, 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "misoku --headless: %v\n", err)
		return 1
	}
	tiles, err := writeHeadlessTiles(page, fonts, pal, cols, rows, base)
	if err != nil {
		fmt.Fprintf(os.Stderr, "misoku --headless: %v\n", err)
		return 1
	}
	payload := headlessPayload{
		Protocol: headlessProtocol, Version: version, Note: note.ID, Theme: th.Name,
		Cols: cols, Rows: rows, PageRows: page.Height,
		WidthPx: cols * fonts.CellW, Tiles: tiles,
	}
	if err := json.NewEncoder(out).Encode(payload); err != nil {
		return 1
	}
	return 0
}

func writeHeadlessTiles(page *rich.Page, fonts *rich.Fonts, pal rich.Palette, cols, rows int, dir string) ([]headlessTile, error) {
	var out []headlessTile
	n := 0
	for first := 0; first < max(page.Height, 1); first += rows {
		height := min(rows, max(page.Height-first, 1))
		img := (&rich.Raster{Page: page, Fonts: fonts, Palette: pal}).Tile(first, height)
		png := rich.EncodePNG(img)
		if len(png) == 0 {
			continue
		}
		name := filepath.Join(dir, fmt.Sprintf("tile-%04d.png", n))
		if err := os.WriteFile(name, png, 0o600); err != nil {
			return nil, err
		}
		out = append(out, headlessTile{
			Row: first, Cols: min(cols, page.Width), Rows: height,
			Width: img.Bounds().Dx(), Height: img.Bounds().Dy(), Path: name,
		})
		n++
	}
	return out, nil
}

// pathDir is path.Dir without importing path in main for one call.
func pathDir(id string) string {
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] == '/' {
			return id[:i]
		}
	}
	return "."
}
