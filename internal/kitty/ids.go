package kitty

import "sync"

// Image ids are a shared, finite resource: a placeholder names its image
// through a 256-colour foreground, so ids run 1…255 and two components that
// draw pictures at once (the preview's inline images and the rich view's
// tiles) must not hand out the same one.
var ids = struct {
	mu   sync.Mutex
	next int
	used map[int]bool
}{next: 1, used: map[int]bool{}}

// AcquireID returns an image id no live picture holds, or false when all 255
// are in use.
func AcquireID() (int, bool) {
	ids.mu.Lock()
	defer ids.mu.Unlock()
	for tries := 0; tries < maxID; tries++ {
		id := ids.next
		ids.next++
		if ids.next > maxID {
			ids.next = 1
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
	if id <= 0 || id > maxID {
		return
	}
	ids.mu.Lock()
	delete(ids.used, id)
	ids.mu.Unlock()
}
