package notificationcenter_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/notificationcenter"
	"github.com/ows4444/tui/widgets"
)

// A notification center queues messages and draws them in one panel over the
// base screen, at most maxVisible at a time.
func Example() {
	m := notificationcenter.New(2, 24)
	m.Push(notificationcenter.Notification{Message: "build passed", Variant: widgets.VariantSuccess})
	m.Push(notificationcenter.Notification{Message: "disk almost full", Variant: widgets.VariantWarning})
	base := strings.TrimRight(strings.Repeat(strings.Repeat(" ", 40)+"\n", 8), "\n")
	out := ansi.StripANSI(m.Render(base))
	fmt.Println(strings.Contains(out, "build passed"), strings.Contains(out, "disk almost full"))
	// Output:
	// true true
}
