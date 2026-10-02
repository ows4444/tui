package widgets_test

import (
	"fmt"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// Widgets are stateless: each is a function from its inputs and a theme to a
// styled string, so it can be called from any View. The examples strip the
// styling so the output is plain text.
func ExampleBox() {
	fmt.Println(ansi.StripANSI(widgets.Box("Status", "all good", theme.DarkTheme(), 20)))
	// Output:
	// ┌──────────────────┐
	// │                  │
	// │ Status           │
	// │ all good         │
	// │                  │
	// └──────────────────┘
}

func ExampleBreadcrumb() {
	fmt.Println(ansi.StripANSI(widgets.Breadcrumb([]string{"home", "docs", "api"}, theme.DarkTheme())))
	// Output: home / docs / api
}
