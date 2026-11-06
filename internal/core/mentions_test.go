package core_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
)

// mentioned returns the text of every mention Mentions finds.
func mentioned(text, title string) []string {
	var out []string
	for _, r := range core.Mentions(text, title) {
		out = append(out, text[r[0]:r[1]])
	}
	return out
}

// S8: a title is matched as a whole word in any case, accents and "ñ"
// included, outside code, links, URLs, comments and the front matter.
func TestMentions(t *testing.T) {
	for _, tc := range []struct {
		name, title, text string
		want              []string
	}{
		{"plain", "Docker", "I use Docker daily", []string{"Docker"}},
		{"any case", "Docker", "docker and DOCKER", []string{"docker", "DOCKER"}},
		{"accented title", "Diseño", "El diseño y el DISEÑO.", []string{"diseño", "DISEÑO"}},
		{"accented letter is no boundary", "Diseño", "rediseño, diseñoso", nil},
		{"partial word", "Docker", "Dockerfile and mydocker", nil},
		{"digit is no boundary", "Go", "Go2 go", []string{"go"}},
		{"regex characters", "C++ (notes)", "see C++ (notes), not C+ (notes) nor Cx+ (notes)", []string{"C++ (notes)"}},
		{"regex dot is literal", "a.b", "axb a.b", []string{"a.b"}},
		{"inline code", "Docker", "run `Docker` here", nil},
		{"fenced code", "Docker", "```\nDocker\n```\nDocker", []string{"Docker"}},
		{"tilde fence", "Docker", "~~~sh\nDocker\n~~~", nil},
		{"wiki link", "Docker", "[[Docker]] and [[Other|Docker]] and ![[Docker]]", nil},
		{"markdown link", "Docker", "[Docker](docker.md) and ![Docker](d.png)", nil},
		{"url", "Docker", "https://example.com/Docker/page", nil},
		{"html comment", "Docker", "<!-- Docker -->Docker", []string{"Docker"}},
		{"comment over lines", "Docker", "<!--\nDocker\n-->\nafter Docker", []string{"Docker"}},
		{"front matter", "Docker", "---\ntitle: Docker\n---\nDocker body", []string{"Docker"}},
		{"crlf", "Docker", "Docker\r\nDocker", []string{"Docker", "Docker"}},
		{"inline tags", "Diseño", "#diseño #Diseño/ux tag", nil},
		{"title inside a tag", "Diseño", "#proj-Diseño #mi_diseño", nil},
		{"after a hash", "Diseño", "##Diseño and a#Diseño", nil},
		{"tag beside a mention", "Diseño", "#diseño Diseño", []string{"Diseño"}},
		{"email", "Zentro", "mail juan@zentro.com", nil},
		{"domain without scheme", "Zentro", "www.zentro.com and zentro.io", nil},
		{"path", "Diseño", "ruta Proyectos/Diseño.md", nil},
		{"title ends an address", "Zentro", "juan@zentro, www.zentro and Proyectos/Zentro", nil},
		{"folder in a path", "Diseño", "Diseño/notas.md", nil},
		{"sentence ends", "Zentro", "I use Zentro. Zentro/ and Zentro.", []string{"Zentro", "Zentro", "Zentro"}},
		{"reference link", "Diseño", "[Diseño][1] and [Diseño]: https://x.io", nil},
		{"reference link with parens", "Diseño", "[Diseño] (x)", []string{"Diseño"}},
		{"block id", "abc", "text ^abc", nil},
		{"title that no link can hold", "C#", "I like C# a lot", nil},
		{"empty title", "", "anything", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := mentioned(tc.text, tc.title); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Mentions(%q, %q) = %q, want %q", tc.text, tc.title, got, tc.want)
			}
		})
	}
}

// S8: the offsets point into the whole text, front matter included.
func TestMentionsOffsets(t *testing.T) {
	text := "---\nk: v\n---\nab Docker"
	got := core.Mentions(text, "Docker")
	if len(got) != 1 || text[got[0][0]:got[0][1]] != "Docker" || got[0][0] != len("---\nk: v\n---\nab ") {
		t.Fatalf("Mentions = %v", got)
	}
}

// S8: every mention becomes [[target]], or [[target|original]] when it is
// written differently; links, code and front matter stay as written.
func TestLinkMentions(t *testing.T) {
	for _, tc := range []struct {
		name, text, title, target, want string
		n                               int
	}{
		{"same case", "about Diseño.", "Diseño", "Diseño", "about [[Diseño]].", 1},
		{"case differs", "el diseño y Diseño", "Diseño", "Diseño", "el [[Diseño|diseño]] y [[Diseño]]", 2},
		{"regex title", "C++ (notes) rock", "C++ (notes)", "C++ (notes)", "[[C++ (notes)]] rock", 1},
		{"path target", "see Docker", "Docker", "tech/Docker", "see [[tech/Docker|Docker]]", 1},
		{"skips the rest", "---\ntitle: Docker\n---\n`Docker` [[Docker]] Docker", "Docker", "Docker",
			"---\ntitle: Docker\n---\n`Docker` [[Docker]] [[Docker]]", 1},
		{"tags stay", "#diseño #Diseño/ux tag", "Diseño", "Diseño", "#diseño #Diseño/ux tag", 0},
		{"emails and domains stay", "mail juan@zentro.com y www.zentro.com", "Zentro", "Zentro", "mail juan@zentro.com y www.zentro.com", 0},
		{"paths stay", "ruta Proyectos/Diseño.md", "Diseño", "Diseño", "ruta Proyectos/Diseño.md", 0},
		{"reference links stay", "[Diseño][1]\n\n[1]: https://x.io", "Diseño", "Diseño", "[Diseño][1]\n\n[1]: https://x.io", 0},
		{"block ids stay", "text ^abc", "abc", "abc", "text ^abc", 0},
		{"nothing to link", "no mention", "Docker", "Docker", "no mention", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, n := core.LinkMentions(tc.text, tc.title, tc.target)
			if got != tc.want || n != tc.n {
				t.Fatalf("LinkMentions = %q, %d; want %q, %d", got, n, tc.want, tc.n)
			}
		})
	}
}

// S8: the notes listed mention the title without linking to it; the note
// itself, notes that link to it (wiki or Markdown link), skipped notes and
// notes that fail to read are left out. Rows are sorted by title and say
// where the first mention is.
func TestFindUnlinkedMentions(t *testing.T) {
	content := map[string]string{
		"Diseño.md":       "# Diseño\n\nDiseño itself",
		"b.md":            "intro\nhablando de diseño, y Diseño otra vez",
		"a.md":            "Diseño here",
		"linked.md":       "Diseño and [[Diseño]]",
		"mdlinked.md":     "Diseño and [x](Diseño.md)",
		"none.md":         "nothing",
		"tpl/Template.md": "Diseño template",
		"broken.md":       "Diseño",
	}
	var notes []core.Page
	for _, id := range []string{"Diseño.md", "a.md", "b.md", "broken.md", "linked.md", "mdlinked.md", "none.md", "tpl/Template.md"} {
		notes = append(notes, core.Page{ID: id, Title: id[:len(id)-3]})
	}
	notes = append(notes, core.Page{ID: "tpl", Title: "tpl", IsFolder: true})
	read := func(id string) (string, error) {
		if id == "broken.md" {
			return "", errors.New("unreadable")
		}
		return content[id], nil
	}
	skip := func(id string) bool { return id == "tpl/Template.md" }
	got := core.FindUnlinkedMentions(notes, core.Page{ID: "Diseño.md", Title: "Diseño"}, read, skip)
	want := []core.UnlinkedMention{
		{ID: "a.md", Title: "a", Line: 1, Text: "Diseño here", Count: 1, Content: "Diseño here"},
		{ID: "b.md", Title: "b", Line: 2, Text: "hablando de diseño, y Diseño otra vez", Count: 2, Content: content["b.md"]},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FindUnlinkedMentions =\n%+v\nwant\n%+v", got, want)
	}
}
