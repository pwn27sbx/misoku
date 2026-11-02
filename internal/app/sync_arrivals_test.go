package app_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// arrivalWindow is a window on a synced vault, with its note open and a
// first sync done, so the next sync's message is the one a test reads.
func arrivalWindow(t *testing.T) (remote, vault string, h *harness) {
	t.Helper()
	remote, vault = syncedVault(t)
	h = window(t, vault, syncConfig(t, vault), app.WithClock(at1405))
	h.keys("1")
	h.key(tea.KeyEsc)
	h.finish(h.update(app.SyncNowMsg{}))
	return remote, vault, h
}

// pushNotes writes the notes on another device and pushes them in one commit.
func pushNotes(t *testing.T, remote string, names ...string) {
	t.Helper()
	desk := clone(t, remote)
	for _, name := range names {
		writeOutside(t, desk, name, "from desk "+name)
	}
	gitIn(t, desk, "add", "-A")
	gitIn(t, desk, "commit", "-q", "-m", "from desk")
	gitIn(t, desk, "push", "-q", "origin", "HEAD:main")
}

// numberedNotes is n note names that sort in order: note01.md, note02.md...
func numberedNotes(n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("note%02d.md", i+1)
	}
	return names
}

// messagesView is the `:messages` list, closed again after reading it.
func messagesView(h *harness) string {
	h.t.Helper()
	h.ex("messages")
	v := h.screen()
	h.key(tea.KeyEsc)
	return v
}

// wideBar is the status bar in a window wide enough to show a whole message:
// at termW the bar's other parts leave a message about 50 columns.
func wideBar(h *harness) string {
	h.t.Helper()
	h.send(tea.WindowSizeMsg{Width: 1000, Height: termH})
	lines := strings.Split(h.screen(), "\n")
	return lines[len(lines)-1]
}

func wantWideBar(t *testing.T, h *harness, want string) {
	t.Helper()
	if bar := wideBar(h); !strings.Contains(bar, want) {
		t.Fatalf("the status bar must show %q:\n%s", want, bar)
	}
}

// S1, A1: the exact words, for a sync asked for.
func TestSyncSaysWhichNotesArrived(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		notes []string
		want  string
	}{
		{"one new", []string{"nota.md"}, "1 new note from another computer: nota.md"},
		{"two new", []string{"nota.md", "otra.md"}, "2 new notes from another computer: nota.md, otra.md"},
		{"one updated", []string{"doc.md"}, "1 note updated from another computer: doc.md"},
		{"two updated", []string{"doc.md", "two.md"}, "2 notes updated from another computer: doc.md, two.md"},
		{"both", []string{"a.md", "b.md", "doc.md"}, "2 new notes, 1 updated from another computer: a.md, b.md, doc.md"},
		{"one of each", []string{"a.md", "doc.md"}, "1 new note, 1 updated from another computer: a.md, doc.md"},
		{"not ASCII, in a folder", []string{"ideas/reunión.md"}, "1 new note from another computer: ideas/reunión.md"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			remote, vault := syncedVault(t)
			writeOutside(t, vault, "two.md", "two")
			gitIn(t, vault, "add", "-A")
			gitIn(t, vault, "commit", "-q", "-m", "second")
			gitIn(t, vault, "push", "-q", "origin", "HEAD:main")
			h := window(t, vault, syncConfig(t, vault), app.WithClock(at1405))
			h.keys("1")
			h.key(tea.KeyEsc)
			h.finish(h.update(app.SyncNowMsg{}))
			pushNotes(t, remote, c.notes...)
			h.finish(h.update(app.SyncNowMsg{}))
			wantWideBar(t, h, c.want)
			if bar := wideBar(h); strings.Contains(bar, "Esc dismiss") {
				t.Errorf("good news is not an error:\n%s", bar)
			}
			if v := messagesView(h); !strings.Contains(v, c.want) {
				t.Errorf(":messages must list it:\n%s", v)
			}
		})
	}
}

// S1: an automatic sync that brought notes speaks.
func TestAutomaticSyncNamesTheNotesThatArrived(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	clock := newTestClock()
	h := window(t, vault, syncConfig(t, vault), app.WithClock(clock.Now))
	pushNotes(t, remote, "nota.md", "otra.md")
	clock.add(5 * time.Minute)
	if n := poll(h); n != 1 {
		t.Fatalf("want the timed sync to run (%d commands ran)", n)
	}
	wantWideBar(t, h, "2 new notes from another computer: nota.md, otra.md")
}

// S2: no note arrived, so an automatic sync stays silent, also when other
// files or a deletion came in.
func TestAutomaticSyncWithNoNoteArrivedStaysSilent(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		push func(t *testing.T, remote string)
	}{
		{"nothing came", func(*testing.T, string) {}},
		{"a file that is not a note", func(t *testing.T, remote string) { pushNotes(t, remote, "assets/pic.png") }},
		{"a deleted note", func(t *testing.T, remote string) {
			desk := clone(t, remote)
			gitIn(t, desk, "rm", "-q", "gone.md")
			gitIn(t, desk, "commit", "-q", "-m", "delete")
			gitIn(t, desk, "push", "-q", "origin", "HEAD:main")
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			remote, vault := syncedVault(t)
			pushNotes(t, remote, "gone.md")
			clock := newTestClock()
			h := window(t, vault, syncConfig(t, vault), app.WithClock(clock.Now))
			h.finish(h.update(app.SyncNowMsg{}))
			c.push(t, remote)
			h.send(core.StatusMsg{Text: "typing along"})
			clock.add(5 * time.Minute)
			if n := poll(h); n != 1 {
				t.Fatalf("want the timed sync to run (%d commands ran)", n)
			}
			wantBar(t, h, "typing along")
			if bar := statusBar(h); strings.Contains(bar, "from another computer") {
				t.Errorf("no note arrived:\n%s", bar)
			}
		})
	}
}

// S2: a sync asked for says exactly what it always said, also when only a
// file that is not a note came.
func TestAskedSyncWithNoNoteArrivedSaysSynced(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		push func(t *testing.T, remote string)
	}{
		{"nothing came", func(*testing.T, string) {}},
		{"a file that is not a note", func(t *testing.T, remote string) { pushNotes(t, remote, "assets/pic.png") }},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			remote, _, h := arrivalWindow(t)
			c.push(t, remote)
			h.finish(h.update(app.SyncNowMsg{}))
			if v := messagesView(h); !strings.Contains(v, "14:05  Synced ") || strings.Contains(v, "another computer") {
				t.Errorf("want the plain line:\n%s", v)
			}
		})
	}
}

// S3: the conflict message stays as it is, with the arrivals added to it.
func TestSyncConflictKeepsItsMessageAndAddsTheArrivals(t *testing.T) {
	t.Parallel()
	remote, vault, h := arrivalWindow(t)
	pushNotes(t, remote, "doc.md", "other.md")
	writeOutside(t, vault, "doc.md", "mine")
	h.finish(h.update(app.SyncNowMsg{}))
	for _, want := range []string{"Synced · 1 conflict copy: doc (conflict ", " · 1 new note from another computer: other.md"} {
		wantWideBar(t, h, want)
	}
}

// A problem in the same sync stays first in the line and stays an error: the
// notes arrived, and the push after them failed.
func TestSyncProblemIsNotHiddenByTheArrivals(t *testing.T) {
	t.Parallel()
	remote, vault, h := arrivalWindow(t)
	pushNotes(t, remote, "nota.md")
	hook := filepath.Join(remote, "hooks", "pre-receive")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeOutside(t, vault, "mine.md", "mine")
	h.finish(h.update(app.SyncNowMsg{}))
	wantWideBar(t, h, "Esc dismiss")
	v := noSpace(messagesView(h))
	first, second := strings.Index(v, noSpace("Sync failed: origin refused")), strings.Index(v, noSpace("1 new note from another computer: nota.md"))
	if first < 0 || second < first {
		t.Errorf("want the failure, then the arrivals (%d, %d):\n%s", first, second, v)
	}
}

// finishSync merges what the workspace check says into the line, and
// `:messages` lists that once: here, the open note was deleted on the desk.
func TestSyncArrivalsKeepTheWorkspaceCheckMessage(t *testing.T) {
	t.Parallel()
	remote, _, h := arrivalWindow(t)
	desk := clone(t, remote)
	gitIn(t, desk, "rm", "-q", "doc.md")
	writeOutside(t, desk, "nota.md", "new")
	gitIn(t, desk, "add", "-A")
	gitIn(t, desk, "commit", "-q", "-m", "from desk")
	gitIn(t, desk, "push", "-q", "origin", "HEAD:main")
	h.finish(h.update(app.SyncNowMsg{}))
	wantWideBar(t, h, "Synced · \"doc.md\" was deleted or renamed outside Misoku")
	wantWideBar(t, h, "before recreating it · 1 new note from another computer: nota.md")
	v := messagesView(h)
	listed := 0
	for _, row := range strings.Split(v, "\n") {
		if strings.Contains(row, "14:05") && strings.Contains(row, "was deleted") {
			listed++
		}
	}
	if listed != 1 {
		t.Errorf(":messages must list the check's message once, in the sync's line:\n%s", v)
	}
}

// A1, S4: the bar cuts a long list and keeps the count; the bar says where
// the rest is.
func TestSyncBarCutsALongListKeepingTheCount(t *testing.T) {
	t.Parallel()
	remote, _, h := arrivalWindow(t)
	pushNotes(t, remote, numberedNotes(5)...)
	h.finish(h.update(app.SyncNowMsg{}))
	wantWideBar(t, h, "5 new notes from another computer: note01.md, note02.md, note03.md ... +2 more · :sync status")
	if bar := wideBar(h); strings.Contains(bar, "note04.md") {
		t.Errorf("the fourth name must be cut:\n%s", bar)
	}
}

// Exactly as many names as the bar shows are not cut.
func TestSyncBarDoesNotCutThreeNames(t *testing.T) {
	t.Parallel()
	remote, _, h := arrivalWindow(t)
	pushNotes(t, remote, numberedNotes(3)...)
	h.finish(h.update(app.SyncNowMsg{}))
	wantWideBar(t, h, "3 new notes from another computer: note01.md, note02.md, note03.md")
	if bar := wideBar(h); strings.Contains(bar, "more") || strings.Contains(bar, "Synced") {
		t.Errorf("nothing was cut, and the names replace a plain Synced:\n%s", bar)
	}
}

// arrivalRows is the note names listed in the `:sync status` dialog, read
// from the dialog's own rows: the explorer on screen also shows the names.
func arrivalRows(view string) []string {
	var names []string
	for _, row := range strings.Split(view, "\n") {
		fields := strings.Split(row, "│")
		if len(fields) < 2 {
			continue
		}
		if cell := strings.TrimSpace(fields[len(fields)-2]); strings.HasPrefix(cell, "note") && strings.HasSuffix(cell, ".md") {
			names = append(names, cell)
		}
	}
	return names
}

// S4: `:sync status` names the notes of the last sync that brought any, even
// after a later sync that brought none, with its usual cap.
func TestSyncStatusListsTheLastArrivals(t *testing.T) {
	t.Parallel()
	remote, _, h := arrivalWindow(t)
	if v := dialogText(openSyncStatus(h)); strings.Contains(v, "from another computer") {
		t.Errorf("no note arrived yet:\n%s", v)
	}
	h.key(tea.KeyEsc)
	pushNotes(t, remote, numberedNotes(5)...)
	h.finish(h.update(app.SyncNowMsg{}))
	h.finish(h.update(app.SyncNowMsg{}))
	v := openSyncStatus(h)
	if !strings.Contains(v, "5 new notes from another computer (just now):") {
		t.Errorf(":sync status must say what the last arrivals were:\n%s", v)
	}
	if got, want := arrivalRows(v), numberedNotes(5); !slices.Equal(got, want) {
		t.Errorf(":sync status lists %q, want %q:\n%s", got, want, v)
	}
}

func TestSyncStatusCapsALongListOfArrivals(t *testing.T) {
	t.Parallel()
	remote, _, h := arrivalWindow(t)
	pushNotes(t, remote, numberedNotes(12)...)
	h.finish(h.update(app.SyncNowMsg{}))
	v := openSyncStatus(h)
	if got, want := arrivalRows(v), numberedNotes(10); !slices.Equal(got, want) {
		t.Errorf(":sync status lists %q, want %q:\n%s", got, want, v)
	}
	if !strings.Contains(v, "… and 2 more") {
		t.Errorf("want the rest counted:\n%s", v)
	}
	if strings.Contains(v, "has them all") {
		t.Errorf("nothing keeps every name of a long list:\n%s", v)
	}
}

// messageRows is the rows of `:messages` only: the explorer on screen also shows
// the names.
func messageRows(h *harness) string {
	h.t.Helper()
	var listed []string
	for _, row := range strings.Split(messagesView(h), "\n") {
		if strings.Contains(row, "14:05") {
			listed = append(listed, row)
		}
	}
	return strings.Join(listed, "\n")
}

// S4: `:messages` lists the names of a list the bar cut, as many as `:sync
// status` does, and counts the rest.
func TestMessagesListTheNamesOfALongList(t *testing.T) {
	t.Parallel()
	remote, _, h := arrivalWindow(t)
	pushNotes(t, remote, numberedNotes(12)...)
	h.finish(h.update(app.SyncNowMsg{}))
	wantWideBar(t, h, "+9 more")
	rows := messageRows(h)
	for _, name := range numberedNotes(10) {
		if !strings.Contains(rows, name) {
			t.Errorf(":messages must list %s:\n%s", name, rows)
		}
	}
	if regexp.MustCompile(`14:05\s+note11`).MatchString(rows) || !strings.Contains(rows, "… and 2 more") {
		t.Errorf(":messages caps the list and counts the rest:\n%s", rows)
	}
}

// A huge list does not flush what `:messages` held.
func TestAHugeListKeepsEarlierMessages(t *testing.T) {
	t.Parallel()
	remote, _, h := arrivalWindow(t)
	h.send(core.StatusMsg{Text: "an earlier message"})
	pushNotes(t, remote, numberedNotes(150)...)
	h.finish(h.update(app.SyncNowMsg{}))
	h.send(tea.WindowSizeMsg{Width: 140, Height: 200})
	rows := messageRows(h)
	if !strings.Contains(rows, "an earlier message") {
		t.Errorf("earlier messages must stay:\n%s", rows)
	}
	if !strings.Contains(rows, "… and 140 more") {
		t.Errorf("the rest is counted:\n%s", rows)
	}
}

// S5: no dialog opens, and keys typed right after go on into the note.
func TestAutomaticSyncArrivalsOpenNoDialog(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	clock := newTestClock()
	h := window(t, vault, syncConfig(t, vault), app.WithClock(clock.Now))
	h.keys("1")
	h.keys("A one")
	pushNotes(t, remote, numberedNotes(5)...)
	clock.add(5 * time.Minute)
	if n := poll(h); n != 1 {
		t.Fatalf("want the timed sync to run (%d commands ran)", n)
	}
	wantBar(t, h, "5 new notes from another computer")
	if v := h.view(); strings.Contains(v, "╭") {
		t.Errorf("no dialog may open:\n%s", v)
	}
	h.keys(" two")
	if got := h.editor(); !strings.Contains(got, "base one two") {
		t.Fatalf("the keys must reach the note, got %q:\n%s", got, h.view())
	}
}

// barAt is the status bar as a window w columns wide draws it.
func barAt(h *harness, w int) string {
	h.t.Helper()
	h.send(tea.WindowSizeMsg{Width: w, Height: termH})
	lines := strings.Split(h.screen(), "\n")
	return lines[len(lines)-1]
}

// S1: with a note open, the notice stays whole at normal widths by giving up
// the right-side parts, and at 80 columns the compact form keeps the names.
func TestArrivalNoticeShowsAtNormalWidths(t *testing.T) {
	t.Parallel()
	const long = "2 new notes from another computer: nota.md, otra.md"
	remote, _, h := arrivalWindow(t)
	pushNotes(t, remote, "nota.md", "otra.md")
	h.finish(h.update(app.SyncNowMsg{}))
	for _, c := range []struct {
		width         int
		want, wantNot []string
	}{
		{160, []string{long, "1 word", "Markdown", "synced just now", "vault", "Ln 1, Col 1"}, nil},
		{120, []string{long, "Ln 1, Col 1"}, []string{"1 word"}},
		{100, []string{long, "Ln 1, Col 1"}, []string{"1 word", "Markdown"}},
		{80, []string{"2 new notes: nota.md, otra.md", "Ln 1, Col 1"}, []string{"another computer", "1 word", "Markdown", "vault"}},
	} {
		bar := barAt(h, c.width)
		for _, want := range c.want {
			if !strings.Contains(bar, want) {
				t.Errorf("at %d columns the bar must show %q:\n%s", c.width, want, bar)
			}
		}
		for _, not := range c.wantNot {
			if strings.Contains(bar, not) {
				t.Errorf("at %d columns the bar must not show %q:\n%s", c.width, not, bar)
			}
		}
	}
}

// A1: the compact form keeps the count, the names and the cut of a long list.
func TestArrivalNoticeCompactForms(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		added  []string
		edited bool // two.md, which is not the note on screen
		width  int
		want   string
	}{
		{"one new", []string{"a-long-name-of-a-note.md"}, false, 80, "1 new note: a-long-name-of-a-note.md"},
		{"one updated", nil, true, 70, "1 note updated: two.md"},
		{"both", []string{"a.md", "b.md"}, true, 90, "2 new notes, 1 updated: a.md, b.md, two.md"},
		{"cut as the last resort", []string{"a.md", "b.md"}, true, 80, "2 new notes, 1 updated: a.md, b.md, tw"},
		{"a long list", numberedNotes(5), false, 120, "5 new notes: note01.md, note02.md, note03.md ... +2 more · :sync status"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			remote, _, h := arrivalWindow(t)
			pushNotes(t, remote, "two.md")
			h.finish(h.update(app.SyncNowMsg{}))
			desk := clone(t, remote)
			for _, name := range c.added {
				writeOutside(t, desk, name, "new")
			}
			if c.edited {
				writeOutside(t, desk, "two.md", "edited")
			}
			gitIn(t, desk, "add", "-A")
			gitIn(t, desk, "commit", "-q", "-m", "from desk")
			gitIn(t, desk, "push", "-q", "origin", "HEAD:main")
			h.finish(h.update(app.SyncNowMsg{}))
			if bar := barAt(h, c.width); !strings.Contains(bar, c.want) {
				t.Errorf("at %d columns the bar must show %q:\n%s", c.width, c.want, bar)
			}
			if v := messagesView(h); !strings.Contains(v, "from another computer") {
				t.Errorf(":messages keeps the long form:\n%s", v)
			}
		})
	}
}

// Any other message draws as it did before the notice existed: these rows
// are what the bar drew on the base commit, 5a3d5c8.
func TestOtherMessagesDrawAsOnTheBase(t *testing.T) {
	t.Parallel()
	const long = "a long message that is not about notes arriving and that does not fit in the status bar at these widths at all"
	_, _, h := arrivalWindow(t)
	for _, c := range []struct {
		isErr bool
		width int
		want  string
	}{
		{false, 80, "  NORMAL  󰈙 doc.md   Markdown  󰓦 synced just now  󰉋  vault   Ln 1, Col 1 "},
		{false, 120, "  NORMAL  󰈙 doc.md   a long message that is not a 󰈭 1 word   Markdown  󰓦 synced just now  󰉋  vault   Ln 1, Col 1 "},
		{true, 80, "  NORMAL  󰈙 doc.md   Markdown  󰓦 synced just now  󰉋  vault   Ln 1, Col 1 "},
		{true, 120, "  NORMAL  󰈙 doc.md   a long message that is not a 󰈭 1 word   Markdown  󰓦 synced just now  󰉋  vault   Ln 1, Col 1 "},
	} {
		h.send(core.StatusMsg{Text: long, Error: c.isErr})
		if bar := barAt(h, c.width); bar != c.want {
			t.Errorf("error=%v at %d columns:\n got %q\nwant %q", c.isErr, c.width, bar, c.want)
		}
	}
}

// Other text in the line keeps its priority: the bar draws as it always does,
// parts included, and the sync's line reads in its long form.
func TestArrivalsSharingTheLineKeepTheParts(t *testing.T) {
	t.Parallel()
	remote, _, h := arrivalWindow(t)
	pushNotes(t, remote, "doc.md", "nota.md")
	h.finish(h.update(app.SyncNowMsg{}))
	bar := barAt(h, 120)
	for _, want := range []string{"Synced · \"doc.md\" reloaded", "Markdown", "Ln 1, Col 1"} {
		if !strings.Contains(bar, want) {
			t.Errorf("the bar must show %q:\n%s", want, bar)
		}
	}
}

// The parts the notice took come back when it clears, on the next key.
func TestRightPartsReturnWhenTheNoticeClears(t *testing.T) {
	t.Parallel()
	remote, _, h := arrivalWindow(t)
	pushNotes(t, remote, "nota.md", "otra.md")
	h.finish(h.update(app.SyncNowMsg{}))
	if bar := barAt(h, 100); strings.Contains(bar, "Markdown") || !strings.Contains(bar, "otra.md") {
		t.Fatalf("setup: the notice takes the parts' room:\n%s", bar)
	}
	h.keys("l")
	bar := barAt(h, 100)
	if strings.Contains(bar, "another computer") {
		t.Errorf("the notice clears on the next key:\n%s", bar)
	}
	for _, want := range []string{"1 word", "Markdown", "vault"} {
		if !strings.Contains(bar, want) {
			t.Errorf("the bar must show %q again:\n%s", want, bar)
		}
	}
}

// On the start screen `:` opens no command line, so the bar keeps the count of
// the rest and leaves the command out, whatever the width.
func TestStartScreenCutsALongListWithoutACommand(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	h := wideWindow(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	pushNotes(t, remote, numberedNotes(13)...)
	h.finish(h.update(app.SyncNowMsg{}))
	wantLine(t, h, "13 new notes from another computer: note01.md, note02.md, note03.md ... +10 more")
	for _, w := range []int{1000, 110, 90} {
		if got := barAt(h, w); strings.Contains(got, ":sync") || !strings.Contains(got, "... +10 more") {
			t.Errorf("at %d columns the bar must keep the count and suggest no command:\n%s", w, got)
		}
	}
}
