package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pwn27sbx/misoku/internal/clipimage"
)

// pasteImageUsage is what `misoku paste-image --help` says.
const pasteImageUsage = "Usage: misoku paste-image [--dir <folder>] [--link]"

// readClipboardImage reads a picture from the clipboard; tests replace it.
var readClipboardImage = clipimage.Read

// runPasteImage handles "misoku paste-image [--dir <folder>] [--link]": it saves
// the picture in the clipboard under <folder>/assets, the folder rules the app
// uses, and prints the path a Markdown link needs (or the whole link with
// --link), so an editor can paste an image the way the app does.
func runPasteImage(args []string) int {
	fs := flag.NewFlagSet("paste-image", flag.ContinueOnError)
	dir := fs.String("dir", ".", "folder of the note the picture belongs to")
	link := fs.Bool("link", false, "print the Markdown link instead of the path")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), pasteImageUsage)
		fs.PrintDefaults()
	}
	rest, err := positionals(fs, args)
	if err != nil {
		return 2
	}
	if len(rest) > 0 { // it takes no note: the folder comes from --dir
		fs.Usage()
		return 2
	}

	target, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "misoku paste-image: %v\n", err)
		return 1
	}
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "misoku paste-image: %s is not a folder\n", *dir)
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	data, ok, err := readClipboardImage(ctx, os.Getenv)
	switch {
	case errors.Is(err, clipimage.ErrNoTool):
		fmt.Fprintf(os.Stderr, "misoku paste-image: needs %s\n", clipimage.Hint())
		return 1
	case err != nil:
		fmt.Fprintf(os.Stderr, "misoku paste-image: %v\n", err)
		return 1
	case !ok:
		fmt.Fprintln(os.Stderr, "misoku paste-image: the clipboard holds no picture")
		return 1
	}
	defer clipimage.Discard(data)

	rel, err := savePasted(target, data.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "misoku paste-image: %v\n", err)
		return 1
	}
	if *link {
		name := rel[len("assets/"):]
		name = name[:len(name)-len(filepath.Ext(name))]
		fmt.Printf("![%s](%s)\n", name, rel)
		return 0
	}
	fmt.Println(rel)
	return 0
}

// savePasted copies the picture into folder/assets and returns the path a link
// in a note of folder uses, with -2, -3 … when the name is taken.
func savePasted(folder, src string) (string, error) {
	assets := filepath.Join(folder, "assets")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		return "", fmt.Errorf("could not create %s: %w", assets, err)
	}
	ext := filepath.Ext(src)
	stem := src[:len(src)-len(ext)]
	name := filepath.Base(stem)
	// Try names until one is free: the create itself is exclusive, so two
	// pastes at the same second cannot take the same name.
	for n := 1; n < 1000; n++ {
		file := name + ext
		if n > 1 {
			file = fmt.Sprintf("%s-%d%s", name, n, ext)
		}
		dst := filepath.Join(assets, file)
		err := copyPasted(src, dst)
		if err == nil {
			return "assets/" + file, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return "", err
		}
	}
	return "", fmt.Errorf("could not find a free name for %s", name+ext)
}

// copyPasted writes src to dst, refusing to overwrite what is already there.
func copyPasted(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("could not write %s: %w", filepath.Base(dst), err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(dst)
		return fmt.Errorf("could not write %s: %w", filepath.Base(dst), err)
	}
	if err := out.Close(); err != nil { // a delayed write error leaves half a picture
		_ = os.Remove(dst)
		return fmt.Errorf("could not write %s: %w", filepath.Base(dst), err)
	}
	return nil
}
