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

// terminalSize is the size of the console window f in cells; ok is false
// when f is not a console.
func terminalSize(f *os.File) (w, h int, ok bool) {
	var info windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(windows.Handle(f.Fd()), &info); err != nil {
		return 0, 0, false
	}
	return int(info.Window.Right-info.Window.Left) + 1, int(info.Window.Bottom-info.Window.Top) + 1, true
}
