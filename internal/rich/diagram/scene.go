package diagram

import (
	"image/color"
	"math"
	"strings"

	"github.com/pwn27sbx/misoku/internal/rich"
)

// CommandKind identifies one native vector primitive.
type CommandKind int

const (
	// CommandRect draws a rectangle.
	CommandRect CommandKind = iota
	// CommandRounded draws a rounded rectangle.
	CommandRounded
	// CommandEllipse draws an ellipse.
	CommandEllipse
	// CommandDiamond draws a diamond.
	CommandDiamond
	// CommandHexagon draws a hexagon.
	CommandHexagon
	// CommandParallelogram draws a parallelogram.
	CommandParallelogram
	// CommandPolygon draws an arbitrary closed polygon.
	CommandPolygon
	// CommandLine draws a line.
	CommandLine
	// CommandArrow draws a line with optional endpoint decorations.
	CommandArrow
	// CommandPolyline draws an open polyline.
	CommandPolyline
	// CommandSector draws a filled pie sector.
	CommandSector
	// CommandText draws a shaped text run.
	CommandText
	// CommandBadge draws a small labelled pill.
	CommandBadge
)

// TextAlign controls text placement inside its layout rectangle.
type TextAlign int

const (
	// AlignLeft aligns text to the left edge.
	AlignLeft TextAlign = iota
	// AlignCenter centers text horizontally and vertically.
	AlignCenter
	// AlignRight aligns text to the right edge.
	AlignRight
)

// Point is a vertex in cell coordinates.
type Point struct{ X, Y int }

// Command is one layout-independent vector instruction in cell units.
type Command struct {
	Kind   CommandKind
	X, Y   int
	W, H   int
	X2, Y2 int
	Points []Point
	Text   string
	SizePx float64
	Mono   bool
	Align  TextAlign
	Color  color.RGBA
	Fill   color.RGBA
	Stroke color.RGBA
	Radius int
	Slant  int
	Dashed bool
	Thick  bool
	Start  float64
	End    float64
	Head   Arrowhead
	Tail   Arrowhead
}

// Scene is a diagram laid out in terminal-cell coordinates.
type Scene struct {
	Width, Height int
	Commands      []Command
}

func newScene() Scene { return Scene{Width: 1, Height: 1} }

// Rect appends a filled or stroked rectangle.
func (s *Scene) Rect(x, y, w, h int, fill, stroke color.RGBA) {
	s.Commands = append(s.Commands, Command{Kind: CommandRect, X: x, Y: y, W: w, H: h, Fill: fill, Stroke: stroke})
	s.grow(x, y, w, h)
}

// Rounded appends a rounded rectangle.
func (s *Scene) Rounded(x, y, w, h, radius int, fill, stroke color.RGBA) {
	s.Commands = append(s.Commands, Command{Kind: CommandRounded, X: x, Y: y, W: w, H: h, Radius: radius, Fill: fill, Stroke: stroke})
	s.grow(x, y, w, h)
}

// Ellipse appends an ellipse.
func (s *Scene) Ellipse(x, y, w, h int, fill, stroke color.RGBA) {
	s.Commands = append(s.Commands, Command{Kind: CommandEllipse, X: x, Y: y, W: w, H: h, Fill: fill, Stroke: stroke})
	s.grow(x, y, w, h)
}

// Diamond appends a diamond.
func (s *Scene) Diamond(x, y, w, h int, fill, stroke color.RGBA) {
	s.Commands = append(s.Commands, Command{Kind: CommandDiamond, X: x, Y: y, W: w, H: h, Fill: fill, Stroke: stroke})
	s.grow(x, y, w, h)
}

// Hexagon appends a hexagon.
func (s *Scene) Hexagon(x, y, w, h int, fill, stroke color.RGBA) {
	s.Commands = append(s.Commands, Command{Kind: CommandHexagon, X: x, Y: y, W: w, H: h, Fill: fill, Stroke: stroke})
	s.grow(x, y, w, h)
}

// Parallelogram appends a parallelogram. A negative slant mirrors it.
func (s *Scene) Parallelogram(x, y, w, h, slant int, fill, stroke color.RGBA) {
	s.Commands = append(s.Commands, Command{Kind: CommandParallelogram, X: x, Y: y, W: w, H: h, Slant: slant, Fill: fill, Stroke: stroke})
	s.grow(x, y, w, h)
}

// Polygon appends a closed polygon.
func (s *Scene) Polygon(points []Point, fill, stroke color.RGBA) {
	s.Commands = append(s.Commands, Command{Kind: CommandPolygon, Points: append([]Point(nil), points...), Fill: fill, Stroke: stroke})
	for _, p := range points {
		s.grow(p.X, p.Y, 1, 1)
	}
}

// Line appends a straight line.
func (s *Scene) Line(x1, y1, x2, y2 int, stroke color.RGBA, dashed, thick bool) {
	s.Commands = append(s.Commands, Command{Kind: CommandLine, X: x1, Y: y1, X2: x2, Y2: y2, Color: stroke, Dashed: dashed, Thick: thick})
	s.grow(min(x1, x2), min(y1, y2), abs(x2-x1)+1, abs(y2-y1)+1)
}

// Arrow appends a connector with endpoint decorations.
func (s *Scene) Arrow(x1, y1, x2, y2 int, stroke color.RGBA, style EdgeStyle, head, tail Arrowhead) {
	if style == EdgeInvisible {
		return
	}
	s.Commands = append(s.Commands, Command{Kind: CommandArrow, X: x1, Y: y1, X2: x2, Y2: y2, Color: stroke,
		Dashed: style == EdgeDotted, Thick: style == EdgeThick, Head: head, Tail: tail})
	s.grow(min(x1, x2)-1, min(y1, y2)-1, abs(x2-x1)+3, abs(y2-y1)+3)
}

// Polyline appends an open polyline.
func (s *Scene) Polyline(points []Point, stroke color.RGBA, dashed, thick bool) {
	s.Commands = append(s.Commands, Command{Kind: CommandPolyline, Points: append([]Point(nil), points...), Color: stroke, Dashed: dashed, Thick: thick})
	for _, p := range points {
		s.grow(p.X, p.Y, 1, 1)
	}
}

// Sector appends a filled circular sector. Angles are radians, zero at three
// o'clock, and positive angles move clockwise in screen coordinates.
func (s *Scene) Sector(x, y, w, h int, start, end float64, fill color.RGBA) {
	s.Commands = append(s.Commands, Command{Kind: CommandSector, X: x, Y: y, W: w, H: h, Start: start, End: end, Fill: fill})
	s.grow(x, y, w, h)
}

// Text appends one text line inside the given cell rectangle.
func (s *Scene) Text(text string, x, y, w, h int, size float64, align TextAlign, col color.RGBA, mono bool) {
	s.Commands = append(s.Commands, Command{Kind: CommandText, X: x, Y: y, W: w, H: h, Text: text, SizePx: size, Align: align, Color: col, Mono: mono})
	s.grow(x, y, w, h)
}

// TextLines appends wrapped lines inside the given cell rectangle.
func (s *Scene) TextLines(lines []string, x, y, w, lineHeight int, size float64, align TextAlign, col color.RGBA, mono bool) {
	for i, line := range lines {
		s.Text(line, x, y+i*lineHeight, w, lineHeight, size, align, col, mono)
	}
}

// Badge appends a small labelled pill.
func (s *Scene) Badge(text string, x, y, w, h int, size float64, fill, col color.RGBA) {
	s.Commands = append(s.Commands, Command{Kind: CommandBadge, X: x, Y: y, W: w, H: h, Text: text, SizePx: size, Fill: fill, Color: col, Align: AlignCenter})
	s.grow(x, y, w, h)
}

// Node appends a graph node and its centred label.
func (s *Scene) Node(n Node, x, y, w, h int, fonts *rich.Fonts, palette rich.Palette, fill, stroke color.RGBA) {
	switch n.Shape {
	case ShapeRounded:
		s.Rounded(x, y, w, h, 1, fill, stroke)
	case ShapeStadium:
		s.Rounded(x, y, w, h, min(w, h)/2, fill, stroke)
	case ShapeSubroutine:
		s.Rect(x, y, w, h, fill, stroke)
		s.Line(x+1, y, x+1, y+h-1, stroke, false, false)
		s.Line(x+w-2, y, x+w-2, y+h-1, stroke, false, false)
	case ShapeCylinder:
		s.Ellipse(x, y, w, 2, fill, stroke)
		s.Rect(x, y+1, w, max(h-2, 1), fill, stroke)
		s.Line(x, y+1, x, y+h-2, stroke, false, false)
		s.Line(x+w-1, y+1, x+w-1, y+h-2, stroke, false, false)
		s.Ellipse(x, y+h-2, w, 2, fill, stroke)
	case ShapeCircle:
		s.Ellipse(x, y, w, h, fill, stroke)
	case ShapeDiamond:
		s.Diamond(x, y, w, h, fill, stroke)
	case ShapeHexagon:
		s.Hexagon(x, y, w, h, fill, stroke)
	case ShapeParallelogram:
		s.Parallelogram(x, y, w, h, 2, fill, stroke)
	case ShapeParallelogramAlt:
		s.Parallelogram(x, y, w, h, -2, fill, stroke)
	case ShapeDoubleCircle:
		s.Ellipse(x, y, w, h, fill, stroke)
		s.Ellipse(x+1, y+1, max(w-2, 1), max(h-2, 1), fill, stroke)
	case ShapeAsymmetric:
		s.Polygon([]Point{{X: x, Y: y}, {X: x + w - 2, Y: y}, {X: x + w, Y: y + h/2}, {X: x + w - 2, Y: y + h - 1}, {X: x, Y: y + h - 1}}, fill, stroke)
	default:
		s.Rect(x, y, w, h, fill, stroke)
	}
	label := n.Label
	if label == "" {
		label = n.ID
	}
	pad := 2
	if n.Shape == ShapeDiamond {
		pad = max(w/3, 2)
	}
	lines, _, lineH := labelSize(label, fonts, max(w-2*pad, 3), fonts.BodyPx, false)
	textH := len(lines) * lineH
	s.TextLines(lines, x+pad, y+(h-textH)/2, w-2*pad, lineH, fonts.BodyPx, AlignCenter, foreground(palette), false)
}

func labelSize(label string, fonts *rich.Fonts, maxCells int, size float64, mono bool) ([]string, int, int) {
	if fonts == nil {
		return []string{label}, max(len([]rune(label)), 1), 1
	}
	if maxCells < 3 {
		maxCells = 3
	}
	lines := fonts.Wrap(strings.TrimSpace(label), maxCells*fonts.CellW, size, mono)
	if len(lines) == 0 {
		lines = []string{""}
	}
	width := 1
	for _, line := range lines {
		// rich.Fonts.Measure returns fixed.Int26_6, so 64 units equal one
		// pixel. Convert pixels to cells before adding edge padding.
		pixels := int(math.Ceil(float64(fonts.Measure(line, size, mono)) / 64))
		width = max(width, (pixels+fonts.CellW-1)/fonts.CellW+1)
	}
	lineH := max(int(math.Ceil(fonts.LineHeightPx(size, mono)/float64(fonts.CellH))), 1)
	return lines, min(width, maxCells), lineH
}

func (s *Scene) grow(x, y, w, h int) {
	if w < 0 {
		x += w
		w = -w
	}
	if h < 0 {
		y += h
		h = -h
	}
	s.Width = max(s.Width, x+w)
	s.Height = max(s.Height, y+h)
}

func foreground(p rich.Palette) color.RGBA {
	if p.Fg.A != 0 {
		return p.Fg
	}
	return color.RGBA{R: 220, G: 220, B: 220, A: 255}
}

func muted(p rich.Palette) color.RGBA {
	if p.Muted.A != 0 {
		return p.Muted
	}
	return foreground(p)
}

func rule(p rich.Palette) color.RGBA {
	if p.Rule.A != 0 {
		return p.Rule
	}
	return muted(p)
}

func accent(p rich.Palette, index int) color.RGBA {
	if len(p.Heading) == 0 {
		return foreground(p)
	}
	i := index % len(p.Heading)
	if i < 0 {
		i += len(p.Heading)
	}
	if p.Heading[i].A != 0 {
		return p.Heading[i]
	}
	return foreground(p)
}

func fade(c color.RGBA, alpha uint8) color.RGBA {
	c.A = alpha
	return c
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
