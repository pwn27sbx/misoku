//go:build !darwin && !windows

package i18n

// Elsewhere the environment alone says the language.
var appleLanguage, windowsLanguage func() string
