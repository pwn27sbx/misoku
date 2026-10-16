package main

import "flag"

// positionals parses args with fs and returns the arguments that are not
// flags, so flags work before, between and after them (the standard flag
// package stops at the first one). A flag error, flag.ErrHelp included,
// is returned as fs.Parse returns it.
func positionals(fs *flag.FlagSet, args []string) ([]string, error) {
	var rest []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return rest, nil
		}
		rest = append(rest, fs.Arg(0))
		args = fs.Args()[1:]
	}
}
