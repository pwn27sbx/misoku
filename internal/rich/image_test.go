package rich

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func writePNG(t *testing.T, path string, w, h int, c color.RGBA) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func blockFor(t *testing.T, p *Page) *Item {
	t.Helper()
	for i := range p.Items {
		if p.Items[i].Kind == ItemBox {
			return &p.Items[i]
		}
	}
	return nil
}

func TestLocalImageDrawsInThePage(t *testing.T) {
	vault := t.TempDir()
	writePNG(t, filepath.Join(vault, "pic.png"), 40, 20, color.RGBA{200, 10, 10, 255})
	f := testFonts(t)
	doc := Document{Blocks: []Block{{Kind: KindImage, Src: "pic.png", Lines: []string{"pic"}, Line: 0, End: 1}}}
	p := Layout(doc, LayoutOptions{Width: 40, Fonts: f, Palette: testPalette(),
		Images: ImageOptions{BaseDir: vault, Root: vault}})
	box := blockFor(t, p)
	if box == nil {
		t.Fatal("the image was not placed in the page")
	}
	if box.W != p.Width-2*pageMargin {
		t.Fatalf("image width = %d cells, want the content width %d", box.W, p.Width-2*pageMargin)
	}
	if box.H < 1 {
		t.Fatalf("image height = %d cells", box.H)
	}
	img := (&Raster{Page: p, Fonts: f, Palette: testPalette()}).Tile(box.Y, box.H)
	if got := img.RGBAAt(box.X*f.CellW+2, 2); got.R < 150 || got.G > 60 {
		t.Fatalf("image pixel = %v, want the picture's red", got)
	}
}

func TestImageOutsideTheVaultIsRefused(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writePNG(t, filepath.Join(outside, "secret.png"), 10, 10, color.RGBA{0, 255, 0, 255})
	f := testFonts(t)
	doc := Document{Blocks: []Block{{Kind: KindImage, Src: "../secret.png", Lines: []string{"s"}, Line: 0, End: 1}}}
	p := Layout(doc, LayoutOptions{Width: 30, Fonts: f, Palette: testPalette(),
		Images: ImageOptions{BaseDir: root, Root: root}})
	if blockFor(t, p) != nil {
		t.Fatal("an image outside the vault was drawn")
	}
}

func TestImageSymlinkOutOfTheVaultIsRefused(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "secret.png")
	writePNG(t, target, 10, 10, color.RGBA{0, 255, 0, 255})
	link := filepath.Join(root, "link.png")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	f := testFonts(t)
	doc := Document{Blocks: []Block{{Kind: KindImage, Src: "link.png", Lines: []string{"s"}, Line: 0, End: 1}}}
	p := Layout(doc, LayoutOptions{Width: 30, Fonts: f, Palette: testPalette(),
		Images: ImageOptions{BaseDir: root, Root: root}})
	if blockFor(t, p) != nil {
		t.Fatal("a symlink pointing outside the vault was drawn")
	}
}

func TestRemoteImagesAreOffByDefault(t *testing.T) {
	f := testFonts(t)
	doc := Document{Blocks: []Block{{Kind: KindImage, Src: "https://example.com/a.png", Lines: []string{"r"}, Line: 0, End: 1}}}
	p := Layout(doc, LayoutOptions{Width: 30, Fonts: f, Palette: testPalette(),
		Images: ImageOptions{BaseDir: t.TempDir(), Root: t.TempDir()}})
	if blockFor(t, p) != nil {
		t.Fatal("a remote image was fetched with remote images off")
	}
}

func TestRemoteImageRefusesPrivateAddresses(t *testing.T) {
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(buf.Bytes())
	}))
	defer srv.Close()
	// The test server listens on 127.0.0.1: a remote fetch must refuse it
	// unless the caller explicitly widens the policy.
	opts := ImageOptions{Remote: true, BaseDir: t.TempDir(), Root: t.TempDir()}
	if _, err := fetchRemote(srv.URL, opts); err == nil {
		t.Fatal("a loopback address was fetched")
	}
	// A caller that widens the check can use it (the injected-client path).
	opts.Check = func(net.IP) bool { return true }
	opts.Client = srv.Client()
	if _, err := fetchRemote(srv.URL, opts); err != nil {
		t.Fatalf("fetch with a widened policy: %v", err)
	}
}
