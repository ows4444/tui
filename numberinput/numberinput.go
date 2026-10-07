// Package numberinput is a thin wrapper over textinput.Model that
// restricts typed input to digits and a single leading '-'. Everything
// else (navigation, deletion, paste, focus, styling, View) behaves
// exactly like textinput.Model.
package numberinput

import (
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

// Update intercepts tui.KeyRunes to reject any rune that isn't an ASCII
// digit, except a single '-' which is accepted only when it would become
// the very first character of the value (criteria #335, #336). Every
// other key or Msg (navigation, deletion, paste, blink ticks, ...) is
// forwarded unchanged to the embedded textinput.Model.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	// The space bar arrives as its own key type, not as KeyRunes.
	if k, ok := msg.(tui.Key); ok && k.Type == tui.KeySpace {
		return m, nil
	}
	if k, ok := msg.(tui.Key); ok && k.Type == tui.KeyRunes {
		filtered := make([]rune, 0, len(k.Text))
		for _, r := range k.Text {
			if isDigit(r) {
				filtered = append(filtered, r)
				continue
			}
			if r == '-' && m.Cursor() == 0 && (len(m.Value()) == 0 || m.Value()[0] != '-') {
				filtered = append(filtered, r)
			}
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

func isDigit(r rune) bool { return r >= '0' && r <= '9' }

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}
