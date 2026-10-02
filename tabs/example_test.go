package tabs_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/tabs"
)

// Tabs track which one is active; the right arrow moves to the next.
func Example() {
	m := tabs.New("files", "search", "settings")
	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	fmt.Println(m.Active())
	// Output: 1
}
