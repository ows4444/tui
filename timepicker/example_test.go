package timepicker_test

import (
	"fmt"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/timepicker"
)

// Type the hours and the minutes, then confirm with Enter.
func Example() {
	t := timepicker.New(time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC))
	t.Focus()
	for _, k := range []string{"1", "4", "3", "0"} {
		t, _ = t.Update(tui.Key{Type: tui.KeyRunes, Text: k})
	}
	fmt.Println(ansi.StripANSI(t.View()))
	_, cmd := t.Update(tui.Key{Type: tui.KeyEnter})
	fmt.Println(cmd().(timepicker.SelectedMsg).Value.Format("15:04 on 2006-01-02"))
	// Output:
	// 14:30
	// 14:30 on 2026-10-17
}
