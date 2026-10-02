package widgets

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func TestHeader(t *testing.T) {
	got := Header("Dashboard", theme.DarkTheme())
	want := ansi.NewStyle().Bold().Foreground(theme.DarkTheme().Primary).Render("Dashboard")
	if got != want {
		t.Errorf("Header() = %q, want %q", got, want)
	}
}

func TestHeaderWithAccessoryNoAccessory(t *testing.T) {
	got := HeaderWithAccessory("Dashboard", "", 40, theme.DarkTheme())
	want := Header("Dashboard", theme.DarkTheme())
	if got != want {
		t.Errorf("HeaderWithAccessory() with empty accessory = %q, want %q", got, want)
	}
}

func TestHeaderWithAccessoryFits(t *testing.T) {
	got := HeaderWithAccessory("Dashboard", "v1.2.3", 20, theme.DarkTheme())
	titleStyled := Header("Dashboard", theme.DarkTheme())
	accessoryStyled := ansi.NewStyle().Foreground(theme.DarkTheme().Muted).Render("v1.2.3")
	// "Dashboard" (9) + gap + "v1.2.3" (6) = 20 -> gap of 5.
	want := titleStyled + "     " + accessoryStyled
	if got != want {
		t.Errorf("HeaderWithAccessory() = %q, want %q", got, want)
	}
}

// TestHeaderWithAccessoryIsALayoutRow proves HeaderWithAccessory's fitting
// path is a real layout.Row with a growing gap, not just visually equivalent
// output: it must match the Row's own output byte-for-byte at several widths.
func TestHeaderWithAccessoryIsALayoutRow(t *testing.T) {
	dt := theme.DarkTheme()
	titleStyled := Header("Dashboard", dt)
	accessoryStyled := ansi.NewStyle().Foreground(dt.Muted).Render("v1.2.3")

	for _, width := range []int{16, 20, 40} {
		got := HeaderWithAccessory("Dashboard", "v1.2.3", width, dt)
		row := layout.Row(0,
			layout.FlexChild{Node: layout.Block(titleStyled)},
			layout.FlexChild{Node: layout.Block(""), Grow: 1},
			layout.FlexChild{Node: layout.Block(accessoryStyled)},
		)
		want := layout.Draw(row, layout.Constraints{MinW: width, MaxW: width, MaxH: layout.Unbounded})
		if got != want {
			t.Errorf("width %d: HeaderWithAccessory() =\n%q\nwant (the Row's own output)\n%q", width, got, want)
		}
		if ansi.Width(got) != width {
			t.Errorf("width %d: result is %d columns wide", width, ansi.Width(got))
		}
	}
}

func TestHeaderWithAccessoryTooNarrowStillSeparated(t *testing.T) {
	got := HeaderWithAccessory("Dashboard", "v1.2.3", 5, theme.DarkTheme())
	titleStyled := Header("Dashboard", theme.DarkTheme())
	accessoryStyled := ansi.NewStyle().Foreground(theme.DarkTheme().Muted).Render("v1.2.3")
	want := titleStyled + " " + accessoryStyled
	if got != want {
		t.Errorf("HeaderWithAccessory() with a too-narrow width = %q, want %q (single-space fallback)", got, want)
	}
}
