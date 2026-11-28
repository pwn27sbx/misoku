//go:build !linux && !darwin

package termsize

// cells is unavailable where the console API reports no pixel geometry, so
// the page keeps its default cell size.
func cells(fd uintptr) (int, int, bool) { return 0, 0, false }
