package tutor

import (
	"os"
	"testing"

	"github.com/pwn27sbx/misoku/internal/i18n"
)

// TestMain runs the package's tests with the Spanish tutorial in use: most
// of them name its lessons, which existed first. Tests about another
// language ask for it with useLang or the *For functions; none of them runs
// in parallel, so switching the language never races.
func TestMain(m *testing.M) {
	i18n.Set(i18n.ES)
	os.Exit(m.Run())
}

// useLang plays the tutorial in l for one test and puts Spanish back after it.
func useLang(t *testing.T, l i18n.Lang) {
	t.Helper()
	i18n.Set(l)
	t.Cleanup(func() { i18n.Set(i18n.ES) })
}
