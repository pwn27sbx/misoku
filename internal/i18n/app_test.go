package i18n

import (
	"strings"
	"testing"
)

// sameInBothLanguages are app entries whose Spanish is, on purpose, the
// same as their English: names, formats and words both languages share.
var sameInBothLanguages = map[Key]bool{
	AppNeovimError:     true, // "Neovim: %s"
	AppPalGroupGeneral: true, // GENERAL
	AppPalGroupEditor:  true, // EDITOR · NORMAL, Vim's mode name
	AppDateMonth:       true, // a format
	AppDateLong:        true, // a format
	AppSyncOffWhy:      true, // a format
	AppSyncInfoArrived: true, // a format
	AppNo:              true, // No
	AppKindVideo:       true, // video
	AppKindAudio:       true, // audio
}

// S1: every app entry has a Spanish text of its own, so a copy of the
// English left behind while translating is caught.
func TestAppEntriesAreTranslated(t *testing.T) {
	var n int
	for key, en := range catalogs[EN] {
		if !strings.HasPrefix(string(key), "app.") {
			continue
		}
		n++
		es := catalogs[ES][key]
		if es == "" && en != "" {
			t.Errorf("%q has no Spanish text", key)
		}
		if es == en && !sameInBothLanguages[key] {
			t.Errorf("%q reads the same in Spanish: %q", key, es)
		}
	}
	if n == 0 {
		t.Fatal("the catalog holds no app entries")
	}
}
