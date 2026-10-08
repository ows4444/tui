package carousel_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/carousel"
)

// Right shows the next slide and fills its dot.
func Example() {
	tips := carousel.New("Press ? for help", "Press / to search", "Press q to quit")
	tips.Focus()
	tips, cmd := tips.Update(tui.Key{Type: tui.KeyRight})
	// Each row is padded to the carousel's width, trimmed here for the
	// comparison.
	for _, row := range strings.Split(ansi.StripANSI(tips.View()), "\n") {
		fmt.Println(strings.TrimSpace(row))
	}
	fmt.Println(cmd().(carousel.ChangedMsg).Index)
	// Output:
	// Press / to search
	// <○ ● ○>
	// 1
}
