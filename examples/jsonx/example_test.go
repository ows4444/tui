package main

import (
	"fmt"
	"strings"
)

// Example finds the path of a nested value.
func Example() {
	root, _ := parse(strings.NewReader(`{"orders":[{"customer":{"name":"Ada"}}]}`))
	m := newModel(root, "x.json")
	m = m.search("ada").firstHitFrom(root)
	fmt.Println(m.selected().path())
	// Output: $.orders[0].customer.name
}
