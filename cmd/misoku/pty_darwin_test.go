package main

import (
	"os"
	"syscall"
	"testing"
	"unsafe"

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
	var name [128]byte
	ctl := func(fd uintptr) {
		for _, req := range []uintptr{unix.TIOCPTYGRANT, unix.TIOCPTYUNLK} {
			if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, req, 0); e != 0 {
				err = e
				return
			}
		}
		if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, unix.TIOCPTYGNAME, uintptr(unsafe.Pointer(&name[0]))); e != 0 {
			err = e
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
	slave, err = os.OpenFile(unix.ByteSliceToString(name[:]), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		t.Fatal(err)
	}
	return master, slave
}
