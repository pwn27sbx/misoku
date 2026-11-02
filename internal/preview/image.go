package preview

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"  // register decoders
	_ "image/jpeg" //
	_ "image/png"  // register the decoder; graphics.go encodes
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/pwn27sbx/misoku/internal/kitty"
	_ "golang.org/x/image/bmp"  //
	_ "golang.org/x/image/webp" //
)

// maxImageRows caps how tall (in terminal rows) an image may be drawn.
const maxImageRows = 24

var (
	mdImageLine  = regexp.MustCompile(`^!\[([^\]]*)\]\(\s*(<[^>]+>|[^)\s]+)(?:\s+"[^"]*")?\s*\)$`)
	htmlImageTag = regexp.MustCompile(`(?i)^<img\b[^>]*>$`)
	srcAttr      = htmlAttrRegex("src")
)

// standaloneImage reports whether a whole line is just an image, returning
// its alt text and source.
func standaloneImage(line string) (alt, src string, ok bool) {
	t := strings.TrimSpace(line)
	if m := mdImageLine.FindStringSubmatch(t); m != nil {
		return m[1], strings.Trim(m[2], "<>"), true
	}
	if htmlImageTag.MatchString(t) {
		if s := attr(srcAttr, t); s != "" {
			return attr(altAttr, t), s, true
		}
	}
	return "", "", false
}

// resolveImage turns a Markdown image source into a local file path.
func resolveImage(src, baseDir string) (string, error) {
	if strings.Contains(src, "://") {
		return "", fmt.Errorf("remote image")
	}
	if u, err := url.PathUnescape(src); err == nil {
		src = u
	}
	if strings.HasPrefix(src, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			src = filepath.Join(home, src[2:])
		}
	}
	if !filepath.IsAbs(src) {
		if baseDir == "" {
			return "", fmt.Errorf("unknown folder")
		}
		src = filepath.Join(baseDir, filepath.FromSlash(src))
	}
	return src, nil
}

type imageKey struct {
	path    string
	mod     time.Time
	width   int
	profile int
	bg      string
}

var (
	imageCacheMu sync.Mutex
	imageCache   = map[imageKey][]string{}
)

// cellBox is the cols x rows box an iw x ih picture fills at width columns. A
// cell is about twice as tall as it is wide, so both renderers divide by two.
func cellBox(iw, ih, width int) (cols, rows int) {
	cols = min(width, iw)
	rows = (cols*ih/iw + 1) / 2
	if rows > maxImageRows {
		rows = maxImageRows
		cols = max(rows*2*iw/ih, 1)
	}
	rows = max(rows, 1)
	return cols, rows
}

// renderImage draws a picture: with the Kitty graphics protocol when the
// terminal can show it, with "▀" half blocks (two pixels per cell) otherwise,
// scaled to fit width columns and maxImageRows rows.
func renderImage(path string, width int, bg lipgloss.Color) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("not found")
	}
	if imageMode() == kitty.Graphics {
		lines, err := renderGraphicsImage(path, info, width)
		if err == nil {
			return lines, nil
		}
		if !errors.Is(err, errImageFallback) {
			return nil, err
		}
	}
	profile := lipgloss.ColorProfile()
	key := imageKey{path, info.ModTime(), width, int(profile), string(bg)}
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
		return nil, fmt.Errorf("unsupported image")
	}

	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	if iw == 0 || ih == 0 {
		return nil, fmt.Errorf("empty image")
	}
	cols, rows := cellBox(iw, ih, width)

	back := parseHex(string(bg))
	sample := func(cx, py int) color.RGBA {
		// Average the source pixels covered by this output pixel.
		x0, x1 := b.Min.X+cx*iw/cols, b.Min.X+(cx+1)*iw/cols
		y0, y1 := b.Min.Y+py*ih/(rows*2), b.Min.Y+(py+1)*ih/(rows*2)
		x1, y1 = max(x1, x0+1), max(y1, y0+1)
		stepX, stepY := max((x1-x0)/4, 1), max((y1-y0)/4, 1)
		var r, g, bl, n uint32
		for y := y0; y < y1; y += stepY {
			for x := x0; x < x1; x += stepX {
				cr, cg, cb, ca := img.At(x, y).RGBA()
				// Composite over the theme background.
				r += (cr + uint32(back.R)*257*(0xffff-ca)/0xffff) >> 8
				g += (cg + uint32(back.G)*257*(0xffff-ca)/0xffff) >> 8
				bl += (cb + uint32(back.B)*257*(0xffff-ca)/0xffff) >> 8
				n++
			}
		}
		return color.RGBA{uint8(r / n), uint8(g / n), uint8(bl / n), 255}
	}

	lines := make([]string, rows)
	for y := 0; y < rows; y++ {
		var sb strings.Builder
		for x := 0; x < cols; x++ {
			fg := profile.Color(hexColor(sample(x, y*2))).Sequence(false)
			bgSeq := profile.Color(hexColor(sample(x, y*2+1))).Sequence(true)
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

func hexColor(c color.RGBA) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

func parseHex(h string) color.RGBA {
	var c color.RGBA
	if len(h) == 7 && h[0] == '#' {
		_, _ = fmt.Sscanf(h[1:], "%02x%02x%02x", &c.R, &c.G, &c.B)
	}
	return c
}
