//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

// isTerminal reports whether f is a terminal a person types in, not a
// file, a pipe or /dev/null (a character device too, as in scripts).
func isTerminal(f *os.File) bool {
	_, err := unix.IoctlGetTermios(int(f.Fd()), ioctlReadTermios)
	return err == nil
}
