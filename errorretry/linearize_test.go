package errorretry

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New("connection lost", 3)
	want := "Error: connection lost\nRetries used: 0 of 3\nPress Enter or r to retry, Escape to dismiss"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	m.retryCount = 3
	want = "Error: connection lost\nRetries used: 3 of 3\nNo retries left. Press Escape to dismiss"
	if got := m.Linearize(); got != want {
		t.Errorf("exhausted =\n%s\nwant\n%s", got, want)
	}
}
