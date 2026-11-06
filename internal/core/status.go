package core

import "strings"

// NoteStatus is the state a note's "status:" front matter field gives it.
type NoteStatus string

// The statuses a note can have; StatusNone is a note without one, or with a
// value Misoku doesn't know.
const (
	StatusNone      NoteStatus = ""
	StatusActive    NoteStatus = "active"
	StatusOnHold    NoteStatus = "onhold"
	StatusCompleted NoteStatus = "completed"
	StatusDropped   NoteStatus = "dropped"
)

// ParseStatus reads a "status:" value regardless of case and edge spaces:
// "active", "onhold" (also "on hold" and "on-hold"), "completed" or
// "dropped". Anything else is StatusNone.
func ParseStatus(value string) NoteStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "active":
		return StatusActive
	case "onhold", "on hold", "on-hold":
		return StatusOnHold
	case "completed":
		return StatusCompleted
	case "dropped":
		return StatusDropped
	}
	return StatusNone
}
