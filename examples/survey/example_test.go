package main

import (
	"fmt"
	"os"
)

// Example prints the summary of one set of answers.
func Example() {
	a := answers{Name: "Ada", Email: "ada@example.com", Years: 12, Role: "backend", Tools: []string{"Linters"}, Score: 9}
	_ = summary(os.Stdout, a, false)
	fmt.Println()
	// Output:
	// Thank you, Ada. Here is what you sent:
	//   Email     ada@example.com
	//   Years     12
	//   Role      backend
	//   Tools     Linters
	//   Score     9 / 10
}
