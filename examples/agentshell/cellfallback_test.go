package main

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/internal/cellcheck"
	"github.com/ows4444/tui/toolapproval"
)

// portStates are the screens this example can show, by name.
func portStates() map[string]tui.Model {
	idle := testModel()

	streaming := testModel()
	streaming.busy, streaming.script = true, []string{"hello "}
	streaming.stream.Append("hello **world**")

	pending := testModel()
	pending.busy, pending.pending = true, true
	pending.tool = &toolCall{name: "run_tests", description: "go test ./...", risk: toolapproval.RiskMedium}
	pending.approve = pending.newApproval()
	pending.prompt.SetValue("line one\nline two")

	return map[string]tui.Model{"idle": idle, "streaming": streaming, "pending": pending}
}

// When every screen of this example is drawn with the cell renderer, the frame
// log shall record zero fallbacks (spec S11).
func TestCellRendererNoFallbacks(t *testing.T) {
	for name, m := range portStates() {
		if fb := cellcheck.Fallbacks(m, 200, 100); len(fb) > 0 {
			t.Errorf("%s: %v", name, fb)
		}
	}
}
