package math

import (
	"encoding/binary"
	"sync"

	"golang.org/x/image/font/sfnt"
)

// The OpenType MATH table: TeX's sizes for delimiters, radicals and large
// operators. A viewer that only scales the base glyph draws a stretched
// parenthesis as a blurred small one; the font already ships the real shapes,
// so the engine asks for them by glyph id.

type glyphVariant struct {
	gid sfnt.GlyphIndex
}

var (
	mathOnce     sync.Once
	mathVert     map[sfnt.GlyphIndex][]sfnt.GlyphIndex
	mathHoriz    map[sfnt.GlyphIndex][]sfnt.GlyphIndex
	mathTableErr error
)

// variantGlyphs returns the vertical size variants the font supplies for r,
// smallest first. An empty slice means the face has none and the caller falls
// back to scaling.
func variantGlyphs(r rune) []sfnt.GlyphIndex {
	mathOnce.Do(loadMathVariants)
	otf, err := mathFont()
	if err != nil {
		return nil
	}
	var buf sfnt.Buffer
	gid, err := otf.GlyphIndex(&buf, r)
	if err != nil || gid == 0 {
		return nil
	}
	return mathVert[gid]
}

func horizontalVariants(r rune) []sfnt.GlyphIndex {
	mathOnce.Do(loadMathVariants)
	otf, err := mathFont()
	if err != nil {
		return nil
	}
	var buf sfnt.Buffer
	gid, err := otf.GlyphIndex(&buf, r)
	if err != nil || gid == 0 {
		return nil
	}
	return mathHoriz[gid]
}

// loadMathVariants parses the embedded face's MATH table once. Malformed or
// missing data leaves the maps empty and the engine keeps its scaling
// fallback; it never fails a formula.
func loadMathVariants() {
	mathVert = map[sfnt.GlyphIndex][]sfnt.GlyphIndex{}
	mathHoriz = map[sfnt.GlyphIndex][]sfnt.GlyphIndex{}
	table, ok := sfntTable(fontData, "MATH")
	if !ok || len(table) < 10 {
		return
	}
	u16 := func(off int) int {
		if off < 0 || off+2 > len(table) {
			return -1
		}
		return int(binary.BigEndian.Uint16(table[off:]))
	}
	mathVariantsOff := u16(8)
	if mathVariantsOff <= 0 || mathVariantsOff >= len(table) {
		return
	}
	mv := mathVariantsOff
	minOverlap := u16(mv) // reserved
	_ = minOverlap
	vertCovOff := u16(mv + 2)
	horizCovOff := u16(mv + 4)
	vertCount := u16(mv + 6)
	horizCount := u16(mv + 8)
	if vertCount < 0 || horizCount < 0 {
		return
	}
	vertOffsets := mv + 10
	horizOffsets := vertOffsets + 2*vertCount
	read := func(covOff, count, offsets int, dst map[sfnt.GlyphIndex][]sfnt.GlyphIndex) {
		if covOff <= 0 || count <= 0 || offsets+2*count > len(table) {
			return
		}
		cov := mv + covOff
		coverIndex := coverageIndex(table, cov)
		for i := 0; i < count; i++ {
			off := u16(offsets + 2*i)
			if off <= 0 || mv+off+4 > len(table) {
				continue
			}
			con := mv + off
			n := u16(con + 2)
			if n <= 0 || con+4+4*n > len(table) {
				continue
			}
			var variants []sfnt.GlyphIndex
			for j := 0; j < n; j++ {
				g := u16(con + 4 + 4*j)
				if g > 0 {
					variants = append(variants, sfnt.GlyphIndex(g))
				}
			}
			for gid, idx := range coverIndex {
				if idx == i {
					dst[gid] = variants
				}
			}
		}
	}
	read(vertCovOff, vertCount, vertOffsets, mathVert)
	read(horizCovOff, horizCount, horizOffsets, mathHoriz)
}

// coverageIndex maps glyph id to its coverage index for format 1 and 2
// coverage tables.
func coverageIndex(table []byte, off int) map[sfnt.GlyphIndex]int {
	out := map[sfnt.GlyphIndex]int{}
	if off+4 > len(table) {
		return out
	}
	format := binary.BigEndian.Uint16(table[off:])
	n := int(binary.BigEndian.Uint16(table[off+2:]))
	switch format {
	case 1:
		for i := 0; i < n; i++ {
			p := off + 4 + 2*i
			if p+2 > len(table) {
				return out
			}
			out[sfnt.GlyphIndex(binary.BigEndian.Uint16(table[p:]))] = i
		}
	case 2:
		for i := 0; i < n; i++ {
			p := off + 4 + 6*i
			if p+6 > len(table) {
				return out
			}
			start := int(binary.BigEndian.Uint16(table[p:]))
			end := int(binary.BigEndian.Uint16(table[p+2:]))
			base := int(binary.BigEndian.Uint16(table[p+4:]))
			for g := start; g <= end; g++ {
				out[sfnt.GlyphIndex(g)] = base + (g - start)
			}
		}
	}
	return out
}

// sfntTable finds one table in a raw font file.
func sfntTable(data []byte, tag string) ([]byte, bool) {
	if len(data) < 12 {
		return nil, false
	}
	n := int(binary.BigEndian.Uint16(data[4:]))
	for i := 0; i < n; i++ {
		p := 12 + 16*i
		if p+16 > len(data) {
			break
		}
		if string(data[p:p+4]) != tag {
			continue
		}
		off := int(binary.BigEndian.Uint32(data[p+8:]))
		length := int(binary.BigEndian.Uint32(data[p+12:]))
		if off < 0 || length < 0 || off+length > len(data) {
			return nil, false
		}
		return data[off : off+length], true
	}
	return nil, false
}
