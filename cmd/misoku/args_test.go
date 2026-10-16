package main

import (
	"errors"
	"flag"
	"io"
	"slices"
	"testing"
)

// positionals collects the arguments that are not flags, with the flags
// allowed before, between and after them.
func TestPositionalsAllowFlagsAnywhere(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		args []string
		want []string
		o    string
	}{
		{nil, nil, ""},
		{[]string{"-o", "x"}, nil, "x"},
		{[]string{"a"}, []string{"a"}, ""},
		{[]string{"-o", "x", "a"}, []string{"a"}, "x"},
		{[]string{"a", "-o", "x"}, []string{"a"}, "x"},
		{[]string{"a", "-o", "x", "b"}, []string{"a", "b"}, "x"},
	} {
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		o := fs.String("o", "", "")
		got, err := positionals(fs, tc.args)
		if err != nil || !slices.Equal(got, tc.want) || *o != tc.o {
			t.Errorf("%q: %q, -o %q, %v; want %q, -o %q", tc.args, got, *o, err, tc.want, tc.o)
		}
	}
}

// A flag error, -h included, is returned as it is.
func TestPositionalsReturnFlagErrors(t *testing.T) {
	t.Parallel()
	for args, want := range map[string]error{"-h": flag.ErrHelp, "-nope": nil} {
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		_, err := positionals(fs, []string{"a", args})
		if err == nil || (want != nil && !errors.Is(err, want)) {
			t.Errorf("%s: error %v, want %v", args, err, want)
		}
	}
}
