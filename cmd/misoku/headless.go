package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/rich"
	"github.com/pwn27sbx/misoku/internal/rich/engines"
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
	Run      string         `json:"run"`
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

// headlessPage lays a note out for headless mode. It goes through the same
// page constructor as the app, so both draw the same fonts, palette and maths.
func headlessPage(store *core.Store, noteID string, cols int, th theme.Theme) (*rich.Page, *rich.Fonts, rich.Palette, string, error) {
	note, err := store.Get(noteID)
	if err != nil {
		return nil, nil, rich.Palette{}, "", err
	}
	fonts, err := rich.LoadFonts(rich.FontConfig{})
	if err != nil {
		return nil, nil, rich.Palette{}, "", err
	}
	pal := rich.PaletteFromTheme(th)
	doc := rich.Parse([]byte(note.Content))
	dir := filepath.Join(store.Root(), filepath.FromSlash(pathDir(note.ID)))
	page := engines.Page(doc, cols, fonts, pal, rich.ImageOptions{BaseDir: dir, Root: store.Root()})
	// The payload reports the store's note, never the raw argument: a host
	// keys its tiles by what the store resolved.
	return page, fonts, pal, note.ID, nil
}

// runHeadless lays the note out at cols cells, writes one PNG per screenful
// and prints the payload. It returns the process exit code.
func runHeadless(store *core.Store, noteID string, cols, rows int, themeName, version string, out io.Writer) int {
	th, ok := theme.Get(themeName)
	if !ok {
		th = theme.DefaultTheme()
	}
	if rows <= 0 {
		rows = 40
	}
	if cols <= 0 {
		cols = 100
	}
	page, fonts, pal, id, err := headlessPage(store, noteID, cols, th)
	if err != nil {
		fmt.Fprintf(os.Stderr, "misoku --headless: %v\n", err)
		return 1
	}
	// One folder per run: two panes (or a re-render) never overwrite each
	// other's tiles, and the payload names the folder it owns.
	run := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	base := filepath.Join(os.TempDir(), "misoku-headless", run)
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
		Protocol: headlessProtocol, Run: run, Version: version, Note: id, Theme: th.Name,
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
			return nil, fmt.Errorf("tile %d encoded to nothing", n)
		}
		name := filepath.Join(dir, fmt.Sprintf("tile-%04d.png", n))
		// Write then rename: a host never reads a half-written PNG.
		tmp := name + ".tmp"
		if err := os.WriteFile(tmp, png, 0o600); err != nil {
			return nil, err
		}
		if err := os.Rename(tmp, name); err != nil {
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
