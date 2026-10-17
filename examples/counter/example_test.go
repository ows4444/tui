package main

import "fmt"

// Example clicks + three times and prints the count.
func Example() {
	m := initialModel()
	for i := 0; i < 3; i++ {
		m = click(m, "inc")
	}
	fmt.Println(m.count)
	// Output: 3
}
