package toolapproval

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New("delete_build", "Delete the build directory", RiskHigh)
	m.Timeout = 30 * time.Second
	want := "Approval needed: delete_build, high risk\n" +
		"Delete the build directory\n" +
		"Approve, option 1 of 3, selected\n" +
		"Deny, option 2 of 3\n" +
		"Always Allow, option 3 of 3\n" +
		"Denies automatically after 30s if unanswered"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	next, _ := m.Update(tui.Key{Type: tui.KeyRight})
	if got := next.Linearize(); !strings.Contains(got, "Deny, option 2 of 3, selected") || strings.Contains(got, "Approve, option 1 of 3, selected") {
		t.Errorf("highlight did not move:\n%s", got)
	}
	plain := New("ls", "", RiskLow)
	if got := plain.Linearize(); strings.Contains(got, "Denies automatically") || !strings.HasPrefix(got, "Approval needed: ls, low risk\n") {
		t.Errorf("plain = %q", got)
	}
}
