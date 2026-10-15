package gitsync

import (
	"os"

	"github.com/pwn27sbx/misoku/internal/config"
)

// Disabled reports whether MISOKU_SYNC=0 turned sync off for every vault,
// without touching the config.
func Disabled() bool { return os.Getenv("MISOKU_SYNC") == "0" }

// Match returns the settings of the vault that is exactly folder, the folder
// Misoku opened. It reports false when sync is disabled or folder is not
// listed: Misoku opens whatever folder it is started in, often a code
// repository, so only a listed folder may ever sync. Match never runs git.
//
// Paths are compared as files, not as text (see sameDir), so a vault listed
// through a symlink, with "/var" for "/private/var" on macOS, or in another
// letter case on a case-insensitive disk still matches. A subfolder of a
// listed vault does not.
func Match(vaults []config.Vault, folder string) (config.Vault, bool) {
	if Disabled() {
		return config.Vault{}, false
	}
	return Listed(vaults, folder)
}

// Listed is Match whatever MISOKU_SYNC says: whether folder is in vaults,
// for `misoku sync init`, which lists it when it is not.
func Listed(vaults []config.Vault, folder string) (config.Vault, bool) {
	for _, v := range vaults {
		if sameDir(v.Path, folder) {
			return v, true
		}
	}
	return config.Vault{}, false
}

// sameDir reports whether a and b are the same existing folder.
//
// It asks the operating system (os.SameFile: device and inode on Unix,
// volume and file index on Windows) instead of comparing resolved path
// strings, because strings disagree for one folder in ways that are hard to
// normalise: symlinks, macOS's /var -> /private/var, Windows 8.3 short
// names, and letter case on case-insensitive file systems (macOS and
// Windows by default), where "Notes" and "notes" are one folder. A path
// that does not exist matches nothing.
func sameDir(a, b string) bool {
	ia, err := os.Stat(a)
	if err != nil || !ia.IsDir() {
		return false
	}
	ib, err := os.Stat(b)
	if err != nil || !ib.IsDir() {
		return false
	}
	return os.SameFile(ia, ib)
}
