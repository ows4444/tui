package widgets

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TagStyle selects how Tag fills its chip.
type TagStyle int

const (
	// TagSolid renders a filled background chip, matching Badge's
	// Background(variant.Color(t))+Foreground(t.TextInverse) convention.
	TagSolid TagStyle = iota
	// TagOutline renders only a colored border/foreground, with no
	// background fill, so it stays visually distinct from TagSolid.
	TagOutline
)

// Tag renders text as a small chip colored by variant (via
// widgets.Variant's existing color mapping — no separate severity type),
// either as a solid filled pill (TagSolid, Badge's Background+TextInverse
// convention) or as an outlined chip with just a colored border/foreground
// and no background fill (TagOutline).
//
// Tag is a pure render function with no Model/Update of its own, following
// widgets.Checkbox's caller-owns-state convention: dismissing/removing a
// tag is the CALLER's responsibility — drop it from the caller's own slice
// of tags and re-render without it. Tag itself has no notion of dismissal
// or interactive state.
//
// Empty text renders a minimal chip (just the outline/pill markers)
// without panicking.
func Tag(text string, style TagStyle, variant Variant, t theme.Theme) string {
	color := variant.Color(t)
	mark := markPrefix(variant, t)
	if text == "" {
		mark = "" // a minimal chip stays minimal
	}

	switch style {
	case TagOutline:
		return ansi.NewStyle().Bold().Foreground(color).Render("[" + mark + text + "]")
	default:
		return ansi.NewStyle().Bold().Background(color).Foreground(t.TextInverse).Render(" " + mark + text + " ")
	}
}
