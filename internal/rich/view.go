package rich

import (
	"errors"
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
	return &View{Fonts: fonts, Palette: pal, Sender: sender, W: max(w, 1), H: max(h, 1), tiles: map[int]*tile{}}
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
	rows := v.H
	if v.degraded {
		rows = v.H
	}
	need := v.neededTiles(rows)
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
	r := &Raster{Page: v.Page, Fonts: v.Fonts, Palette: v.Palette}
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
		v.nextID = 0
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

// Close releases every tile and id.
func (v *View) Close() { v.release() }

func (v *View) release() {
	for first, t := range v.tiles {
		v.releaseTile(first, t)
	}
	v.nextID = 0
}

var _ = strconv.Itoa
