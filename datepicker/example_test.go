package datepicker_test

import (
	"fmt"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/datepicker"
)

// Arrow keys move the cursor by a day or a week; Enter selects the date.
func Example() {
	m := datepicker.New(time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC))
	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	fmt.Println(m.Cursor().Format("2006-01-02"))
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	fmt.Println(cmd().(datepicker.SelectedMsg).Date.Format("Mon 2 Jan"))
	// Output:
	// 2026-03-18
	// Wed 18 Mar
}
