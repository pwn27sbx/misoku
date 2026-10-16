//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package main

import "golang.org/x/sys/unix"

// ioctlReadTermios reads a terminal's settings on macOS and the BSDs.
const ioctlReadTermios = unix.TIOCGETA
