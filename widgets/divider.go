package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Divider returns a plain horizontal rule, width columns of '─'. Unlike
// Badge/StatusIndicator/KeyHint, it's returned unstyled — a divider is
// normally one uniform color, so the caller can just wrap the whole result
// in ansi.Style.Render themselves instead of this function taking a style
// parameter.
func Divider(width int) string { return DividerWith(width, theme.Theme{}) }

// DividerWith is Divider drawn with t's rule glyph (theme.Glyphs.RuleH), so an
// ASCII theme gets a row of '-'. Divider is DividerWith on the default theme.
func DividerWith(width int, t theme.Theme) string {
	if width <= 0 {
		return ""
	}
	return strings.Repeat(t.GlyphSet().RuleH, width)
}

// DividerLabel returns a horizontal rule width columns wide with label
// centered in it, e.g. "── Section ───". If the label (plus one space of
// padding on each side) doesn't fit in width, it's truncated instead.
func DividerLabel(width int, label string) string {
	return DividerLabelWith(width, label, theme.Theme{})
}

// DividerLabelWith is DividerLabel drawn with t's rule glyph.
func DividerLabelWith(width int, label string, t theme.Theme) string {
	if width <= 0 {
		return ""
	}
	text := " " + label + " "
	textWidth := ansi.Width(text)
	if textWidth >= width {
		return ansi.Truncate(text, width)
	}
	remaining := width - textWidth
	left := remaining / 2
	right := remaining - left
	rule := t.GlyphSet().RuleH
	return strings.Repeat(rule, left) + text + strings.Repeat(rule, right)
}
