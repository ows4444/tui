package clipboard

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New("secret-token-123", "Copy token")
	if got := m.Linearize(); got != "Copy token, button" {
		t.Errorf("idle = %q", got)
	}
	m.copied = true
	if got := m.Linearize(); got != "Copied to clipboard" {
		t.Errorf("copied = %q", got)
	}
	if got := m.Linearize(); contains(got, "secret-token") {
		t.Errorf("must not speak the copied text: %q", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
