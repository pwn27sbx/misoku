package diagram

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sort"

	"github.com/pwn27sbx/misoku/internal/rich"
)

const (
	maxSceneCells  = 20_000
	maxScenePixels = 4_000_000
)

func validateScene(scene Scene, fonts *rich.Fonts) error {
	if fonts == nil {
		return fmt.Errorf("diagram: no fonts")
	}
	if scene.Width <= 0 || scene.Height <= 0 {
		return fmt.Errorf("diagram: invalid scene size %dx%d", scene.Width, scene.Height)
	}
	if scene.Width > maxSceneCells || scene.Height > maxSceneCells || scene.Width*scene.Height > maxSceneCells {
		return fmt.Errorf("diagram: scene %dx%d exceeds native cell budget", scene.Width, scene.Height)
	}
	pixels := int64(scene.Width) * int64(fonts.CellW) * int64(scene.Height) * int64(fonts.CellH)
	if pixels > maxScenePixels {
		return fmt.Errorf("diagram: scene %dx%d exceeds native pixel budget", scene.Width, scene.Height)
	}
	return nil
}

func renderScene(scene Scene, fonts *rich.Fonts) *image.RGBA {
	if validateScene(scene, fonts) != nil {
		return nil
	}
	w := max(scene.Width, 1) * fonts.CellW
	h := max(scene.Height, 1) * fonts.CellH
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for _, cmd := range scene.Commands {
		drawCommand(img, cmd, fonts)
	}
	if text := sceneTextPage(scene, fonts); text != nil {
		textImg := (&rich.Raster{Page: text, Fonts: fonts}).Tile(0, text.Height)
		draw.Draw(img, img.Bounds(), textImg, image.Point{}, draw.Over)
	}
	return img
}

func drawCommand(img *image.RGBA, c Command, fonts *rich.Fonts) {
	x, y := c.X*fonts.CellW, c.Y*fonts.CellH
	w, h := max(c.W, 0)*fonts.CellW, max(c.H, 0)*fonts.CellH
	switch c.Kind {
	case CommandRect:
		if c.Fill.A != 0 {
			fillRect(img, x, y, w, h, c.Fill)
		}
		if c.Stroke.A != 0 {
			strokeRect(img, x, y, w, h, c.Stroke)
		}
	case CommandRounded:
		drawRounded(img, x, y, w, h, c.Radius*fonts.CellW, c.Fill, c.Stroke)
	case CommandEllipse:
		if c.Radius > 0 {
			r := max(c.Radius, 2)
			drawEllipse(img, x-r, y-r, 2*r+1, 2*r+1, true, c.Fill, c.Stroke)
		} else {
			drawEllipse(img, x, y, w, h, c.Circle, c.Fill, c.Stroke)
		}
	case CommandDiamond:
		drawPolygon(img, []Point{{X: x + w/2, Y: y}, {X: x + w, Y: y + h/2}, {X: x + w/2, Y: y + h}, {X: x, Y: y + h/2}}, c.Fill, c.Stroke)
	case CommandHexagon:
		cut := min(w/4, h/2)
		drawPolygon(img, []Point{{X: x + cut, Y: y}, {X: x + w - cut, Y: y}, {X: x + w, Y: y + h/2}, {X: x + w - cut, Y: y + h}, {X: x + cut, Y: y + h}, {X: x, Y: y + h/2}}, c.Fill, c.Stroke)
	case CommandParallelogram:
		slant := c.Slant * fonts.CellW
		if abs(slant) > w/2 {
			slant = w / 2
		}
		drawPolygon(img, []Point{{X: x + slant, Y: y}, {X: x + w, Y: y}, {X: x + w - slant, Y: y + h}, {X: x, Y: y + h}}, c.Fill, c.Stroke)
	case CommandPolygon:
		drawPolygon(img, pixelPoints(c.Points, fonts), c.Fill, c.Stroke)
	case CommandLine:
		drawLine(img, x, y, c.X2*fonts.CellW, c.Y2*fonts.CellH, c.Color, c.Dashed, c.Thick)
	case CommandArrow:
		x2, y2 := c.X2*fonts.CellW, c.Y2*fonts.CellH
		drawLine(img, x, y, x2, y2, c.Color, c.Dashed, c.Thick)
		drawArrowhead(img, x2, y2, math.Atan2(float64(y2-y), float64(x2-x)), c.Head, c.Color, c.Thick)
		drawArrowhead(img, x, y, math.Atan2(float64(y-y2), float64(x-x2)), c.Tail, c.Color, c.Thick)
	case CommandPolyline:
		for i := 1; i < len(c.Points); i++ {
			a, b := pixelPoint(c.Points[i-1], fonts), pixelPoint(c.Points[i], fonts)
			drawLine(img, a.X, a.Y, b.X, b.Y, c.Color, c.Dashed, c.Thick)
		}
	case CommandSector:
		drawSector(img, x, y, w, h, c.Start, c.End, c.Fill)
	case CommandBadge:
		drawRounded(img, x, y, w, h, min(h/2, 12), c.Fill, color.RGBA{})
	}
}

func sceneTextPage(scene Scene, fonts *rich.Fonts) *rich.Page {
	page := &rich.Page{Width: max(scene.Width, 1), Height: max(scene.Height, 1)}
	for _, cmd := range scene.Commands {
		if cmd.Kind != CommandText && cmd.Kind != CommandBadge {
			continue
		}
		if cmd.Text == "" || cmd.W <= 0 || cmd.H <= 0 || cmd.Color.A == 0 {
			continue
		}
		size := cmd.SizePx
		if size <= 0 {
			size = fonts.BodyPx
		}
		page.Items = append(page.Items, rich.Item{
			Kind: rich.ItemText, X: cmd.X, Y: cmd.Y, W: cmd.W, H: cmd.H,
			XOffPx: alignedTextXOffPx(cmd, fonts, size),
			Text:   cmd.Text, SizePx: size, Mono: cmd.Mono, Bold: cmd.Bold, Color: cmd.Color,
			BaselinePx: textBaseline(fonts, size, cmd.Mono, cmd.H),
		})
	}
	if len(page.Items) == 0 {
		return nil
	}
	return page
}

func alignedTextXOffPx(cmd Command, fonts *rich.Fonts, size float64) int {
	if cmd.Align == AlignLeft {
		return 0
	}
	textWidth := int(math.Ceil(float64(fonts.Measure(cmd.Text, size, cmd.Mono)) / 64))
	available := cmd.W * fonts.CellW
	offset := 0
	switch cmd.Align {
	case AlignCenter:
		offset = (available - textWidth) / 2
	case AlignRight:
		offset = available - textWidth
	}
	return min(max(offset, 0), max(available-1, 0))
}

func textBaseline(fonts *rich.Fonts, size float64, mono bool, cells int) int {
	line := fonts.LineHeightPx(size, mono)
	ascent := fonts.AscentPx(size, mono)
	// Match rich.layouter: centre the font's ascent/descent box vertically,
	// then bias by half the leading so the glyph body looks optically centred.
	return max(int(math.Round((float64(cells*fonts.CellH)+ascent-line/2)/2)), 0)
}

func pixelPoint(p Point, fonts *rich.Fonts) Point {
	return Point{X: p.X * fonts.CellW, Y: p.Y * fonts.CellH}
}

func pixelPoints(points []Point, fonts *rich.Fonts) []Point {
	out := make([]Point, len(points))
	for i, p := range points {
		out[i] = pixelPoint(p, fonts)
	}
	return out
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	r := image.Rect(x, y, x+w, y+h).Intersect(img.Bounds())
	if r.Empty() {
		return
	}
	draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Over)
}

func strokeRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	drawLine(img, x, y, x+w-1, y, c, false, false)
	drawLine(img, x, y+h-1, x+w-1, y+h-1, c, false, false)
	drawLine(img, x, y, x, y+h-1, c, false, false)
	drawLine(img, x+w-1, y, x+w-1, y+h-1, c, false, false)
}

func drawRounded(img *image.RGBA, x, y, w, h, radius int, fill, stroke color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	radius = min(max(radius, 0), min(w, h)/2)
	if fill.A != 0 {
		for row := 0; row < h; row++ {
			inset := 0
			if row < radius {
				dy := radius - row
				inset = radius - int(math.Sqrt(float64(radius*radius-dy*dy)))
			} else if row >= h-radius {
				dy := row - (h - radius - 1)
				inset = radius - int(math.Sqrt(float64(radius*radius-dy*dy)))
			}
			fillRect(img, x+inset, y+row, w-2*inset, 1, fill)
		}
	}
	if stroke.A == 0 {
		return
	}
	if radius == 0 {
		strokeRect(img, x, y, w, h, stroke)
		return
	}
	drawLine(img, x+radius, y, x+w-radius-1, y, stroke, false, false)
	drawLine(img, x+radius, y+h-1, x+w-radius-1, y+h-1, stroke, false, false)
	drawLine(img, x, y+radius, x, y+h-radius-1, stroke, false, false)
	drawLine(img, x+w-1, y+radius, x+w-1, y+h-radius-1, stroke, false, false)
	corners := []struct {
		point Point
		start float64
	}{
		{Point{X: x + radius, Y: y + radius}, math.Pi},
		{Point{X: x + w - radius - 1, Y: y + radius}, 1.5 * math.Pi},
		{Point{X: x + radius, Y: y + h - radius - 1}, math.Pi / 2},
		{Point{X: x + w - radius - 1, Y: y + h - radius - 1}, 0},
	}
	for _, corner := range corners {
		drawArc(img, corner.point.X, corner.point.Y, radius, corner.start, corner.start+math.Pi/2, stroke)
	}
}

func drawEllipse(img *image.RGBA, x, y, w, h int, circle bool, fill, stroke color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	if circle {
		d := min(w, h)
		x += (w - d) / 2
		y += (h - d) / 2
		w, h = d, d
	}
	points := ellipsePoints(x, y, w, h, 72)
	drawPolygon(img, points, fill, stroke)
}

func ellipsePoints(x, y, w, h, segments int) []Point {
	points := make([]Point, segments)
	cx, cy := float64(x)+float64(w-1)/2, float64(y)+float64(h-1)/2
	rx, ry := float64(w-1)/2, float64(h-1)/2
	for i := range points {
		a := 2 * math.Pi * float64(i) / float64(segments)
		points[i] = Point{X: int(math.Round(cx + rx*math.Cos(a))), Y: int(math.Round(cy + ry*math.Sin(a)))}
	}
	return points
}

func drawSector(img *image.RGBA, x, y, w, h int, start, end float64, fill color.RGBA) {
	if fill.A == 0 || w <= 0 || h <= 0 {
		return
	}
	if end <= start {
		end += 2 * math.Pi
	}
	end = min(end, start+2*math.Pi)
	r := float64(min(w, h)-1) / 2
	cx, cy := float64(x)+float64(w-1)/2, float64(y)+float64(h-1)/2
	segments := max(int((end-start)*24), 12)
	points := make([]Point, 0, segments+2)
	points = append(points, Point{X: int(math.Round(cx)), Y: int(math.Round(cy))})
	for i := 0; i <= segments; i++ {
		a := start + (end-start)*float64(i)/float64(segments)
		points = append(points, Point{X: int(math.Round(cx + r*math.Cos(a))), Y: int(math.Round(cy + r*math.Sin(a)))})
	}
	drawPolygon(img, points, fill, color.RGBA{})
}

func drawPolygon(img *image.RGBA, points []Point, fill, stroke color.RGBA) {
	if len(points) < 2 {
		return
	}
	if fill.A != 0 && len(points) >= 3 {
		minY, maxY := points[0].Y, points[0].Y
		for _, p := range points {
			minY, maxY = min(minY, p.Y), max(maxY, p.Y)
		}
		for y := minY; y <= maxY; y++ {
			var xs []int
			for i, a := range points {
				b := points[(i+1)%len(points)]
				if (a.Y <= y && b.Y > y) || (b.Y <= y && a.Y > y) {
					x := int(math.Round(float64(a.X) + float64(y-a.Y)*float64(b.X-a.X)/float64(b.Y-a.Y)))
					xs = append(xs, x)
				}
			}
			sort.Ints(xs)
			for i := 0; i+1 < len(xs); i += 2 {
				if xs[i+1] > xs[i] {
					fillRect(img, xs[i], y, xs[i+1]-xs[i]+1, 1, fill)
				}
			}
		}
	}
	if stroke.A != 0 {
		for i, a := range points {
			b := points[(i+1)%len(points)]
			drawLine(img, a.X, a.Y, b.X, b.Y, stroke, false, false)
		}
	}
}

func drawArc(img *image.RGBA, cx, cy, radius int, start, end float64, c color.RGBA) {
	steps := max(int((end-start)*float64(radius)), 4)
	var prev Point
	for i := 0; i <= steps; i++ {
		a := start + (end-start)*float64(i)/float64(steps)
		p := Point{X: int(math.Round(float64(cx) + float64(radius)*math.Cos(a))), Y: int(math.Round(float64(cy) + float64(radius)*math.Sin(a)))}
		if i > 0 {
			drawLine(img, prev.X, prev.Y, p.X, p.Y, c, false, false)
		}
		prev = p
	}
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA, dashed, thick bool) {
	if c.A == 0 {
		return
	}
	length := math.Hypot(float64(x2-x1), float64(y2-y1))
	brush := 1
	if thick {
		brush = 3
	}
	if dashed {
		steps := max(int(length), 1)
		for i := 0; i <= steps; i++ {
			if (i/7)%2 != 0 {
				continue
			}
			t := float64(i) / float64(steps)
			brushAt(img, int(math.Round(float64(x1)+(float64(x2-x1))*t)), int(math.Round(float64(y1)+(float64(y2-y1))*t)), brush, c)
		}
		return
	}
	dx, sx := abs(x2-x1), 1
	if x2 < x1 {
		sx = -1
	}
	dy, sy := -abs(y2-y1), 1
	if y2 < y1 {
		sy = -1
	}
	err := dx + dy
	for {
		brushAt(img, x1, y1, brush, c)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x1 += sx
		}
		if e2 <= dx {
			err += dx
			y1 += sy
		}
	}
}

func brushAt(img *image.RGBA, x, y, size int, c color.RGBA) {
	half := size / 2
	fillRect(img, x-half, y-half, size, size, c)
}

func drawArrowhead(img *image.RGBA, x, y int, angle float64, head Arrowhead, c color.RGBA, thick bool) {
	if head == ArrowNone || c.A == 0 {
		return
	}
	size := 8
	if thick {
		size = 10
	}
	ux, uy := math.Cos(angle), math.Sin(angle)
	px, py := -uy, ux
	switch head {
	case ArrowFilled:
		baseX, baseY := int(math.Round(float64(x)-ux*float64(size))), int(math.Round(float64(y)-uy*float64(size)))
		drawPolygon(img, []Point{{X: x, Y: y}, {X: int(math.Round(float64(baseX) + px*4)), Y: int(math.Round(float64(baseY) + py*4))}, {X: int(math.Round(float64(baseX) - px*4)), Y: int(math.Round(float64(baseY) - py*4))}}, c, c)
	case ArrowOpen:
		baseX, baseY := int(math.Round(float64(x)-ux*float64(size))), int(math.Round(float64(y)-uy*float64(size)))
		drawLine(img, x, y, int(math.Round(float64(baseX)+px*4)), int(math.Round(float64(baseY)+py*4)), c, false, thick)
		drawLine(img, x, y, int(math.Round(float64(baseX)-px*4)), int(math.Round(float64(baseY)-py*4)), c, false, thick)
	case ArrowCircle:
		drawEllipse(img, x-4, y-4, 9, 9, true, c, c)
	case ArrowCross:
		drawLine(img, x-4, y-4, x+4, y+4, c, false, thick)
		drawLine(img, x+4, y-4, x-4, y+4, c, false, thick)
	}
}
