package theme

import (
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestResolvedTypographyDefaultsAndOverride(t *testing.T) {
	ty := DarkTheme().ResolvedTypography()
	if ty.H1 != ansi.NewStyle().Foreground(DarkTheme().Primary).Bold().Underline() {
		t.Error("H1 default wrong")
	}
	if ty.Link != ansi.NewStyle().Foreground(DarkTheme().Info).Underline() {
		t.Error("Link default wrong")
	}
	custom := ansi.NewStyle().Italic()
	th := DarkTheme()
	th.Typography.H1 = custom
	if th.ResolvedTypography().H1 != custom {
		t.Error("explicit H1 not kept")
	}
}
