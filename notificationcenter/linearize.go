package notificationcenter

import (
	"github.com/ows4444/tui/ansi"
	"strconv"
	"strings"

	"github.com/ows4444/tui/widgets"
)

// Linearize renders the queue as plain text for accessible output (see
// tui.Linearizer): "Notifications, N" and one line per notification, oldest
// first, "<kind> K of N: <message>" with the Variant in words, or "" when the
// queue is empty.
func (m Model) Linearize() string {
	if len(m.Notifications) == 0 {
		return ""
	}
	kinds := map[widgets.Variant]string{
		widgets.VariantNeutral: "notice", widgets.VariantInfo: "info",
		widgets.VariantSuccess: "success", widgets.VariantWarning: "warning", widgets.VariantError: "error",
	}
	n := strconv.Itoa(len(m.Notifications))
	lines := []string{"Notifications, " + n}
	for i, nt := range m.Notifications {
		text := ansi.Clean(m.Raw, nt.Message)
		if len(nt.Text) > 0 {
			text = nt.Text.Plain()
		}
		lines = append(lines, kinds[nt.Variant]+" "+strconv.Itoa(i+1)+" of "+n+": "+text)
	}
	return strings.Join(lines, "\n")
}
