package tutor

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/i18n"
)

// hardWraps reports the 1-based numbers of plain paragraph lines that are
// followed, with no blank line between, by another plain line of the same
// paragraph. Printing a paragraph one line per paragraph is what the lesson
// needs: the editor soft-wraps it, while a hand-made break cuts the text short
// against the preview (MISOKU-04).
//
// A line is not plain when it is blank, a heading, a list item, an indented
// continuation, a table row, a quote, a thematic break or a fence; fences are
// blanked by stripCode, and the front matter is skipped.
func hardWraps(text string) []int {
	lines := stripCode(text)
	front := false
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		front = true
	}
	var wraps []int
	prevPlain := false
	for i, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		if front {
			// The front matter ends at the next "---"; its own opening line
			// counts as skipped.
			if i > 0 && strings.TrimSpace(line) == "---" {
				front = false
			}
			prevPlain = false
			continue
		}
		switch {
		case strings.TrimSpace(line) == "":
			prevPlain = false
		case strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t"):
			// An indented line continues the block above it (a list item, a
			// paragraph); it is never a hand-made wrap.
			prevPlain = false
		case isBlockLine(line):
			prevPlain = false
		default:
			if prevPlain {
				wraps = append(wraps, i+1)
			}
			prevPlain = true
		}
	}
	return wraps
}

// isBlockLine reports whether line starts a Markdown block of its own.
func isBlockLine(line string) bool {
	t := strings.TrimSpace(line)
	if t == "" {
		return true
	}
	if strings.HasPrefix(t, "#") {
		return true
	}
	if strings.HasPrefix(t, ">") {
		return true
	}
	if strings.HasPrefix(t, "|") {
		return true
	}
	for _, p := range []string{"- ", "* ", "+ ", "---", "***", "___"} {
		if strings.HasPrefix(t, p) {
			return true
		}
	}
	// An ordered list item: digits, then "." or ")", then a space.
	i := 0
	for i < len(t) && t[i] >= '0' && t[i] <= '9' {
		i++
	}
	if i > 0 && i+1 < len(t) && (t[i] == '.' || t[i] == ')') && t[i+1] == ' ' {
		return true
	}
	return false
}

// bundledLessons lists the embedded note files, vault-relative paths
// included, sorted by WalkDir. The lessons live one folder per route.
func bundledLessons(t *testing.T, bundled fs.FS) []string {
	t.Helper()
	var names []string
	err := fs.WalkDir(bundled, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".md") {
			names = append(names, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return names
}

// S7: every bundled lesson keeps one paragraph per line.
func TestLessonsAreNotHardWrapped(t *testing.T) {
	forEachLang(t, func(t *testing.T, _ i18n.Lang, _ Catalog, bundled fs.FS) {
		names := bundledLessons(t, bundled)
		if len(names) != 102 {
			t.Fatalf("bundled notes = %d, want 102 (97 lessons + 4 indexes + the reference note)", len(names))
		}
		for _, name := range names {
			data, err := fs.ReadFile(bundled, name)
			if err != nil {
				t.Errorf("%s: %v", name, err)
				continue
			}
			for _, line := range hardWraps(string(data)) {
				t.Errorf("%s:%d: paragraph wrapped by hand; put it on one line", name, line)
			}
		}
	})
}

// The checker itself must catch a wrapped paragraph (and leave lists alone).
func TestHardWrapsChecker(t *testing.T) {
	wrapped := "# T\n\nUn párrafo que alguien partió\nen dos líneas a mano.\n\n- una lista\n  con su continuación\n- otra\n\n> una cita\n\n| a | b |\n| --- | --- |\n"
	if got := hardWraps(wrapped); len(got) != 1 || got[0] != 4 {
		t.Fatalf("hardWraps = %v, want [4]", got)
	}
	unwrapped := "---\ntitle: x\nwrapped\n---\n\n# T\n\nUn párrafo entero en una línea.\n\n- uno\n- dos\n\n```\ncódigo\npartido\n```\n"
	if got := hardWraps(unwrapped); len(got) != 0 {
		t.Fatalf("hardWraps = %v, want none", got)
	}
}
