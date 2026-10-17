package main

import "fmt"

// Example types 2+3*4 and presses Enter.
func Example() {
	m := initialModel()
	for _, s := range []string{"2", "+", "3", "*", "4"} {
		m = typeKey(m, s)
	}
	m, _ = m.press("=")
	fmt.Println(m.expr)
	// Output: 14
}
