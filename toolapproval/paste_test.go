package toolapproval

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/input"
)

// #18: pasted text, newline included, never approves or rejects. It goes
// through the real decoder so the test covers what the terminal delivers with
// bracketed paste on (the default): one PasteEvent, not an Enter key.
func TestPasteWithNewlineDoesNotResolveTheCall(t *testing.T) {
	for _, paste := range []string{"y\n", "\n", "yes\r\nyes\n", "\x1b[200~ignored"} {
		ev, err := input.NewReader(strings.NewReader("\x1b[200~" + paste + "\x1b[201~")).ReadEvent()
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := ev.(tui.PasteEvent); !ok {
			t.Fatalf("%q decoded to %#v, want a PasteEvent", paste, ev)
		}
		m := New("rm", "delete a file", RiskHigh)
		m.highlighted = ChoiceApprove
		got, cmd := m.Update(ev)
		if cmd != nil {
			t.Errorf("paste %q returned a Cmd, so it resolved or scheduled something", paste)
		}
		if got.highlighted != ChoiceApprove {
			t.Errorf("paste %q moved the highlight to %v", paste, got.highlighted)
		}
	}
}
