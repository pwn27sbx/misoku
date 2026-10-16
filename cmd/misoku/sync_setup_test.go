package main

import "testing"

// The help lists the assistant as the recommended way to set up sync,
// and init for any other git URL.
func TestHelpListsSyncSetup(t *testing.T) {
	t.Parallel()
	code, out := runMain(t, "--help")
	if code != 0 {
		t.Fatalf("misoku --help: exit %d:\n%s", code, out)
	}
	wantOutput(t, out, "misoku sync setup [--dir <folder>]", "(recommended)", "misoku sync init <url> [--dir <folder>]")
	code, out = runMain(t, "sync", "--help")
	if code != 0 {
		t.Fatalf("misoku sync --help: exit %d:\n%s", code, out)
	}
	wantOutput(t, out, "misoku sync setup")
}
