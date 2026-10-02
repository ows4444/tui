// Package emailinput is a thin wrapper over textinput.Model that rejects
// whitespace runes as typed input. Everything else (navigation, deletion,
// paste, focus, styling, View) behaves exactly like textinput.Model. It
// also exposes a lightweight Valid heuristic for whether the current
// value looks like an email address.
package emailinput

import (
	"strings"
	"unicode"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/textinput"
)

// Model wraps a textinput.Model, filtering the runes it accepts. Its promoted
// Mouse and Bounds fields turn on click-to-place-cursor and drag-select exactly
// as on textinput.Model.
type Model struct {
	textinput.Model
}

// New returns a Model with textinput's default styling.
func New() Model {
	return Model{Model: textinput.New()}
}

// Update intercepts tui.KeyRunes to reject any whitespace rune (space,
// tab, and everything else unicode.IsSpace reports) so it can never be
// inserted into the value (criterion #514). Every other key or Msg
// (navigation, deletion, paste, blink ticks, ...) is forwarded unchanged
// to the embedded textinput.Model.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if k, ok := msg.(tui.Key); ok && k.Type == tui.KeyRunes {
		filtered := make([]rune, 0, len(k.Text))
		for _, r := range k.Text {
			if unicode.IsSpace(r) {
				continue
			}
			filtered = append(filtered, r)
		}
		if len(filtered) == 0 {
			return m, nil
		}
		k.Text, k.Code = string(filtered), filtered[0]
		msg = k
	}

	// A tui.MouseEvent is not filtered: it goes to the embedded textinput, which
	// handles it when Mouse is on (a click places the cursor, a drag selects).
	// Mouse and Bounds are the embedded model's, promoted onto Model.
	if ev, ok := msg.(tui.MouseEvent); ok {
		msg = ev
	}

	next, cmd := m.Model.Update(msg)
	m.Model = next
	return m, cmd
}

// Valid reports whether the current value looks like an email address,
// via a lightweight heuristic: exactly one '@' with a non-empty part
// before it, and a '.' with a non-empty part after it somewhere
// following the '@'. This is not full RFC 5322 validation (criterion
// #515).
func (m Model) Valid() bool {
	v := m.Value()

	at := strings.Count(v, "@")
	if at != 1 {
		return false
	}

	i := strings.IndexByte(v, '@')
	local, domain := v[:i], v[i+1:]
	if local == "" || domain == "" {
		return false
	}

	dot := strings.IndexByte(domain, '.')
	if dot < 0 {
		return false
	}
	if dot == 0 || dot == len(domain)-1 {
		return false
	}

	return true
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}
