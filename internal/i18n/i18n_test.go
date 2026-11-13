package i18n

import (
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// env is a fake environment for Resolve.
func env(vars map[string]string) func(string) string {
	return func(name string) string { return vars[name] }
}

// fixed returns a reader of a system setting that answers v and counts its
// calls.
func fixed(v string, calls *int) func() string {
	return func() string {
		*calls++
		return v
	}
}

// S2-S4: the saved choice first, then the system language in its order.
func TestResolve(t *testing.T) {
	for _, c := range []struct {
		name           string
		saved          string
		vars           map[string]string
		apple, windows string
		want           Lang
	}{
		{"nothing set is English", "", nil, "", "", EN},
		{"saved en wins over the system", "en", map[string]string{"LANGUAGE": "es"}, "es-PE", "es-PE", EN},
		{"saved es wins over the system", "es", map[string]string{"LANG": "en_US.UTF-8"}, "en-US", "", ES},
		{"an unknown saved value counts as Auto", "fr", map[string]string{"LANG": "es_PE.UTF-8"}, "", "", ES},
		{"LANG es_PE", "", map[string]string{"LANG": "es_PE"}, "", "", ES},
		{"LANG es-419", "", map[string]string{"LANG": "es-419"}, "", "", ES},
		{"LANG es_ES.UTF-8", "", map[string]string{"LANG": "es_ES.UTF-8"}, "", "", ES},
		{"LANG en_GB", "", map[string]string{"LANG": "en_GB.UTF-8"}, "", "", EN},
		{"LANGUAGE es:en takes its first entry", "", map[string]string{"LANGUAGE": "es:en", "LANG": "en_US.UTF-8"}, "", "", ES},
		{"LANGUAGE en:es takes its first entry", "", map[string]string{"LANGUAGE": "en:es", "LANG": "es_PE.UTF-8"}, "es-PE", "", EN},
		{"LANGUAGE before LC_ALL", "", map[string]string{"LANGUAGE": "es", "LC_ALL": "en_US.UTF-8"}, "", "", ES},
		{"LC_ALL before LC_MESSAGES", "", map[string]string{"LC_ALL": "es_MX.UTF-8", "LC_MESSAGES": "en_US.UTF-8"}, "", "", ES},
		{"LC_MESSAGES before AppleLanguages", "", map[string]string{"LC_MESSAGES": "en_US.UTF-8"}, "es-PE", "", EN},
		// Measured on the owner's Mac: the terminal sets LANG, the system
		// language is in AppleLanguages.
		{"AppleLanguages before LANG", "", map[string]string{"LANG": "en_US.UTF-8"}, "es-PE", "", ES},
		{"Windows before LANG", "", map[string]string{"LANG": "en_US.UTF-8"}, "", "es-PE", ES},
		{"C counts as unset", "", map[string]string{"LC_ALL": "C", "LANG": "es_PE.UTF-8"}, "", "", ES},
		{"C.UTF-8 counts as unset", "", map[string]string{"LC_ALL": "C.UTF-8", "LANG": "es_PE.UTF-8"}, "", "", ES},
		{"POSIX counts as unset", "", map[string]string{"LC_ALL": "POSIX", "LANG": "es_PE.UTF-8"}, "", "", ES},
		{"another language is English (S3)", "", map[string]string{"LANG": "fr_FR.UTF-8"}, "", "", EN},
		{"another language decides, later ones do not count", "", map[string]string{"LC_ALL": "de_DE.UTF-8", "LANG": "es_PE.UTF-8"}, "es-PE", "", EN},
		{"unknown everywhere is English", "", map[string]string{"LANG": "ja_JP.UTF-8"}, "ja-JP", "", EN},
	} {
		t.Run(c.name, func(t *testing.T) {
			var calls int
			got := Resolve(c.saved, env(c.vars), fixed(c.apple, &calls), fixed(c.windows, &calls))
			if got != c.want {
				t.Fatalf("Resolve(%q, %v, %q, %q) = %q, want %q", c.saved, c.vars, c.apple, c.windows, got, c.want)
			}
		})
	}
}

// S4.2: AppleLanguages runs only when the saved choice is Auto, and only
// when no variable set on purpose decided.
func TestResolveReadsTheSystemOnlyWhenNeeded(t *testing.T) {
	var calls int
	Resolve("es", env(map[string]string{"LANG": "en_US"}), fixed("en", &calls), fixed("en", &calls))
	if calls != 0 {
		t.Fatalf("a saved choice read the system language %d times", calls)
	}
	Resolve("", env(map[string]string{"LANGUAGE": "es"}), fixed("en", &calls), fixed("en", &calls))
	if calls != 0 {
		t.Fatalf("LANGUAGE decided, yet the system language was read %d times", calls)
	}
	if got := Resolve("", env(nil), nil, nil); got != EN {
		t.Fatalf("no readers: got %q, want en", got)
	}
}

func TestParseAppleLanguages(t *testing.T) {
	for _, c := range []struct{ out, want string }{
		{"(\n    \"es-PE\"\n)\n", "es-PE"},
		{"(\n    \"en-US\",\n    \"es-PE\"\n)\n", "en-US"},
		{"(\n    en,\n    es\n)\n", "en"},
		{"(\n)\n", ""},
		{"", ""},
	} {
		if got := parseAppleLanguages(c.out); got != c.want {
			t.Errorf("parseAppleLanguages(%q) = %q, want %q", c.out, got, c.want)
		}
	}
}

func TestParse(t *testing.T) {
	for _, c := range []struct {
		in   string
		want Lang
		ok   bool
	}{{"en", EN, true}, {"es", ES, true}, {"", "", false}, {"fr", "", false}, {"ES", "", false}} {
		if got, ok := Parse(c.in); got != c.want || ok != c.ok {
			t.Errorf("Parse(%q) = %q, %v; want %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

// testKeys registers its own entries, so these tests do not depend on what
// the app's catalog holds.
const (
	testHello Key = "test.hello"
	testNotes Key = "test.notes"
	testOnly  Key = "test.only_en"
)

func useTestKeys(t *testing.T) {
	t.Helper()
	en := map[Key]string{testHello: "Hello, %s", testNotes + one: "%d note", testNotes + other: "%d notes", testOnly: "only English"}
	es := map[Key]string{testHello: "Hola, %s", testNotes + one: "%d nota", testNotes + other: "%d notas"}
	register(EN, en)
	register(ES, es)
	t.Cleanup(func() {
		// Leave the catalog as the app has it: the parity test must not see
		// the deliberately missing entry.
		for key := range en {
			delete(catalogs[EN], key)
		}
		for key := range es {
			delete(catalogs[ES], key)
		}
	})
}

func useLang(t *testing.T, l Lang) {
	t.Helper()
	old := Current()
	Set(l)
	t.Cleanup(func() { Set(old) })
}

func TestTranslate(t *testing.T) {
	useTestKeys(t)
	useLang(t, ES)
	if got := T(testHello, "Ana"); got != "Hola, Ana" {
		t.Errorf("T in Spanish = %q", got)
	}
	if got := TFor(EN, testHello, "Ana"); got != "Hello, Ana" {
		t.Errorf("TFor(EN) = %q", got)
	}
	if got := N(testNotes, 1, 1); got != "1 nota" {
		t.Errorf("N(1) = %q", got)
	}
	if got := N(testNotes, 3, 3); got != "3 notas" {
		t.Errorf("N(3) = %q", got)
	}
	if got := NFor(EN, testNotes, 0, 0); got != "0 notes" {
		t.Errorf("NFor(EN, 0) = %q", got)
	}
	if got := T(testOnly); got != "only English" {
		t.Errorf("a key missing in Spanish should fall back to English, got %q", got)
	}
	if got := T("no.such.key"); got != "no.such.key" {
		t.Errorf("an unknown key should read as itself, got %q", got)
	}
	Set(EN)
	if got := T(testHello, "Ana"); got != "Hello, Ana" {
		t.Errorf("after Set(EN): %q", got)
	}
}

func TestCurrentDefaultsToEnglishAndIgnoresUnknown(t *testing.T) {
	useLang(t, ES)
	Set("fr")
	if got := Current(); got != EN {
		t.Fatalf("Set of an unknown language: Current = %q, want en", got)
	}
}

// Current is read by background commands while Update may switch it.
func TestCurrentIsSafeAcrossGoroutines(t *testing.T) {
	useLang(t, EN)
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				if i%2 == 0 {
					Set(ES)
				} else {
					_ = T(SettingsTitle)
				}
			}
		}()
	}
	wg.Wait()
}

func TestNames(t *testing.T) {
	if Name(EN) != "English" || Name(ES) != "Español" {
		t.Fatalf("Name: %q, %q", Name(EN), Name(ES))
	}
}

var verb = regexp.MustCompile(`%[-+# 0]*(\[\d+\])?(\d+|\*)?(\.(\d+|\*))?[a-zA-Z%]`)

// Every key exists in both languages, with the same fmt verbs in the same
// order, and every plural has both forms.
func TestCatalogParity(t *testing.T) {
	for _, l := range Langs {
		if len(catalogs[l]) == 0 {
			t.Fatalf("no catalog for %q", l)
		}
	}
	for _, a := range Langs {
		for _, b := range Langs {
			if a == b {
				continue
			}
			for key, text := range catalogs[a] {
				other, ok := catalogs[b][key]
				if !ok {
					t.Errorf("%q has no %s entry", key, b)
					continue
				}
				if va, vb := verb.FindAllString(text, -1), verb.FindAllString(other, -1); !slices.Equal(va, vb) {
					t.Errorf("%q: %s verbs %v, %s verbs %v", key, a, va, b, vb)
				}
			}
		}
		for key := range catalogs[a] {
			if base, ok := strings.CutSuffix(string(key), string(one)); ok {
				if _, has := catalogs[a][Key(base)+other]; !has {
					t.Errorf("%s: %q has no %q form", a, base, other)
				}
			}
			if base, ok := strings.CutSuffix(string(key), string(other)); ok {
				if _, has := catalogs[a][Key(base)+one]; !has {
					t.Errorf("%s: %q has no %q form", a, base, one)
				}
			}
		}
	}
}

// The parity check itself catches a missing entry and a verb mismatch.
func TestCatalogParityCatchesDifferences(t *testing.T) {
	a := map[Key]string{"x": "%d of %s"}
	b := map[Key]string{"x": "%s de %d"}
	if slices.Equal(verb.FindAllString(a["x"], -1), verb.FindAllString(b["x"], -1)) {
		t.Fatal("reordered verbs compared equal")
	}
	if got := verb.FindAllString("%[1]d %-8s %.2f %%", -1); !slices.Equal(got, []string{"%[1]d", "%-8s", "%.2f", "%%"}) {
		t.Fatalf("verbs: %v", got)
	}
}

// useSystemReader replaces how the system language is read, counting the
// reads, and forgets the cached system language before and after.
func useSystemReader(t *testing.T, l Lang) *int {
	t.Helper()
	calls := new(int)
	old := readSystem
	readSystem = func() Lang {
		*calls++
		return l
	}
	SetSystem("")
	t.Cleanup(func() {
		readSystem = old
		SetSystem("")
	})
	return calls
}

// The system language is read once, by DetectSystem at startup; System
// only returns what was read and never reads it itself, so the settings
// menu can call it from Update.
func TestSystemIsReadOnceAndCached(t *testing.T) {
	calls := useSystemReader(t, ES)
	if got := DetectSystem(); got != ES || *calls != 1 {
		t.Fatalf("DetectSystem = %q after %d reads, want es after 1", got, *calls)
	}
	for range 3 {
		if got := System(); got != ES {
			t.Fatalf("System = %q, want the cached es", got)
		}
	}
	if *calls != 1 {
		t.Fatalf("System read the system language again: %d reads", *calls)
	}
}

// Before DetectSystem ran (a saved choice at startup reads no system
// setting), System falls back to the environment alone, without reading
// the system.
func TestSystemBeforeDetectionUsesTheEnvironmentOnly(t *testing.T) {
	calls := useSystemReader(t, EN)
	for _, name := range []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES", "LANG"} {
		t.Setenv(name, "")
	}
	t.Setenv("LANGUAGE", "es")
	if got := System(); got != ES {
		t.Fatalf("System before detection = %q, want es from LANGUAGE", got)
	}
	if *calls != 0 {
		t.Fatalf("System read the system language %d times", *calls)
	}
}

// Detect with a saved choice reads no system setting and caches nothing;
// with Auto it reads it once and caches it.
func TestDetectCachesTheSystemOnlyWhenAuto(t *testing.T) {
	calls := useSystemReader(t, ES)
	if got := Detect("en"); got != EN || *calls != 0 {
		t.Fatalf("Detect(en) = %q after %d reads", got, *calls)
	}
	if got := Detect(""); got != ES || *calls != 1 {
		t.Fatalf("Detect(auto) = %q after %d reads", got, *calls)
	}
	if got := System(); got != ES || *calls != 1 {
		t.Fatalf("System after Detect(auto) = %q after %d reads", got, *calls)
	}
}

// Startup with a saved choice returns it at once and reads the system
// language in the background, so Auto chosen later is the system's; with
// Auto it reads the system language before returning.
func TestStartupReadsTheSystemInTheBackgroundWhenSaved(t *testing.T) {
	calls := useSystemReader(t, ES)
	release := make(chan struct{})
	done := make(chan struct{})
	readSystem = func() Lang {
		<-release
		*calls++
		close(done)
		return ES
	}
	got := make(chan Lang, 1)
	go func() { got <- Startup("en") }()
	select {
	case l := <-got:
		if l != EN {
			t.Fatalf("Startup(en) = %q", l)
		}
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("Startup(en) waited for the system language")
	}
	close(release)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Startup(en) never read the system language")
	}
	deadline := time.Now().Add(5 * time.Second)
	for System() != ES {
		if time.Now().After(deadline) {
			t.Fatal("the system language read in the background was not kept")
		}
		time.Sleep(time.Millisecond)
	}
	if *calls != 1 {
		t.Fatalf("%d reads, want 1", *calls)
	}
}

func TestStartupReadsTheSystemFirstWhenAuto(t *testing.T) {
	calls := useSystemReader(t, ES)
	if got := Startup(""); got != ES || *calls != 1 || System() != ES {
		t.Fatalf("Startup(auto) = %q after %d reads, System %q", got, *calls, System())
	}
}
