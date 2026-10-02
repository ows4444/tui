package keymap_test

import (
	"fmt"

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
