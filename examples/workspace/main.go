// Command workspace is an editor-style window split into three panes with tabs:
// an explorer on the left, an editor at the top right and a panel at the
// bottom right. Drag the dividers to resize the panes, double-click one to
// collapse the pane next to it (a thin strip stays to bring it back).
//
// Mouse: click a pane or a tab, drag a divider, double-click a divider to
// collapse or restore, scroll the wheel over a pane. Keyboard: Tab and
// Shift+Tab move between panes, . and , change tab, arrows and PgUp PgDn scroll,
// [ ] move the vertical divider, - = move the horizontal one, c collapses the
// focused pane, z zooms it, 0 resets the layout, ? shows help, q or Ctrl+C
// quits. Small windows hide the panes that no longer fit.
//
//	go run ./examples/workspace
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
)

func main() {
	if _, err := tui.NewProgram(newModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
