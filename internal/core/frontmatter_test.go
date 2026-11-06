package core_test

import (
	"reflect"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
)

func TestParseFrontMatterClosersAndBody(t *testing.T) {
	cases := []struct {
		name, text, closer, body string
		end                      int
	}{
		{"dashes", "---\ntitle: A\n---\nbody", "---", "body", 2},
		{"dots", "---\ntitle: A\n...\nbody", "...", "body", 2},
		{"crlf", "---\r\ntitle: A\r\n---\r\nbody\r\n", "---", "body\r\n", 2},
		{"empty block", "---\n---\nbody", "---", "body", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fm := core.ParseFrontMatter(c.text)
			if fm.StartLine != 0 || fm.EndLine != c.end || fm.Closer != c.closer || fm.Body != c.body {
				t.Fatalf("got start=%d end=%d closer=%q body=%q", fm.StartLine, fm.EndLine, fm.Closer, fm.Body)
			}
		})
	}
}

func TestParseFrontMatterNone(t *testing.T) {
	for _, text := range []string{"", "# Title\nbody", "---\nnot closed\n# x", "---", "text\n---\nmore\n---"} {
		fm := core.ParseFrontMatter(text)
		if fm.StartLine != -1 || fm.EndLine != -1 || fm.Body != text || len(fm.Fields) != 0 || fm.Closer != "" {
			t.Errorf("%q: got %+v", text, fm)
		}
	}
}

func TestParseFrontMatterFields(t *testing.T) {
	text := "---\r\nTitle: \"Hello: world\"\r\ntags: [go, 'tui', ]\r\naliases:\r\n  - one\r\n  - two\r\nstatus: active\r\n---\r\nbody"
	fm := core.ParseFrontMatter(text)
	want := []core.Field{
		{Key: "Title", Value: "Hello: world", Line: 1, EndLine: 1},
		{Key: "tags", Value: "[go, 'tui', ]", List: []string{"go", "tui"}, Line: 2, EndLine: 2},
		{Key: "aliases", Value: "", List: []string{"one", "two"}, Line: 3, EndLine: 5},
		{Key: "status", Value: "active", Line: 6, EndLine: 6},
	}
	if !reflect.DeepEqual(fm.Fields, want) {
		t.Fatalf("Fields =\n%#v\nwant\n%#v", fm.Fields, want)
	}
}

func TestFrontMatterGetIsCaseInsensitive(t *testing.T) {
	fm := core.ParseFrontMatter("---\nTitle: A\nCOVER: c.png\n---\n")
	if f, ok := fm.Get("title"); !ok || f.Value != "A" || f.Key != "Title" {
		t.Errorf("Get(title) = %+v, %v", f, ok)
	}
	if f, ok := fm.Get("Cover"); !ok || f.Value != "c.png" {
		t.Errorf("Get(Cover) = %+v, %v", f, ok)
	}
	if _, ok := fm.Get("missing"); ok {
		t.Error("Get(missing) should report false")
	}
}

func TestFieldItems(t *testing.T) {
	fm := core.ParseFrontMatter("---\na: x, y ,, z\nb: [p, q]\nc:\n- r\n---\n")
	for key, want := range map[string][]string{"a": {"x", "y", "z"}, "b": {"p", "q"}, "c": {"r"}} {
		f, _ := fm.Get(key)
		if got := f.Items(); !reflect.DeepEqual(got, want) {
			t.Errorf("%s Items = %q, want %q", key, got, want)
		}
	}
}

func TestFrontMatterEndAndPredicates(t *testing.T) {
	cases := []struct {
		lines []string
		want  int
	}{
		{[]string{"---", "a: 1", "---", "x"}, 2},
		{[]string{"---\r", "a: 1\r", "...\r", "x"}, 2},
		{[]string{"---", "a: 1"}, -1},
		{[]string{"---"}, -1},
		{[]string{"x", "---", "---"}, -1},
		{nil, -1},
	}
	for _, c := range cases {
		if got := core.FrontMatterEnd(c.lines); got != c.want {
			t.Errorf("FrontMatterEnd(%q) = %d, want %d", c.lines, got, c.want)
		}
	}
	if !core.IsFrontMatterOpen("---\r") || core.IsFrontMatterOpen("...") {
		t.Error("IsFrontMatterOpen")
	}
	if !core.IsFrontMatterClose("...") || !core.IsFrontMatterClose("---") || core.IsFrontMatterClose("----") {
		t.Error("IsFrontMatterClose")
	}
}

func TestSetFrontMatterField(t *testing.T) {
	cases := []struct {
		name, text, key, value, want string
	}{
		{"replace keeps key spelling", "---\nCover: old.png\ntitle: T\n---\nbody", "cover", "new.png",
			"---\nCover: new.png\ntitle: T\n---\nbody"},
		{"append before dashes", "---\ntitle: T\n---\nbody", "cover", "c.png",
			"---\ntitle: T\ncover: c.png\n---\nbody"},
		{"append before dots closer", "---\ntitle: T\n...\nbody", "cover", "c.png",
			"---\ntitle: T\ncover: c.png\n...\nbody"},
		{"create block", "body", "cover", "c.png", "---\ncover: c.png\n---\nbody"},
		{"create block on empty note", "", "status", "active", "---\nstatus: active\n---\n"},
		{"unclosed is not a block", "---\nbody", "status", "active", "---\nstatus: active\n---\n---\nbody"},
		{"crlf append", "---\r\ntitle: T\r\n---\r\nbody", "status", "done",
			"---\r\ntitle: T\r\nstatus: done\r\n---\r\nbody"},
		{"crlf create", "a\r\nb", "status", "done", "---\r\nstatus: done\r\n---\r\na\r\nb"},
		{"replace list", "---\ntags:\n  - a\n  - b\nx: 1\n---\n", "tags", "c",
			"---\ntags: c\nx: 1\n---\n"},
		{"empty value", "---\n---\n", "status", "", "---\nstatus:\n---\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := core.SetFrontMatterField(c.text, c.key, c.value); got != c.want {
				t.Fatalf("got\n%q\nwant\n%q", got, c.want)
			}
		})
	}
}

func TestSetFrontMatterFieldQuoting(t *testing.T) {
	cases := map[string]string{
		"plain":      "plain",
		"a: b":       `"a: b"`,
		"#tag":       `"#tag"`,
		"[x]":        `"[x]"`,
		"{x}":        `"{x}"`,
		"'q'":        `"'q'"`,
		`"q"`:        `'"q"'`,
		" edge":      `" edge"`,
		"edge ":      `"edge "`,
		"a:b":        "a:b",
		"mid # hash": "mid # hash",
	}
	for value, want := range cases {
		got := core.SetFrontMatterField("---\n---\n", "k", value)
		if got != "---\nk: "+want+"\n---\n" {
			t.Errorf("%q: got %q", value, got)
		}
		if f, _ := core.ParseFrontMatter(got).Get("k"); value != `"q"` && value != "'q'" && f.Value != value {
			t.Errorf("%q does not round-trip: %q", value, f.Value)
		}
	}
}
