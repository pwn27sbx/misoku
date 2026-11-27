package rich

import (
	"errors"
	"image/color"
	"strconv"
	"strings"

	"github.com/pwn27sbx/misoku/internal/kitty"
)

// Sender is the one output images go through, so a transmit never splits a
// frame (the app's chain provides it).
type Sender interface {
	Send(seqs ...string) error
}

// ErrDegraded means the terminal refused an image: the view keeps drawing the
// screenful it can and reports why, instead of showing a half-scrolled page.
var ErrDegraded = errors.New("rich: terminal refused a tile; one tile per screen")

// View shows a Page as placeholder cells. It owns scroll in cells, the tile
// cache and its quota, and every transmit. It draws nothing itself: View()
// returns exactly H lines of W cells, and a pane that cannot show it must fall
// back to text for that frame.
type View struct {
	Fonts   *Fonts
	Palette Palette
	Page    *Page
	Sender  Sender

	W, H   int
	scroll int

	nextID   int
	tiles    map[int]*tile // key: first row
	degraded bool

	wideOff  int // cells the wide blocks are scrolled sideways
	hints    map[string]Hint
	hintOn   bool
	overlay  []Item // search marks and selection, drawn after the page
	search   []int  // matched rows
	searchAt int
	selA     int
	selB     int
}

type tile struct {
	id    int
	first int
	rows  int
	cols  int
}

const maxTiles = 4

// NewView makes an empty view for a pane of w×h cells.
func NewView(w, h int, fonts *Fonts, pal Palette, sender Sender) *View {
	return &View{Fonts: fonts, Palette: pal, Sender: sender, W: max(w, 1), H: max(h, 1),
		tiles: map[int]*tile{}, searchAt: -1, selA: -1, selB: -1}
}

// SetSize changes the pane size; tiles no longer matching the grid are
// released and the scroll stays inside the page.
func (v *View) SetSize(w, h int) {
	if w == v.W && h == v.H {
		return
	}
	v.W, v.H = max(w, 1), max(h, 1)
	v.release()
	v.clamp()
}

// SetPage shows a new page, releasing the old tiles.
func (v *View) SetPage(p *Page) {
	v.Page = p
	v.release()
	v.clamp()
}

// Ready reports whether there is a page to draw. False means the caller draws
// the text reading view instead.
func (v *View) Ready() bool { return v.Page != nil && v.Fonts != nil && v.Sender != nil }

// Degraded reports that the terminal refused a tile and the view fell back to
// one image per screen.
func (v *View) Degraded() bool { return v.degraded }

// Scroll moves the view by delta cells, clamped to the page.
func (v *View) Scroll(delta int) {
	v.scroll += delta
	v.clamp()
}

// ScrollTop is the first visible cell row.
func (v *View) ScrollTop() int { return v.scroll }

// MaxScroll is the last possible scroll position.
func (v *View) MaxScroll() int {
	if v.Page == nil {
		return 0
	}
	return max(v.Page.Height-v.H, 0)
}

func (v *View) clamp() {
	if v.scroll > v.MaxScroll() {
		v.scroll = v.MaxScroll()
	}
	if v.scroll < 0 {
		v.scroll = 0
	}
}

// SourceLine is the note line at the top of the pane.
func (v *View) SourceLine() int {
	if v.Page == nil {
		return 0
	}
	return v.Page.SourceLine(v.scroll)
}

// GoToLine scrolls so the block that starts at line is at the top.
func (v *View) GoToLine(line int) {
	if v.Page != nil {
		v.scroll = v.Page.RowOf(line)
		v.clamp()
	}
}

// Prepare rasterises, transmits and caches the tiles the pane needs. It must
// run before View, so placeholders never name an image the terminal does not
// hold.
func (v *View) Prepare() error {
	if !v.Ready() {
		return nil
	}
	// Tiles are one screenful and aligned to the view height, so a degraded
	// terminal and a healthy one ask for the same single screenful; the flag
	// only says why a tile was refused.
	need := v.neededTiles(v.H)
	// Evict farthest tiles first, keeping the visible ones.
	for first, t := range v.tiles {
		if _, ok := need[first]; ok && len(v.tiles) <= maxTiles {
			continue
		}
		if _, ok := need[first]; ok {
			continue
		}
		v.releaseTile(first, t)
	}
	var firstErr error
	for first, rowCount := range need {
		if _, ok := v.tiles[first]; ok {
			continue
		}
		if err := v.makeTile(first, rowCount); err != nil {
			firstErr = err
			break
		}
	}
	return firstErr
}

// neededTiles says which tiles the frame shows and how many rows each holds.
// Tiles are aligned to the view height so a scroll only adds one at the edge.
func (v *View) neededTiles(rows int) map[int]int {
	need := map[int]int{}
	if v.Page == nil {
		return need
	}
	last := max(v.Page.Height, 1)
	for first := (v.scroll / rows) * rows; first < v.scroll+rows && first < last; first += rows {
		if first < 0 {
			continue
		}
		n := min(rows, last-first)
		need[first] = n
	}
	return need
}

func (v *View) makeTile(first, rows int) error {
	r := &Raster{Page: v.Page, Fonts: v.Fonts, Palette: v.Palette, Overlay: v.overlay, OffX: v.wideOff}
	img := r.Tile(first, rows)
	png := EncodePNG(img)
	if len(png) == 0 {
		return nil
	}
	cols := min(v.Page.Width, v.W)
	id, ok := v.acquireID()
	if !ok {
		return ErrDegraded
	}
	if err := v.Sender.Send(kitty.Transmit(id, cols, rows, png)...); err != nil {
		kitty.ReleaseID(id) // the transmit never landed: the id is free again
		v.degraded = true
		v.release()
		return ErrDegraded
	}
	v.tiles[first] = &tile{id: id, first: first, rows: rows, cols: cols}
	return nil
}

// acquireID asks the shared kitty pool for an id, so rich tiles and the
// preview's pictures never collide.
func (v *View) acquireID() (int, bool) {
	id, ok := kitty.AcquireID()
	return id, ok
}

func (v *View) releaseTile(first int, t *tile) {
	if v.Sender != nil {
		_ = v.Sender.Send(kitty.Delete(t.id))
	}
	kitty.ReleaseID(t.id)
	delete(v.tiles, first)
}

// View draws exactly H lines of W cells: placeholder cells where a tile is
// cached, spaces elsewhere. Call Prepare first.
func (v *View) View() string {
	lines := make([]string, v.H)
	for i := range lines {
		lines[i] = strings.Repeat(" ", v.W)
	}
	if v.Page == nil {
		return strings.Join(lines, "\n")
	}
	for first, t := range v.tiles {
		if first > v.scroll+v.H || first+t.rows <= v.scroll {
			continue
		}
		ph := kitty.Placeholders(t.id, t.cols, t.rows)
		for k := 0; k < t.rows; k++ {
			screenRow := first + k - v.scroll
			if screenRow < 0 || screenRow >= v.H || k >= len(ph) {
				continue
			}
			lines[screenRow] = padCells(ph[k], t.cols, v.W)
		}
	}
	return strings.Join(lines, "\n")
}

// invalidateAll frees every cached tile: an overlay change that spans the page
// (a search) redraws them all on the next Prepare.
func (v *View) invalidateAll() {
	for first, t := range v.tiles {
		v.releaseTile(first, t)
	}
}

// invalidateRows frees the tiles that hold the given cell rows, so a selection
// redraws only what it covers.
func (v *View) invalidateRows(a, b int) {
	if a > b {
		a, b = b, a
	}
	for first, t := range v.tiles {
		if first <= b && a < first+t.rows {
			v.releaseTile(first, t)
		}
	}
}

// colorAlpha is c with its alpha replaced: overlays tint, they never hide the
// text they mark.
func colorAlpha(c color.RGBA, a uint8) color.RGBA { c.A = a; return c }

// padCells keeps a placeholder row exactly w cells wide. The row's own ANSI
// reset closes the image colour before the padding.
func padCells(s string, cols, w int) string {
	if cols >= w {
		return clipCells(s, w)
	}
	return s + strings.Repeat(" ", w-cols)
}

// clipCells cuts an escaped placeholder row to w cells, counting the private
// use runes only.
func clipCells(s string, w int) string {
	cells := 0
	for i, r := range s {
		if r == '\U0010EEEE' {
			if cells == w {
				return s[:i]
			}
			cells++
		}
	}
	return s
}

// Search marks every row of the page that contains q, ignoring case, and
// scrolls to the first match. It returns how many rows matched.
func (v *View) Search(q string) int {
	v.search = nil
	v.searchAt = -1
	v.overlay = nil
	if v.Page == nil || strings.TrimSpace(q) == "" {
		return 0
	}
	needle := strings.ToLower(q)
	for _, it := range v.Page.Items {
		if it.Kind != ItemText || !strings.Contains(strings.ToLower(it.Text), needle) {
			continue
		}
		if len(v.search) > 0 && v.search[len(v.search)-1] == it.Y {
			continue
		}
		v.search = append(v.search, it.Y)
		v.overlay = append(v.overlay, Item{Kind: ItemRect, X: it.X, Y: it.Y, W: it.W, H: max(it.H, 1),
			Color: colorAlpha(v.Palette.Stripe, 0x60)})
	}
	if len(v.search) == 0 {
		return 0
	}
	v.searchAt = 0
	v.scroll = max(v.search[0]-v.H/3, 0)
	v.clamp()
	v.invalidateAll()
	return len(v.search)
}

// SearchNext and SearchPrev move between matches, wrapping.
func (v *View) SearchNext(delta int) bool {
	if len(v.search) == 0 {
		return false
	}
	v.searchAt = (v.searchAt + delta + len(v.search)) % len(v.search)
	v.scroll = max(v.search[v.searchAt]-v.H/3, 0)
	v.clamp()
	return true
}

// Outline is the page's headings, for a table of contents.
func (v *View) Outline() []Heading {
	if v.Page == nil {
		return nil
	}
	return v.Page.Headings
}

// Select marks cell rows [a, b] (in either order) and returns the text they
// hold, so copy works on the rendered page.
func (v *View) Select(a, b int) string {
	if v.Page == nil {
		return ""
	}
	if a > b {
		a, b = b, a
	}
	a = max(a, 0)
	b = min(b, max(v.Page.Height-1, 0))
	// A selection replaces the search marks: keeping them would leave
	// matches on screen that n/N no longer address.
	v.search = nil
	v.searchAt = -1
	v.selA, v.selB = a, b
	defer v.invalidateAll()
	v.overlay = nil
	v.overlay = append(v.overlay, Item{Kind: ItemRect, X: 0, Y: a, W: v.Page.Width, H: b - a + 1,
		Color: colorAlpha(v.Palette.Link, 0x40)})
	return v.textRows(a, b)
}

// textRows is the text of cell rows a..b, in order, without touching the
// selection or the overlay.
func (v *View) textRows(a, b int) string {
	var lines []string
	for row := a; row <= b; row++ {
		var text strings.Builder
		for _, it := range v.Page.Items {
			if it.Kind == ItemText && it.Y == row {
				if text.Len() > 0 {
					text.WriteString(" ")
				}
				text.WriteString(it.Text)
			}
		}
		lines = append(lines, text.String())
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

// SelectionRows is the selected row range, or (-1,-1).
func (v *View) SelectionRows() (int, int) { return v.selA, v.selB }

// ClearSelection drops the selection and the search marks.
func (v *View) ClearSelection() {
	a, b := v.selA, v.selB
	v.selA, v.selB = -1, -1
	v.search = nil
	v.searchAt = -1
	v.overlay = nil
	if a >= 0 || b >= 0 {
		v.invalidateAll()
	}
}

// Hint is one labelled link the page shows while the hint key is on.
type Hint struct {
	Label  string
	Row    int
	X0, X1 int
	Target string
}

// ShowHints labels every link the pane shows with a letter, draws the labels
// over the page and returns them. Typing a letter follows that link.
func (v *View) ShowHints() []Hint {
	v.ClearHints()
	if v.Page == nil || len(v.Page.Links) == 0 {
		return nil
	}
	v.hints = map[string]Hint{}
	v.hintOn = true
	var out []Hint
	for _, l := range v.Page.Links {
		if l.Row < v.scroll || l.Row >= v.scroll+v.H {
			continue
		}
		label := string(rune('a' + len(out)))
		if len(out) >= 26 {
			break
		}
		h := Hint{Label: label, Row: l.Row, X0: l.X0, X1: l.X1, Target: l.Target}
		out = append(out, h)
		v.hints[label] = h
		v.overlay = append(v.overlay,
			Item{Kind: ItemRect, X: l.X0, Y: l.Row, W: max(l.X1-l.X0, 1), H: 1, Color: colorAlpha(v.Palette.Link, 0xC0)},
			Item{Kind: ItemText, X: l.X0, Y: l.Row, W: 1, H: 1, Text: label,
				SizePx: v.Fonts.BodyPx, Color: color.RGBA{R: 255, G: 255, B: 255, A: 255}, BaselinePx: v.Fonts.CellH * 3 / 4},
		)
	}
	v.invalidateAll()
	return out
}

// HintFor is the link a typed letter names, while hints are on.
func (v *View) HintFor(label string) (Hint, bool) {
	h, ok := v.hints[label]
	return h, ok
}

// HintsOn reports whether the labels are showing.
func (v *View) HintsOn() bool { return v.hintOn }

// ClearHints drops the labels and redraws the page without them.
func (v *View) ClearHints() {
	if v.hintOn || len(v.hints) > 0 {
		v.invalidateAll()
	}
	v.hintOn = false
	v.hints = nil
	v.overlay = nil
}

// WideScroll moves the page's wide blocks sideways and redraws them.
func (v *View) WideScroll(delta int) {
	if v.Page == nil || len(v.Page.Wide) == 0 {
		return
	}
	limit := 0
	for _, it := range v.Page.Items {
		if !wideRow(v.Page, it.Y) {
			continue
		}
		if end := it.X + it.W - v.W + 2; end > limit {
			limit = end
		}
	}
	v.wideOff = min(max(v.wideOff+delta, 0), max(limit, 0))
	v.invalidateAll()
}

// WideOffset is how far the wide blocks are scrolled, in cells.
func (v *View) WideOffset() int { return v.wideOff }

// PageLine is the note line a page row belongs to.
func (v *View) PageLine(row int) int {
	if v.Page == nil {
		return 0
	}
	return v.Page.SourceLine(row)
}

// VisibleText is the text of the rows the pane shows, for a copy without a
// selection.
func (v *View) VisibleText() string {
	if v.Page == nil {
		return ""
	}
	return v.textRows(v.scroll, min(v.scroll+v.H-1, v.Page.Height-1))
}

// RowAt maps a pane row (0 is the top of the view) to the page's cell row.
func (v *View) RowAt(screenRow int) int { return v.scroll + screenRow }

// Close releases every tile and id.
func (v *View) Close() { v.release() }

func (v *View) release() {
	for first, t := range v.tiles {
		v.releaseTile(first, t)
	}
	v.nextID = 0
}

var _ = strconv.Itoa
