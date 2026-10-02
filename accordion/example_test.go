package accordion_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/accordion"
	"github.com/ows4444/tui/ansi"
)

// Sections start collapsed; Toggle opens one and shows its content.
func Example() {
	m := accordion.New(accordion.Section{Title: "Details", Content: "the body"})
	fmt.Println(strings.Contains(ansi.StripANSI(m.View()), "the body"))
	m.Toggle(0)
	fmt.Println(strings.Contains(ansi.StripANSI(m.View()), "the body"))
	// Output:
	// false
	// true
}
