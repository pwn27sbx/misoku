package preview

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/kitty"
)

// Limits for the pictures held by the terminal. The terminal keeps the pixels,
// so they are released image by image; the registry never grows past these.
const (
	maxGraphicsImages = 32       // images kept for the open note
	maxGraphicsTotal  = 64 << 20 // every image together
)

// maxGraphicsBytes caps one image, after encoding. A variable so a test can
// lower it instead of building an 8 MiB picture.
var maxGraphicsBytes = 8 << 20

// errImageFallback means "draw this one with half blocks instead": no graphics
// output installed, an image too large to send, or a terminal that refused it.
var errImageFallback = errors.New("graphics fallback")

var (
	graphicsMu    sync.Mutex
	graphicsMode  = kitty.Blocks
	graphicsSink  kitty.Sender
	graphicsSeen  = map[graphicsKey]*graphicsImage{}
	graphicsOrder []*graphicsImage
	graphicsBytes int
	graphicsID    = 1

	// renderDepth counts nested compiles (embeds and columns compile inside
	// one another) and renderUsed holds the images the current render draws:
	// those must never be released while their placeholders are on screen.
	renderDepth int
	renderUsed  map[int]bool
)

// graphicsKey identifies one transmitted image: the same picture at a new width
// is a new box, so it is sent again and the old one released.
type graphicsKey struct {
	path  string
	mod   time.Time
	width int
}

type graphicsImage struct {
	key        graphicsKey
	id, cols   int
	rows, size int
}

// SetImageMode chooses how pictures are drawn. Changing it releases whatever the
// terminal holds, since those images belong to the other renderer.
func SetImageMode(m kitty.Mode) {
	graphicsMu.Lock()
	defer graphicsMu.Unlock()
	if m == graphicsMode {
		return
	}
	releaseLocked()
	graphicsMode = m
}

// SetImageSink installs the output that carries the graphics sequences: the same
// writer Bubble Tea draws its frames with, so a frame can never land inside a
// transmit. Passing nil turns the graphics path off.
func SetImageSink(s kitty.Sender) {
	graphicsMu.Lock()
	defer graphicsMu.Unlock()
	releaseLocked()
	graphicsSink = s
}

// ClearImages releases every picture the terminal holds for Misoku. The app
// calls it when the preview goes away and once more when it quits.
func ClearImages() {
	graphicsMu.Lock()
	defer graphicsMu.Unlock()
	releaseLocked()
}

// ConfigureImages applies the "ui.images" setting, with MISOKU_IMAGES still
// winning over it. Called once, before the program starts.
func ConfigureImages(setting string, isTTY bool) {
	mode := kitty.Detect(os.Getenv, isTTY)
	if os.Getenv("MISOKU_IMAGES") == "" {
		switch strings.ToLower(strings.TrimSpace(setting)) {
		case "kitty", "graphics", "on":
			mode = kitty.Graphics
		case "blocks", "half", "off":
			mode = kitty.Blocks
		}
	}
	SetImageMode(mode)
}

// imageMode reports the renderer in use.
func imageMode() kitty.Mode {
	graphicsMu.Lock()
	defer graphicsMu.Unlock()
	return graphicsMode
}

// releaseLocked deletes every image the terminal holds and forgets the registry.
// The caller holds graphicsMu. Failures are ignored on purpose: this is cleanup
// of best-effort state that the terminal frees by itself when Misoku exits.
func releaseLocked() {
	if graphicsSink != nil {
		for _, img := range graphicsOrder {
			_ = graphicsSink.Send(kitty.Delete(img.id))
		}
	}
	graphicsSeen = map[graphicsKey]*graphicsImage{}
	graphicsOrder = nil
	graphicsBytes = 0
}

// renderGraphicsImage returns the placeholder cells for one picture, sending it
// to the terminal the first time this path, width and modification time are
// seen. It returns errImageFallback when the picture must be drawn with half
// blocks instead.
func renderGraphicsImage(path string, info os.FileInfo, width int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	cfg, format, err := image.DecodeConfig(f)
	f.Close()
	if err != nil {
		return nil, errors.New(i18n.T(i18n.PreviewImageUnsupported))
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, errors.New(i18n.T(i18n.PreviewImageEmpty))
	}
	cols, rows := cellBox(cfg.Width, cfg.Height, width)

	key := graphicsKey{path: path, mod: info.ModTime(), width: width}
	if img := lookupGraphics(key); img != nil {
		return kitty.Placeholders(img.id, img.cols, img.rows), nil
	}
	data, err := imagePNGBytes(path, format)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > maxGraphicsBytes {
		return nil, errImageFallback
	}
	img, err := registerGraphics(key, data, cols, rows)
	if err != nil {
		return nil, errImageFallback
	}
	return kitty.Placeholders(img.id, img.cols, img.rows), nil
}

// imagePNGBytes returns the image as PNG: a PNG file is sent untouched, the
// other formats are decoded and encoded once (the protocol transfers only PNG,
// RGB and RGBA).
func imagePNGBytes(path, format string) ([]byte, error) {
	if format == "png" {
		return os.ReadFile(path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, errors.New(i18n.T(i18n.PreviewImageUnsupported))
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, errors.New(i18n.T(i18n.PreviewImageUnsupported))
	}
	return buf.Bytes(), nil
}

// lookupGraphics returns the image already sent for key, if any, and marks it as
// the most recently used one.
func lookupGraphics(key graphicsKey) *graphicsImage {
	graphicsMu.Lock()
	defer graphicsMu.Unlock()
	img := graphicsSeen[key]
	if img == nil {
		return nil
	}
	for i, other := range graphicsOrder {
		if other == img {
			graphicsOrder = append(graphicsOrder[:i], graphicsOrder[i+1:]...)
			break
		}
	}
	graphicsOrder = append(graphicsOrder, img)
	markUsedLocked(img.id)
	return img
}

// registerGraphics sends data under a fresh id and remembers it. Room is made by
// releasing images the current render does not draw, never one of its own: a
// picture whose placeholders are already in the frame must keep its image. When
// there is no such image left the picture falls back to half blocks.
func registerGraphics(key graphicsKey, data []byte, cols, rows int) (*graphicsImage, error) {
	graphicsMu.Lock()
	defer graphicsMu.Unlock()
	if img := graphicsSeen[key]; img != nil {
		markUsedLocked(img.id)
		return img, nil
	}
	if graphicsSink == nil {
		return nil, errImageFallback
	}
	for len(graphicsOrder) >= maxGraphicsImages || graphicsBytes+len(data) > maxGraphicsTotal {
		if !releaseStaleLocked() {
			return nil, errImageFallback
		}
	}
	id, ok := nextGraphicsIDLocked()
	if !ok {
		return nil, errImageFallback
	}
	if err := graphicsSink.Send(kitty.Transmit(id, cols, rows, data)...); err != nil {
		return nil, err
	}
	img := &graphicsImage{key: key, id: id, cols: cols, rows: rows, size: len(data)}
	graphicsSeen[key] = img
	graphicsOrder = append(graphicsOrder, img)
	graphicsBytes += len(data)
	markUsedLocked(id)
	return img, nil
}

// markUsedLocked records that id is part of the frame being built. The caller
// holds graphicsMu.
func markUsedLocked(id int) {
	if renderUsed != nil {
		renderUsed[id] = true
	}
}

// releaseStaleLocked frees the oldest image the current render does not draw and
// reports whether one was freed. An image whose release fails stays registered,
// so a later render or ClearImages can try again instead of leaking it silently.
func releaseStaleLocked() bool {
	for i, img := range graphicsOrder {
		if renderUsed[img.id] {
			continue
		}
		if graphicsSink != nil {
			if err := graphicsSink.Send(kitty.Delete(img.id)); err != nil {
				return false
			}
		}
		graphicsOrder = append(graphicsOrder[:i], graphicsOrder[i+1:]...)
		delete(graphicsSeen, img.key)
		graphicsBytes -= img.size
		return true
	}
	return false
}

// beginImageRender opens the scope of one compile and returns the function that
// closes it: whatever the render did not draw is released then, so a note change
// or a resize does not leave the previous pictures in the terminal. Nested
// compiles share the outermost scope.
func beginImageRender() func() {
	graphicsMu.Lock()
	if renderDepth == 0 {
		renderUsed = map[int]bool{}
	}
	renderDepth++
	graphicsMu.Unlock()
	return func() {
		graphicsMu.Lock()
		defer graphicsMu.Unlock()
		renderDepth--
		if renderDepth > 0 {
			return
		}
		for releaseStaleLocked() {
		}
		renderUsed = nil
	}
}

// nextGraphicsIDLocked hands out an id the placeholder colour can carry (a
// 256-colour foreground holds 1…255) that no live image is using.
func nextGraphicsIDLocked() (int, bool) {
	for tries := 0; tries < 255; tries++ {
		id := graphicsID
		graphicsID++
		if graphicsID > 255 {
			graphicsID = 1
		}
		used := false
		for _, img := range graphicsOrder {
			if img.id == id {
				used = true
				break
			}
		}
		if !used {
			return id, true
		}
	}
	return 0, false
}
