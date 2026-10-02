package widgets

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Variant selects which of a Theme's semantic colors Badge and
// StatusIndicator render with.
type Variant int

const (
	// VariantNeutral uses the Theme's Muted colour. It is the zero value.
	VariantNeutral Variant = iota
	// VariantInfo uses the Theme's Info colour.
	VariantInfo
	// VariantSuccess uses the Theme's Success colour.
	VariantSuccess
	// VariantWarning uses the Theme's Warning colour.
	VariantWarning
	// VariantError uses the Theme's Error colour.
	VariantError
)

// Color resolves v to one of t's semantic colors — exported so other
// packages (e.g. toast) can share the same Variant-to-color mapping
// instead of re-implementing it.
func (v Variant) Color(t theme.Theme) ansi.Color {
	switch v {
	case VariantInfo:
		return t.Info
	case VariantSuccess:
		return t.Success
	case VariantWarning:
		return t.Warning
	case VariantError:
		return t.Error
	default:
		return t.Muted
	}
}
