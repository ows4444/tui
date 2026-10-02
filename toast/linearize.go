package toast

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/widgets"
)

// Linearize renders the toast as plain text for accessible output (see
// tui.Linearizer): "Notification, <kind>: <message>", or "" while it is
// closed. The kind is the Variant in words, so meaning does not rely on
// colour.
func (m Model) Linearize() string {
	if !m.open {
		return ""
	}
	kind := map[widgets.Variant]string{
		widgets.VariantNeutral: "notice", widgets.VariantInfo: "info",
		widgets.VariantSuccess: "success", widgets.VariantWarning: "warning", widgets.VariantError: "error",
	}[m.Variant]
	return "Notification, " + kind + ": " + ansi.Clean(m.Raw, m.Message)
}
