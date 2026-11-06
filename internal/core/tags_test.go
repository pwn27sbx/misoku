package core

import (
	"reflect"
	"testing"
)

func TestTagSpans(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []string // the text each span covers, "#" included
	}{
		{"plain", "learn #go today", []string{"#go"}},
		{"line start", "#go", []string{"#go"}},
		{"several", "#a and #b", []string{"#a", "#b"}},
		{"unicode letters", "sobre #diseño gráfico", []string{"#diseño"}},
		{"nested and symbols", "#lang/go_1-x.", []string{"#lang/go_1-x"}},
		{"trailing slash dropped", "#go/ next", []string{"#go"}},
		{"heading", "# Heading", nil},
		{"double hash", "##go", nil},
		{"digit first", "#123 issue", nil},
		{"after word char", "a#b", nil},
		{"after slash", "x/#y", nil},
		{"html entity", "&#123; and &#x41;", nil},
		{"url fragment", "see https://x/#frag", nil},
		{"url query fragment", "see https://x.com/a?b=#c", nil},
		{"code span", "run `#go` now", nil},
		{"code span then tag", "run `x` #go", []string{"#go"}},
		{"wiki link", "[[Note#Section]] and [[#heading]]", nil},
		{"embed", "![[pic.png#x]]", nil},
		{"markdown link", "[#tag text](note.md#part) #real", []string{"#real"}},
		{"html comment", "<!-- #hidden --> #shown", []string{"#shown"}},
		{"after punctuation", "(#go), #rust!", []string{"#go", "#rust"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, s := range TagSpans(tt.line) {
				got = append(got, tt.line[s[0]:s[1]])
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TagSpans(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestTags(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{"none", "just text", nil},
		{"inline lowercased and deduplicated", "#Go then #go and #Rust", []string{"go", "rust"}},
		{"front matter scalar", "---\ntags: a, #B\n---\nbody #c", []string{"a", "b", "c"}},
		{"front matter brackets", "---\ntags: [x, \"#y\"]\n---\n", []string{"x", "y"}},
		{"front matter list", "---\ntitle: T\ntags:\n  - one\n  - '#two'\n---\n#one", []string{"one", "two"}},
		{"front matter key any case", "---\nTags: a\n---\n", []string{"a"}},
		{"hash in front matter is not inline", "---\ntitle: #nope\n---\n", nil},
		{"fence skipped", "```\n#code\n```\n#after", []string{"after"}},
		{"tilde fence skipped", "~~~go\n#code\n~~~\n", nil},
		{"multi-line comment", "<!--\n#hidden\n-->#shown", []string{"shown"}},
		{"crlf", "#a\r\n#b\r\n", []string{"a", "b"}},
		{"heading not a tag", "# Title\n## Sub #real", []string{"real"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Tags(tt.text); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Tags(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}
