package core

import (
	"reflect"
	"testing"
	"time"
)

// suggestNotes is a small vault: Docker at the root, AWS Notes under tech.
func suggestNotes() []LinkNote {
	at := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	return []LinkNote{
		{ID: "Docker.md", Title: "Docker", Text: "# Docker\n\ncontainers", UpdatedAt: at.Add(-time.Hour)},
		{ID: "tech/AWS Notes.md", Title: "AWS Notes", Text: awsText(), UpdatedAt: at},
	}
}

func awsText() string {
	return "# AWS Notes\n\n## S3 Storage\n\nstores the files\n\n^s3-basics\n\n## EC2 Compute\n\nvirtual machines\n"
}

// inserts lists what a suggestion list would write between "[[" and "]]".
func inserts(list []LinkSuggestion) []string {
	out := make([]string, len(list))
	for i, s := range list {
		out[i] = s.Insert
	}
	return out
}

// S1: a bare name lists notes.
func TestLinkSuggestionsListNotes(t *testing.T) {
	notes := suggestNotes()
	if got := inserts(LinkSuggestions(notes, "tech/AWS Notes.md", awsText(), "")); !reflect.DeepEqual(got, []string{"AWS Notes", "Docker"}) {
		t.Fatalf("empty query = %v, want the notes, newest first", got)
	}
	if got := inserts(LinkSuggestions(notes, "", "", "dock")); !reflect.DeepEqual(got, []string{"Docker"}) {
		t.Fatalf("dock = %v", got)
	}
	if got := inserts(LinkSuggestions(notes, "", "", "AWS Notes")); !reflect.DeepEqual(got, []string{"AWS Notes"}) {
		t.Fatalf("AWS Notes = %v", got)
	}
	if got := LinkSuggestions(notes, "", "", "nothing like it"); len(got) != 0 {
		t.Fatalf("no note should match: %v", inserts(got))
	}
}

// S1: "#" lists the headings of the note being edited, filtered.
func TestLinkSuggestionsHeadingsOfTheCurrentNote(t *testing.T) {
	notes := suggestNotes()
	got := LinkSuggestions(notes, "tech/AWS Notes.md", awsText(), "#")
	want := []string{"#AWS Notes", "#S3 Storage", "#EC2 Compute"}
	if inserts(got) == nil || !reflect.DeepEqual(inserts(got), want) {
		t.Fatalf("# = %v, want %v", inserts(got), want)
	}
	if got[1].Kind != LinkKindHeading || got[1].Label != "S3 Storage" || got[1].NoteID != "tech/AWS Notes.md" {
		t.Fatalf("a heading suggestion should name its note: %+v", got[1])
	}
	if got = LinkSuggestions(notes, "tech/AWS Notes.md", awsText(), "#s3"); !reflect.DeepEqual(inserts(got), []string{"#S3 Storage"}) {
		t.Fatalf("#s3 = %v", inserts(got))
	}
	// The editor's text wins over the one the notes list holds.
	live := awsText() + "\n## Brand New\n"
	if got = LinkSuggestions(notes, "tech/AWS Notes.md", live, "#brand"); !reflect.DeepEqual(inserts(got), []string{"#Brand New"}) {
		t.Fatalf("live text = %v", inserts(got))
	}
}

// S1: "Note#" lists the headings of that note, resolved like a link.
func TestLinkSuggestionsHeadingsOfAnotherNote(t *testing.T) {
	notes := suggestNotes()
	got := LinkSuggestions(notes, "", "# Docker", "AWS Notes#ec2")
	if !reflect.DeepEqual(inserts(got), []string{"AWS Notes#EC2 Compute"}) {
		t.Fatalf("AWS Notes#ec2 = %v", inserts(got))
	}
	if got[0].Detail != "AWS Notes" || got[0].Label != "EC2 Compute" {
		t.Fatalf("the note should say which note the heading is in: %+v", got[0])
	}
	if got := LinkSuggestions(notes, "", "", "Missing#ec2"); len(got) != 0 {
		t.Fatalf("a note that does not exist has no headings: %v", inserts(got))
	}
	if got := LinkSuggestions(notes, "", "", "AWS Notes#nothing"); len(got) != 0 {
		t.Fatalf("no heading should match: %v", inserts(got))
	}
}

// S1: "#^" lists block ids, each with the line it marks.
func TestLinkSuggestionsBlocks(t *testing.T) {
	notes := suggestNotes()
	got := LinkSuggestions(notes, "tech/AWS Notes.md", awsText(), "#^")
	if !reflect.DeepEqual(inserts(got), []string{"#^s3-basics"}) {
		t.Fatalf("#^ = %v", inserts(got))
	}
	if got[0].Label != "^s3-basics" || got[0].Detail != "stores the files" || got[0].Kind != LinkKindBlock {
		t.Fatalf("a block should show its id and line: %+v", got[0])
	}
	if got = LinkSuggestions(notes, "", "", "AWS Notes#^s3"); !reflect.DeepEqual(inserts(got), []string{"AWS Notes#^s3-basics"}) {
		t.Fatalf("AWS Notes#^s3 = %v", inserts(got))
	}
	if got = LinkSuggestions(notes, "tech/AWS Notes.md", awsText(), "#^nope"); len(got) != 0 {
		t.Fatalf("no block should match: %v", inserts(got))
	}
}

// S1: headings match ignoring case and keeping accents.
func TestLinkSuggestionsHeadingMatchIgnoresCase(t *testing.T) {
	notes := []LinkNote{{ID: "a.md", Title: "A", Text: "## Configuración inicial\n\n## CONFIG extra\n"}}
	got := LinkSuggestions(notes, "a.md", notes[0].Text, "#CONFIG")
	want := []string{"#Configuración inicial", "#CONFIG extra"}
	if !reflect.DeepEqual(inserts(got), want) {
		t.Fatalf("CONFIG = %v, want %v", inserts(got), want)
	}
}

// S1: two headings with the same text complete the same link once.
func TestLinkSuggestionsDuplicateHeadingsOnce(t *testing.T) {
	notes := []LinkNote{{ID: "a.md", Title: "A", Text: "## Setup\n\n## Setup\n\n## Deploy\n"}}
	if got := inserts(LinkSuggestions(notes, "a.md", notes[0].Text, "#")); !reflect.DeepEqual(got, []string{"#Setup", "#Deploy"}) {
		t.Fatalf("# = %v, want each heading once", got)
	}
}

// S1: notes with the same title link by path, as the finder's picker does.
func TestLinkSuggestionsDuplicateTitlesUseThePath(t *testing.T) {
	notes := []LinkNote{
		{ID: "one/Note.md", Title: "Note"},
		{ID: "two/Note.md", Title: "Note"},
	}
	want := []string{"one/Note", "two/Note"}
	if got := inserts(LinkSuggestions(notes, "", "", "note")); !reflect.DeepEqual(got, want) {
		t.Fatalf("note = %v, want %v", got, want)
	}
	if notes := []LinkNote{{ID: "one/Note.md", Title: "Note", Text: "## H\n"}}; len(LinkSuggestions(notes, "one/Note.md", notes[0].Text, "#")) != 1 {
		t.Fatal("a single note links by name")
	}
}

// S1: a note with no headings or blocks suggests nothing, and does not panic.
func TestLinkSuggestionsEmptyNote(t *testing.T) {
	notes := []LinkNote{{ID: "a.md", Title: "A", Text: "plain text only"}}
	for _, q := range []string{"#", "#^", "#x"} {
		if got := LinkSuggestions(notes, "a.md", notes[0].Text, q); len(got) != 0 {
			t.Fatalf("%s = %v, want nothing", q, inserts(got))
		}
	}
}

// A5: a heading with "[", "]", "|" or a leading "^" is written so
// ParseWikiLink reads it back and LinkLine still finds it.
func TestLinkHeadingIsSafeToInsert(t *testing.T) {
	for _, heading := range []string{
		"A [draft] section",
		"Pipes | here",
		"^ caret start",
		"^^ double caret",
		"Hash # inside",
		"S3 Storage!",
	} {
		text := "## " + heading + "\n\nbody\n"
		notes := []LinkNote{{ID: "a.md", Title: "A", Text: text}}
		got := LinkSuggestions(notes, "a.md", text, "#")
		if len(got) != 1 {
			t.Fatalf("%q: %v", heading, inserts(got))
		}
		l := ParseWikiLink(got[0].Insert)
		if l.Target != "" || l.Block != "" || l.Heading == "" {
			t.Fatalf("%q inserted %q, which parses as %+v", heading, got[0].Insert, l)
		}
		if line := LinkLine(text, l); line != 0 {
			t.Fatalf("%q inserted %q, which points at line %d", heading, got[0].Insert, line)
		}
		if _, ok := LinkSection(text, l); !ok {
			t.Fatalf("%q inserted %q, which finds no section", heading, got[0].Insert)
		}
	}
}

// S1: Enter with no suggestion completes a plain name and nothing else.
func TestLinkCompletion(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"", ""},
		{"   ", ""},
		{"Brand New", "Brand New"},
		{" docker ", "docker"},
		{"#Top", ""},
		{"Note#Top", ""},
		{"#^block", ""},
	} {
		if got := LinkCompletion(c.in); got != c.want {
			t.Errorf("LinkCompletion(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// A4: headings and block ids skip fenced code.
func TestHeadingsAndBlockIDsSkipCode(t *testing.T) {
	text := "# Real\n\n```\n## Not a heading\n^not-a-block\n```\n\n## Also real\n\nend ^real\n"
	if got := Headings(text); !reflect.DeepEqual(got, []string{"Real", "Also real"}) {
		t.Fatalf("Headings = %v", got)
	}
	got := BlockIDs(text)
	if len(got) != 1 || got[0].ID != "real" || got[0].Text != "end" {
		t.Fatalf("BlockIDs = %+v", got)
	}
}
