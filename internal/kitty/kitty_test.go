package kitty_test

import (
	"bytes"
	"encoding/base64"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/kitty"
)

func envOf(pairs map[string]string) func(string) string {
	return func(k string) string { return pairs[k] }
}

func TestDetectChoosesGraphicsOnlyWhereItWorks(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		env  map[string]string
		tty  bool
		want kitty.Mode
	}{
		{"not a terminal", map[string]string{"TERM_PROGRAM": "ghostty"}, false, kitty.Blocks},
		{"ghostty", map[string]string{"TERM_PROGRAM": "ghostty"}, true, kitty.Graphics},
		{"kitty by TERM", map[string]string{"TERM": "xterm-kitty"}, true, kitty.Graphics},
		{"kitty window id", map[string]string{"KITTY_WINDOW_ID": "3"}, true, kitty.Graphics},
		{"wezterm", map[string]string{"TERM_PROGRAM": "WezTerm"}, true, kitty.Graphics},
		{"iterm2", map[string]string{"TERM_PROGRAM": "iTerm.app"}, true, kitty.Graphics},
		{"konsole", map[string]string{"KONSOLE_VERSION": "240800"}, true, kitty.Graphics},
		{"rex", map[string]string{"TERM_PROGRAM": "rex"}, true, kitty.Graphics},
		{"konsole under tmux stays blocks", map[string]string{"KONSOLE_VERSION": "240800", "TMUX": "/tmp/x"}, true, kitty.Blocks},
		{"herdr masks the outer terminal", map[string]string{"TERM": "xterm-256color", "HERDR_ENV": "1"}, true, kitty.Graphics},
		{"tmux never passes graphics", map[string]string{"TERM_PROGRAM": "ghostty", "TMUX": "/tmp/x"}, true, kitty.Blocks},
		{"screen", map[string]string{"TERM": "screen-256color", "TERM_PROGRAM": "ghostty"}, true, kitty.Blocks},
		{"zellij is not rendered", map[string]string{"ZELLIJ": "0", "TERM_PROGRAM": "ghostty"}, true, kitty.Blocks},
		{"plain xterm", map[string]string{"TERM": "xterm-256color"}, true, kitty.Blocks},
		{"no environment at all", map[string]string{}, true, kitty.Blocks},
		{"MISOKU_IMAGES=blocks beats ghostty", map[string]string{"TERM_PROGRAM": "ghostty", "MISOKU_IMAGES": "blocks"}, true, kitty.Blocks},
		{"MISOKU_IMAGES=kitty beats tmux", map[string]string{"TMUX": "/tmp/x", "MISOKU_IMAGES": "kitty"}, true, kitty.Graphics},
		{"MISOKU_IMAGES=kitty beats no tty", map[string]string{"MISOKU_IMAGES": "kitty"}, false, kitty.Graphics},
		{"garbage MISOKU_IMAGES falls through", map[string]string{"TERM_PROGRAM": "ghostty", "MISOKU_IMAGES": "maybe"}, true, kitty.Graphics},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := kitty.Detect(envOf(tc.env), tc.tty); got != tc.want {
				t.Fatalf("Detect = %v, want %v", got, tc.want)
			}
		})
	}
}

// payloadOf returns the base64 data carried by the sequences and the parameter
// lists, in order.
func payloadOf(t *testing.T, seqs []string) (string, []string) {
	t.Helper()
	var data strings.Builder
	var params []string
	for _, s := range seqs {
		if !strings.HasPrefix(s, "\x1b_G") || !strings.HasSuffix(s, "\x1b\\") {
			t.Fatalf("not an APC graphics sequence: %q", s)
		}
		body := strings.TrimSuffix(strings.TrimPrefix(s, "\x1b_G"), "\x1b\\")
		p, payload, ok := strings.Cut(body, ";")
		if !ok {
			t.Fatalf("no payload separator in %q", s)
		}
		params = append(params, p)
		data.WriteString(payload)
	}
	return data.String(), params
}

func TestTransmitSendsThePngInQuietChunks(t *testing.T) {
	t.Parallel()
	png := bytes.Repeat([]byte{0x89, 'P', 'N', 'G'}, 4000) // 16 000 bytes → 21 336 base64 bytes
	seqs := kitty.Transmit(7, 40, 10, png)
	if len(seqs) != 6 {
		t.Fatalf("expected 6 chunks of 4096, got %d", len(seqs))
	}
	payload, params := payloadOf(t, seqs)
	if want := base64.StdEncoding.EncodeToString(png); payload != want {
		t.Fatal("the chunks do not reassemble to the image")
	}
	head := params[0]
	for _, want := range []string{"a=T", "f=100", "t=d", "i=7", "U=1", "c=40", "r=10", "q=2", "m=1"} {
		if !strings.Contains(head, want) {
			t.Errorf("first chunk lacks %q: %q", want, head)
		}
	}
	for i, p := range params[1 : len(params)-1] {
		if p != "m=1" {
			t.Errorf("chunk %d = %q, want m=1", i+2, p)
		}
	}
	if last := params[len(params)-1]; last != "m=0" {
		t.Errorf("last chunk = %q, want m=0", last)
	}
}

func TestTransmitRefusesNothingToSend(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		id, c, r int
		png      []byte
	}{
		{"no data", 1, 10, 5, nil},
		{"no columns", 1, 0, 5, []byte("x")},
		{"no rows", 1, 10, 0, []byte("x")},
		{"no id", 0, 10, 5, []byte("x")},
		{"id past the placeholder colour", 256, 10, 5, []byte("x")},
	} {
		if seqs := kitty.Transmit(tc.id, tc.c, tc.r, tc.png); seqs != nil {
			t.Errorf("%s: expected no sequences, got %v", tc.name, seqs)
		}
	}
}

func TestDeleteFreesOnlyThatImage(t *testing.T) {
	t.Parallel()
	got := kitty.Delete(42)
	if got != "\x1b_Ga=d,d=I,i=42,q=2\x1b\\" {
		t.Fatalf("Delete = %q", got)
	}
	for _, forbidden := range []string{"d=A", "d=a"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("Delete must not remove other programs' images: %q", got)
		}
	}
}

func TestPlaceholdersAreOneCellPerPixelCell(t *testing.T) {
	t.Parallel()
	lines := kitty.Placeholders(9, 6, 3)
	if len(lines) != 3 {
		t.Fatalf("rows = %d, want 3", len(lines))
	}
	for i, l := range lines {
		if w := ansi.StringWidth(l); w != 6 {
			t.Fatalf("row %d is %d cells wide, want 6 (%q)", i, w, l)
		}
		if !strings.HasPrefix(l, "\x1b[38;5;9m") {
			t.Errorf("row %d does not carry the image id as its colour: %q", i, l)
		}
		if !strings.HasSuffix(l, "\x1b[39m") {
			t.Errorf("row %d does not reset the colour: %q", i, l)
		}
		if strings.Contains(l, "\x1b_G") {
			t.Errorf("row %d carries graphics escapes into the view: %q", i, l)
		}
		if n := strings.Count(l, "\U0010EEEE"); n != 6 {
			t.Errorf("row %d has %d placeholders, want 6", i, n)
		}
	}
	// Row 1, column 2: the table's entries 1 and 2.
	if want := "\U0010EEEE\u030D\u030E"; !strings.Contains(lines[1], want) {
		t.Fatalf("missing row/column diacritics %q in %q", want, lines[1])
	}
	// Row 0, column 0.
	if want := "\U0010EEEE\u0305\u0305"; !strings.HasPrefix(lines[0], "\x1b[38;5;9m"+want) {
		t.Fatalf("missing top-left diacritics %q in %q", want, lines[0])
	}
}

func TestPlaceholdersRefuseAnEmptyBox(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ id, c, r int }{{0, 4, 2}, {256, 4, 2}, {1, 0, 2}, {1, 4, 0}} {
		if got := kitty.Placeholders(tc.id, tc.c, tc.r); got != nil {
			t.Errorf("Placeholders(%d,%d,%d) = %v, want nil", tc.id, tc.c, tc.r, got)
		}
	}
}

func TestPlaceholdersPastTheDiacriticTableStillFillTheBox(t *testing.T) {
	t.Parallel()
	lines := kitty.Placeholders(1, 400, 1)
	if len(lines) != 1 {
		t.Fatalf("rows = %d", len(lines))
	}
	if w := ansi.StringWidth(lines[0]); w != 400 {
		t.Fatalf("wide box = %d cells, want 400", w)
	}
}

// recorder collects everything written, so an interleaving is visible.
type recorder struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (r *recorder) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.Write(p)
}

func (r *recorder) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.String()
}

func TestWriterKeepsAChunkSetTogether(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	w := kitty.NewWriter(rec)
	seqs := make([]string, 40)
	for i := range seqs {
		seqs[i] = strings.Repeat("s", 50)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := w.Send(seqs...); err != nil {
			t.Error(err)
		}
	}()
	go func() {
		defer wg.Done()
		for range 200 {
			if _, err := w.Write([]byte(strings.Repeat("f", 400))); err != nil {
				t.Error(err)
				return
			}
			runtime.Gosched()
		}
	}()
	wg.Wait()
	if !strings.Contains(rec.String(), strings.Join(seqs, "")) {
		t.Fatal("a frame landed inside the chunk set")
	}
}

// guardWriter fails the test if two writes are ever inside at once, which is
// what the lock exists to prevent.
type guardWriter struct {
	t      *testing.T
	inside atomic.Int32
}

func (g *guardWriter) Write(p []byte) (int, error) {
	if g.inside.Add(1) != 1 {
		g.t.Error("a second write started before the first finished")
	}
	time.Sleep(200 * time.Microsecond) // widen the window a broken lock would race through
	g.inside.Add(-1)
	return len(p), nil
}

func TestWriterNeverOverlapsAChunkSetAndAFrame(t *testing.T) {
	t.Parallel()
	w := kitty.NewWriter(&guardWriter{t: t})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 30 {
			if err := w.Send("aa", "bb", "cc"); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for range 30 {
			if _, err := w.Write([]byte("frame")); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	wg.Wait()
}

func TestWriterDiscardsWithoutAnOutput(t *testing.T) {
	t.Parallel()
	w := kitty.NewWriter(nil)
	if _, err := w.Write([]byte("frame")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Send("\x1b_Ga=d,q=2\x1b\\"); err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestWriterReportsAFailedSend(t *testing.T) {
	t.Parallel()
	w := kitty.NewWriter(&failingWriter{allow: 1})
	if err := w.Send("one", "two"); err == nil {
		t.Fatal("expected the write error")
	}
}

type failingWriter struct{ allow int }

func (f *failingWriter) Write(p []byte) (int, error) {
	if f.allow <= 0 {
		return 0, errBoom{}
	}
	f.allow--
	return len(p), nil
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }

// tty is a writer that also looks like a terminal file.
type tty struct {
	bytes.Buffer
	fd uintptr
}

func (t *tty) Fd() uintptr { return t.fd }

// Bubble Tea finds the terminal through the program output and reads the window
// size from its descriptor: the wrapper must not hide it, or the program never
// gets a size and stays on its first frame.
func TestWriterKeepsTheTerminalVisibleToBubbleTea(t *testing.T) {
	t.Parallel()
	f := &tty{fd: 7}
	w := kitty.NewWriter(f)
	if got := w.Fd(); got != 7 {
		t.Fatalf("Fd = %d, want 7", got)
	}
	if _, err := w.Read(make([]byte, 1)); err == nil {
		t.Fatal("Read on the program output should report the end of the stream")
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	plain := kitty.NewWriter(&bytes.Buffer{})
	if got := plain.Fd(); got != ^uintptr(0) {
		t.Fatalf("a buffer has no descriptor, got %d", got)
	}
}

// tmux only passes graphics through its DCS, with every escape byte inside
// the payload doubled.
func TestWrapPassthrough(t *testing.T) {
	seq := "\x1b_Ga=T,f=100;AAAA\x1b\\"
	want := "\x1bPtmux;\x1b\x1b_Ga=T,f=100;AAAA\x1b\x1b\\\x1b\\"
	if got := kitty.WrapPassthrough(seq); got != want {
		t.Fatalf("wrapped = %q, want %q", got, want)
	}
}

// Inside tmux the rich view draws only when passthrough is on and the outer
// terminal speaks Kitty; every other mix stays on the text view.
func TestDetectInTmux(t *testing.T) {
	on := func() bool { return true }
	off := func() bool { return false }
	cases := []struct {
		name string
		env  map[string]string
		pass func() bool
		want kitty.Mode
	}{
		{"tmux on inside ghostty", map[string]string{"TMUX": "/tmp/tmux-1/default,1,0", "TERM": "tmux-256color", "TERM_PROGRAM": "ghostty"}, on, kitty.Graphics},
		{"tmux on inside kitty", map[string]string{"TMUX": "/tmp/x", "TERM": "tmux-256color", "TERM_PROGRAM": "kitty"}, on, kitty.Graphics},
		{"tmux on inside konsole", map[string]string{"TMUX": "/tmp/x", "TERM": "tmux-256color", "KONSOLE_VERSION": "240800"}, on, kitty.Graphics},
		{"tmux on inside plain xterm", map[string]string{"TMUX": "/tmp/x", "TERM": "tmux-256color"}, on, kitty.Blocks},
		{"tmux inside herdr", map[string]string{"TMUX": "/tmp/x", "TERM": "tmux-256color", "HERDR_ENV": "1"}, on, kitty.Graphics},
		{"tmux passthrough off", map[string]string{"TMUX": "/tmp/x", "TERM": "tmux-256color", "TERM_PROGRAM": "ghostty"}, off, kitty.Blocks},
		{"tmux unknown passthrough", map[string]string{"TMUX": "/tmp/x", "TERM": "tmux-256color", "TERM_PROGRAM": "ghostty"}, nil, kitty.Blocks},
		{"screen stays text", map[string]string{"TERM": "screen-256color", "TERM_PROGRAM": "ghostty"}, on, kitty.Blocks},
		{"herdr is unchanged", map[string]string{"HERDR_ENV": "1", "TERM": "xterm-256color"}, on, kitty.Graphics},
		{"plain ghostty is unchanged", map[string]string{"TERM_PROGRAM": "ghostty"}, on, kitty.Graphics},
	}
	for _, c := range cases {
		if got := kitty.DetectIn(envOf(c.env), true, c.pass); got != c.want {
			t.Errorf("%s: mode %v, want %v", c.name, got, c.want)
		}
	}
}

// Inside a real tmux pane, TERM is tmux-256color and TERM_PROGRAM is tmux.
// DetectTmux asks the outer terminal probe so Ghostty/kitty/WezTerm/Konsole/herdr
// are still recognised when passthrough is on.
func TestDetectTmuxRealPaneEnv(t *testing.T) {
	t.Parallel()
	on := func() bool { return true }
	off := func() bool { return false }
	paneEnv := map[string]string{
		"TMUX":         "/tmp/tmux-1000/default,123,0",
		"TERM":         "tmux-256color",
		"TERM_PROGRAM": "tmux",
	}
	cases := []struct {
		name  string
		env   map[string]string
		pass  func() bool
		outer map[string]string
		want  kitty.Mode
	}{
		{"ghostty via client_termname", paneEnv, on, map[string]string{"TERM": "xterm-ghostty"}, kitty.Graphics},
		{"ghostty via global TERM_PROGRAM", paneEnv, on, map[string]string{"TERM_PROGRAM": "ghostty"}, kitty.Graphics},
		{"kitty via client_termname", paneEnv, on, map[string]string{"TERM": "xterm-kitty"}, kitty.Graphics},
		{"wezterm via global TERM_PROGRAM", paneEnv, on, map[string]string{"TERM_PROGRAM": "WezTerm"}, kitty.Graphics},
		{"konsole via global KONSOLE_VERSION", paneEnv, on, map[string]string{"KONSOLE_VERSION": "240800"}, kitty.Graphics},
		{"herdr via global HERDR_ENV", paneEnv, on, map[string]string{"HERDR_ENV": "1"}, kitty.Graphics},
		{"plain xterm outer stays blocks", paneEnv, on, map[string]string{"TERM": "xterm-256color"}, kitty.Blocks},
		{"passthrough off ignores outer ghostty", paneEnv, off, map[string]string{"TERM": "xterm-ghostty", "TERM_PROGRAM": "ghostty"}, kitty.Blocks},
		{"MISOKU_IMAGES=blocks beats outer ghostty", map[string]string{
			"TMUX":          "/tmp/tmux-1000/default,123,0",
			"TERM":          "tmux-256color",
			"TERM_PROGRAM":  "tmux",
			"MISOKU_IMAGES": "blocks",
		}, on, map[string]string{"TERM": "xterm-ghostty", "TERM_PROGRAM": "ghostty"}, kitty.Blocks},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := kitty.DetectTmux(envOf(tc.env), true, tc.pass, envOf(tc.outer)); got != tc.want {
				t.Fatalf("DetectTmux = %v, want %v", got, tc.want)
			}
		})
	}
}

// Send wraps the transmit and nothing else: frames never carry a DCS.
func TestPassthroughWriterWrapsOnlyGraphics(t *testing.T) {
	var buf bytes.Buffer
	w := kitty.NewWriter(&buf)
	w.SetPassthrough(true)
	if err := w.Send("\x1b_Ga=T;x\x1b\\"); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); !strings.HasPrefix(got, "\x1bPtmux;") || !strings.HasSuffix(got, "\x1b\\") {
		t.Fatalf("send not wrapped: %q", got)
	}
	buf.Reset()
	if _, err := w.Write([]byte("frame")); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "frame" {
		t.Fatalf("frame wrapped: %q", got)
	}
}

// tmux rewrites 256-colour foreground indices 0…15 into basic 8/16-colour SGR
// codes (30…37, 90…97), which Ghostty does not recognise as Kitty placeholder
// image ids. AcquireID must hand out ids >= 16 so 38;5;N survives tmux verbatim.
func TestAcquireIDSkipsBasicANSIColours(t *testing.T) {
	var held []int
	t.Cleanup(func() {
		for _, id := range held {
			kitty.ReleaseID(id)
		}
	})
	for i := 0; i < 240; i++ {
		id, ok := kitty.AcquireID()
		if !ok {
			t.Fatalf("AcquireID exhausted early at %d", i)
		}
		held = append(held, id)
		if id < 16 || id > 255 {
			t.Fatalf("AcquireID = %d, want 16..255 so tmux never collapses 38;5;N to basic SGR", id)
		}
	}
	if id, ok := kitty.AcquireID(); ok {
		t.Fatalf("AcquireID returned %d after all 240 ids were taken", id)
	}
}
