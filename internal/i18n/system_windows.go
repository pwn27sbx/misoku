package i18n

import "golang.org/x/sys/windows"

// appleLanguage is macOS only.
var appleLanguage func() string

// windowsLanguage is the first of the user's preferred UI languages:
// Windows usually sets no LANG.
func windowsLanguage() string {
	langs, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME)
	if err != nil || len(langs) == 0 {
		return ""
	}
	return langs[0]
}
