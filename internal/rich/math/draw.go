package math

import (
	"image"
	"image/color"
	"image/draw"

	xdraw "golang.org/x/image/draw"
	"math"
)

// render draws a laid-out box into a transparent image. The formula's
// baseline sits padY pixels from the top; x starts padX in.
func render(b box, fg color.RGBA) (img *image.RGBA, padX, padY, baseline int) {
	padX = int(math.Max(1, math.Round(0.05*maxf(b.w, 1))))
	padY = 2
	w := int(math.Ceil(b.w)) + 2*padX
	h := int(math.Ceil(b.h+b.d)) + 2*padY
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	img = image.NewRGBA(image.Rect(0, 0, w, h))
	baseline = padY + int(math.Ceil(b.h))
	for _, it := range b.items {
		switch it.kind {
		case itGlyph:
			if it.gid != 0 {
				drawGlyphID(img, it.gid, it.size, float64(padX)+it.x,
					float64(baseline)-it.y, fg)
				break
			}
			drawGlyph(img, it.r, it.size, float64(padX)+it.x,
				float64(baseline)-it.y, it.sx, it.sy, fg)
		case itRule:
			x0 := int(math.Round(float64(padX) + it.x))
			y0 := int(math.Round(float64(baseline) - it.y))
			rw := int(math.Round(it.w))
			rh := int(math.Round(it.h))
			if rh < 1 {
				rh = 1
			}
			if rw > 0 {
				draw.Draw(img, image.Rect(x0, y0, x0+rw, y0+rh), image.NewUniform(fg), image.Point{}, draw.Over)
			}
		case itTofu:
			x0 := int(math.Round(float64(padX) + it.x))
			y0 := int(math.Round(float64(baseline) - it.h))
			rw := int(math.Round(it.w))
			rh := int(math.Round(it.h))
			drawTofu(img, x0, y0, rw, rh, fg)
		case itSlash:
			x0 := float64(padX) + it.x
			y0 := float64(baseline) - it.y
			drawLine(img, x0, y0, x0+it.w, y0-it.h, maxf(1, it.size), fg)
		}
	}
	return img, padX, padY, baseline
}

// drawTofu draws the hollow box a missing glyph gets, so a CJK word inside
// \text shows as boxes instead of silently vanishing.
func drawTofu(dst *image.RGBA, x, y, w, h int, col color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	u := image.NewUniform(col)
	draw.Draw(dst, image.Rect(x, y, x+w, y+1), u, image.Point{}, draw.Over)
	draw.Draw(dst, image.Rect(x, y+h-1, x+w, y+h), u, image.Point{}, draw.Over)
	draw.Draw(dst, image.Rect(x, y, x+1, y+h), u, image.Point{}, draw.Over)
	draw.Draw(dst, image.Rect(x+w-1, y, x+w, y+h), u, image.Point{}, draw.Over)
}

// drawLine draws a one-pixel staircase between two points; only \not uses it.
func drawLine(dst *image.RGBA, x0, y0, x1, y1, thickness float64, col color.RGBA) {
	dx := x1 - x0
	dy := y1 - y0
	steps := int(math.Max(math.Abs(dx), math.Abs(dy)))
	if steps <= 0 {
		return
	}
	u := image.NewUniform(col)
	th := int(math.Round(thickness))
	if th < 1 {
		th = 1
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Round(x0 + dx*t))
		y := int(math.Round(y0 + dy*t))
		draw.Draw(dst, image.Rect(x, y, x+th, y+th), u, image.Point{}, draw.Over)
	}
}

// scaleImage shrinks an image to fit widthPx, keeping the aspect ratio.
func scaleImage(src *image.RGBA, widthPx int) *image.RGBA {
	if widthPx <= 0 || src.Bounds().Dx() <= widthPx {
		return src
	}
	sw := src.Bounds().Dx()
	sh := src.Bounds().Dy()
	tw := widthPx
	th := int(math.Round(float64(sh) * float64(tw) / float64(sw)))
	if th < 1 {
		th = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	xdraw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	return dst
}
