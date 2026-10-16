package main

import "fmt"

// Example lists the files two ticked features would create.
func Example() {
	got := files(answers{Name: "demo", Language: "Go", Features: []string{"README", ".gitignore"}})
	fmt.Println(len(got), got["README.md"] != "", got[".gitignore"] != "")
	// Output: 2 true true
}
