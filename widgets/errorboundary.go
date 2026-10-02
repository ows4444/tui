package widgets

import "github.com/ows4444/tui/theme"

// ErrorBoundary calls render and returns its output unchanged when it
// returns normally. If render panics, ErrorBoundary recovers the panic
// and returns fallback styled as a VariantError Alert instead of letting
// the panic propagate to the caller. This is a widgets-layer helper only
// (no Program-level rendering primitive involved): it wraps a single
// func() string call.
func ErrorBoundary(render func() string, fallback string, t theme.Theme) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = Alert(fallback, VariantError, t, 0)
		}
	}()
	return render()
}
