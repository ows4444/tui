package textinput

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/input"
)

// Shift+a under KeyboardReportAllKeys with associated text arrives as
// ESC[97;2;65u; the decoded key must insert 'A', not 'a'.
func TestShiftLetterFromKittyAssociatedTextInsertsCapital(t *testing.T) {
	rd := input.NewReader(strings.NewReader("\x1b[97;2;65u\x1b[98;1;98u"))
	m := focused()
	for i := 0; i < 2; i++ {
		ev, err := rd.ReadEvent()
		if err != nil {
			t.Fatal(err)
		}
		m, _ = m.Update(ev)
	}
	if got := m.Value(); got != "Ab" {
		t.Fatalf("Value = %q, want %q", got, "Ab")
	}
}
