package app_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// guideFooter is the leader guide's last line: it is on screen exactly
// while the guide is.
const guideFooter = "These keys come from config.json"

func guideOpen(h *harness) bool { return strings.Contains(h.view(), guideFooter) }

// tickGuide delivers the timer of the latest Space press, as if the delay
// had passed.
func tickGuide(h *harness) {
	h.t.Helper()
	h.send(app.LeaderGuideTick(app.LeaderSeq(h.m)))
}

// guideGroup returns the group whose column holds text in the guide: the
// header row names the three groups, and text belongs to the rightmost
// header that starts at or before it.
func guideGroup(t *testing.T, view, text string) string {
	t.Helper()
	header := -1
	lines := strings.Split(view, "\n")
	for y, l := range lines {
		if strings.Contains(l, "NOTE") && strings.Contains(l, "VIEW") && strings.Contains(l, " GO") {
			header = y
			break
		}
	}
	if header < 0 {
		t.Fatalf("no NOTE / VIEW / GO header on screen:\n%s", view)
	}
	x, y := find(t, view, text)
	if y <= header {
		t.Fatalf("%q is not below the group headers:\n%s", text, view)
	}
	got := ""
	for _, g := range []string{"NOTE", "VIEW", " GO"} {
		gx, _ := find(t, lines[header], g)
		if gx <= x+1 {
			got = strings.TrimSpace(g)
		}
	}
	return got
}

// guideRow reports whether a guide row shows key and then help.
func guideRow(view, key, help string) bool {
	re := regexp.MustCompile(`(^|\s)` + regexp.QuoteMeta(key) + `\s+` + regexp.QuoteMeta(help))
	return re.MatchString(view)
}

func TestLeaderGuideAppearsAfterTheDelay(t *testing.T) {
	h := openNote(t)
	if app.LeaderGuideDelay != 400*time.Millisecond {
		t.Fatalf("delay is %v, want 400ms (D1)", app.LeaderGuideDelay)
	}
	// The real timer: Space returns a command that waits the delay and
	// then delivers the tick of that press.
	start := time.Now()
	m, cmd := h.m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	h.m = m
	if cmd == nil {
		t.Fatal("Space started no timer")
	}
	if guideOpen(h) {
		t.Fatalf("the guide showed before the delay:\n%s", h.view())
	}
	msg := cmd()
	if d := time.Since(start); d < app.LeaderGuideDelay {
		t.Fatalf("the timer fired after %v, want at least %v", d, app.LeaderGuideDelay)
	}
	if msg != app.LeaderGuideTick(app.LeaderSeq(h.m)) {
		t.Fatalf("the timer delivered %#v", msg)
	}
	h.send(msg)
	if !guideOpen(h) {
		t.Fatalf("the guide did not show after the delay:\n%s", h.view())
	}
}

func TestLeaderGuideGroupsTheFollowingKeys(t *testing.T) {
	h := openNote(t)
	h.keys(" ")
	tickGuide(h)
	v := h.view()
	cases := map[string]string{
		"New note":       "NOTE",
		"Export note":    "NOTE",
		"Toggle preview": "VIEW",
		"Themes":         "VIEW",
		"Focus explorer": "GO",
		"Home screen":    "GO",
	}
	for text, want := range cases {
		if got := guideGroup(t, v, text); got != want {
			t.Errorf("%q is under %s, want %s:\n%s", text, got, want, v)
		}
	}
	// Each row starts with the key that follows Space.
	if !guideRow(v, "Shift+P", "Export note") {
		t.Errorf("the export row should start with its key Shift+P:\n%s", v)
	}
	// Ctrl-only actions have no leader key and stay out.
	if strings.Contains(v, "Pane to the left") {
		t.Errorf("an action without a leader key is listed:\n%s", v)
	}
}

func TestLeaderGuideFollowsTheConfiguredKeys(t *testing.T) {
	h := openNote(t, app.WithKeys(map[string][]string{
		"toggle_explorer": {"space e"},
		"focus_explorer":  {},
		"toggle_preview":  {"space v"},
		"themes":          {},
	}))
	h.keys(" ")
	tickGuide(h)
	v := h.view()
	if !guideRow(v, "v", "Toggle preview") {
		t.Errorf("the moved toggle_preview should show its new key v:\n%s", v)
	}
	if !guideRow(v, "e", "Toggle explorer") {
		t.Errorf("toggle_explorer moved to space e should show its key e:\n%s", v)
	}
	if got := guideGroup(t, v, "Toggle explorer"); got != "VIEW" {
		t.Errorf("toggle_explorer moved to space e should show under VIEW, got %s:\n%s", got, v)
	}
	for _, gone := range []string{"Themes", "Focus explorer"} {
		if strings.Contains(v, gone) {
			t.Errorf("unbound %q is still listed:\n%s", gone, v)
		}
	}
}

func TestLeaderGuideEscClosesWithoutRunningAnything(t *testing.T) {
	h := openNote(t)
	before := h.view()
	h.keys(" ")
	tickGuide(h)
	if !guideOpen(h) {
		t.Fatalf("setup: the guide should be open:\n%s", h.view())
	}
	h.key(tea.KeyEsc)
	if after := h.view(); after != before {
		t.Fatalf("Esc should close the guide and change nothing else:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestLeaderGuideUnknownKeyClosesAndWarns(t *testing.T) {
	h := openNote(t)
	// Before the guide shows, an unknown key ends the sequence quietly, as
	// it always did.
	h.keys(" z")
	if v := h.view(); strings.Contains(v, "not a leader key") {
		t.Fatalf("a fast unknown key should not warn:\n%s", v)
	}
	h.keys(" ")
	tickGuide(h)
	h.keys("z")
	v := h.view()
	if strings.Contains(v, guideFooter) {
		t.Fatalf("an unknown key should close the guide:\n%s", v)
	}
	if !strings.Contains(v, "z: not a leader key") {
		t.Fatalf("an unknown key should say so:\n%s", v)
	}
}

func TestLeaderGuideNeverShowsForAFastSequence(t *testing.T) {
	h := openNote(t)
	h.keys(" ")
	seq := app.LeaderSeq(h.m)
	h.keys("p") // toggle preview, before the delay
	if !strings.Contains(h.view(), "hello") {
		t.Fatalf("setup: the note should be on screen:\n%s", h.view())
	}
	previewHidden := h.view()
	h.send(app.LeaderGuideTick(seq)) // the timer fires after the sequence ended
	if v := h.view(); strings.Contains(v, guideFooter) || v != previewHidden {
		t.Fatalf("a tick after the sequence ended must change nothing:\n%s", v)
	}
}

func TestLeaderGuideIgnoresAStaleTick(t *testing.T) {
	h := openNote(t)
	h.keys(" ")
	first := app.LeaderSeq(h.m)
	h.key(tea.KeyEsc) // cancel the first sequence
	h.keys(" ")       // a new one starts and is still pending
	if app.LeaderSeq(h.m) == first {
		t.Fatal("a new Space press should get a new sequence number")
	}
	h.send(app.LeaderGuideTick(first))
	if guideOpen(h) {
		t.Fatalf("the first press's timer opened the guide of the second:\n%s", h.view())
	}
	tickGuide(h)
	if !guideOpen(h) {
		t.Fatalf("the current press's timer should open the guide:\n%s", h.view())
	}
}

func TestLeaderKeyStillRunsWhileTheGuideIsOpen(t *testing.T) {
	h := openNote(t)
	before := h.view()
	h.keys(" ")
	tickGuide(h)
	h.keys("p") // toggle_preview hides the preview
	v := h.view()
	if strings.Contains(v, guideFooter) {
		t.Fatalf("running an action should close the guide:\n%s", v)
	}
	if v == before {
		t.Fatalf("space p with the guide open should toggle the preview:\n%s", v)
	}
	h.keys(" ")
	if guideOpen(h) {
		t.Fatalf("the next Space should wait again before showing the guide:\n%s", h.view())
	}
	h.keys("p") // and a fast space p brings the preview back
	if v := h.view(); v != before {
		t.Fatalf("a fast space p should toggle the preview back:\n%s", v)
	}
}

// On a small terminal the groups stack and the footer still shows.
func TestLeaderGuideFitsASmallTerminal(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "hello"); err != nil {
		t.Fatal(err)
	}
	const w, ht = 60, 24
	h := newHarnessSize(t, w, ht, store)
	h.keys("1")
	h.key(tea.KeyEsc)
	h.keys(" ")
	tickGuide(h)
	v := ansi.Strip(h.m.View())
	lines := strings.Split(v, "\n")
	if len(lines) != ht {
		t.Fatalf("frame has %d rows, want %d", len(lines), ht)
	}
	for i, l := range lines {
		if lw := ansi.StringWidth(l); lw != w {
			t.Fatalf("row %d is %d cells wide, want %d:\n%s", i, lw, w, v)
		}
	}
	if !strings.Contains(v, guideFooter) {
		t.Fatalf("the footer should show on a small terminal:\n%s", v)
	}
}
