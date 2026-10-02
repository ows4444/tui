package appshell_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/appshell"
)

// An appshell frames content with a title, a scrolling area and an input line.
func Example() {
	m := appshell.New("my app", 40, 5)
	fmt.Println(strings.Contains(ansi.StripANSI(m.View()), "my app"))
	// Output:
	// true
}
