package widgets_test

import (
	"errors"
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

// A footer lists the focused widget's bindings on one line and drops the last
// hints, with a "…", when the line is too narrow.
func ExampleFooter() {
	hints := []widgets.Hint{{Key: "enter", Action: "submit"}, {Key: "esc", Action: "cancel"}, {Key: "?", Action: "help"}}
	fmt.Println(ansi.StripANSI(widgets.Footer(80, hints...)))
	fmt.Println(ansi.StripANSI(widgets.Footer(24, hints...)))
	// Output:
	// [enter] submit  [esc] cancel  [?] help
	// [enter] submit  …
}

// Tree prints nested entries with connector lines, for output that is not
// navigated.
func ExampleTree() {
	fmt.Println(widgets.Tree([]widgets.TreeNode{
		{Label: "cmd", Children: []widgets.TreeNode{{Label: "tui"}}},
		{Label: "go.mod"},
	}))
	// Output:
	// ├── cmd
	// │   └── tui
	// └── go.mod
}

// ErrorTrace shows what each layer of a wrapped error added, and its type.
func ExampleErrorTrace() {
	err := fmt.Errorf("save: %w", errors.New("disk full"))
	fmt.Println(ansi.StripANSI(widgets.ErrorTrace(err)))
	// Output:
	// ✗ save (*fmt.wrapError)
	// └── disk full (*errors.errorString)
}
