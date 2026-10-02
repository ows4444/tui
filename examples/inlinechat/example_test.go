package main

import (
	"fmt"
	"strings"
)

// Example builds the inlinechat example's model and renders its first frame.
func Example() {
	m := model{}
	fmt.Println(strings.TrimSpace(m.View()) != "")
	// Output: true
}
