package main

import "fmt"

// Example sorts by name and prints the first process.
func Example() {
	m := initialModel().sortBy(colName)
	fmt.Println(m.shown()[0].name)
	// Output: bluetoothd
}
