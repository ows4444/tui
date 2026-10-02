package theme

import "github.com/ows4444/tui/ansi"

// Typography is the set of text styles a theme gives to headings and inline
// emphasis. A zero Style in any field means "derive it from the colour
// roles", which Theme.ResolvedTypography does, so a Theme built before
// Typography existed reads the same.
type Typography struct {
	// H1 and H2 style top-level and second-level headings (markdown
	// headings, dialog titles).
	H1, H2 ansi.Style
	// Emphasis and Strong style italic and bold inline text. A style set
	// here replaces the surrounding text colour.
	Emphasis, Strong ansi.Style
	// Code styles inline code; Link styles links.
	Code, Link ansi.Style
}

// ResolvedTypography returns t.Typography with each empty style replaced by
// its default: H1 bold underlined Primary, H2 bold Primary, Emphasis italic,
// Strong bold, Code in Secondary, Link underlined Info. A style the theme set
// explicitly is kept as it is.
func (t Theme) ResolvedTypography() Typography {
	ty := t.Typography
	var zero ansi.Style
	if ty.H1 == zero {
		ty.H1 = ansi.NewStyle().Foreground(t.Primary).Bold().Underline()
	}
	if ty.H2 == zero {
		ty.H2 = ansi.NewStyle().Foreground(t.Primary).Bold()
	}
	if ty.Emphasis == zero {
		ty.Emphasis = ansi.NewStyle().Italic()
	}
	if ty.Strong == zero {
		ty.Strong = ansi.NewStyle().Bold()
	}
	if ty.Code == zero {
		ty.Code = ansi.NewStyle().Foreground(t.Secondary)
	}
	if ty.Link == zero {
		ty.Link = ansi.NewStyle().Foreground(t.Info).Underline()
	}
	return ty
}
