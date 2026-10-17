package gitsync

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// The home folder, the root of a disk and the system's folders are no
// place for notes: syncing the home folder would push ~/.ssh and every
// other private file in it. The welcome never offers them, and setup and
// init refuse to set them up.

// UnsafeFolder reports whether dir is no place to keep notes in: the home
// folder (home; empty means none is known), the root of a disk, or a
// system folder.
func UnsafeFolder(dir, home string) bool {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return true
	}
	if home != "" && SameFolder(abs, home) {
		return true
	}
	return filepath.Dir(abs) == abs || systemFolder(abs, home)
}

// RefuseSyncFolder is the refusal setup and init give for dir when it is
// an UnsafeFolder, and nil otherwise. home is the user's home folder;
// empty means the one the operating system gives.
func RefuseSyncFolder(dir, home string) error {
	if home == "" {
		home, _ = os.UserHomeDir() // none known: the root and system folders are still refused
	}
	if !UnsafeFolder(dir, home) {
		return nil
	}
	return fmt.Errorf("%s is your home folder, the root of a disk or a system folder: sync a folder of its own for your notes, such as ~/Notes (misoku sync setup --dir ~/Notes).", dir)
}

// systemFolder reports whether dir is a folder of the operating system:
// one of the system's own trees (/usr, /etc, C:\Windows, Program Files…)
// or one of the folders that only hold others (/home, /Volumes, /tmp,
// C:\Users…). Folders inside the latter, such as a disk under /Volumes,
// are not, and neither is anything in the home folder.
func systemFolder(dir, home string) bool {
	trees, holders := systemFolders()
	for _, p := range candidates(dir) {
		if home != "" && within(p, home) {
			return false
		}
		for _, t := range trees {
			if within(p, t) {
				return true
			}
		}
		for _, h := range holders {
			if samePath(p, h) {
				return true
			}
		}
	}
	return false
}

// systemFolders lists this OS's system trees and holder folders.
func systemFolders() (trees, holders []string) {
	if runtime.GOOS == "windows" {
		drive := os.Getenv("SystemDrive")
		if drive == "" {
			drive = "C:"
		}
		trees = []string{envOr("SystemRoot", drive+`\Windows`), envOr("ProgramFiles", drive+`\Program Files`),
			envOr("ProgramFiles(x86)", drive+`\Program Files (x86)`), envOr("ProgramData", drive+`\ProgramData`)}
		return trees, []string{drive + `\Users`}
	}
	trees = []string{"/bin", "/boot", "/dev", "/etc", "/lib", "/lib32", "/lib64", "/libx32", "/proc", "/sbin", "/sys", "/usr",
		"/System", "/Library", "/private/etc"}
	holders = []string{"/home", "/Users", "/Volumes", "/mnt", "/media", "/run", "/opt", "/srv", "/snap", "/tmp", "/var",
		"/Applications", "/private", "/private/tmp", "/private/var"}
	return trees, holders
}

// envOr is the environment variable key, or def when it is empty.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// candidates is dir cleaned and, when it differs, with its symlinks
// resolved (macOS's /etc is /private/etc).
func candidates(dir string) []string {
	out := []string{filepath.Clean(dir)}
	if real, err := filepath.EvalSymlinks(dir); err == nil && real != out[0] {
		out = append(out, real)
	}
	return out
}

// foldCase: letter case does not tell folders apart on Windows and, by
// default, on macOS.
var foldCase = runtime.GOOS == "windows" || runtime.GOOS == "darwin"

// samePath compares two cleaned paths as the file system would.
func samePath(a, b string) bool {
	if foldCase {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// within reports whether p is base or inside it.
func within(p, base string) bool {
	if foldCase {
		p, base = strings.ToLower(p), strings.ToLower(base)
	}
	rel, err := filepath.Rel(filepath.Clean(base), p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
