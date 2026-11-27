package rich

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	gtfont "github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Fonts holds the faces the page draws with and the cell grid it lays out in.
// Body text is shaped with go-text/typesetting (kerning, clusters) and drawn
// with x/image/opentype at the same advances, so measuring and drawing agree.
type Fonts struct {
	CellW, CellH int     // one terminal cell, in pixels
	BodyPx       float64 // body size in pixels
	MonoPx       float64

	bodySh, monoSh *gtfont.Face
	bodyDr, monoDr font.Face
	bodyBytes      []byte
	monoBytes      []byte
	boldBytes      []byte
	italicBytes    []byte
	faces          map[faceKey]font.Face
	shaper         shaping.HarfbuzzShaper
}

type faceKey struct {
	size   float64
	mono   bool
	bold   bool
	italic bool
}

// FontConfig says where the faces come from. Empty paths resolve the system's
// monospace and sans-serif fonts; a caller that cannot find them gets an error
// and the rich view falls back to text.
type FontConfig struct {
	BodyPath string
	MonoPath string
	CellW    int
	CellH    int
	BodyPx   float64
	MonoPx   float64
}

// NewFonts builds a Fonts from font bytes; tests use it with the Go fonts.
func NewFonts(body, mono []byte, cfg FontConfig) (*Fonts, error) {
	return NewFontsWithStyles(body, mono, nil, nil, cfg)
}

// NewFontsWithBold is NewFonts with a bold face.
func NewFontsWithBold(body, mono, bold []byte, cfg FontConfig) (*Fonts, error) {
	return NewFontsWithStyles(body, mono, bold, nil, cfg)
}

// NewFontsWithStyles is NewFonts with bold and italic faces. Without a bold
// face, bold text is struck twice one pixel over; without an italic face,
// italic text is drawn upright (rare on the systems this ships for).
func NewFontsWithStyles(body, mono, bold, italic []byte, cfg FontConfig) (*Fonts, error) {
	f := &Fonts{CellW: cfg.CellW, CellH: cfg.CellH, BodyPx: cfg.BodyPx, MonoPx: cfg.MonoPx,
		bodyBytes: body, monoBytes: mono, boldBytes: bold, italicBytes: italic, faces: map[faceKey]font.Face{}}
	if f.CellW <= 0 {
		f.CellW = 12
	}
	if f.CellH <= 0 {
		f.CellH = 24
	}
	if f.BodyPx <= 0 {
		// The body text has to read like the terminal's own: a terminal cell
		// is about 1.2 times its font size, so 0.83 of the cell height is the
		// size the reader sees in the file list beside the page.
		f.BodyPx = math.Round(float64(f.CellH) * 0.83)
		if f.BodyPx < 8 {
			f.BodyPx = 8
		}
	}
	if f.MonoPx <= 0 {
		f.MonoPx = f.BodyPx * 0.85
	}
	var err error
	if f.bodySh, err = gtfont.ParseTTF(bytes.NewReader(body)); err != nil {
		return nil, fmt.Errorf("rich: body font: %w", err)
	}
	if f.monoSh, err = gtfont.ParseTTF(bytes.NewReader(mono)); err != nil {
		return nil, fmt.Errorf("rich: code font: %w", err)
	}
	if f.bodyDr, err = drawFace(body, f.BodyPx); err != nil {
		return nil, err
	}
	if f.monoDr, err = drawFace(mono, f.MonoPx); err != nil {
		return nil, err
	}
	if len(bold) > 0 {
		if f.faces[faceKey{size: f.BodyPx, bold: true}], err = drawFace(bold, f.BodyPx); err != nil {
			return nil, err
		}
	}
	if len(italic) > 0 {
		if f.faces[faceKey{size: f.BodyPx, italic: true}], err = drawFace(italic, f.BodyPx); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// Face returns the body or code face at size px, cached: headings are drawn
// bigger than body text, and shaping already uses the same size.
func (f *Fonts) Face(sizePx float64, mono bool) font.Face {
	if !mono && sizePx == f.BodyPx {
		return f.bodyDr
	}
	if mono && sizePx == f.MonoPx {
		return f.monoDr
	}
	key := faceKey{size: sizePx, mono: mono}
	if face, ok := f.faces[key]; ok {
		return face
	}
	b := f.bodyBytes
	if mono {
		b = f.monoBytes
	}
	face, err := drawFace(b, sizePx)
	if err != nil {
		if mono {
			return f.monoDr
		}
		return f.bodyDr
	}
	f.faces[key] = face
	return face
}

// ItalicFace returns the italic face at size px; ok is false when none was
// loaded, and the caller draws upright.
func (f *Fonts) ItalicFace(sizePx float64) (font.Face, bool) {
	if len(f.italicBytes) == 0 {
		return nil, false
	}
	key := faceKey{size: sizePx, italic: true}
	if face, ok := f.faces[key]; ok {
		return face, true
	}
	face, err := drawFace(f.italicBytes, sizePx)
	if err != nil {
		return nil, false
	}
	f.faces[key] = face
	return face, true
}

// BoldFace returns the bold face at size px. ok is false when no bold face
// was loaded: the raster then draws the body glyph twice.
func (f *Fonts) BoldFace(sizePx float64) (font.Face, bool) {
	if len(f.boldBytes) == 0 {
		return nil, false
	}
	key := faceKey{size: sizePx, bold: true}
	if face, ok := f.faces[key]; ok {
		return face, true
	}
	face, err := drawFace(f.boldBytes, sizePx)
	if err != nil {
		return nil, false
	}
	f.faces[key] = face
	return face, true
}

// LoadFonts finds the faces on this machine, in the order the plan fixes:
// configured paths, the environment, then the system's own discovery.
func LoadFonts(cfg FontConfig) (*Fonts, error) {
	body, mono := cfg.BodyPath, cfg.MonoPath
	if body == "" {
		body = os.Getenv("MISOKU_RICH_FONT")
	}
	if mono == "" {
		mono = os.Getenv("MISOKU_RICH_MONO_FONT")
	}
	if mono == "" {
		mono = systemFont("monospace")
	}
	if body == "" {
		body = systemFont("sans-serif")
	}
	if body == "" || mono == "" {
		return nil, errors.New("rich: no system font found; set ui.rich_font and ui.rich_mono_font")
	}
	bodyBytes, err := os.ReadFile(body)
	if err != nil {
		return nil, fmt.Errorf("rich: body font %s: %w", body, err)
	}
	monoBytes, err := os.ReadFile(mono)
	if err != nil {
		return nil, fmt.Errorf("rich: code font %s: %w", mono, err)
	}
	italic := systemFontStyle("sans-serif", "italic")
	bold := systemFontStyle("sans-serif", "bold")
	italicBytes, _ := os.ReadFile(italic)
	boldBytes, _ := os.ReadFile(bold)
	return NewFontsWithStyles(bodyBytes, monoBytes, boldBytes, italicBytes, cfg)
}

// systemFont asks fontconfig on Linux, and looks in the usual folders on
// macOS and Windows. Bold and italic faces are left to the drawing face's
// synthetic styles; the plan keeps one face per family until the text task.
// systemFontStyle asks fontconfig for a styled face on Linux and falls back
// to the plain family elsewhere (the raster then draws upright or synthetic).
func systemFontStyle(family, style string) string {
	if runtime.GOOS == "linux" {
		if out, err := exec.Command("fc-match", "-f", "%{file}", family+":style="+style).Output(); err == nil {
			p := strings.TrimSpace(string(out))
			if p != "" {
				return p
			}
		}
	}
	return ""
}

func systemFont(family string) string {
	if runtime.GOOS == "linux" {
		if out, err := exec.Command("fc-match", "-f", "%{file}", family).Output(); err == nil {
			p := strings.TrimSpace(string(out))
			if p != "" {
				return p
			}
		}
	}
	var dirs []string
	switch runtime.GOOS {
	case "darwin":
		dirs = []string{"/System/Library/Fonts", "/Library/Fonts", filepath.Join(os.Getenv("HOME"), "Library/Fonts")}
		family = strings.ToLower(family)
		if strings.Contains(family, "mono") {
			return firstExisting(filepath.Join(dirs[0], "SFNSMono.ttf"), filepath.Join(dirs[0], "Menlo.ttc"))
		}
		return firstExisting(filepath.Join(dirs[0], "SFNS.ttf"), filepath.Join(dirs[0], "Helvetica.ttc"))
	case "windows":
		dir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
		if strings.Contains(strings.ToLower(family), "mono") {
			return firstExisting(filepath.Join(dir, "consola.ttf"), filepath.Join(dir, "cour.ttf"))
		}
		return firstExisting(filepath.Join(dir, "segoeui.ttf"), filepath.Join(dir, "arial.ttf"))
	}
	for _, dir := range []string{filepath.Join(os.Getenv("HOME"), ".local/share/fonts"), "/usr/share/fonts", "/usr/local/share/fonts"} {
		_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(p))
			if ext == ".ttf" || ext == ".otf" {
				if strings.Contains(strings.ToLower(filepath.Base(p)), strings.ToLower(family)) {
					if found == "" {
						found = p
					}
				}
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	return ""
}

var found string

func firstExisting(paths ...string) string {
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func drawFace(b []byte, sizePx float64) (font.Face, error) {
	f, err := opentype.Parse(b)
	if err != nil {
		return nil, err
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: sizePx, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}
	return face, nil
}

// Shaped is one positioned glyph: Rune is the rune at the pen, X the offset
// from the line's pen before drawing it, Advance the pen move after it.
type Shaped struct {
	R       rune
	X       fixed.Int26_6
	Advance fixed.Int26_6
}

// Shape shapes s at size px. It is the only measurement path: wrapping and
// drawing both use its advances, so a line never changes width between the
// two.
func (f *Fonts) Shape(s string, sizePx float64, mono bool) []Shaped {
	face, dr := f.bodySh, f.bodyDr
	if mono {
		face, dr = f.monoSh, f.monoDr
	}
	_ = dr
	runes := []rune(s)
	if len(runes) == 0 {
		return nil
	}
	out := f.shaper.Shape(shaping.Input{
		Text: runes, RunStart: 0, RunEnd: len(runes),
		Direction: 0, Face: face,
		Size:     fixed.Int26_6(sizePx * 64),
		Script:   language.Latin,
		Language: language.NewLanguage("en"),
	})
	shaped := make([]Shaped, 0, len(out.Glyphs))
	for _, g := range out.Glyphs {
		i := g.TextIndex()
		if i < 0 || i >= len(runes) {
			i = 0
		}
		shaped = append(shaped, Shaped{R: runes[i], X: g.XOffset, Advance: g.Advance})
	}
	return shaped
}

// Measure is the shaped width of s in 26.6 pixels.
func (f *Fonts) Measure(s string, sizePx float64, mono bool) fixed.Int26_6 {
	var w fixed.Int26_6
	for _, g := range f.Shape(s, sizePx, mono) {
		w += g.Advance
	}
	return w
}

// WithTextScale returns the same faces on the same cell grid with the body and
// mono sizes multiplied by k. A diagram laid out with it needs about k times
// more cells and reads k times bigger: that is how it fills a pane.
func (f *Fonts) WithTextScale(k float64) (*Fonts, error) {
	if k <= 0 {
		k = 1
	}
	cfg := FontConfig{CellW: f.CellW, CellH: f.CellH, BodyPx: f.BodyPx * k, MonoPx: f.MonoPx * k}
	return NewFontsWithStyles(f.bodyBytes, f.monoBytes, f.boldBytes, f.italicBytes, cfg)
}

// MeasureStyled is Measure for a run drawn in the bold or italic face: their
// advances come from that face, the same ones the raster uses to move the pen,
// so the cells a run claims hold its ink. Without a bold or italic face the
// run draws with the body face (synthetic bold keeps the body advances) and
// Measure is exact.
func (f *Fonts) MeasureStyled(s string, sizePx float64, mono, bold, italic bool) int {
	face, styled := f.styledFace(sizePx, bold, italic)
	if !styled {
		w := int(f.Measure(s, sizePx, mono)) / 64
		if bold && !mono && w > 0 {
			// Synthetic bold strikes every glyph one pixel over: claim that
			// pixel so the next run never starts inside the ink.
			w++
		}
		return w
	}
	w := 0
	for _, g := range f.Shape(s, sizePx, mono) {
		step := g.Advance.Round()
		// A styled face without the glyph draws nothing useful and the
		// raster keeps the shaped advance, so the claim must keep it too.
		if adv, ok := face.GlyphAdvance(g.R); ok {
			step = adv.Round()
		}
		w += step
	}
	return w
}

// styledFace is the face a run draws with and whether it differs from the
// shaped body face (only then do advances come from the face).
func (f *Fonts) styledFace(sizePx float64, bold, italic bool) (font.Face, bool) {
	if bold {
		if face, ok := f.BoldFace(sizePx); ok {
			return face, true
		}
		return f.Face(sizePx, false), false
	}
	if italic {
		if face, ok := f.ItalicFace(sizePx); ok {
			return face, true
		}
	}
	return f.Face(sizePx, false), false
}

// Wrap breaks s into lines no wider than maxPx, honouring the note's own
// breaks. A word wider than the line is split by rune, never overflowed.
func (f *Fonts) Wrap(s string, maxPx int, sizePx float64, mono bool) []string {
	var out []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := ""
		for _, w := range words {
			cand := w
			if line != "" {
				cand = line + " " + w
			}
			if f.Measure(cand, sizePx, mono) <= fixed.I(maxPx) {
				line = cand
				continue
			}
			if line != "" {
				out = append(out, line)
			}
			// Split a word that does not fit on a line of its own.
			line = ""
			for _, r := range w {
				cand = line + string(r)
				if line != "" && f.Measure(cand, sizePx, mono) > fixed.I(maxPx) {
					out = append(out, line)
					line = string(r)
					continue
				}
				line = cand
			}
		}
		out = append(out, line)
	}
	return out
}

// LineHeightPx is the height of one text line at size px.
func (f *Fonts) LineHeightPx(sizePx float64, mono bool) float64 {
	m := f.Face(sizePx, mono).Metrics()
	return float64(m.Ascent+m.Descent) / 64
}

// AscentPx is the distance from the line's top to the baseline.
func (f *Fonts) AscentPx(sizePx float64, mono bool) float64 {
	return float64(f.Face(sizePx, mono).Metrics().Ascent) / 64
}
