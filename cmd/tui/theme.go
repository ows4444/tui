package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/ows4444/tui/theme"
)

// runTheme runs `tui theme check [-min ratio] <file.json>` and `tui theme list`.
func runTheme(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	switch args[0] {
	case "list":
		if len(args) != 1 {
			return errors.New(usage)
		}
		for _, n := range theme.PresetNames() {
			fmt.Fprintln(out, n)
		}
		return nil
	case "check":
		return checkTheme(args[1:], out)
	}
	return errors.New(usage)
}

// checkTheme parses a theme file and lists every text or accent role below the
// contrast ratio; it fails if there is one, or if the file does not parse.
func checkTheme(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("theme check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	min := fs.Float64("min", 4.5, "smallest contrast ratio a role may have (WCAG AA is 4.5)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%v\n%s", err, usage)
	}
	if fs.NArg() != 1 {
		return errors.New(usage)
	}
	fh, err := os.Open(fs.Arg(0)) // #nosec G304 -- the file the user named
	if err != nil {
		return err
	}
	defer fh.Close()
	t, err := theme.Parse(fh)
	if err != nil {
		return err
	}
	issues := t.Check(*min)
	if len(issues) == 0 {
		fmt.Fprintf(out, "%s: every text and accent role reaches %.1f:1\n", fs.Arg(0), *min)
		return nil
	}
	for _, is := range issues {
		fmt.Fprintf(out, "%s: %s (want %.1f)\n", fs.Arg(0), is, *min)
	}
	return fmt.Errorf("%d role(s) below %.1f:1", len(issues), *min)
}
