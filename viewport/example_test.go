package viewport_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/viewport"
)

// A viewport shows a window onto longer content and scrolls it.
func Example() {
	m := viewport.New(10, 3)
	m.SetContent(strings.Join([]string{"one", "two", "three", "four", "five"}, "\n"))
	fmt.Println(m.View())
	m.LineDown(2)
	fmt.Println("--")
	fmt.Println(m.View())
	// Output:
	// one
	// two
	// three
	// --
	// three
	// four
	// five
}
