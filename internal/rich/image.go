package rich

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tdcanvas "github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// Images in the page. Local files are confined to one root (the vault), as the
// text preview confines them; remote pictures are off unless the caller turns
// them on, and then every resolved address must be public, checked per hop.

// maxRemoteBytes caps one download, as the plan fixes.
const maxRemoteBytes = 8 << 20

// maxImagePixels caps a decoded picture (4096×4096): a stitched screenshot or
// a hostile file must not allocate the heap during a page render.
const maxImagePixels = 4096 * 4096

// ImageOptions says what a page may load and where from.
type ImageOptions struct {
	BaseDir string // the note's folder, for relative sources
	Root    string // the vault; no file outside it is drawn
	Remote  bool   // fetch http(s) sources (off by default)
	// Check decides whether a resolved address may be contacted. Nil means
	// public addresses only; tests swap it.
	Check func(net.IP) bool
	// Client is the HTTP client for remote pictures; tests pass one.
	Client *http.Client
}

// publicOnly is the default address policy: no loopback, private, link-local,
// multicast or unspecified address.
func publicOnly(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified()
}

var remoteCache = struct {
	mu    sync.Mutex
	m     map[string]image.Image
	order []string
}{m: map[string]image.Image{}}

// imageBox builds the page's box for an image block. ok is false when the
// source cannot be used (outside the vault, unsupported format, remote and
// disabled): the caller keeps the placeholder.
func imageBox(b Block, opts ImageOptions, widthCells, cellW, cellH int) (*Box, bool) {
	src := strings.TrimSpace(b.Src)
	if src == "" {
		return nil, false
	}
	var img image.Image
	if isRemote(src) {
		if !opts.Remote {
			return nil, false
		}
		var err error
		img, err = fetchRemote(src, opts)
		if err != nil {
			return nil, false
		}
	} else {
		p, ok := confinedPath(src, opts.BaseDir, opts.Root)
		if !ok {
			return nil, false
		}
		if strings.EqualFold(filepath.Ext(p), ".svg") {
			data, err := os.ReadFile(p)
			if err != nil {
				return nil, false
			}
			drawn, err := svgImage(data, widthCells*cellW)
			if err != nil {
				return nil, false
			}
			img = drawn
		} else {
			f, err := os.Open(p)
			if err != nil {
				return nil, false
			}
			defer f.Close()
			decoded, _, err := image.Decode(f)
			if err != nil {
				return nil, false
			}
			img = decoded
		}
	}
	bounds := img.Bounds()
	widthPx := widthCells * cellW
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 || widthPx <= 0 || cellH <= 0 {
		return nil, false
	}
	heightPx := bounds.Dy() * widthPx / bounds.Dx()
	scaled := scaleImage(img, widthPx, heightPx)
	cells := max((heightPx+cellH-1)/cellH, 1)
	return &Box{
		W: widthCells,
		H: cells,
		Draw: func(dst draw.Image, x, y int) {
			draw.Draw(dst, image.Rect(x, y, x+scaled.Bounds().Dx(), y+scaled.Bounds().Dy()), scaled, image.Point{}, draw.Over)
		},
	}, true
}

func isRemote(src string) bool {
	return strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://")
}

// confinedPath resolves src against the note's folder and refuses anything
// outside root, symlinks included (both are cleaned to absolute paths).
func confinedPath(src, baseDir, root string) (string, bool) {
	if root == "" {
		return "", false
	}
	p := src
	if !filepath.IsAbs(p) {
		p = filepath.Join(baseDir, filepath.FromSlash(src))
	}
	p = filepath.Clean(p)
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(absRoot, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	// A symlink inside the vault must not point outside it.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		rel, err := filepath.Rel(absRoot, resolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", false
		}
	}
	return abs, true
}

// fetchRemote downloads a picture after checking every address it would touch:
// the host, and each redirect target. It connects only to checked addresses.
func fetchRemote(src string, opts ImageOptions) (image.Image, error) {
	check := opts.Check
	if check == nil {
		check = publicOnly
	}
	u, err := url.Parse(src)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("rich: bad image URL")
	}
	// The address policy runs before the cache: an entry stored by a caller
	// that widened it is never served to one that did not.
	if err := checkHost(u, check); err != nil {
		return nil, err
	}
	if img, ok := cachedRemote(src); ok {
		return img, nil
	}
	client := opts.Client
	if client == nil {
		dialer := &net.Dialer{Timeout: 5 * time.Second}
		client = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					host, port, err := net.SplitHostPort(addr)
					if err != nil {
						return nil, err
					}
					ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
					if err != nil {
						return nil, err
					}
					for _, ip := range ips {
						if !check(ip) {
							continue
						}
						return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
					}
					return nil, errors.New("rich: no public address")
				},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return checkHost(req.URL, check)
			},
		}
	}
	resp, err := client.Get(src)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rich: image status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxRemoteBytes))
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if b := img.Bounds(); b.Dx()*b.Dy() > maxImagePixels {
		return nil, errors.New("rich: image too large")
	}
	remoteCache.mu.Lock()
	if len(remoteCache.order) >= 32 {
		oldest := remoteCache.order[0]
		remoteCache.order = remoteCache.order[1:]
		delete(remoteCache.m, oldest)
	}
	remoteCache.m[src] = img
	remoteCache.order = append(remoteCache.order, src)
	remoteCache.mu.Unlock()
	return img, nil
}

func checkHost(u *url.URL, check func(net.IP) bool) error {
	host := u.Hostname()
	if host == "" {
		return errors.New("rich: no host")
	}
	if ip := net.ParseIP(host); ip != nil {
		if !check(ip) {
			return errors.New("rich: address not allowed")
		}
		return nil
	}
	ips, err := net.DefaultResolver.LookupIP(context.Background(), "ip", host)
	if err != nil || len(ips) == 0 {
		return errors.New("rich: host does not resolve")
	}
	for _, ip := range ips {
		if !check(ip) {
			return errors.New("rich: address not allowed")
		}
	}
	return nil
}

func cachedRemote(src string) (image.Image, bool) {
	remoteCache.mu.Lock()
	defer remoteCache.mu.Unlock()
	img, ok := remoteCache.m[src]
	return img, ok
}

// svgImage rasterises an SVG at widthPx with tdewolff/canvas, which draws
// text and gradients where oksvg drew no text at all.
func svgImage(data []byte, widthPx int) (image.Image, error) {
	c, err := tdcanvas.ParseSVG(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if c.W <= 0 || c.H <= 0 || widthPx <= 0 {
		return nil, errors.New("rich: empty SVG")
	}
	res := float64(widthPx) / c.W
	return rasterizer.Draw(c, tdcanvas.Resolution(res), tdcanvas.LinearColorSpace{}), nil
}

// scaleImage resizes img to w×h with a good filter.
func scaleImage(img image.Image, w, h int) *image.RGBA {
	if w <= 0 || h <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst
}
