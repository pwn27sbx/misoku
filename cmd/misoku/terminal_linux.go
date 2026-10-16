package main

import "golang.org/x/sys/unix"

// ioctlReadTermios reads a terminal's settings on Linux.
const ioctlReadTermios = unix.TCGETS
