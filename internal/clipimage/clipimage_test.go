package clipimage_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/clipimage"
)

// fake answers the clipboard programs of one session.
type fake struct {
	types, data string
	missing     map[string]bool // programs that are not installed
	fail        map[string]bool // programs that fail
	calls       []string
}

func (f *fake) run(ctx context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	if f.missing[name] {
		return nil, errors.New("exec: \"" + name + "\": executable file not found in $PATH")
	}
	if f.fail[name] {
		return nil, errors.New(name + ": exit status 1")
	}
	switch name {
	case "wl-paste", "xclip":
		if len(args) > 0 && (args[0] == "--list-types" || strings.Contains(strings.Join(args, " "), "TARGETS")) {
			if f.types == "" {
				return nil, errors.New(name + ": exit status 1")
			}
			return []byte(f.types), nil
		}
	}
	return []byte(f.data), nil
}

func waylandEnv(kv map[string]string) func(string) string {
	return func(k string) string { return kv[k] }
}

func TestWaylandReadsTheFirstAcceptedPictureType(t *testing.T) {
	t.Parallel()
	f := &fake{types: "text/plain\nimage/webp\nimage/png\n", data: "PNGDATA"}
	d, ok, err := clipimage.ReadWith(context.Background(), waylandEnv(map[string]string{"WAYLAND_DISPLAY": "wayland-0"}), f.run)
	if err != nil || !ok {
		t.Fatalf("Read = %v, %v", ok, err)
	}
	if d.MIME != "image/png" || d.Ext != ".png" {
		t.Fatalf("picked %s (%s), want image/png first", d.MIME, d.Ext)
	}
	t.Cleanup(func() { clipimage.Discard(d) })
	if !strings.HasPrefix(d.Path, os.TempDir()) || !strings.Contains(d.Path, "pasted-") {
		t.Fatalf("unexpected temp path %q", d.Path)
	}
	data, err := os.ReadFile(d.Path)
	if err != nil || string(data) != "PNGDATA" {
		t.Fatalf("file = %q, %v", data, err)
	}
	if d.Bytes != 7 {
		t.Fatalf("Bytes = %d", d.Bytes)
	}
}

func TestWaylandWithoutAPictureIsNotAnError(t *testing.T) {
	t.Parallel()
	f := &fake{types: "text/plain\n"}
	_, ok, err := clipimage.ReadWith(context.Background(), waylandEnv(map[string]string{"WAYLAND_DISPLAY": "wayland-0"}), f.run)
	if ok || err != nil {
		t.Fatalf("a clipboard holding text only: ok=%v err=%v", ok, err)
	}
	if len(f.calls) != 1 || !strings.Contains(f.calls[0], "--list-types") {
		t.Fatalf("the picture type should not be requested: %q", f.calls)
	}
}

func TestMimeParametersAreIgnored(t *testing.T) {
	t.Parallel()
	f := &fake{types: "image/png;charset=binary\n", data: "x"}
	d, ok, err := clipimage.ReadWith(context.Background(), waylandEnv(map[string]string{"WAYLAND_DISPLAY": "1"}), f.run)
	if err != nil || !ok || d.MIME != "image/png" {
		t.Fatalf("Read = %+v, %v, %v", d, ok, err)
	}
	clipimage.Discard(d)
}

func TestX11IsUsedWithoutWayland(t *testing.T) {
	t.Parallel()
	f := &fake{types: "TARGETS\nimage/jpeg\n", data: "JPEG"}
	d, ok, err := clipimage.ReadWith(context.Background(), waylandEnv(nil), f.run)
	if err != nil || !ok {
		t.Fatalf("Read = %v, %v", ok, err)
	}
	if d.MIME != "image/jpeg" || d.Ext != ".jpg" {
		t.Fatalf("picked %s (%s)", d.MIME, d.Ext)
	}
	for _, call := range f.calls {
		if strings.HasPrefix(call, "wl-paste") {
			t.Fatalf("wl-paste should not run without Wayland: %q", f.calls)
		}
	}
	clipimage.Discard(d)
}

func TestWaylandFallsBackToX11WhenItsToolIsMissing(t *testing.T) {
	t.Parallel()
	f := &fake{types: "image/png\n", data: "PNG", missing: map[string]bool{"wl-paste": true}}
	d, ok, err := clipimage.ReadWith(context.Background(), waylandEnv(map[string]string{"WAYLAND_DISPLAY": "1"}), f.run)
	if err != nil || !ok {
		t.Fatalf("Read = %v, %v", ok, err)
	}
	clipimage.Discard(d)
}

func TestNoToolAtAllIsReported(t *testing.T) {
	t.Parallel()
	f := &fake{missing: map[string]bool{"wl-paste": true, "xclip": true}}
	_, ok, err := clipimage.ReadWith(context.Background(), waylandEnv(map[string]string{"WAYLAND_DISPLAY": "1"}), f.run)
	if ok || !errors.Is(err, clipimage.ErrNoTool) {
		t.Fatalf("ok=%v err=%v, want ErrNoTool", ok, err)
	}
}

func TestAFailingToolIsReportedAndIsNotAPicture(t *testing.T) {
	t.Parallel()
	f := &fake{types: "image/png\n", data: "x", fail: map[string]bool{"wl-paste": true}}
	_, ok, err := clipimage.ReadWith(context.Background(), waylandEnv(map[string]string{"WAYLAND_DISPLAY": "1"}), f.run)
	if ok || err == nil {
		t.Fatalf("ok=%v err=%v, want the tool error", ok, err)
	}
	if errors.Is(err, clipimage.ErrNoTool) {
		t.Fatal("a failing tool is not a missing one")
	}
}

func TestReadWithoutARunnerSaysNoTool(t *testing.T) {
	t.Parallel()
	if _, _, err := clipimage.ReadWith(context.Background(), waylandEnv(nil), nil); !errors.Is(err, clipimage.ErrNoTool) {
		t.Fatalf("err = %v", err)
	}
}

func TestDiscardRemovesTheTemporaryFolder(t *testing.T) {
	t.Parallel()
	f := &fake{types: "image/png\n", data: "x"}
	d, ok, err := clipimage.ReadWith(context.Background(), waylandEnv(map[string]string{"WAYLAND_DISPLAY": "1"}), f.run)
	if err != nil || !ok {
		t.Fatalf("Read = %v, %v", ok, err)
	}
	dir := d.Path[:strings.LastIndex(d.Path, string(os.PathSeparator))]
	clipimage.Discard(d)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("temporary folder still there: %v", err)
	}
}
