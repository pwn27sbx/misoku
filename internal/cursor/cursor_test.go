package cursor_test

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/cursor"
	"github.com/pwn27sbx/misoku/internal/syncout"
)

// frame is what Bubble Tea's renderer writes in the alt screen: the cursor
// homes, then the lines paint.
const frame = "\x1b[H\x1b[2Jthe frame"

// recorder keeps every Write it receives, one string each.
type recorder struct{ writes []string }

func (r *recorder) Write(p []byte) (int, error) {
	r.writes = append(r.writes, string(p))
	return len(p), nil
}

func (r *recorder) all() string { return strings.Join(r.writes, "") }

func (r *recorder) count(seq string) int { return strings.Count(r.all(), seq) }

func TestSetShowsTheCursorInTheModeShape(t *testing.T) {
	r := &recorder{}
	w := cursor.NewWriter(r)

	w.Set(3, 4, cursor.Block, true)

	// x=3 y=4 are 0-based; CUP is 1-based.
	if want := "\x1b[?25h\x1b[2 q\x1b[5;4H"; r.all() != want {
		t.Fatalf("first placement = %q, want %q", r.all(), want)
	}
}

func TestInsertIsABar(t *testing.T) {
	r := &recorder{}
	w := cursor.NewWriter(r)

	w.Set(0, 0, cursor.Bar, true)

	if r.count("\x1b[6 q") != 1 {
		t.Fatalf("bar shape not emitted once: %q", r.all())
	}
	if r.count("\x1b[2 q") != 0 {
		t.Fatalf("block shape emitted for INSERT: %q", r.all())
	}
}

// The renderer skips a frame whose text did not change, so a repeated state
// must not repeat its bytes.
func TestSameStateWritesNothing(t *testing.T) {
	r := &recorder{}
	w := cursor.NewWriter(r)
	w.Set(3, 4, cursor.Block, true)
	r.writes = nil

	w.Set(3, 4, cursor.Block, true)
	w.Set(3, 4, cursor.Block, true)

	if got := r.all(); got != "" {
		t.Fatalf("repeated state wrote %q, want nothing", got)
	}
}

func TestModeSwitchEmitsOnlyTheNewShape(t *testing.T) {
	r := &recorder{}
	w := cursor.NewWriter(r)
	w.Set(3, 4, cursor.Block, true)
	r.writes = nil

	w.Set(3, 4, cursor.Bar, true)

	// The cell did not move and the cursor is already visible: only DECSCUSR.
	if want := "\x1b[6 q"; r.all() != want {
		t.Fatalf("mode switch = %q, want %q", r.all(), want)
	}
}

func TestMovingTheCursorEmitsOnlyThePosition(t *testing.T) {
	r := &recorder{}
	w := cursor.NewWriter(r)
	w.Set(3, 4, cursor.Block, true)
	r.writes = nil

	w.Set(4, 4, cursor.Block, true)

	if want := "\x1b[5;5H"; r.all() != want {
		t.Fatalf("move = %q, want %q", r.all(), want)
	}
}

// The frame leaves the cursor at its last line; every frame must put it back,
// in the same write so the placement lands inside the frame's sync bracket.
func TestFramePutsTheCursorBack(t *testing.T) {
	r := &recorder{}
	w := cursor.NewWriter(r)
	w.Set(3, 4, cursor.Block, true)
	r.writes = nil

	if _, err := w.Write([]byte(frame)); err != nil {
		t.Fatal(err)
	}

	if len(r.writes) != 1 {
		t.Fatalf("frame and placement went out in %d writes, want 1", len(r.writes))
	}
	if want := frame + "\x1b[5;4H"; r.writes[0] != want {
		t.Fatalf("frame = %q, want %q", r.writes[0], want)
	}
}

func TestShapeIsEmittedOnceAcrossFrames(t *testing.T) {
	r := &recorder{}
	w := cursor.NewWriter(r)
	w.Set(3, 4, cursor.Bar, true)

	for range 5 {
		if _, err := w.Write([]byte(frame)); err != nil {
			t.Fatal(err)
		}
	}

	if n := r.count("\x1b[6 q"); n != 1 {
		t.Fatalf("bar shape emitted %d times across frames, want 1: %q", n, r.all())
	}
	if n := r.count("\x1b[5;4H"); n != 6 {
		t.Fatalf("placement emitted %d times (1 set + 5 frames), want 6", n)
	}
}

// Loose sequences (renderer.execute, image sends) are not frames: they must
// pass through untouched.
func TestLooseWritesPassThrough(t *testing.T) {
	r := &recorder{}
	w := cursor.NewWriter(r)
	w.Set(3, 4, cursor.Block, true)
	r.writes = nil

	for _, s := range []string{"\x1b[?25l", "\x1b_Ga=d,i=1\x1b\\", "plain"} {
		if _, err := w.Write([]byte(s)); err != nil {
			t.Fatal(err)
		}
		if got := r.writes[len(r.writes)-1]; got != s {
			t.Fatalf("loose write %q came out as %q", s, got)
		}
	}
	if n := r.count("\x1b[5;4H"); n != 0 {
		t.Fatalf("a loose write was treated as a frame: %q", r.all())
	}
}

func TestHiddenCursorIsHiddenOnceAndFramesStayClean(t *testing.T) {
	r := &recorder{}
	w := cursor.NewWriter(r)
	w.Set(3, 4, cursor.Block, true)
	r.writes = nil

	w.Set(0, 0, cursor.Block, false)
	w.Set(0, 0, cursor.Block, false)
	if got := r.all(); got != "\x1b[?25l" {
		t.Fatalf("hide = %q, want one hide sequence", got)
	}

	r.writes = nil
	if _, err := w.Write([]byte(frame)); err != nil {
		t.Fatal(err)
	}
	if got := r.all(); got != frame {
		t.Fatalf("frame with the cursor hidden = %q, want the frame alone", got)
	}
}

// The start screen hides a cursor that was never shown, so it was never given
// a shape either: the first visible state must still send the DECSCUSR.
func TestShowingAfterAHiddenStartEmitsTheShape(t *testing.T) {
	r := &recorder{}
	w := cursor.NewWriter(r)
	w.Set(0, 0, cursor.Block, false)
	if got := r.all(); got != "" {
		t.Fatalf("hiding a cursor that was never shown wrote %q", got)
	}
	r.writes = nil

	w.Set(3, 4, cursor.Block, true)

	if want := "\x1b[?25h\x1b[2 q\x1b[5;4H"; r.all() != want {
		t.Fatalf("first visible state = %q, want %q", r.all(), want)
	}
}

func TestRestoreResetsTheShapeAndShowsTheCursor(t *testing.T) {
	r := &recorder{}
	w := cursor.NewWriter(r)
	w.Set(3, 4, cursor.Bar, true)
	r.writes = nil

	if err := w.Restore(); err != nil {
		t.Fatal(err)
	}

	if want := "\x1b[0 q\x1b[?25h"; r.all() != want {
		t.Fatalf("restore = %q, want %q", r.all(), want)
	}
}

// After an external program has used the terminal, the next state must be
// re-applied even though it did not change.
func TestRestoreForgetsTheState(t *testing.T) {
	r := &recorder{}
	w := cursor.NewWriter(r)
	w.Set(3, 4, cursor.Block, true)
	if err := w.Restore(); err != nil {
		t.Fatal(err)
	}
	r.writes = nil

	w.Set(3, 4, cursor.Block, true)

	if want := "\x1b[?25h\x1b[2 q\x1b[5;4H"; r.all() != want {
		t.Fatalf("re-apply = %q, want %q", r.all(), want)
	}
}

// The placement has to sit inside the frame's synchronized update.
func TestPlacementStaysInsideTheSyncBracket(t *testing.T) {
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pr.Close()

	w := cursor.NewWriter(syncout.New(pw))
	w.Set(3, 4, cursor.Block, true)
	if _, err := w.Write([]byte(frame)); err != nil {
		t.Fatal(err)
	}
	if err := pw.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(pr)
	if err != nil {
		t.Fatal(err)
	}

	got := string(out)
	i := strings.Index(got, frame)
	if i < 0 {
		t.Fatalf("the frame never reached the terminal: %q", got)
	}
	begin := strings.LastIndex(got[:i], "\x1b[?2026h")
	end := strings.Index(got[i:], "\x1b[?2026l")
	if begin < 0 || end < 0 {
		t.Fatalf("the frame is not inside a sync bracket: %q", got)
	}
	if !strings.HasPrefix(got[i+len(frame):], "\x1b[5;4H") {
		t.Fatalf("the frame's placement is not inside its bracket: %q", got)
	}
}

func TestNilWriterDiscardsEverything(t *testing.T) {
	w := cursor.NewWriter(nil)
	w.Set(3, 4, cursor.Block, true)
	if n, err := w.Write([]byte(frame)); n != len(frame) || err != nil {
		t.Fatalf("Write = %d, %v; want %d, nil", n, err, len(frame))
	}
	if err := w.Restore(); err != nil {
		t.Fatal(err)
	}
}

func TestShortWriteIsReported(t *testing.T) {
	w := cursor.NewWriter(shortWriter{})
	w.Set(0, 0, cursor.Block, true)
	n, err := w.Write([]byte(frame))
	if n > len(frame) {
		t.Fatalf("Write reported %d bytes of %d", n, len(frame))
	}
	if err == nil {
		t.Fatal("a short write went unreported")
	}
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return 0, nil }

func TestEnabled(t *testing.T) {
	env := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}
	cases := []struct {
		name    string
		setting string
		env     map[string]string
		isTTY   bool
		want    bool
	}{
		{"auto on a terminal", "auto", nil, true, true},
		{"unset on a terminal", "", nil, true, true},
		{"block turns it off", "block", nil, true, false},
		{"auto is not a terminal", "auto", nil, false, false},
		{"env auto wins over block", "block", map[string]string{"MISOKU_CURSOR_SHAPE": "auto"}, true, true},
		{"env block wins over auto", "auto", map[string]string{"MISOKU_CURSOR_SHAPE": "block"}, true, false},
		{"env is case and space insensitive", "auto", map[string]string{"MISOKU_CURSOR_SHAPE": " Block "}, true, false},
		{"an unknown setting means auto", "nonsense", nil, true, true},
		{"TERM=dumb emits nothing", "auto", map[string]string{"TERM": "dumb"}, true, false},
		{"TERM=dumb with block", "block", map[string]string{"TERM": "dumb"}, true, false},
		{"a nil getenv is harmless", "auto", nil, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			getenv := env(c.env)
			if c.env == nil {
				getenv = nil
			}
			if got := cursor.Enabled(c.setting, getenv, c.isTTY); got != c.want {
				t.Fatalf("Enabled(%q, %v, %v) = %v, want %v", c.setting, c.env, c.isTTY, got, c.want)
			}
		})
	}
}
