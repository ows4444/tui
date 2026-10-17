package keymap_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/keymap"
)

// A Registry collects bindings so a help screen can list them and so two
// actions bound to the same key in one scope are reported.
func Example() {
	var r keymap.Registry
	r.Add(keymap.NewBinding("quit", "q"))
	conflicts, _ := r.Add(keymap.NewBinding("close", "q"))
	fmt.Println(len(conflicts))
	for _, h := range r.Hints("") {
		fmt.Println(h.Key, h.Desc)
	}
	// Output:
	// 1
	// q quit
	// q close
}

// A user's key file rebinds actions by name; Apply says which names matched
// nothing.
func ExampleApply() {
	ov, _ := keymap.Parse(strings.NewReader(`{"quit": ["x"], "hlep": ["h"]}`))
	out, unknown := keymap.Apply([]keymap.Binding{keymap.NewBinding("quit", "q"), keymap.NewBinding("help", "?")}, ov)
	fmt.Println(out[0].Keys, out[1].Keys, unknown)
	// Output: [x] [?] [hlep]
}
