//go:build linux || darwin

package termsize

import "golang.org/x/sys/unix"

// cells asks the kernel for the window size: TIOCGWINSZ carries both the cell
// grid and the window's pixels, which is exactly the ratio we need.
func cells(fd uintptr) (int, int, bool) {
	ws, err := unix.IoctlGetWinsize(int(fd), unix.TIOCGWINSZ)
	if err != nil || ws == nil {
		return 0, 0, false
	}
	return FromWinsize(int(ws.Col), int(ws.Row), int(ws.Xpixel), int(ws.Ypixel))
}
