package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/ows4444/tui/keymap"
)

// runKeys runs `tui keys check <file.json>`.
func runKeys(args []string, out io.Writer) error {
	if len(args) != 2 || args[0] != "check" {
		return errors.New(usage)
	}
	fh, err := os.Open(args[1]) // #nosec G304 -- the file the user named
	if err != nil {
		return err
	}
	defer fh.Close()
	ov, err := keymap.Parse(fh)
	if err != nil {
		return fmt.Errorf("%s: %w", args[1], err)
	}
	// A key given to two actions is allowed (they may be in different
	// scopes), so it is reported and does not fail the check.
	byKey := map[string][]string{}
	actions := make([]string, 0, len(ov))
	for a, keys := range ov {
		actions = append(actions, a)
		for _, k := range keys {
			byKey[k] = append(byKey[k], a)
		}
	}
	sort.Strings(actions)
	for _, a := range actions {
		if len(ov[a]) == 0 {
			fmt.Fprintf(out, "%s: %s: unbound\n", args[1], a)
			continue
		}
		fmt.Fprintf(out, "%s: %s: %s\n", args[1], a, strings.Join(ov[a], ", "))
	}
	shared := make([]string, 0)
	for k, as := range byKey {
		if len(as) > 1 {
			sort.Strings(as)
			shared = append(shared, fmt.Sprintf("%s: %s is given to %s; fine only if they are in different scopes", args[1], k, strings.Join(as, " and ")))
		}
	}
	sort.Strings(shared)
	for _, s := range shared {
		fmt.Fprintln(out, s)
	}
	fmt.Fprintf(out, "%s: %d action(s), every key name is valid\n", args[1], len(actions))
	return nil
}
