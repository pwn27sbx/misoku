package main

import (
	"os"
	"strconv"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

// openPTY opens a pseudo-terminal: master is the terminal window's side,
// slave the side Misoku runs on.
func openPTY(t *testing.T) (master, slave *os.File) {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	var n uint32
	ctl := func(fd uintptr) {
		if err = unix.IoctlSetPointerInt(int(fd), unix.TIOCSPTLCK, 0); err == nil {
			n, err = unix.IoctlGetUint32(int(fd), unix.TIOCGPTN)
		}
	}
	if rc, cerr := master.SyscallConn(); cerr != nil {
		t.Fatal(cerr)
	} else if cerr = rc.Control(ctl); cerr != nil {
		t.Fatal(cerr)
	}
	if err != nil {
		master.Close()
		t.Fatal(err)
	}
	slave, err = os.OpenFile("/dev/pts/"+strconv.Itoa(int(n)), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		t.Fatal(err)
	}
	return master, slave
}
