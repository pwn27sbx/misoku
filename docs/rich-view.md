# Rich reading view

`:rich` (or `Space V`) draws the active note as a page of pixels over the
terminal's graphics protocol, behind rows of placeholder cells, with the text
reading view as the fallback everywhere. This document is the user-facing map;
`docs/rich-contracts.md` is the internal seam between the packages.

## Pipeline

```
note text ─rich.Parse─▶ Document ─rich.Layout─▶ Page ─rich.Raster─▶ tiles ─rich.View─▶ placeholder rows
```

- `internal/rich` parses the note into blocks (each keeps its source lines),
  lays them out in terminal cells, and rasterises slices of the page to PNG.
- `internal/rich/engines` is the one place the engines are installed. The app,
  `--headless` and the page tests all lay pages out through it, so they cannot
  drift apart. It owns the maths config and the diagram renderer, whose caches
  survive page rebuilds; a theme change (new palette) rebuilds them.
- `internal/rich/math` draws TeX-style maths (its own parser, Latin Modern Math
  bundled) as block boxes and as inline ink on a text baseline.
- `internal/rich/diagram` draws the thirteen Mermaid families natively; a block
  it cannot parse keeps its source panel with the reason.
- `internal/rich/view` owns scroll, tiles, the four-tile quota, placeholders,
  search marks, selection, link hints and the zoomed diagram page.

## Keys

| Key | What it does |
| --- | --- |
| `j` / `k`, `d` / `u`, `g` / `G` | Scroll one row, half a page, to the top/bottom |
| `/`, `n` / `N` | Search the page, next/previous match |
| `f` then a letter | Follow a link hint |
| `H` / `L` | Scroll a wide code block or table sideways |
| `+` / `=`, `-` | Zoom the page in or out (1.15× steps) |
| `enter` | Zoom the selected diagram (or the first on screen) to the pane |
| click | Zoom a diagram; any other row opens the text view at that line |
| `y` | Copy the selection, or the visible page |
| `esc` | Leave a zoom/search/selection, then back to the text view at the visible line |

While a diagram is zoomed, `esc`, `enter`, `i`, `q` or a click return to the
page at the row you left.

## Terminals

Kitty graphics are chosen from the environment (`internal/kitty`): Ghostty,
kitty, WezTerm, iTerm2, Konsole and Rex, plus herdr panes, which render
graphics themselves. `MISOKU_IMAGES=kitty|graphics|on` forces them and
`blocks|half|off` disables them. screen and zellij are not rendered through
(they get the text view); Warp is not supported.

tmux 3.3 or newer works when it lets graphics through: with
`set -g allow-passthrough on`, the transmissions go wrapped in tmux's
passthrough DCS (the Unicode placeholders survive tmux's redraws), and with it
off `space V` says how to turn it on.

The page is drawn at the terminal's real cell size in pixels (TIOCGWINSZ):
body text is 0.83 of the cell height, the terminal's own size. A terminal that
reports no pixels (some multiplexers answer zeros) falls back to 12x24; set
`ui.rich_cell_px` (or `MISOKU_RICH_CELL_PX`, e.g. `10x19`) to tell it.

Without graphics the rich view refuses to open and says why; the text reading
view is always one key away and never blank.

## Fonts

`ui.rich_font` and `ui.rich_mono_font` (or `MISOKU_RICH_FONT` /
`MISOKU_RICH_MONO_FONT`) point at faces; otherwise the system's sans-serif and
monospace are discovered (fontconfig on Linux, the usual folders on macOS and
Windows). Body text is shaped with go-text/typesetting; bold and italic use the
system faces when present, and synthetic bold otherwise. Maths always uses the
bundled Latin Modern Math (GUST licence); text faces are never bundled.

## Images

Local images are confined to the vault (symlinks resolved). Remote images are
off by default; `ui.rich_remote_images = on` fetches them with the same
public-address check, an 8 MB cap and a cache, and reveals the reader's IP to
the host — see `docs/configuration.md`.

## Performance

Measured with `go test ./internal/rich -bench Rich -run TestRichPaintBudgets`
on a 1000-line note at 120×40 (Go test fonts, 1× = 12×24 cells, 2× = 24×48):

| Work | 1× | 2× | Budget |
| --- | --- | --- | --- |
| First paint (parse, lay out, screen tile, PNG) | 31 ms | 74 ms | 250 ms |
| Scroll frame p100 (fixed 16-row tile bands) | 9 ms | 28 ms | 33 ms |

The view caches fixed 16-row tile bands and reuses their encoded PNGs across
small scrolls, so only band crossings encode a new strip and both 1× and 2×
zoomed scroll frames stay within the 33 ms budget.

herdr client CPU is a manual live check (herdr 0.9.1: 15 % with kitty pictures
vs 1 % with `MISOKU_IMAGES=blocks` in the 2026-11-03 method; 0.9.5: 2.0 % / 2.2 %).
The rich view sends at most four tiles, one per screen when the terminal
refuses more, so it never multiplies what herdr has to decode.

## Headless mode

`misoku --headless --note <id> --cols N --rows M --theme T` lays the note out
through the same constructor, writes one PNG per screenful to a folder under
the system temporary directory and prints one JSON line (protocol 1): the run
folder, the note, the size in cells and pixels, and each tile's path, size and
first row. A host draws the tiles and deletes them; `plugins/claude-code/`
documents the pane plugin.
