package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/i18n"
)

// runMainLang is runMain in the language lang: LANGUAGE decides before
// the machine's own language (i18n.Resolve).
func runMainLang(t *testing.T, lang string, args ...string) (int, string) {
	t.Helper()
	home := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), mainArgsEnv+"="+strings.Join(args, "\n"), "MISOKU_WORKSPACE=", "LANGUAGE="+lang,
		"HOME="+home, "USERPROFILE="+home, "XDG_CACHE_HOME="+home, "XDG_CONFIG_HOME="+home, "LOCALAPPDATA="+home, "APPDATA="+home)
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return code, string(out)
}

// S1: misoku --help, its flags and a subcommand's usage speak the
// interface language.
func TestHelpSpeaksTheLanguage(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		lang string
		args []string
		want []string
	}{
		{"en", []string{"--help"}, []string{"Usage:\n", "open your notes", "Flags:\n", "print the Misoku version and exit"}},
		{"es", []string{"--help"}, []string{"Uso:\n", "abre tus notas", "Opciones:\n", "muestra la versión de Misoku y termina"}},
		{"en", []string{"sync", "nonsense"}, []string{"Usage: misoku sync init <url> [--dir <folder>]"}},
		{"es", []string{"sync", "nonsense"}, []string{"Uso: misoku sync init <url> [--dir <carpeta>]"}},
		{"es", []string{"tutor", "--route", "Z"}, []string{"misoku tutor: ruta desconocida \"Z\""}},
	} {
		_, out := runMainLang(t, c.lang, c.args...)
		for _, w := range c.want {
			if !strings.Contains(out, w) {
				t.Errorf("LANGUAGE=%s misoku %v: no %q in:\n%s", c.lang, c.args, w, out)
			}
		}
	}
}

// useLang sets the interface language for one test, which must not run in
// parallel: the language is the whole process's.
func useLang(t *testing.T, l i18n.Lang) {
	t.Helper()
	old := i18n.Current()
	i18n.Set(l)
	t.Cleanup(func() { i18n.Set(old) })
}

// S1: the text welcome speaks Spanish in Spanish.
func TestWelcomeInSpanish(t *testing.T) {
	useLang(t, i18n.ES)
	g := newGreeting(t)
	g.missing = "/gone/Notes"
	_, _, out := g.run(t, "9\n")
	for _, w := range []string{"Te damos la bienvenida a Misoku.", "No encuentro tus notas en /gone/Notes.",
		"¿Dónde están tus notas?", "  1) Empezar mis notas", "  2) Traer mis notas de otro equipo",
		"Número (Enter cancela): ", "! Escribe 1 o 2.", "No se cambió nada. Vuelve a ejecutar misoku cuando quieras."} {
		if !strings.Contains(out, w) {
			t.Errorf("no %q in:\n%s", w, out)
		}
	}
	if got := missingNotesMessage("/n"); got != "Falta tu carpeta de notas /n; abre otra con misoku <carpeta>" {
		t.Errorf("missingNotesMessage = %q", got)
	}
}
