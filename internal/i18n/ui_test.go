package i18n

import "testing"

// The tables of ui.go are folded into maps in init, where a repeated id
// would silently drop a text; every text is written in both languages.
func TestUITablesHaveUniqueIDsAndEveryText(t *testing.T) {
	seen := map[Key]bool{}
	check := func(key Key, texts ...string) {
		t.Helper()
		if seen[key] {
			t.Errorf("%q appears twice", key)
		}
		seen[key] = true
		for _, s := range texts {
			if s == "" {
				t.Errorf("%q has an empty text", key)
			}
		}
	}
	for _, x := range keymapTexts {
		check(KeymapHelp(x.action), x.enHelp, x.enLong, x.esHelp, x.esLong)
	}
	for _, x := range commandTexts {
		check(CommandHelp(x.id), x.enHelp, x.enLong, x.esHelp, x.esLong)
	}
	for _, x := range slashTexts {
		check(x.key, x.en, x.es)
	}
}

// The month and weekday lists the preview splits on commas have one name
// per month and per day in every language.
func TestDateNamesAreComplete(t *testing.T) {
	for _, l := range Langs {
		for key, want := range map[Key]int{PreviewMonths: 12, PreviewMonthsShort: 12, PreviewWeekdays: 7} {
			n := 1
			for _, r := range TFor(l, key) {
				if r == ',' {
					n++
				}
			}
			if n != want {
				t.Errorf("%s %q: %d names, want %d", l, key, n, want)
			}
		}
	}
}
