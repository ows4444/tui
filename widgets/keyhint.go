package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/keymap"
)

var (
	keyHintKeyStyle = ansi.NewStyle().Bold()
	keyHintDimStyle = ansi.NewStyle().Faint()
)

// KeyHint renders a single "[key] action" hint, e.g. "[enter] submit",
// with the brackets and action dim and the key itself bold.
func KeyHint(key, action string) string {
	return keyHintDimStyle.Render("[") + keyHintKeyStyle.Render(key) + keyHintDimStyle.Render("] "+action)
}

// Hint is one KeyHint's worth of arguments, for building a help line out
// of several with KeyHints.
type Hint struct {
	Key    string
	Action string
}

// KeyHints joins several KeyHint results with sep, e.g. for a status-line
// help string like "[up/down] move  [enter] select  [q] quit".
func KeyHints(sep string, hints ...Hint) string {
	parts := make([]string, len(hints))
	for i, h := range hints {
		parts[i] = KeyHint(h.Key, h.Action)
	}
	return strings.Join(parts, sep)
}

// HintsFromKeymap returns the bindings of scope in reg as Hints, so a key-hint
// line reads the same registry as every other help widget.
func HintsFromKeymap(reg *keymap.Registry, scope string) []Hint {
	var out []Hint
	for _, h := range reg.Hints(scope) {
		out = append(out, Hint{Key: h.Key, Action: h.Desc})
	}
	return out
}
