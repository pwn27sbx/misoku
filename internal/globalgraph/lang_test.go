package globalgraph

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/i18n"
)

// S1: the vault map's title and hints speak Spanish in Spanish.
func TestTitleAndHintsInSpanish(t *testing.T) {
	old := i18n.Current()
	i18n.Set(i18n.ES)
	t.Cleanup(func() { i18n.Set(old) })
	v := ansi.Strip(newModel(t, 200, 20).View())
	for _, w := range []string{"11 notas · 12 enlaces · 1 huérfana · 1 sin resolver", " / filtrar ", "hjkl mover"} {
		if !strings.Contains(v, w) {
			t.Errorf("no %q in:\n%s", w, v)
		}
	}
}
