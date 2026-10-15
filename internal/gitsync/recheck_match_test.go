package gitsync_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/gitsync"
)

func TestRecheckListedFallbackIsNarrow(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	existing := filepath.Join(root, "notes")
	if err := os.Mkdir(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "afile")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if runtime.GOOS != "windows" {
		if err := os.Symlink(existing, link); err != nil {
			t.Fatal(err)
		}
	}
	cwd, _ := os.Getwd()
	cases := []struct {
		name         string
		listed, open string
		want         bool
	}{
		{"missing a vs missing b", filepath.Join(root, "a"), filepath.Join(root, "b"), false},
		{"missing same path", filepath.Join(root, "a"), filepath.Join(root, "a"), true},
		{"missing same path, trailing slash and dot", filepath.Join(root, "a") + string(filepath.Separator), filepath.Join(root, ".", "a"), true},
		{"relative vs absolute missing", "rel-missing", filepath.Join(cwd, "rel-missing"), false},
		{"listed missing, open existing", filepath.Join(root, "gone"), existing, false},
		{"listed existing, open missing", existing, filepath.Join(root, "gone"), false},
		{"existing same", existing, existing, true},
		{"existing vs another existing", existing, root, false},
		{"file vs file", file, file, false},
		{"subfolder of missing", filepath.Join(root, "a"), filepath.Join(root, "a", "sub"), false},
	}
	if runtime.GOOS != "windows" {
		cases = append(cases, struct {
			name         string
			listed, open string
			want         bool
		}{"existing through symlink (SameFile)", link, existing, true})
	}
	for _, c := range cases {
		_, got := gitsync.Listed([]config.Vault{{Path: c.listed}}, c.open)
		if got != c.want {
			t.Errorf("%s: Listed(%q, %q) = %v, want %v", c.name, c.listed, c.open, got, c.want)
		}
	}
}
