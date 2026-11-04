package main

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/cursor"
	"github.com/pwn27sbx/misoku/internal/kitty"
)

// slowWriter is a terminal that takes a moment over every write and notices a
// write that starts before the previous one ended.
type slowWriter struct {
	busy    atomic.Bool
	overlap atomic.Bool
	mu      sync.Mutex
	writes  []string
	started chan struct{} // closed on the first write
	once    sync.Once
}

func (s *slowWriter) Write(p []byte) (int, error) {
	if !s.busy.CompareAndSwap(false, true) {
		s.overlap.Store(true)
	}
	defer s.busy.Store(false)
	if s.started != nil {
		s.once.Do(func() { close(s.started) })
	}
	time.Sleep(2 * time.Millisecond)
	s.mu.Lock()
	s.writes = append(s.writes, string(p))
	s.mu.Unlock()
	return len(p), nil
}

func (s *slowWriter) has(seq string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range s.writes {
		if strings.Contains(w, seq) {
			return true
		}
	}
	return false
}

// interleaved reports a write that landed between two chunks of one upload.
func (s *slowWriter) interleaved() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	first, last := -1, -1
	for i, w := range s.writes {
		if strings.HasPrefix(w, "\x1b_G") {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	for i := first; i >= 0 && i < last; i++ {
		if !strings.HasPrefix(s.writes[i], "\x1b_G") {
			return true
		}
	}
	return false
}

// A chunked picture upload holds the output while its chunks go out; a cursor
// placement written meanwhile would land between two chunks and break the
// image. The cursor writer must sit above the picture writer, so its write
// waits for the lock the upload holds.
func TestCursorWritesWaitForAnImageUpload(t *testing.T) {
	dst := &slowWriter{started: make(chan struct{})}
	out, images, cur := chain(dst, true)
	if cur == nil {
		t.Fatal("no cursor writer")
	}
	if out != cur {
		t.Fatalf("the program output is %T, want the cursor writer on top", out)
	}

	cur.Set(3, 4, cursor.Block, true)
	chunks := kitty.Transmit(1, 2, 2, make([]byte, 16000))
	if len(chunks) < 3 {
		t.Fatalf("setup: want a multi-chunk upload, got %d chunks", len(chunks))
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := images.Send(chunks...); err != nil {
			t.Errorf("Send: %v", err)
		}
	}()
	<-dst.started // the upload holds the output from here on
	cur.Set(5, 4, cursor.Block, true)
	<-done

	if dst.interleaved() || dst.overlap.Load() {
		t.Fatal("a cursor write landed inside a picture upload")
	}
	if !dst.has("\x1b[5;6H") {
		t.Fatal("the placement never reached the terminal")
	}
}

// Without a cursor writer the program output is the picture writer itself, so
// Bubble Tea still finds the terminal through it.
func TestChainWithoutTheCursor(t *testing.T) {
	out, images, cur := chain(&slowWriter{}, false)
	if cur != nil {
		t.Fatal("a cursor writer was built with the feature off")
	}
	if out != images {
		t.Fatalf("program output = %T, want the picture writer", out)
	}
	if out.(interface{ Fd() uintptr }).Fd() != ^uintptr(0) {
		t.Fatal("the chain should report no file for a plain writer")
	}
}
