package markdown_test

import (
	"fmt"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/markdown"
	"github.com/ows4444/tui/theme"
)

func ExampleRender() {
	out := markdown.Render("# Title\n\nSome *emphasis* and `code`.\n\n- one\n- two", 40, theme.DarkTheme())
	fmt.Println(ansi.StripANSI(out))
	// Output:
	// Title
	//
	// Some emphasis and code.
	//
	// • one
	// • two
}
