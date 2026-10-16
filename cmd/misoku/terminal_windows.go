package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// isTerminal reports whether f is the console a person types in, not a
// file, a pipe or NUL.
func isTerminal(f *os.File) bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(f.Fd()), &mode) == nil
}
