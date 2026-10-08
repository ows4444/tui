package inputgroup_test

import (
	"fmt"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/inputgroup"
)

// A field between a fixed prefix and suffix.
func Example() {
	site := inputgroup.New("https://", ".com")
	site.SetValue("example")
	fmt.Println(ansi.StripANSI(site.View()))
	fmt.Println(site.FullValue())
	// Output:
	// https://example.com
	// https://example.com
}
