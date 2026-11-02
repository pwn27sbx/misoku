package preview_test

import (
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/kitty"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// sink records the graphics sequences a compile sends.
type sink struct {
	mu   sync.Mutex
	seqs []string
}

func (s *sink) Send(seqs ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seqs = append(s.seqs, seqs...)
	return nil
}

func (s *sink) all() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.seqs...)
}

// graphics turns the terminal on for one test, hands back the recording and
// restores half blocks afterwards.
func graphics(t *testing.T) *sink {
	t.Helper()
	s := &sink{}
	preview.SetImageMode(kitty.Graphics)
	preview.SetImageSink(s)
	t.Cleanup(func() {
		preview.SetImageMode(kitty.Blocks)
		preview.SetImageSink(nil)
		preview.ClearImages()
	})
	return s
}

// picture writes a w x h png into dir/assets and returns its Markdown line.
func picture(t *testing.T, dir string, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 3), uint8(y * 6), 200, 255})
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, "assets", "pic.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return "![A picture](assets/pic.png)"
}

func TestGraphicsDrawsPlaceholdersInsteadOfHalfBlocks(t *testing.T) {
	dir := t.TempDir()
	md := picture(t, dir, 80, 40)
	s := graphics(t)

	out := preview.CompileIn(md, theme.DefaultTheme(), 40, dir)

	if strings.Contains(out, "▀") {
		t.Fatal("still drawing half blocks with graphics on")
	}
	plain := ansi.Strip(out)
	if !strings.ContainsRune(plain, 0x10EEEE) {
		t.Fatalf("no placeholders in:\n%q", plain)
	}
	if strings.Contains(out, "\x1b_G") {
		t.Fatal("graphics escapes leaked into the view text")
	}
	var rows int
	for _, l := range strings.Split(out, "\n") {
		if strings.ContainsRune(l, 0x10EEEE) {
			rows++
			if w := ansi.StringWidth(l); w > 40 {
				t.Fatalf("placeholder row is %d cells wide, pane is 40", w)
			}
		}
	}
	if rows != 10 {
		t.Fatalf("expected a 40x10 box, got %d rows", rows)
	}

	seqs := s.all()
	if len(seqs) != 1 {
		t.Fatalf("expected one transmit, got %d sequences", len(seqs))
	}
	head := seqs[0]
	for _, want := range []string{"a=T", "f=100", "t=d", "U=1", "c=40", "r=10", "q=2"} {
		if !strings.Contains(head, want) {
			t.Errorf("transmit lacks %q: %q", want, head)
		}
	}
	id := placeholderID(t, out)
	if want := "i=" + id; !strings.Contains(head, want) {
		t.Fatalf("transmit %q does not carry the placeholder id %s", head, id)
	}
}

// placeholderID is the image id the placeholder cells name in their foreground.
func placeholderID(t *testing.T, out string) string {
	t.Helper()
	for _, l := range strings.Split(out, "\n") {
		if !strings.ContainsRune(l, 0x10EEEE) {
			continue
		}
		i := strings.Index(l, "\x1b[38;5;")
		if i < 0 {
			t.Fatalf("placeholder row without an id colour: %q", l)
		}
		rest := l[i+len("\x1b[38;5;"):]
		j := strings.IndexByte(rest, 'm')
		if j < 0 {
			t.Fatalf("unterminated colour in %q", l)
		}
		return rest[:j]
	}
	t.Fatal("no placeholder row")
	return ""
}

func TestGraphicsSendsEachPictureOnce(t *testing.T) {
	dir := t.TempDir()
	md := picture(t, dir, 80, 40)
	s := graphics(t)

	first := preview.CompileIn(md, theme.DefaultTheme(), 40, dir)
	second := preview.CompileIn(md, theme.DefaultTheme(), 40, dir)

	if n := len(s.all()); n != 1 {
		t.Fatalf("compiling twice sent %d sequences, want 1", n)
	}
	if first != second {
		t.Fatal("the same picture compiled differently the second time")
	}
}

func TestGraphicsSendsThePictureAgainAtAnotherWidth(t *testing.T) {
	dir := t.TempDir()
	md := picture(t, dir, 80, 40)
	s := graphics(t)

	preview.CompileIn(md, theme.DefaultTheme(), 40, dir)
	preview.CompileIn(md, theme.DefaultTheme(), 20, dir)

	seqs := s.all()
	if len(seqs) < 2 {
		t.Fatalf("a new width needs a new transmit, got %d sequences", len(seqs))
	}
	if !strings.Contains(seqs[1], "c=20") && !strings.Contains(seqs[0], "c=20") {
		t.Fatalf("no transmit for the 20-column box: %q", seqs)
	}
}

func TestGraphicsEncodesOtherFormatsAsPng(t *testing.T) {
	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for i := range img.Pix {
		img.Pix[i] = uint8(i)
	}
	f, err := os.Create(filepath.Join(dir, "photo.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, img, nil); err != nil {
		t.Fatal(err)
	}
	f.Close()
	s := graphics(t)

	preview.CompileIn("![](photo.jpg)", theme.DefaultTheme(), 20, dir)

	seqs := s.all()
	if len(seqs) != 1 {
		t.Fatalf("expected one transmit, got %d", len(seqs))
	}
	body := strings.TrimSuffix(strings.TrimPrefix(seqs[0], "\x1b_G"), "\x1b\\")
	_, payload, ok := strings.Cut(body, ";")
	if !ok {
		t.Fatalf("no payload in %q", seqs[0])
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("payload is not base64: %v", err)
	}
	if !strings.HasPrefix(string(data), "\x89PNG") {
		t.Fatal("a jpeg was not re-encoded as PNG")
	}
}

func TestGraphicsKeepsHalfBlocksWhenNoOutputIsInstalled(t *testing.T) {
	dir := t.TempDir()
	md := picture(t, dir, 80, 40)
	preview.SetImageMode(kitty.Graphics)
	preview.SetImageSink(nil)
	t.Cleanup(func() {
		preview.SetImageMode(kitty.Blocks)
	})

	out := preview.CompileIn(md, theme.DefaultTheme(), 40, dir)
	if !strings.Contains(out, "▀") {
		t.Fatal("expected half blocks without a graphics output")
	}
}

func TestClearImagesReleasesEveryPicture(t *testing.T) {
	dir := t.TempDir()
	md := picture(t, dir, 80, 40)
	s := graphics(t)

	out := preview.CompileIn(md, theme.DefaultTheme(), 40, dir)
	id := placeholderID(t, out)
	s.mu.Lock()
	s.seqs = nil
	s.mu.Unlock()

	preview.ClearImages()

	seqs := s.all()
	if len(seqs) != 1 || seqs[0] != kitty.Delete(mustAtoi(t, id)) {
		t.Fatalf("expected a delete for image %s, got %q", id, seqs)
	}
}

func mustAtoi(t *testing.T, s string) int {
	t.Helper()
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			t.Fatalf("not a number: %q", s)
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func TestChangingTheNoteReleasesThePictures(t *testing.T) {
	dir := t.TempDir()
	md := picture(t, dir, 80, 40)
	s := graphics(t)

	m := preview.New(theme.DefaultTheme())
	m.SetSize(80, 24)
	m.SetBaseDir(dir)
	m.SetPage(core.Page{ID: "notes/a.md", Title: "a", Content: md})
	m.SetPage(core.Page{ID: "notes/b.md", Title: "b", Content: "nothing here"})

	found := false
	for _, seq := range s.all() {
		if strings.Contains(seq, "a=d,d=I,i=") {
			found = true
		}
	}
	if !found {
		t.Fatalf("opening another note did not release the picture: %q", s.all())
	}
}

// The configured setting picks the renderer, and MISOKU_IMAGES still wins over
// it, so a bug report is one shell line away.
func TestConfigureImagesPrefersTheEnvironment(t *testing.T) {
	dir := t.TempDir()
	md := picture(t, dir, 80, 40)
	graphics(t)

	t.Setenv("MISOKU_IMAGES", "")
	preview.ConfigureImages("blocks", true)
	if out := preview.CompileIn(md, theme.DefaultTheme(), 40, dir); !strings.Contains(out, "\u2580") {
		t.Fatal("ui.images=blocks did not fall back to half blocks")
	}
	preview.ConfigureImages("kitty", true)
	if out := preview.CompileIn(md, theme.DefaultTheme(), 40, dir); strings.Contains(out, "\u2580") {
		t.Fatal("ui.images=kitty still drew half blocks")
	}

	t.Setenv("MISOKU_IMAGES", "blocks")
	preview.ConfigureImages("kitty", true)
	if out := preview.CompileIn(md, theme.DefaultTheme(), 40, dir); !strings.Contains(out, "\u2580") {
		t.Fatal("MISOKU_IMAGES did not win over ui.images")
	}
}

// idsIn returns the image ids the placeholder cells of out name.
func idsIn(t *testing.T, out string) map[string]bool {
	t.Helper()
	ids := map[string]bool{}
	for _, l := range strings.Split(out, "\n") {
		for i := 0; i+len("\x1b[38;5;") < len(l); i++ {
			if l[i:i+len("\x1b[38;5;")] != "\x1b[38;5;" {
				continue
			}
			rest := l[i+len("\x1b[38;5;"):]
			j := strings.IndexByte(rest, 'm')
			if j < 0 {
				continue
			}
			if strings.Contains(rest[j:], "\U0010EEEE") || strings.Contains(rest[j:min(len(rest), j+20)], "\U0010EEEE") {
				ids[rest[:j]] = true
			}
		}
	}
	return ids
}

// deletedIDs returns the ids the sequences released.
func deletedIDs(seqs []string) map[string]bool {
	ids := map[string]bool{}
	for _, s := range seqs {
		if !strings.Contains(s, "a=d,d=I,i=") {
			continue
		}
		rest := s[strings.Index(s, "a=d,d=I,i=")+len("a=d,d=I,i="):]
		end := strings.IndexByte(rest, ',')
		if end < 0 {
			end = strings.IndexByte(rest, '\x1b')
		}
		if end > 0 {
			ids[rest[:end]] = true
		}
	}
	return ids
}

// A note with more pictures than the terminal can keep must not lose the ones
// its frame already draws: the extra pictures fall back to half blocks instead,
// and no id that appears on screen is ever released.
func TestGraphicsNeverReleasesAPictureItsFrameDraws(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	var md strings.Builder
	for i := 0; i < 35; i++ {
		img := image.NewRGBA(image.Rect(0, 0, 40, 20))
		for p := range img.Pix {
			img.Pix[p] = uint8(p * (i + 1))
		}
		name := "assets/p" + strconv.Itoa(i) + ".png"
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
		f.Close()
		md.WriteString("![](" + name + ")\n\n")
	}
	s := graphics(t)

	out := preview.CompileIn(md.String(), theme.DefaultTheme(), 40, dir)

	drawn := idsIn(t, out)
	if len(drawn) == 0 {
		t.Fatal("no picture was drawn")
	}
	for id := range drawn {
		if deletedIDs(s.all())[id] {
			t.Fatalf("image %s was released while its placeholders are on screen", id)
		}
	}
	if !strings.Contains(out, "\u2580") {
		t.Fatal("the pictures past the cap should fall back to half blocks")
	}
}

// Pictures a render no longer draws are released when that render ends.
func TestGraphicsReleasesWhatTheNextRenderDrops(t *testing.T) {
	dir := t.TempDir()
	md := picture(t, dir, 80, 40)
	s := graphics(t)

	first := preview.CompileIn(md, theme.DefaultTheme(), 40, dir)
	if got := len(idsIn(t, first)); got != 1 {
		t.Fatalf("expected one picture, got %d", got)
	}
	s.mu.Lock()
	s.seqs = nil
	s.mu.Unlock()

	preview.CompileIn("just text", theme.DefaultTheme(), 40, dir)
	if n := len(deletedIDs(s.all())); n != 1 {
		t.Fatalf("expected one release, got %q", s.all())
	}
}

// Resizing re-sends the picture for the new box and releases the old one.
func TestGraphicsResizeReplacesTheOldBox(t *testing.T) {
	dir := t.TempDir()
	md := picture(t, dir, 80, 40)
	s := graphics(t)

	preview.CompileIn(md, theme.DefaultTheme(), 40, dir)
	wide := preview.CompileIn(md, theme.DefaultTheme(), 20, dir)

	if !strings.Contains(strings.Join(s.all(), ""), "c=20") {
		t.Fatal("no transmit for the new width")
	}
	deleted := deletedIDs(s.all())
	for id := range idsIn(t, wide) {
		if deleted[id] {
			t.Fatalf("the new box %s was released", id)
		}
	}
	if len(deleted) == 0 {
		t.Fatal("the old box was never released")
	}
}
