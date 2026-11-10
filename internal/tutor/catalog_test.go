package tutor

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"
)

// validCatalog is a minimal catalog every validation case starts from.
func validCatalog() Catalog {
	return Catalog{Routes: []Route{{
		ID: "B", Name: "Básico", Index: "B00 Básico.md",
		Lessons: []Lesson{{
			ID: "B01", Title: "Títulos", File: "B01 Títulos.md",
			Steps: []Step{{Text: "Haz algo", Check: Check{Kind: KindNone}}},
		}},
	}}}
}

func TestBundledRoutes(t *testing.T) {
	cat, err := BundledRoutes()
	if err != nil {
		t.Fatalf("the embedded catalog is invalid: %v", err)
	}
	if len(cat.Routes) == 0 {
		t.Fatal("no route is embedded")
	}
	var b *Route
	for i := range cat.Routes {
		if cat.Routes[i].ID == "B" {
			b = &cat.Routes[i]
		}
	}
	if b == nil {
		t.Fatal("route B is not embedded")
	}
	if b.Index != "B00 Básico.md" {
		t.Errorf("index = %q", b.Index)
	}
	if len(b.Lessons) != 26 {
		t.Fatalf("route B has %d lessons, want 26", len(b.Lessons))
	}
	for i, l := range b.Lessons {
		wantID := fmt.Sprintf("B%02d", i+1)
		if l.ID != wantID {
			t.Errorf("lesson %d = %q, want %q", i, l.ID, wantID)
		}
		if len(l.Steps) == 0 {
			t.Errorf("%s has no steps", l.ID)
		}
	}
	want := map[string]int{"B": 26, "I": 30, "A": 21, "E": 20}
	for _, r := range cat.Routes {
		if n, ok := want[r.ID]; !ok || n != len(r.Lessons) {
			t.Errorf("route %s has %d lessons, want %d", r.ID, len(r.Lessons), want[r.ID])
		}
		delete(want, r.ID)
	}
	if len(want) != 0 {
		t.Errorf("missing routes: %v", want)
	}
}

// S7/D8: every embedded note is an index, a lesson or the final reference
// note, and every index and lesson is embedded.
func TestEmbeddedFilesCovered(t *testing.T) {
	cat, err := BundledRoutes()
	if err != nil {
		t.Fatal(err)
	}
	described := map[string]bool{ReferenceNote: true}
	for _, r := range cat.Routes {
		described[r.Index] = true
		for _, l := range r.Lessons {
			described[l.File] = true
		}
	}
	entries, err := fs.ReadDir(Bundled(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if !described[e.Name()] {
			t.Errorf("%s is embedded but not in the catalog", e.Name())
		}
		delete(described, e.Name())
	}
	for name := range described {
		if _, err := fs.Stat(Bundled(), name); err != nil {
			t.Errorf("%s is in the catalog but not embedded", name)
		}
	}
}

func TestCatalogValidation(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Catalog)
	}{
		{"unknown route", func(c *Catalog) { c.Routes[0].ID = "X" }},
		{"index prefix", func(c *Catalog) { c.Routes[0].Index = "C00 Básico.md" }},
		{"duplicate lesson", func(c *Catalog) {
			c.Routes[0].Lessons = append(c.Routes[0].Lessons, c.Routes[0].Lessons[0])
		}},
		{"file with a slash", func(c *Catalog) { c.Routes[0].Lessons[0].File = "B01/a.md" }},
		{"file with a colon", func(c *Catalog) { c.Routes[0].Lessons[0].File = "B01: a.md" }},
		{"no steps", func(c *Catalog) { c.Routes[0].Lessons[0].Steps = nil }},
		{"empty text", func(c *Catalog) { c.Routes[0].Lessons[0].Steps[0].Text = "  " }},
		{"long text", func(c *Catalog) { c.Routes[0].Lessons[0].Steps[0].Text = strings.Repeat("á", 61) }},
		{"unknown kind", func(c *Catalog) { c.Routes[0].Lessons[0].Steps[0].Check.Kind = "bogus" }},
		{"unknown pred", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindText, Pred: "bogus", Args: []string{"x"}}
		}},
		{"unknown event", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindAction, Event: "bogus"}
		}},
		{"unknown fence", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindTextFence, Fence: "bogus", Pattern: "x"}
		}},
		{"pred arity", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindText, Pred: "line-contains"}
		}},
		{"n-lines arity", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindText, Pred: "n-lines-start", Args: []string{"2"}}
		}},
		{"heading level", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindText, Pred: "heading-exists", Args: []string{"0", "x"}}
		}},
		{"table cells", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindText, Pred: "table-shape", Args: []string{"two"}}
		}},
		{"unknown disk pred", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindDisk, Pred: "bogus", Args: []string{"x"}}
		}},
		{"config-key arity", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindDisk, Pred: "config-key", Args: []string{"a", "b", "c"}}
		}},
		{"empty all", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindAll}
		}},
		{"all with none", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindAll, Checks: []Check{{Kind: KindNone}}}
		}},
		{"duplicate action value", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindAction, Event: "next_tab", Value: Values{"a", "a"}}
		}},
		{"heading-under arity", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindText, Pred: "heading-under", Args: []string{"2", "Viaje", "3"}}
		}},
		{"heading-under level order", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindText, Pred: "heading-under", Args: []string{"3", "Viaje", "2", "Día 1"}}
		}},
		{"task-status state", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindText, Pred: "task-status", Args: []string{"bogus", "comprar pan"}}
		}},
		{"task-status text", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindText, Pred: "task-status", Args: []string{"done", " "}}
		}},
		{"action with a pred", func(c *Catalog) {
			c.Routes[0].Lessons[0].Steps[0].Check = Check{Kind: KindAction, Event: "next_tab", Pred: "line-contains"}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cat := validCatalog()
			tc.edit(&cat)
			if err := cat.validate(); err == nil {
				t.Fatal("validate accepted an invalid catalog")
			}
		})
	}
}

func TestValidCatalogPasses(t *testing.T) {
	if err := validCatalog().validate(); err != nil {
		t.Fatalf("the minimal catalog should be valid: %v", err)
	}
}

// L4/E13: a text check ignores every line the embedded lesson already has.
func TestWithoutOriginal(t *testing.T) {
	data, err := fs.ReadFile(Bundled(), "B13 Títulos.md")
	if err != nil {
		t.Fatal(err)
	}
	original := string(data)
	ownHeading := Check{Kind: KindText, Pred: "heading-exists", Args: []string{"2", "Tu turno"}}
	if !ownHeading.Pass(original, Env{}) {
		t.Fatal("fixture: the original has its own ## Tu turno")
	}
	if ownHeading.Pass(withoutOriginal("B13 Títulos.md", original), Env{}) {
		t.Fatal("a lesson passed on its own text")
	}
	wanted := Check{Kind: KindText, Pred: "heading-exists", Args: []string{"2", "Viaje"}}
	if !wanted.Pass(withoutOriginal("B13 Títulos.md", original+"\n## Viaje\n"), Env{}) {
		t.Fatal("the player's own heading should pass")
	}
}

// A fixture lesson that is not embedded has no original to subtract.
func TestWithoutOriginalFixture(t *testing.T) {
	const text = "## Viaje\n"
	if got := withoutOriginal("B99 Fixture.md", text); got != text {
		t.Fatalf("withoutOriginal changed a fixture: %q", got)
	}
}

// L4/E13, one case per check family: original lines are never the evidence,
// while a heading, block, table or fence the lesson already has may still be
// the target or the structure.
func TestBaselinePerCheck(t *testing.T) {
	cases := []struct {
		name     string
		check    Check
		full     string
		filtered string
		want     bool
	}{
		{
			name:     "a new line passes line-contains",
			check:    Check{Kind: KindText, Pred: "line-contains", Args: []string{"hoy"}},
			full:     "hoy es el día\n",
			filtered: "hoy es el día\n",
			want:     true,
		},
		{
			name:     "the original line does not",
			check:    Check{Kind: KindText, Pred: "line-contains", Args: []string{"hoy"}},
			full:     "hoy es el día\n",
			filtered: "\n",
			want:     false,
		},
		{
			name:     "absent sees the original line that must go",
			check:    Check{Kind: KindText, Pred: "absent", Args: []string{"borrador"}},
			full:     "borrador\n",
			filtered: "\n",
			want:     false,
		},
		{
			name:     "absent after deleting it",
			check:    Check{Kind: KindText, Pred: "absent", Args: []string{"borrador"}},
			full:     "limpio\n",
			filtered: "limpio\n",
			want:     true,
		},
		{
			name:     "an original heading is not a new heading",
			check:    Check{Kind: KindText, Pred: "heading-exists", Args: []string{"2", "Viaje"}},
			full:     "## Viaje\n",
			filtered: "\n",
			want:     false,
		},
		{
			name:     "a new link may target the lesson's own heading",
			check:    Check{Kind: KindText, Pred: "headinglink-exists", Args: []string{"Equipaje"}},
			full:     "## Equipaje\n\n[[#Equipaje]]\n",
			filtered: "\n\n[[#Equipaje]]\n",
			want:     true,
		},
		{
			name:     "the lesson's own link text is not evidence",
			check:    Check{Kind: KindText, Pred: "headinglink-exists", Args: []string{"Equipaje"}},
			full:     "## Equipaje\n\n[[#Equipaje]]\n",
			filtered: "## Equipaje\n\n\n",
			want:     false,
		},
		{
			name:     "an original block can be linked by a new line",
			check:    Check{Kind: KindText, Pred: "blockref-exists", Args: []string{"reserva"}},
			full:     "La mesa ^reserva\n\n[[#^reserva]]\n",
			filtered: "\n\n[[#^reserva]]\n",
			want:     true,
		},
		{
			name:     "an untouched table is not evidence",
			check:    Check{Kind: KindText, Pred: "table-shape", Args: []string{"2"}},
			full:     "| a | b |\n| --- | --- |\n| 1 | 2 |\n",
			filtered: "\n\n\n",
			want:     false,
		},
		{
			name:     "an edited table row is",
			check:    Check{Kind: KindText, Pred: "table-shape", Args: []string{"2"}},
			full:     "| a | b |\n| --- | --- |\n| 1 | 3 |\n",
			filtered: "\n\n| 1 | 3 |\n",
			want:     true,
		},
		{
			name:     "an untouched board fence is not evidence",
			check:    Check{Kind: KindTextFence, Fence: "board", Pattern: "- "},
			full:     "```board\n## Columna\n- tarjeta\n```\n",
			filtered: "\n\n\n\n",
			want:     false,
		},
		{
			name:     "a new child under the lesson's own heading",
			check:    Check{Kind: KindText, Pred: "heading-under", Args: []string{"2", "Viaje", "3", "Día 1"}},
			full:     "## Viaje\n\n### Día 1\n",
			filtered: "\n\n### Día 1\n",
			want:     true,
		},
		{
			name:     "the lesson's own child is not evidence",
			check:    Check{Kind: KindText, Pred: "heading-under", Args: []string{"2", "Viaje", "3", "Día 1"}},
			full:     "## Viaje\n\n### Día 1\n",
			filtered: "## Viaje\n\n\n",
			want:     false,
		},
		{
			name:     "the lesson's own task is not evidence",
			check:    Check{Kind: KindText, Pred: "task-status", Args: []string{"done", "comprar pan"}},
			full:     "- [x] comprar pan\n",
			filtered: "\n",
			want:     false,
		},
		{
			name:     "a new card in the lesson's board is",
			check:    Check{Kind: KindTextFence, Fence: "board", Pattern: "- nueva"},
			full:     "```board\n## Columna\n- nueva\n```\n",
			filtered: "\n\n- nueva\n\n",
			want:     true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := Env{full: c.full}
			if got := c.check.Pass(c.filtered, env); got != c.want {
				t.Fatalf("Pass = %v, want %v", got, c.want)
			}
		})
	}
}

func TestMigrationNoticeTexts(t *testing.T) {
	if got := migrationNoticeFor(true); !strings.Contains(got, "papelera") {
		t.Errorf("Trash notice = %q", got)
	}
	if got := migrationNoticeFor(false); !strings.Contains(got, "borraron") {
		t.Errorf("delete notice = %q", got)
	}
}
