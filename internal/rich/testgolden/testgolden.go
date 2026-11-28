// Package testgolden compares golden PNGs the way several architectures can
// agree on: the same size and ink, and no pixel differing by more than an
// antialiasing step. arm64 and amd64 round the rasterisers differently, so a
// byte-for-byte comparison holds locally and fails on CI's other machine.
package testgolden

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
)

// Close reports nil when got is the picture want shows, within antialiasing
// tolerance.
func Close(want, got []byte) error {
	w, err := png.Decode(bytes.NewReader(want))
	if err != nil {
		return fmt.Errorf("golden is not a PNG: %w", err)
	}
	g, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		return fmt.Errorf("new image is not a PNG: %w", err)
	}
	if !w.Bounds().Eq(g.Bounds()) {
		return fmt.Errorf("size %v, want %v", g.Bounds().Size(), w.Bounds().Size())
	}
	bw, iw := InkBox(w), InkBox(g)
	if bw != iw {
		return fmt.Errorf("ink covers %v, want %v", iw, bw)
	}
	bounds := w.Bounds()
	total := bounds.Dx() * bounds.Dy()
	wantInk, gotInk, differing := 0, 0, 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			wr, wg, wb, wa := w.At(x, y).RGBA()
			gr, gg, gb, ga := g.At(x, y).RGBA()
			if wa|wg|wb != 0 {
				wantInk++
			}
			if ga|gg|gb != 0 {
				gotInk++
			}
			delta := 0
			for _, d := range []int{absDiff(wr, gr), absDiff(wg, gg), absDiff(wb, gb), absDiff(wa, ga)} {
				if d > delta {
					delta = d
				}
			}
			if delta == 0 {
				continue
			}
			// RGBA() is 16 bits per channel.
			if delta > 32*257 {
				return fmt.Errorf("pixel %d,%d differs by %d levels", x, y, delta/257)
			}
			differing++
		}
	}
	if margin := max(wantInk/50, 40); gotInk < wantInk-margin || gotInk > wantInk+margin {
		return fmt.Errorf("ink %d pixels, want %d", gotInk, wantInk)
	}
	if differing > max(total/20, 200) {
		return fmt.Errorf("%d pixels differ, more than antialiasing explains", differing)
	}
	return nil
}

// InkBox is the bounding box of every non-transparent pixel, or the empty
// rectangle for a blank image.
func InkBox(img image.Image) image.Rectangle {
	b := img.Bounds()
	first := image.Point{X: -1, Y: -1}
	last := image.Point{}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a == 0 {
				continue
			}
			if first.X < 0 {
				first = image.Point{X: x, Y: y}
			}
			last = image.Point{X: x, Y: y}
		}
	}
	if first.X < 0 {
		return image.Rectangle{}
	}
	return image.Rect(first.X, first.Y, last.X+1, last.Y+1)
}

func absDiff(a, b uint32) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}
