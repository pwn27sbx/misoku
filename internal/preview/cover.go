package preview

import (
	"errors"
	"fmt"
	"image"
	"os"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/i18n"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// coverRows is the banner height in terminal rows (two pixels each).
const coverRows = 8

// SplitFrontMatter separates a leading "---" YAML-style block of simple
// "key: value" pairs from the Markdown body. Keys are lowercased; it is a
// thin view of core.ParseFrontMatter.
func SplitFrontMatter(input string) (map[string]string, string) {
	if !strings.HasPrefix(input, "---\n") && !strings.HasPrefix(input, "---\r\n") {
		return nil, input
	}
	fm := core.ParseFrontMatter(input)
	if fm.EndLine < 0 {
		return nil, input // unterminated: treat as normal text
	}
	meta := make(map[string]string, len(fm.Fields))
	for _, f := range fm.Fields {
		meta[strings.ToLower(f.Key)] = f.Value
	}
	return meta, fm.Body
}

// renderCover draws a banner: the image cropped to fill width × coverRows
// ("object-fit: cover"), sampled nearest-neighbour and posterised so it
// reads as pixel art.
func renderCover(path string, width int, bg lipgloss.Color) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, errors.New(i18n.T(i18n.PreviewImageMissing))
	}
	profile := lipgloss.ColorProfile()
	key := imageKey{"cover:" + path, info.ModTime(), width, int(profile), string(bg)}
	imageCacheMu.Lock()
	if lines, ok := imageCache[key]; ok {
		imageCacheMu.Unlock()
		return lines, nil
	}
	imageCacheMu.Unlock()

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, errors.New(i18n.T(i18n.PreviewImageUnsupported))
	}

	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	if iw == 0 || ih == 0 || width <= 0 {
		return nil, errors.New(i18n.T(i18n.PreviewImageEmpty))
	}
	pw, ph := width, coverRows*2
	// Crop the source to the banner's aspect ratio, centred.
	cw, ch := iw, iw*ph/pw
	if ch > ih {
		ch, cw = ih, ih*pw/ph
	}
	x0, y0 := b.Min.X+(iw-cw)/2, b.Min.Y+(ih-ch)/2

	back := parseHex(string(bg))
	pixel := func(px, py int) string {
		sx := x0 + (px*cw+cw/2)/pw
		sy := y0 + (py*ch+ch/2)/ph
		r, g, bl, a := img.At(sx, sy).RGBA()
		mix := func(c uint32, bgc uint8) uint8 {
			v := (c + uint32(bgc)*257*(0xffff-a)/0xffff) >> 8
			return posterize(uint8(v))
		}
		return fmt.Sprintf("#%02x%02x%02x", mix(r, back.R), mix(g, back.G), mix(bl, back.B))
	}

	lines := make([]string, coverRows)
	for y := 0; y < coverRows; y++ {
		var sb strings.Builder
		for x := 0; x < pw; x++ {
			fg := profile.Color(pixel(x, y*2)).Sequence(false)
			bgSeq := profile.Color(pixel(x, y*2+1)).Sequence(true)
			if fg == "" {
				sb.WriteString("▀")
				continue
			}
			sb.WriteString("\x1b[" + fg + ";" + bgSeq + "m▀")
		}
		if profile != termenv.Ascii {
			sb.WriteString("\x1b[0m")
		}
		lines[y] = sb.String()
	}

	imageCacheMu.Lock()
	imageCache[key] = lines
	imageCacheMu.Unlock()
	return lines, nil
}

// posterize snaps a channel to 8 levels for a retro pixel-art palette.
func posterize(v uint8) uint8 {
	const step = 255.0 / 7
	level := int(float64(v)/step + 0.5)
	return uint8(float64(level) * step)
}
