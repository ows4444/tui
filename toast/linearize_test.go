package toast

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/widgets"
)

var _ tui.Linearizer = Model{}

func TestLinearizeStatesTheKindInWords(t *testing.T) {
	m := New("Saved")
	if got := m.Linearize(); got != "" {
		t.Errorf("closed = %q", got)
	}
	m.open = true
	for v, kind := range map[widgets.Variant]string{
		widgets.VariantNeutral: "notice", widgets.VariantInfo: "info", widgets.VariantSuccess: "success",
		widgets.VariantWarning: "warning", widgets.VariantError: "error",
	} {
		m.Variant = v
		if got := m.Linearize(); got != "Notification, "+kind+": Saved" {
			t.Errorf("variant %d = %q", v, got)
		}
	}
}
