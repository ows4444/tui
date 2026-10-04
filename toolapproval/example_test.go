package toolapproval_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/keymap"
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

// A prompt with two options, labels of the app's own, the command as its body,
// and y and n answering at once.
func Example_choices() {
	m := toolapproval.New("Bash", "wants to run", toolapproval.RiskLow)
	m.Choices = []toolapproval.Choice{toolapproval.ChoiceApprove, toolapproval.ChoiceDeny}
	m.Labels = map[toolapproval.Choice]string{toolapproval.ChoiceApprove: "Allow"}
	m.Body = "$ go test ./..."
	m.KeyMap.Approve = keymap.NewBinding("allow", "y")
	m.KeyMap.Deny = keymap.NewBinding("deny", "n")

	fmt.Println(m.Linearize())
	_, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "n"})
	fmt.Println(cmd().(toolapproval.ResolvedMsg).Choice == toolapproval.ChoiceDeny)
	// Output:
	// Approval needed: Bash, low risk
	// wants to run
	// $ go test ./...
	// Allow, option 1 of 2, selected
	// Deny, option 2 of 2
	// true
}
