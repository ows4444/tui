package toolapproval_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/toolapproval"
)

// A prompt asking whether a tool call may run. Approve is highlighted at first;
// Enter resolves it.
func Example() {
	m := toolapproval.New("write_file", "Write main.go", toolapproval.RiskLow)
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	fmt.Println(cmd().(toolapproval.ResolvedMsg).Choice == toolapproval.ChoiceApprove)
	// Output:
	// true
}
