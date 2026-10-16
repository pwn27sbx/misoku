package app

import "time"

// WithClock makes the app read the time from now, so a test can say how
// long ago the last sync was.
func WithClock(now func() time.Time) Option {
	return func(m *Model) { m.now = now }
}

// SaveCommitDelay is how long after the last save the commit waits.
const SaveCommitDelay = saveCommitDelay
