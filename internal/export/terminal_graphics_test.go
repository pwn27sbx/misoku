package export

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/kitty"
	"github.com/pwn27sbx/misoku/internal/preview"
)

// A PDF is printed, not drawn by the terminal: nothing from the graphics
// protocol may leak into it, even when the preview is set to draw pictures with
// placeholders.
func TestPDFHasNoTerminalGraphics(t *testing.T) {
	dir := t.TempDir()
	// 1×1 PNG.
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\xcf\xc0\xf0\x1f\x00\x05\x00\x01\xff\x89\x99=\x1d\x00\x00\x00\x00IEND\xaeB`\x82")
	if err := os.WriteFile(filepath.Join(dir, "dot.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}

	preview.SetImageMode(kitty.Graphics)
	preview.SetImageSink(&discardSink{})
	t.Cleanup(func() {
		preview.SetImageMode(kitty.Blocks)
		preview.SetImageSink(nil)
		preview.ClearImages()
	})

	var buf bytes.Buffer
	if err := PDF("![dot](dot.png)", Options{BaseDir: dir}, &buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("/Subtype /Image")) {
		t.Error("the picture is no longer embedded")
	}
	if strings.ContainsRune(buf.String(), 0x10EEEE) {
		t.Error("a placeholder cell reached the PDF")
	}
	if bytes.Contains(buf.Bytes(), []byte("\x1b_G")) {
		t.Error("a graphics escape reached the PDF")
	}
}

type discardSink struct{}

func (discardSink) Send(seqs ...string) error { return nil }
