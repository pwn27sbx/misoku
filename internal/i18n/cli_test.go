package i18n

import (
	"errors"
	"fmt"
	"testing"
)

// An Err reads in the language in use when it is read, not when it was
// made, and stays itself for errors.Is however it is wrapped.
func TestErrTranslatesWhenRead(t *testing.T) {
	useTestKeys(t)
	useLang(t, EN)
	err := Err(testOnly)
	other := Err(testOnly)
	wrapped := fmt.Errorf("x: %w", err)
	if !errors.Is(wrapped, err) || errors.Is(wrapped, other) {
		t.Fatal("an Err must match itself only")
	}
	if got := Err(testHello).Error(); got != "Hello, %s" {
		t.Errorf("EN: %q", got)
	}
	Set(ES)
	if got := Err(testHello).Error(); got != "Hola, %s" {
		t.Errorf("ES: %q", got)
	}
}
