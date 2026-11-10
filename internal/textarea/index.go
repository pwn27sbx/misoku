package textarea

import (
	"encoding/binary"
	"hash/maphash"
	"sort"
	"sync/atomic"
	"unsafe"
)

// misoku: a cached index of soft-wrapped rows, so mapping between visual
// rows and text positions is a binary search instead of wrapping the whole
// note on every key press.

// versions hands out text versions. It is global so two copies of a Model
// edited apart never end up with the same version for different text.
var versions atomic.Uint64

// rowIndex holds, for the text version, width and display it was built for,
// the first visual row of every logical line, then the row past the text; the
// last element is the total row count, which a display's footer adds to.
type rowIndex struct {
	version    uint64
	width      int
	generation uint64
	starts     []int
}

// touch records that the text changed. Every write to m.value must call it
// after the write and before anything reads the row index.
func (m *Model) touch() { m.version = versions.Add(1) }

// Version identifies the current text: it changes on every edit and stays
// the same while only the cursor or the view moves.
func (m Model) Version() uint64 { return m.version }

// rows returns the row index for the current text, width and display,
// rebuilding it only when one changed since it was last built.
func (m Model) rows() []int {
	idx := m.index
	if idx == nil {
		idx = &rowIndex{}
	}
	generation := m.displayGeneration()
	if idx.starts != nil && idx.version == m.version && idx.width == m.width && idx.generation == generation {
		return idx.starts
	}
	// A fresh slice, so a caller still holding the old one is unaffected.
	starts := make([]int, 0, len(m.value)+2)
	acc := 0
	for i, l := range m.value {
		starts = append(starts, acc)
		if d := m.displayed(i); d != nil {
			acc += len(d.Rows)
		} else {
			acc += len(m.memoizedWrap(l, m.width))
		}
	}
	starts = append(starts, acc)
	if d := m.activeDisplay(); d != nil && len(d.Footer) > 0 {
		starts = append(starts, acc+len(d.Footer))
	}
	idx.starts = starts
	idx.version, idx.width, idx.generation = m.version, m.width, generation
	return idx.starts
}

// lineAtRow returns the logical line that visual row v belongs to, or
// len(m.value) when v is past the last row.
func (m Model) lineAtRow(v int) int {
	starts := m.rows()
	return sort.Search(len(m.value), func(r int) bool { return starts[r+1] > v })
}

var wrapSeed = maphash.MakeSeed()

// Hash returns the memo key of a line: a 64-bit hash of its runes plus its
// length and the wrap width.
func (w line) Hash() string {
	var b []byte
	if len(w.runes) > 0 {
		b = unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(w.runes))), len(w.runes)*4)
	}
	var k [24]byte
	binary.LittleEndian.PutUint64(k[0:], maphash.Bytes(wrapSeed, b))
	binary.LittleEndian.PutUint64(k[8:], uint64(len(w.runes)))
	binary.LittleEndian.PutUint64(k[16:], uint64(w.width))
	return string(k[:])
}
