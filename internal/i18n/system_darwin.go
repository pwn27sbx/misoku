package i18n

import (
	"context"
	"os/exec"
	"time"
)

// appleLanguage is the first of the user's preferred languages in macOS,
// which the terminal's LANG often does not follow. It gives up after a
// second.
func appleLanguage() string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "defaults", "read", "-g", "AppleLanguages").Output()
	if err != nil {
		return ""
	}
	return parseAppleLanguages(string(out))
}

// windowsLanguage is Windows only.
var windowsLanguage func() string
