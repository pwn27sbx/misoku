package app_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// templateNote is the template of templateStore, with front matter,
// placeholders and a tag only it carries.
const templateNote = "---\nstatus: onhold\ncreated: {{fecha}}\ntags: [plantilla]\n---\n# {{titulo}}\nDía {{date}} #solo-plantilla\n"

// templateStore is a vault with one template in Plantillas and a note,
// Notas/plan, with its own front matter.
func templateStore(t *testing.T) *core.Store {
	t.Helper()
	store := newTestStore(t)
	for _, n := range [][3]string{
		{"Plantillas", "Diario", templateNote},
		{"Notas", "plan", "---\nstatus: active\n---\nintro #real\n"},
	} {
		if _, err := store.SaveAs(n[0], n[1], n[2]); err != nil {
			t.Fatal(err)
		}
	}
	return store
}

// openPlan opens Notas/plan.md, dated dailyDay, with the cursor on its
// empty last line, in NORMAL mode.
func openPlan(t *testing.T, store *core.Store, opts ...app.Option) *harness {
	t.Helper()
	h := dailyHarness(t, store, opts...)
	h.keys("f")
	h.keys("plan")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.keys("G")
	return h
}

// S6: "/plantilla" lists the templates; the one picked joins its front
// matter to the note's (one block, the note's keys win), its placeholders
// take today and the title, its body lands at the cursor, and one u takes
// all of it back.
func TestSlashTemplateInsertsThePickedTemplate(t *testing.T) {
	store := templateStore(t)
	h := openPlan(t, store)
	h.keys("i/plantilla")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "Templates") || !strings.Contains(v, "Diario") {
		t.Fatalf("the template picker should list Diario:\n%s", v)
	}
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	p, _ := store.Get("Notas/plan.md")
	want := "---\nstatus: active\ncreated: 2026-11-06\ntags: [plantilla]\n---\nintro #real\n# plan\nDía 2026-11-06 #solo-plantilla\n"
	if p.Content != want {
		t.Fatalf("after the template the note is\n%q\nwant\n%q", p.Content, want)
	}
	h.keys("u")
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if p, _ := store.Get("Notas/plan.md"); p.Content != "---\nstatus: active\n---\nintro #real\n" {
		t.Fatalf("one u should undo the whole template, the note is %q", p.Content)
	}
}

// S6: templates in subfolders are listed by their path in the folder.
func TestTemplatePickerListsSubfoldersByRelativePath(t *testing.T) {
	store := templateStore(t)
	if _, err := store.SaveAs("Plantillas/Trabajo", "Reunión", "Agenda"); err != nil {
		t.Fatal(err)
	}
	h := openPlan(t, store)
	h.keys("i/plantilla")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "Trabajo/Reunión") {
		t.Fatalf("the picker should list Trabajo/Reunión:\n%s", v)
	}
}

// S6: without templates the bar names the folder to create them in.
func TestSlashTemplateWithoutTemplatesNamesTheFolder(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("Notas", "plan", "intro\n"); err != nil {
		t.Fatal(err)
	}
	h := openPlan(t, store)
	h.keys("i/plantilla")
	h.key(tea.KeyEnter)
	if bar := statusBar(h); !strings.Contains(bar, "Plantillas") {
		t.Fatalf("the bar should name the templates folder:\n%s", bar)
	}
}

// S6: templates_dir picks the folder; one outside the vault is refused with
// a message and the default folder is used.
func TestTemplatesDirOptionAndRefusal(t *testing.T) {
	store := templateStore(t)
	if _, err := store.SaveAs("Modelos", "Acta", "Acta {{title}}"); err != nil {
		t.Fatal(err)
	}
	h := openPlan(t, store, app.WithTemplatesDir("Modelos"))
	h.keys("i/plantilla")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "Acta") || strings.Contains(v, "Diario") {
		t.Fatalf("templates_dir Modelos should list Acta only:\n%s", v)
	}

	h = dailyHarness(t, store, app.WithTemplatesDir("../fuera"))
	if bar := statusBar(h); !strings.Contains(bar, "templates_dir") {
		t.Fatalf("the bar should name the refused setting:\n%s", bar)
	}
}

// S4/S6: a tag only a template carries is not listed by ":tags".
func TestTagsLeaveTheTemplatesOut(t *testing.T) {
	h := openPlan(t, templateStore(t))
	h.keys(":tags")
	h.key(tea.KeyEnter)
	v := h.view()
	if !strings.Contains(v, "#real · 1 note") || strings.Contains(v, "solo-plantilla") || strings.Contains(v, "#plantilla") {
		t.Fatalf(":tags should list #real and nothing of the template:\n%s", v)
	}
}

// S4/S6: tag: and status: in the finder leave the templates out, while a
// plain search still finds them.
func TestFinderQualifiersLeaveTheTemplatesOut(t *testing.T) {
	store := templateStore(t)
	for _, q := range []string{"tag:solo-plantilla", "status:onhold", "#plantilla"} {
		h := newHarness(t, store)
		h.keys("f")
		h.keys(q)
		if v := h.view(); !strings.Contains(v, "0 results") {
			t.Fatalf("%s should leave the template out:\n%s", q, v)
		}
	}
	h := newHarness(t, store)
	h.keys("f")
	h.keys("Día")
	if v := h.view(); !strings.Contains(v, "Plantillas/Diario.md") {
		t.Fatalf("a plain search should still find the template:\n%s", v)
	}
}
