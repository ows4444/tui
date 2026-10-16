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

// HintsFromBindings returns the enabled bindings that have a description as
// Hints, each showing its first key name. Feed it a focused widget's Bindings
// (tui.BindingsProvider), so the footer follows focus and follows the widget's
// KeyMap when the app rebinds it.
func HintsFromBindings(bs []keymap.Binding) []Hint {
	var out []Hint
	for _, b := range bs {
		if !b.Enabled() || b.Desc == "" || len(b.Keys) == 0 {
			continue
		}
		out = append(out, Hint{Key: b.Keys[0], Action: b.Desc})
	}
	return out
}

// Footer renders hints as one line of at most width cells, separated by two
// spaces. Hints that do not fit are dropped from the end and a dim "…" marks
// that some were, so the most important hints go first. A width of 0 or less
// keeps them all. If not even the first hint fits, it is cut to width.
func Footer(width int, hints ...Hint) string {
	const sep = "  "
	parts := make([]string, len(hints))
	for i, h := range hints {
		parts[i] = KeyHint(h.Key, h.Action)
	}
	if width <= 0 {
		return strings.Join(parts, sep)
	}
	fits := func(n int, more bool) bool { // would the first n parts (and "…") fit?
		w := 0
		for i := 0; i < n; i++ {
			if i > 0 {
				w += len(sep)
			}
			w += ansi.Width(parts[i])
		}
		if more {
			w += len(sep) + 1
		}
		return w <= width
	}
	n := len(parts)
	if fits(n, false) {
		return strings.Join(parts, sep)
	}
	for n > 0 && !fits(n, true) {
		n--
	}
	if n == 0 {
		if len(parts) == 0 {
			return ""
		}
		return ansi.Truncate(parts[0], width)
	}
	return strings.Join(parts[:n], sep) + sep + keyHintDimStyle.Render("…")
}
