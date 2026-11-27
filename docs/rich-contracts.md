# Rich reading view — contracts (end of S1)

This is the seam the parallel worktrees build against. It is a contract, not a
tutorial: the shape of each type, who owns it, and what a change to it means.
Code lives in `internal/rich`; the app wiring in `internal/app/richview.go`.

## Pipeline and ownership

```
note text ──rich.Parse──▶ Document ──rich.Layout──▶ Page ──rich.Raster──▶ tiles ──rich.View──▶ placeholder rows
             (doc.go)                 (layout.go)        (raster.go)          (view.go)
```

| Piece | Owner after S1 | May change |
| --- | --- | --- |
| `Document`, `Block`, `Parse` | S1 (`rich-view`) | add fields/block kinds; never change `Line`/`End` meaning |
| `Page`, `Item`, `Box`, `Layout` | S1 (`rich-view`) | **contract**: item kinds and the `Box` seam are frozen for S2–S6 |
| `Fonts` | S1 (`rich-view`) | add faces/sizes; `Shape`/`Measure`/`Wrap` signatures frozen |
| `Raster`, `EncodePNG` | S1 (`rich-view`) | draw more item kinds; `Tile(first, rows)` frozen |
| `View` | A (`rich-view-ui`) | S2/S3 add input, search, TOC here |
| Maths engine | B (`rich-math`) | new `internal/rich/math`; integrates **only** through `Box` |
| Diagram engines | C (`rich-diagram`) | new `internal/rich/diagram`; integrates **only** through `Box` |
| `go.mod`, this file | Claude merges | one owner; writers report needed changes |

## Document (parse)

```go
type Kind int // Paragraph, Heading, ListItem, Quote, Code, Rule, Table, Image, Properties, Math, Diagram
type Block struct {
    Kind   Kind
    Level  int      // heading level 1–6
    Lines  []string // flattened inline text, one entry per source line
    Code   []string // code / math / diagram source
    Lang   string   // fence language
    Rows   [][]string
    Src    string   // image source
    Line, End int   // 0-based source lines, End exclusive
    Blocks []Block  // list items, quote contents
}
type Document struct{ Blocks []Block }
func Parse(src []byte) Document
```

Rules: `Line`/`End` are the note's own lines (front matter and `$$` fences are
line-mapped), so `Page.SourceLine`/`RowOf` keep working. Inline Markdown is
flattened; wiki links become their alias text. `Block.Kind == KindMath` for
`$$…$$` and ```` ```math ````; `KindDiagram` for ```` ```mermaid ```` and
```` ```flow ````.

## Page (layout)

```go
type ItemKind int // ItemText, ItemRect, ItemBox
type Item struct {
    Kind ItemKind
    X, Y, W, H int   // cells from the page's top-left
    Text string
    SizePx float64
    Mono bool
    Thin bool        // rect drawn as a 2 px line
    Color color.RGBA // zero = draw nothing (transparent theme)
    BaselinePx int   // baseline below Y's top edge, in pixels
    Box *Box
}
type Page struct { Items []Item; Width, Height int }
func (p *Page) SourceLine(row int) int
func (p *Page) RowOf(line int) int
func Layout(doc Document, opts LayoutOptions) *Page
```

`LayoutOptions.Math` and `.Diagram` are the **only** integration point for the
engine worktrees:

```go
type Box struct {
    W, H int                                  // cells
    Draw func(dst draw.Image, x, y int)       // pixels; obey the box
}
opts.Math    = func(b Block) *Box { … }       // nil → bordered source placeholder
opts.Diagram = func(b Block) *Box { … }
```

The page clamps `Box.W` to the content width; an engine that needs a taller box
returns the height it wants. `Draw` runs inside `Raster.Tile`, so it must be
side-effect free and must not assume the whole tile belongs to it.

## Fonts and test font

```go
type FontConfig struct{ BodyPath, MonoPath string; CellW, CellH int; BodyPx, MonoPx float64 }
func NewFonts(body, mono []byte, cfg FontConfig) (*Fonts, error) // tests
func LoadFonts(cfg FontConfig) (*Fonts, error)                    // system discovery + MISOKU_RICH_FONT / MISOKU_RICH_MONO_FONT
func (f *Fonts) Shape(s string, sizePx float64, mono bool) []Shaped
func (f *Fonts) Measure(s string, sizePx float64, mono bool) fixed.Int26_6
func (f *Fonts) Wrap(s string, maxPx int, sizePx float64, mono bool) []string
```

Measuring and drawing both use `Shape`'s advances. Tests **must** use
`golang.org/x/image/font/gofont/goregular` + `gomono` through `NewFonts`, never
a system font, so golden pixels are identical on Linux, macOS and Windows.
`internal/rich/fonts_test.go` has `testFonts(t)` for that.

## Raster

```go
type Raster struct{ Page *Page; Fonts *Fonts; Palette Palette }
func (r *Raster) Tile(first, rows int) *image.RGBA // one strip, cell-aligned
func EncodePNG(img *image.RGBA) []byte             // png.BestSpeed
```

Tiles are stateless: a freed tile is drawn again from the page. A golden test
pins `Tile` + `EncodePNG` bytes for the Go fonts.

## View and the terminal

```go
type Sender interface{ Send(seqs ...string) error }
type View struct { /* Fonts, Palette, Page, Sender, W, H, scroll */ }
func NewView(w, h int, f *Fonts, p Palette, s Sender) *View
func (v *View) SetPage(*Page); SetSize(w, h int); Scroll(delta int)
func (v *View) SourceLine() int; GoToLine(line int); MaxScroll() int
func (v *View) Prepare() error   // raster+transmit+cache; call before View()
func (v *View) View() string     // exactly H lines of W cells (placeholders/spaces)
func (v *View) Close()           // delete every tile
var ErrDegraded error            // the terminal refused a tile
```

Rules: `Prepare` before `View` every frame; ids come from `kitty.AcquireID`
(shared with the preview's images) and are released with `kitty.ReleaseID`;
tiles are aligned to the view height, at most `maxTiles` (4) alive, and a
refusal degrades to one screenful and reports `ErrDegraded`. A pane that cannot
draw the rich page shows the text reading view — never blank.

## App wiring (S1)

- `:rich` and `Space V` (`keymap.RichView`) toggle `viewRich`; `Space p`/`Space v`
  keep the editable views. `toggleRichView` refuses without Kitty graphics and
  says why (`AppRichUnavailable`).
- `app.WithImageSender(sender)` is the one output for tiles;
  `app.WithRichFonts(body, mono, cellW, cellH)` overrides discovery.
  `main.go` passes the chain's `kitty.Writer` and `preview.GraphicsOK()` gates
  availability.
- `refreshRich` rebuilds the page on open, note change, resize and theme change;
  `prepareRich` runs after every update and falls back to `viewReading` if a
  tile is refused. Typing does **not** rebuild yet (`ui.rich_view` and the
  debounced re-render are the UI worktree's task).

## What S2–S6 may not do

- Change `Item`/`Box`/`Sender`/`View` signatures without asking Claude (this
  file is edited only at merge time).
- Import `internal/rich` from `internal/preview` or vice versa.
- Add a second image-id allocator or a second output path.
- Use a system font in a test.
