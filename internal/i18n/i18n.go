// Package i18n holds the interface's words in every language Misoku speaks
// and the language in use.
//
// Each area of the app keeps its keys and both translations in one file of
// this package (see settings.go) and registers them in init. A key missing
// in a language reads in English; a test checks that never happens.
package i18n

import (
	"fmt"
	"sync/atomic"
)

// Lang is a language of the interface.
type Lang string

// The languages Misoku speaks.
const (
	EN Lang = "en"
	ES Lang = "es"
)

// Langs lists the languages, in the order the settings menu shows them.
var Langs = []Lang{EN, ES}

// Key names one sentence of the interface.
type Key string

// The suffixes of a plural's two forms: a catalog holds key+one and
// key+other, and N picks one.
const (
	one   Key = ".one"
	other Key = ".other"
)

// catalogs holds every language's sentences. It is filled in init and only
// read afterwards.
var catalogs = map[Lang]map[Key]string{EN: {}, ES: {}}

// register adds entries to l's catalog. A key registered twice is a bug.
func register(l Lang, entries map[Key]string) {
	c := catalogs[l]
	for key, text := range entries {
		if _, dup := c[key]; dup {
			panic(fmt.Sprintf("i18n: %s key %q registered twice", l, key))
		}
		c[key] = text
	}
}

// current is the language in use. Background commands read it while
// Update may switch it, hence the atomic.
var current atomic.Value

// Set makes l the language in use; a language Misoku does not speak is
// English.
func Set(l Lang) {
	if _, ok := catalogs[l]; !ok {
		l = EN
	}
	current.Store(l)
}

// Current is the language in use, English until Set.
func Current() Lang {
	if l, ok := current.Load().(Lang); ok {
		return l
	}
	return EN
}

// T is key in the language in use, formatted with args.
func T(key Key, args ...any) string { return TFor(Current(), key, args...) }

// TFor is key in l, formatted with args. A key l lacks reads in English;
// a key no language has reads as itself.
func TFor(l Lang, key Key, args ...any) string {
	text, ok := catalogs[l][key]
	if !ok {
		if text, ok = catalogs[EN][key]; !ok {
			text = string(key)
		}
	}
	if len(args) == 0 {
		return text
	}
	return fmt.Sprintf(text, args...)
}

// N is the form of the plural key that fits n, in the language in use,
// formatted with args (which usually start with n).
func N(key Key, n int, args ...any) string { return NFor(Current(), key, n, args...) }

// NFor is N in l.
func NFor(l Lang, key Key, n int, args ...any) string {
	form := other
	if n == 1 { // English and Spanish: one is singular, 0 and the rest plural
		form = one
	}
	return TFor(l, key+form, args...)
}

// Parse reads a saved language choice: "en" or "es". Anything else, the
// empty Auto included, is no choice.
func Parse(saved string) (Lang, bool) {
	switch l := Lang(saved); l {
	case EN, ES:
		return l, true
	}
	return "", false
}

// Name is a language's own name for itself, the same in every language.
func Name(l Lang) string {
	switch l {
	case ES:
		return "Español"
	}
	return "English"
}
