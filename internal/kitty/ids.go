package kitty

import "sync"

// minID is the smallest image id AcquireID hands out. Indices 0…15 are the
// basic ANSI palette, which tmux rewrites from 38;5;N into 30…37 / 90…97;
// terminals such as Ghostty ignore basic SGR colours on Unicode placeholders.
// Starting at 16 keeps every id in 38;5;N verbatim through tmux.
const minID = 16

// Image ids are a shared, finite resource: a placeholder names its image
// through a 256-colour foreground, so ids run 16…255 and two components that
// draw pictures at once (the preview's inline images and the rich view's
// tiles) must not hand out the same one.
var ids = struct {
	mu   sync.Mutex
	next int
	used map[int]bool
}{next: minID, used: map[int]bool{}}

// AcquireID returns an image id no live picture holds, or false when all 240
// are in use.
func AcquireID() (int, bool) {
	ids.mu.Lock()
	defer ids.mu.Unlock()
	for tries := 0; tries < maxID-minID+1; tries++ {
		id := ids.next
		ids.next++
		if ids.next > maxID {
			ids.next = minID
		}
		if !ids.used[id] {
			ids.used[id] = true
			return id, true
		}
	}
	return 0, false
}

// ReleaseID frees an id once its image is deleted from the terminal.
func ReleaseID(id int) {
	if id < minID || id > maxID {
		return
	}
	ids.mu.Lock()
	delete(ids.used, id)
	ids.mu.Unlock()
}
