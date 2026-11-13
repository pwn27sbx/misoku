package i18n

import (
	"os"
	"strings"
	"sync/atomic"
)

// Resolve decides the interface language. A saved choice ("en", "es")
// wins. Otherwise the system decides, the first of these that is set:
//
//  1. LANGUAGE (its first entry), LC_ALL, LC_MESSAGES: set on purpose;
//  2. apple: the first of macOS's AppleLanguages;
//  3. windows: the first of Windows' preferred UI languages;
//  4. LANG.
//
// "C" and "POSIX" count as unset. A value starting with "es" is Spanish,
// "en" English, and any other language English too. apple and windows are
// only called when the steps before them did not decide; either may be
// nil.
func Resolve(saved string, getenv func(string) string, apple, windows func() string) Lang {
	if l, ok := Parse(saved); ok {
		return l
	}
	first, _, _ := strings.Cut(getenv("LANGUAGE"), ":")
	for _, v := range []string{first, getenv("LC_ALL"), getenv("LC_MESSAGES")} {
		if l, ok := fromLocale(v); ok {
			return l
		}
	}
	for _, read := range []func() string{apple, windows} {
		if read == nil {
			continue
		}
		if l, ok := fromLocale(read()); ok {
			return l
		}
	}
	if l, ok := fromLocale(getenv("LANG")); ok {
		return l
	}
	return EN
}

// fromLocale reads a locale or language tag (es_PE.UTF-8, es-419, en):
// false when it is unset, C or POSIX.
func fromLocale(v string) (Lang, bool) {
	v = strings.TrimSpace(v)
	base, _, _ := strings.Cut(v, ".")
	base, _, _ = strings.Cut(base, "@")
	if base == "" || base == "C" || base == "POSIX" {
		return "", false
	}
	if strings.HasPrefix(strings.ToLower(base), "es") {
		return ES, true
	}
	return EN, true // English, and any language Misoku does not speak
}

// Detect resolves the language for this process at startup: saved, or
// else the system language, which DetectSystem reads and keeps.
func Detect(saved string) Lang {
	if l, ok := Parse(saved); ok {
		return l
	}
	return DetectSystem()
}

// readSystem reads the system language: the process's environment and the
// system's settings. On macOS that runs `defaults`, up to a second. Tests
// replace it.
var readSystem = func() Lang { return Resolve("", os.Getenv, appleLanguage, windowsLanguage) }

// system is the system language DetectSystem read; "" until then.
var system atomic.Value

// DetectSystem reads the system language and keeps it for System. It may
// run an external program: call it at startup, never from Update.
func DetectSystem() Lang {
	l := readSystem()
	SetSystem(l)
	return l
}

// SetSystem keeps l as the system language; "" forgets it.
func SetSystem(l Lang) { system.Store(l) }

// System is the language Auto means, safe to call from Update: the one
// DetectSystem read, or, when it never ran (a saved choice at startup
// reads no system setting), the one the environment alone says. It never
// reads the system's settings itself.
func System() Lang {
	if l, _ := system.Load().(Lang); l != "" {
		return l
	}
	return Resolve("", os.Getenv, nil, nil)
}

// parseAppleLanguages takes the first language of `defaults read -g
// AppleLanguages`, a property list array such as ( "es-PE", en ).
func parseAppleLanguages(out string) string {
	for _, line := range strings.Split(out, "\n") {
		v := strings.Trim(strings.TrimSpace(line), `(),"`)
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

// Startup decides the language when Misoku starts. With Auto it reads the
// system language first. With a saved choice it returns that choice at
// once and reads the system language in the background, so the system
// read never delays startup and Auto chosen later in :settings (System)
// is the system's.
func Startup(saved string) Lang {
	if l, ok := Parse(saved); ok {
		go DetectSystem()
		return l
	}
	return DetectSystem()
}
