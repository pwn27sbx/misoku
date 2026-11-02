// Package clipimage reads a picture from the system clipboard.
//
// Every platform tool is optional: a missing one is reported as ErrNoTool so
// the caller can say what to install, and a clipboard that holds text is not an
// error at all. The bytes are written to a temporary file, which the caller
// hands to the notes store and then removes.
package clipimage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Data is the picture read from the clipboard, already in a temporary file.
type Data struct {
	Path  string
	MIME  string
	Ext   string
	Bytes int
}

// ErrNoTool says no clipboard tool is installed for the current session.
var ErrNoTool = errors.New("no clipboard image tool")

// errNotFound marks a run whose program does not exist.
var errNotFound = errors.New("program not found")

// isMissing reports whether err says the program is not installed, either from
// Exec or from a runner that reports it its own way.
func isMissing(err error) bool {
	return errors.Is(err, errNotFound) || errors.Is(err, exec.ErrNotFound) ||
		strings.Contains(err.Error(), "executable file not found")
}

// timeout bounds every clipboard program: a clipboard owner that does not
// answer must never freeze the editor.
const timeout = 3 * time.Second

// Runner runs one program and returns its standard output. Tests pass a fake.
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// mimes are the picture formats accepted from the clipboard, best first. SVG,
// HEIC and AVIF are left out: the preview cannot decode them, so accepting
// them would promise a picture nobody can see.
var mimes = []struct{ mime, ext string }{
	{"image/png", ".png"},
	{"image/jpeg", ".jpg"},
	{"image/webp", ".webp"},
	{"image/gif", ".gif"},
	{"image/bmp", ".bmp"},
}

// Read looks for a picture in the clipboard with the tools of this session.
// ok is false when the clipboard holds no picture (text, or nothing); err
// reports a missing tool or a tool that failed.
func Read(ctx context.Context, env func(string) string) (Data, bool, error) {
	return ReadWith(ctx, env, Exec)
}

// ReadWith is Read with the program runner injected.
func ReadWith(ctx context.Context, env func(string) string, run Runner) (Data, bool, error) {
	if env == nil {
		env = os.Getenv
	}
	if run == nil {
		return Data{}, false, ErrNoTool
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch runtime.GOOS {
	case "darwin":
		return readDarwin(ctx, run)
	case "windows":
		return readWindows(ctx, run)
	}
	// Wayland first: it is the clipboard of the session Misoku runs in, so a
	// Wayland answer — picture or not — is final. X11 is tried only when there
	// is no Wayland display or no Wayland clipboard tool at all.
	if env("WAYLAND_DISPLAY") != "" {
		data, ok, err := readWayland(ctx, run)
		if !errors.Is(err, ErrNoTool) {
			return data, ok, err
		}
	}
	return readX11(ctx, run)
}

// Exec runs a program with the package timeout.
func Exec(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}
	if isMissing(err) {
		return nil, errNotFound
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		// A non-zero exit is how every clipboard tool says "nothing there".
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return nil, err
}

func readWayland(ctx context.Context, run Runner) (Data, bool, error) {
	types, err := run(ctx, "wl-paste", "--list-types")
	if err != nil {
		if isMissing(err) {
			return Data{}, false, ErrNoTool
		}
		return Data{}, false, err // an empty clipboard exits non-zero
	}
	mime, ext, ok := chooseMIME(string(types))
	if !ok {
		return Data{}, false, nil
	}
	data, err := run(ctx, "wl-paste", "--type", mime)
	if err != nil {
		return Data{}, false, err
	}
	return save(data, mime, ext)
}

func readX11(ctx context.Context, run Runner) (Data, bool, error) {
	targets, err := run(ctx, "xclip", "-selection", "clipboard", "-t", "TARGETS", "-o")
	if err != nil {
		if isMissing(err) {
			return Data{}, false, ErrNoTool
		}
		return Data{}, false, err
	}
	mime, ext, ok := chooseMIME(string(targets))
	if !ok {
		return Data{}, false, nil
	}
	data, err := run(ctx, "xclip", "-selection", "clipboard", "-t", mime, "-o")
	if err != nil {
		return Data{}, false, err
	}
	return save(data, mime, ext)
}

// readDarwin uses pngpaste when it is installed, and AppleScript otherwise: it
// writes the clipboard's PNG flavor into our temporary file.
func readDarwin(ctx context.Context, run Runner) (Data, bool, error) {
	if data, err := run(ctx, "pngpaste", "-"); err == nil && len(data) > 0 {
		return save(data, "image/png", ".png")
	} else if err != nil && !isMissing(err) {
		return Data{}, false, nil
	}
	path, err := tempPNGPath()
	if err != nil {
		return Data{}, false, err
	}
	script := fmt.Sprintf("set f to (POSIX file %q)\n"+
		"set fh to open for access f with write permission\n"+
		"write (the clipboard as «class PNGf») to fh\n"+
		"close access fh", path)
	if _, err := run(ctx, "osascript", "-e", script); err != nil {
		_ = os.Remove(path)
		if isMissing(err) {
			return Data{}, false, ErrNoTool
		}
		return Data{}, false, nil
	}
	data, err := os.ReadFile(path)
	_ = os.Remove(path)
	if err != nil || len(data) == 0 {
		return Data{}, false, nil
	}
	return save(data, "image/png", ".png")
}

// readWindows asks PowerShell for the clipboard image and saves it as PNG.
func readWindows(ctx context.Context, run Runner) (Data, bool, error) {
	path, err := tempPNGPath()
	if err != nil {
		return Data{}, false, err
	}
	script := fmt.Sprintf("Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; "+
		"$img = [System.Windows.Forms.Clipboard]::GetImage(); if ($null -eq $img) { exit 1 }; "+
		"$img.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png)", strings.ReplaceAll(path, "'", "''"))
	if _, err := run(ctx, "powershell", "-NoProfile", "-STA", "-Command", script); err != nil {
		_ = os.Remove(path)
		if isMissing(err) {
			return Data{}, false, ErrNoTool
		}
		return Data{}, false, nil
	}
	data, err := os.ReadFile(path)
	_ = os.Remove(path)
	if err != nil || len(data) == 0 {
		return Data{}, false, nil
	}
	return save(data, "image/png", ".png")
}

// chooseMIME picks the first accepted picture type from a clipboard type list.
func chooseMIME(list string) (mime, ext string, ok bool) {
	have := map[string]bool{}
	for _, line := range strings.Split(list, "\n") {
		t := strings.ToLower(strings.TrimSpace(line))
		if i := strings.IndexByte(t, ';'); i >= 0 { // "image/png;charset=…"
			t = strings.TrimSpace(t[:i])
		}
		if t != "" {
			have[t] = true
		}
	}
	for _, m := range mimes {
		if have[m.mime] {
			return m.mime, m.ext, true
		}
		if m.mime == "image/jpeg" && have["image/jpg"] {
			return m.mime, m.ext, true
		}
	}
	return "", "", false
}

// save writes the picture into a fresh temporary folder under the name a pasted
// picture gets, so the note's assets/ keeps it.
func save(data []byte, mime, ext string) (Data, bool, error) {
	if len(data) == 0 {
		return Data{}, false, nil
	}
	dir, err := os.MkdirTemp("", "misoku-paste")
	if err != nil {
		return Data{}, false, err
	}
	name := "pasted-" + time.Now().Format("2006-01-02_15-04-05") + ext
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return Data{}, false, err
	}
	return Data{Path: path, MIME: mime, Ext: ext, Bytes: len(data)}, true, nil
}

// Discard removes the temporary folder of a picture that was read.
func Discard(d Data) {
	if d.Path == "" {
		return
	}
	_ = os.RemoveAll(filepath.Dir(d.Path))
}

func tempPNGPath() (string, error) {
	f, err := os.CreateTemp("", "misoku-paste-*.png")
	if err != nil {
		return "", err
	}
	path := f.Name()
	f.Close()
	return path, nil
}
