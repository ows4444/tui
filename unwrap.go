package tui

import "github.com/ows4444/tui/theme"

// Unwrapper is implemented by a root Model that wraps another Model and adds
// behaviour around it (a test harness, a logger, a layout shell) without
// re-implementing the optional interfaces of what it wraps.
//
// When the Program looks for an optional interface (CursorPlacer, Linearizer,
// Themeable, BindingsProvider) it asks the root Model first, then the Model
// returned by Unwrap, and so on down the chain; the first match wins. A
// wrapper therefore needs only Init, Update, View and Unwrap, however many
// optional interfaces the model it wraps has.
//
// A wrapper that also implements Rewrapper takes part in Themeable: the
// Program re-themes the wrapped Model and puts the result back through
// Rewrap. Without Rewrap a theme set through the wrapper is not applied,
// because the wrapper cannot say how to hold the new Model.
type Unwrapper interface {
	// Unwrap returns the Model this one wraps, or nil if there is none.
	Unwrap() Model
}

// Rewrapper is the optional companion of Unwrapper for a wrapper that can be
// rebuilt around a different inner Model. Rewrap returns a copy of the
// receiver that wraps inner, the way Update returns the next model.
type Rewrapper interface {
	Rewrap(inner Model) Model
}

// maxUnwrapDepth bounds the walk down an Unwrap chain, so a Model whose Unwrap
// returns itself (or a cycle) ends in "not found" instead of a hang.
const maxUnwrapDepth = 32

// find returns the first Model in m's Unwrap chain, m included, that
// implements T.
func find[T any](m Model) (T, bool) {
	for range maxUnwrapDepth {
		if t, ok := m.(T); ok {
			return t, true
		}
		u, ok := m.(Unwrapper)
		if !ok {
			break
		}
		if m = u.Unwrap(); m == nil {
			break
		}
	}
	var zero T
	return zero, false
}

// setTheme applies t at the first Themeable in m's Unwrap chain and returns
// the rebuilt chain. ok is false when there is no Themeable, or a wrapper
// above it cannot be rebuilt (it lacks Rewrapper); m is then returned as it was.
func setTheme(m Model, t theme.Theme, depth int) (next Model, ok bool) {
	if th, is := m.(Themeable); is {
		return th.SetTheme(t), true
	}
	u, is := m.(Unwrapper)
	if !is || depth >= maxUnwrapDepth {
		return m, false
	}
	inner := u.Unwrap()
	if inner == nil {
		return m, false
	}
	rw, is := m.(Rewrapper)
	if !is {
		return m, false
	}
	themed, ok := setTheme(inner, t, depth+1)
	if !ok {
		return m, false
	}
	return rw.Rewrap(themed), true
}
