package main

import (
	"fmt"
	"io"
	"os"
)

// The kitty keyboard protocol, when another program in the same terminal
// left it on, makes keys such as Cmd+C arrive as escape sequences
// (ESC [ 99 ; 9 u) that a line read takes as typed text. While misoku
// asks in the terminal it pushes the classic mode onto the terminal's
// stack of keyboard modes and pops it after, which gives back whatever
// mode was there; terminals that do not know the protocol ignore both.
const (
	// classicKeys pushes the classic keyboard mode (CSI > 0 u).
	classicKeys = "\x1b[>0u"
	// restoreKeys pops it (CSI < u).
	restoreKeys = "\x1b[<u"
)

// classicKeysOn reports whether the questions switch to the classic
// keyboard mode: only on a terminal, and never on Windows, whose console
// without VT processing would print the sequences.
func classicKeysOn(goos string, terminal bool) bool {
	return terminal && goos != "windows"
}

// onTerminal reports whether in and out are both a terminal.
func onTerminal(in io.Reader, out io.Writer) bool {
	fin, okIn := in.(*os.File)
	fout, okOut := out.(*os.File)
	return okIn && okOut && isTerminal(fin) && isTerminal(fout)
}

// classicKeyboard pushes the classic keyboard mode on out when on is
// set, and returns what pops it; the caller defers that, so Ctrl+C at a
// question gives the mode back too.
func classicKeyboard(out io.Writer, on bool) (restore func()) {
	if !on {
		return func() {}
	}
	fmt.Fprint(out, classicKeys)
	return func() { fmt.Fprint(out, restoreKeys) }
}
