//go:build !(darwin || dragonfly || freebsd || linux || netbsd || openbsd || windows)

package main

import "os"

// isTerminal is false where Misoku cannot tell: the welcome, which needs
// someone to answer it, is then never shown.
func isTerminal(*os.File) bool { return false }

// terminalSize does not know the size here: the welcome, when shown,
// would be the text menu.
func terminalSize(*os.File) (w, h int, ok bool) { return 0, 0, false }
