package main

import "fmt"

// Example ticks the first task by clicking its row and prints how many are left.
func Example() {
	m := click(initialModel(), "row-1")
	_, open := m.counts()
	fmt.Println(open)
	// Output: 4
}
