package widgets

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// Header renders a styled title bar. With no accessory it's just the
// styled title; with one, the title is left-aligned and the accessory
// (e.g. a version string or a Badge/StatusIndicator) is right-aligned
// within width, using the theme's Muted color.
func Header(title string, t theme.Theme) string {
	return ansi.NewStyle().Bold().Foreground(t.Primary).Render(title)
}

// HeaderWithAccessory is Header plus a right-aligned accessory: the gap
// between them is a growing layout.Row child, filling whatever width
// title and accessory don't use themselves. If title and accessory
// together don't fit in width (leaving no room for even one space), they
// fall back to a single-space separator instead of being jammed together.
func HeaderWithAccessory(title, accessory string, width int, t theme.Theme) string {
	titleStyled := Header(title, t)
	if accessory == "" {
		return titleStyled
	}
	accessoryStyled := ansi.NewStyle().Foreground(t.Muted).Render(accessory)

	if ansi.Width(title)+ansi.Width(accessory)+1 > width {
		return titleStyled + " " + accessoryStyled
	}
	row := layout.Row(0,
		layout.FlexChild{Node: layout.Block(titleStyled)},
		layout.FlexChild{Node: layout.Block(""), Grow: 1},
		layout.FlexChild{Node: layout.Block(accessoryStyled)},
	)
	return layout.Draw(row, layout.Constraints{MinW: width, MaxW: width, MaxH: layout.Unbounded})
}
