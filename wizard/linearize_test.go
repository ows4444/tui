package wizard

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New("Account", "Profile", "Confirm")
	if got := m.Linearize(); got != "Account, step 1 of 3, current\nProfile, step 2 of 3, upcoming\nConfirm, step 3 of 3, upcoming" {
		t.Errorf("start = %q", got)
	}
	if err := m.Next(nil); err != nil {
		t.Fatal(err)
	}
	if got := m.Linearize(); got != "Account, step 1 of 3, completed\nProfile, step 2 of 3, current\nConfirm, step 3 of 3, upcoming" {
		t.Errorf("middle = %q", got)
	}
	if got := New().Linearize(); got != "No steps" {
		t.Errorf("empty = %q", got)
	}
}
