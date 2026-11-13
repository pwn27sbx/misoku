package graph

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/i18n"
)

// S1: the local graph's title and hints speak Spanish in Spanish.
func TestTitleAndHintsInSpanish(t *testing.T) {
	old := i18n.Current()
	i18n.Set(i18n.ES)
	t.Cleanup(func() { i18n.Set(old) })
	v := drawn(newSample(t, 120, 20))
	for _, w := range []string{"Hub · 2 enlaces · 2 retroenlaces · 1 paso", " 0 centrar ", " esc volver ", "hjkl mover · enter abrir"} {
		if !strings.Contains(v, w) {
			t.Errorf("no %q in:\n%s", w, v)
		}
	}
}
