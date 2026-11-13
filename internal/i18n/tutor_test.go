package i18n

import (
	"strings"
	"testing"
)

// L16: the English B00 quotes the tutor's bar texts word for word, and the
// Spanish B00 quotes the ones the bar always had.
func TestTutorBarTextsMatchTheLessons(t *testing.T) {
	for _, c := range []struct {
		lang Lang
		got  string
		want string
	}{
		{EN, TFor(EN, TutorStepLine, "B03", 2, 4, "…"), "B03 · Step 2/4 · …"},
		{EN, TFor(EN, TutorStepFlash, 2), "✓ Step 2"},
		{EN, TFor(EN, TutorLessonPassed, "B03", "Lists"), "✓ B03 passed · next: Lists"},
		{EN, TFor(EN, TutorRouteDone, "Basic"), "🏆 Basic route complete"},
		{ES, TFor(ES, TutorStepLine, "B03", 2, 4, "…"), "B03 · Paso 2/4 · …"},
		{ES, TFor(ES, TutorStepFlash, 2), "✓ Paso 2"},
		{ES, TFor(ES, TutorLessonPassed, "B03", "Listas"), "✓ B03 superado · siguiente: Listas"},
		{ES, TFor(ES, TutorRouteDone, "Básico"), "🏆 Ruta Básico completa"},
	} {
		if c.got != c.want {
			t.Errorf("%s: %q, want %q", c.lang, c.got, c.want)
		}
	}
}

// The first-run dialog counts the Basic route's lessons in each language.
func TestWelcomeRouteCountsLessons(t *testing.T) {
	if got := NFor(EN, WelcomeNewRoute, 26, "Basic", 26); got != "Basic route · 26 lessons" {
		t.Errorf("en = %q", got)
	}
	if got := NFor(ES, WelcomeNewRoute, 1, "Básico", 1); got != "Ruta Básico · 1 lección" {
		t.Errorf("es = %q", got)
	}
	for _, l := range Langs {
		if strings.Contains(TFor(l, WelcomeNew), "welcome.") {
			t.Errorf("%s has no first-run dialog", l)
		}
	}
}
