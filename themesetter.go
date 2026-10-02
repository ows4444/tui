package tui

import "github.com/ows4444/tui/theme"

// ThemeSetter is the contract for a widget that takes a theme.Theme and can
// be re-themed after construction. SetTheme returns the widget's own type T,
// the same way Component.Update does, so a value-type widget can implement
// it (tui.Themeable cannot: it returns Model).
//
// A package proves it satisfies the contract with
//
//	var _ tui.ThemeSetter[Model] = Model{}
//
// A root model that implements Themeable (see WithTheme) forwards the theme
// to its widgets with their SetTheme methods.
type ThemeSetter[T any] interface {
	SetTheme(theme.Theme) T
}
