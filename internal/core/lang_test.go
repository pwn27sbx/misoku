package core_test

import (
	"errors"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/i18n"
)

// S1, A4: the errors core shows in the bar speak the interface language;
// ErrPageNotFound still matches with errors.Is.
func TestErrorsInEachLanguage(t *testing.T) {
	old := i18n.Current()
	t.Cleanup(func() { i18n.Set(old) })
	for _, c := range []struct {
		lang            i18n.Lang
		empty, notFound string
	}{
		{i18n.EN, "file name is empty", "page not found"},
		{i18n.ES, "el nombre del archivo está vacío", "no se encontró la página"},
	} {
		i18n.Set(c.lang)
		if _, err := core.NormalizeNotePath("", " "); err == nil || err.Error() != c.empty {
			t.Errorf("%s: empty name: %v", c.lang, err)
		}
		s, err := core.NewStore(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		_, err = s.Get("missing.md")
		if !errors.Is(err, core.ErrPageNotFound) || err.Error() != c.notFound {
			t.Errorf("%s: Get missing: %v", c.lang, err)
		}
	}
}
