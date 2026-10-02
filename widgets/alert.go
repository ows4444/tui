package widgets

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// alertIcon returns the leading severity glyph for a variant, distinct
// per variant so severity is legible even without color: ✗ error, ✓
// success, ⚠ warning, ℹ info, • neutral.
func alertIcon(v Variant) string { return alertGlyph(v, theme.UnicodeGlyphSet()) }

// alertGlyph is alertIcon drawn from g, so an ASCII theme gets ASCII icons.
func alertGlyph(v Variant, g theme.Glyphs) string {
	switch v {
	case VariantError:
		return g.Cross
	case VariantSuccess:
		return g.Check
	case VariantWarning:
		return g.Warning
	case VariantInfo:
		return g.Info
	default:
		return g.Bullet
	}
}

// Icon returns the glyph from g that stands for v without colour: the icons
// Alert uses (Cross, Check, Warning, Info, and Bullet for VariantNeutral).
func (v Variant) Icon(g theme.Glyphs) string { return alertGlyph(v, g) }

// Mark is Icon, except that VariantNeutral has none and returns "", so a
// neutral widget keeps its plain look while every other variant stays
// distinguishable as plain text.
func (v Variant) Mark(g theme.Glyphs) string {
	if v == VariantNeutral {
		return ""
	}
	return alertGlyph(v, g)
}

// markPrefix is Variant.Mark plus a space, or "" for VariantNeutral.
func markPrefix(v Variant, t theme.Theme) string {
	if m := v.Mark(t.GlyphSet()); m != "" {
		return m + " "
	}
	return ""
}

// Alert renders message in a bordered box colored by variant via
// variant.Color(t) (not t.BorderColor), like Box/Panel but with a leading
// severity icon that visibly differs by variant (e.g. "✗" for
// VariantError vs "✓" for VariantSuccess), so severity reads even without
// color. width behaves exactly as it does for Box/Panel: 0 sizes to
// content, a positive width wraps content so every rendered line's
// ansi.Width stays within it.
func Alert(message string, variant Variant, t theme.Theme, width int) string {
	color := variant.Color(t)
	icon := alertGlyph(variant, t.GlyphSet())
	text := icon + " " + message

	cw, auto := boxContentWidth(width)

	body := text
	if !auto {
		body = ansi.WrapStyled(text, cw)
	}

	b := layout.NewBox().Border(t.Border).BorderColor(color).PaddingAll(1)
	if !auto {
		b = b.Width(cw)
	}
	return b.Render(body)
}
