// Package wizard provides a step-navigation state machine for multi-step
// flows: Next/Back advance or retreat the current step (clamped at the
// bounds), and View renders the step indicator via widgets.Stepper. It
// pairs naturally with widgets.Form/FormField for per-step validation
// errors, but owns no field state itself — the caller keeps that.
package wizard

import (
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// Model holds the ordered step titles and which one is current. Like the
// rest of this library, it's a plain value the caller stores and mutates
// via the returned Model — there's no hidden state.
type Model struct {
	titles  []string
	current int
	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
}

// New constructs a Model at step 0 for the given titles.
func New(titles ...string) Model {
	return Model{titles: titles}
}

// Current returns the current step index. For a zero-step Model it
// returns 0.
func (m Model) Current() int {
	return m.current
}

// Next runs validate; if it returns a non-nil error, the step does not
// advance and the error is returned unchanged. If validate returns nil,
// the current step advances by one, clamped so it never exceeds the last
// step index.
func (m *Model) Next(validate func() error) error {
	if validate != nil {
		if err := validate(); err != nil {
			return err
		}
	}
	if last := len(m.titles) - 1; m.current < last {
		m.current++
	}
	return nil
}

// Back retreats the current step by one, clamped at 0.
func (m *Model) Back() {
	if m.current > 0 {
		m.current--
	}
}

// View renders the step indicator via widgets.Stepper, with t resolved for
// theme.ComponentWizard and the tokens set by WithTokens.
func (m Model) View(t theme.Theme) string {
	return widgets.Stepper(m.titles, m.current, t.Resolve(theme.ComponentWizard, m.tokens))
}

// Tokens returns the colour tokens the step bar renders with under t: t's
// roles, overridden by any theme.WithTokens(theme.ComponentWizard, ...) and
// then by WithTokens.
func (m Model) Tokens(t theme.Theme) theme.Tokens {
	return t.Resolve(theme.ComponentWizard, m.tokens).TokensFor("")
}

// WithTokens returns m with tok as its per-instance colour override. Nil
// fields inherit from the theme passed to View, so only the roles tok names
// change. A second call replaces the first.
func (m Model) WithTokens(tok theme.Tokens) Model {
	m.tokens = tok
	return m
}
