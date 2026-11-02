package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/clipimage"
)

// clipboardPicture makes readClipboardImage return a picture in a temporary
// file named the way a paste names it.
func clipboardPicture(t *testing.T, want, name string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}
	old := readClipboardImage
	readClipboardImage = func(context.Context, func(string) string) (clipimage.Data, bool, error) {
		return clipimage.Data{Path: path, MIME: "image/png", Ext: filepath.Ext(name), Bytes: len(want)}, true, nil
	}
	t.Cleanup(func() { readClipboardImage = old })
}

// clipboardWith makes readClipboardImage report whatever a real one could.
func clipboardWith(t *testing.T, ok bool, err error) {
	t.Helper()
	old := readClipboardImage
	readClipboardImage = func(context.Context, func(string) string) (clipimage.Data, bool, error) {
		return clipimage.Data{}, ok, err
	}
	t.Cleanup(func() { readClipboardImage = old })
}

// stderr runs f and returns what it wrote to the error stream.
func stderr(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	done := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(r)
		done <- string(data)
	}()
	f()
	os.Stderr = old
	w.Close()
	return <-done
}

// stdout runs f and returns what it printed.
func stdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(r) // the read side closing ends it
		done <- string(data)
	}()
	f()
	os.Stdout = old
	w.Close()
	return <-done
}

func TestPasteImageSavesThePictureAndPrintsThePath(t *testing.T) {
	clipboardPicture(t, "png", "pasted-2026-11-01_10-00-00.png")
	notes := t.TempDir()

	var code int
	out := stdout(t, func() { code = runPasteImage([]string{"--dir", notes}) })

	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	want := "assets/pasted-2026-11-01_10-00-00.png"
	if got := strings.TrimSpace(out); got != want {
		t.Fatalf("printed %q, want %q", got, want)
	}
	data, err := os.ReadFile(filepath.Join(notes, want))
	if err != nil || string(data) != "png" {
		t.Fatalf("picture = %q, %v", data, err)
	}
}

func TestPasteImagePrintsTheLinkWhenAsked(t *testing.T) {
	clipboardPicture(t, "png", "pasted-2026-11-01_10-00-00.png")
	notes := t.TempDir()

	out := stdout(t, func() { _ = runPasteImage([]string{"--dir", notes, "--link"}) })

	want := "![pasted-2026-11-01_10-00-00](assets/pasted-2026-11-01_10-00-00.png)\n"
	if out != want {
		t.Fatalf("printed %q, want %q", out, want)
	}
}

// Two pictures with the same name never overwrite each other.
func TestPasteImageNeverOverwritesAPicture(t *testing.T) {
	clipboardPicture(t, "second", "pasted-2026-11-01_10-00-00.png")
	notes := t.TempDir()
	if err := os.MkdirAll(filepath.Join(notes, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(notes, "assets", "pasted-2026-11-01_10-00-00.png")
	if err := os.WriteFile(first, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}

	var code int
	out := stdout(t, func() { code = runPasteImage([]string{"--dir", notes}) })

	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if got := strings.TrimSpace(out); got != "assets/pasted-2026-11-01_10-00-00-2.png" {
		t.Fatalf("printed %q, want the -2 name", got)
	}
	if data, _ := os.ReadFile(first); string(data) != "first" {
		t.Fatalf("the first picture was overwritten: %q", data)
	}
}

func TestPasteImageWithNoPictureFails(t *testing.T) {
	clipboardWith(t, false, nil)
	if code := runPasteImage([]string{"--dir", t.TempDir()}); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
}

func TestPasteImageWithoutAToolSaysWhatToInstall(t *testing.T) {
	clipboardWith(t, false, clipimage.ErrNoTool)
	var code int
	msg := stderr(t, func() { code = runPasteImage([]string{"--dir", t.TempDir()}) })
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	// The hint names the tool of this system, so ask the package for it.
	if !strings.Contains(msg, "needs") || !strings.Contains(msg, clipimage.Hint()) {
		t.Fatalf("stderr = %q, want the install hint for this system", msg)
	}
}

// A clipboard that cannot be read says why, and does not pretend it was empty.
func TestPasteImageReportsAReadFailure(t *testing.T) {
	clipboardWith(t, false, errors.New("wl-paste: exit status 1"))
	var code int
	msg := stderr(t, func() { code = runPasteImage([]string{"--dir", t.TempDir()}) })
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(msg, "wl-paste") {
		t.Fatalf("stderr = %q, want the tool's reason", msg)
	}
}

func TestPasteImageRejectsAFolderThatIsNotThere(t *testing.T) {
	clipboardPicture(t, "png", "pasted.png")
	if code := runPasteImage([]string{"--dir", filepath.Join(t.TempDir(), "nope")}); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
}

// It takes no note: a stray argument is a mistake, and nothing is written.
func TestPasteImageTakesNoPositionalArguments(t *testing.T) {
	notes := t.TempDir()
	clipboardPicture(t, "png", "pasted.png")
	if code := runPasteImage([]string{"--dir", notes, "extra"}); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if entries, err := os.ReadDir(filepath.Join(notes, "assets")); err == nil && len(entries) > 0 {
		t.Fatalf("nothing should be saved: %v", entries)
	}
}
