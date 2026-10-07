// Package autocomplete is a text input with a filtered suggestion
// dropdown — InkUI's "Autocomplete".
//
// It composes tui/textinput for the text-editing half, but does *not*
// compose tui/picker for the dropdown despite both being list-with-cursor
// widgets: picker.Model's cursor is clamped against a fixed Items slice,
// while Autocomplete's suggestion list is a different length on every
// keystroke (each one re-filters against the typed text), which would
// mean rebuilding a picker.Model every Update call just to throw its
// cursor state away again. A plain highlight index, reset whenever the
// filtered set can have changed, is simpler and more obviously correct
// than forcing picker into a shape it wasn't designed for.
package autocomplete

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
)

// Model is a textinput.Model plus a filtered dropdown of Suggestions.
type Model struct {
	Input       textinput.Model
	Suggestions []string
	Theme       theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys that drive the open dropdown. New fills it with
	// DefaultKeyMap; a Model built as a struct literal with a zero KeyMap
	// behaves as if it held DefaultKeyMap. Keys it does not name go to Input.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left click on a
	// suggestion accepts it, the wheel moves the highlight, and a click on the
	// input row goes to Input (cursor placement). Off (the default) sends mouse
	// events to Input, whose own Mouse setting governs them.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the field: the input on
	// its first row and the suggestion rows below it, so make it tall enough for
	// the dropdown. Clicks outside it are ignored.
	Bounds hittest.Rect

	highlight    int
	justAccepted bool // suppresses the dropdown right after Tab/Enter/Esc
}

// New returns a focused Model (typing works immediately) with an
// unstarted cursor blink — call Init to start it, the same as
// textinput.Model.Focus would require.
func New(suggestions ...string) Model {
	m := Model{Suggestions: suggestions, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
	m.Input = textinput.New()
	m.Input.Focus()
	return m
}

// KeyMap names the keys of each dropdown action of a Model.
type KeyMap struct {
	Up     keymap.Binding // move the highlight up
	Down   keymap.Binding // move the highlight down
	Accept keymap.Binding // take the highlighted suggestion
	Close  keymap.Binding // close the dropdown without accepting
}

// DefaultKeyMap returns the keys a Model used before KeyMap existed.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:     keymap.NewBinding("previous suggestion", "up"),
		Down:   keymap.NewBinding("next suggestion", "down"),
		Accept: keymap.NewBinding("accept suggestion", "tab", "enter"),
		Close:  keymap.NewBinding("close suggestions", "esc"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	if len(km.Up.Keys)+len(km.Down.Keys)+len(km.Accept.Keys)+len(km.Close.Keys) == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the dropdown actions followed by the editing actions of
// Input, with descriptions, for help text.
func (m Model) Bindings() []keymap.Binding {
	km := m.keys()
	return append([]keymap.Binding{km.Up, km.Down, km.Accept, km.Close}, m.Input.Bindings()...)
}

// Init returns the Cmd that starts the embedded input's cursor blink.
// Without running it, the input still renders and accepts typing, just
// with a static (non-blinking) cursor.
//
// Deprecated: use Focus, which returns the same Cmd. A component has no Init;
// only a program's root model does.
func (m Model) Init() tui.Cmd {
	return m.Input.Focus()
}

// Focus and Blur delegate to the embedded Input, so a parent composing
// several fields (some textinput.Model, some autocomplete.Model) can
// drive focus across all of them the same way — see examples/form.
func (m *Model) Focus() tui.Cmd { return m.Input.Focus() }

// Blur unfocuses the embedded Input; see Focus.
func (m *Model) Blur() { m.Input.Blur() }

// IsOpen reports whether the suggestion dropdown is currently showing —
// useful for a parent deciding whether Enter on this field should be
// forwarded to accept a suggestion, or treated as "done with this field."
func (m Model) IsOpen() bool { return len(m.filtered()) > 0 }

// filtered returns the Suggestions whose text starts with the current
// input value, case-insensitively. It returns nil (no dropdown) for an
// empty value or right after an accept/close, until the next keystroke.
func (m Model) filtered() []string {
	if m.justAccepted {
		return nil
	}
	q := strings.ToLower(m.Input.Value())
	if q == "" {
		return nil
	}
	var out []string
	for _, s := range m.Suggestions {
		if strings.HasPrefix(strings.ToLower(s), q) {
			out = append(out, s)
		}
	}
	return out
}

// AcceptedMsg is delivered (via the Cmd Update returns) when a suggestion
// is accepted with Tab or Enter.
type AcceptedMsg struct {
	Value string
}

// Update forwards most keys to Input, intercepting Up/Down/Tab/Enter/Esc
// to drive the dropdown while it's open (has at least one filtered
// suggestion). If your app also uses Esc to quit, check that globally
// before forwarding to Autocomplete (the same pattern examples/list and
// examples/form already use for Ctrl+C/Esc) — otherwise Esc will only
// ever close the dropdown, never reach your quit handling, since
// Autocomplete consumes it whenever the dropdown is open. With Mouse on, a
// tui.MouseEvent is handled as described on Mouse.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok && m.Mouse {
		return m.updateMouse(ev)
	}
	if _, ok := msg.(tui.Key); !ok {
		var cmd tui.Cmd
		m.Input, cmd = m.Input.Update(msg)
		return m, cmd
	}

	if filtered := m.filtered(); len(filtered) > 0 {
		km := m.keys()
		switch {
		case keymap.Matches(msg, km.Up):
			if m.highlight > 0 {
				m.highlight--
			}
			return m, nil
		case keymap.Matches(msg, km.Down):
			if m.highlight < len(filtered)-1 {
				m.highlight++
			}
			return m, nil
		case keymap.Matches(msg, km.Accept):
			value := filtered[m.highlight]
			m.Input.SetValue(value)
			m.justAccepted = true
			m.highlight = 0
			return m, func() tui.Msg { return AcceptedMsg{Value: value} }
		case keymap.Matches(msg, km.Close):
			m.justAccepted = true
			return m, nil
		}
	}

	m.justAccepted = false
	m.highlight = 0
	var cmd tui.Cmd
	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}

// updateMouse handles a mouse event with Mouse on: a left press on a suggestion
// row accepts it, the wheel moves the highlight while the dropdown is open, and
// anything on the input row is forwarded to Input with its Bounds set to that
// row.
func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	filtered := m.filtered()
	inside := m.Bounds.Contains(ev.X, ev.Y)
	switch {
	case inside && len(filtered) > 0 && ev.Button == tui.MouseButtonWheelUp:
		if m.highlight > 0 {
			m.highlight--
		}
		return m, nil
	case inside && len(filtered) > 0 && ev.Button == tui.MouseButtonWheelDown:
		if m.highlight < len(filtered)-1 {
			m.highlight++
		}
		return m, nil
	}
	if inside && ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonLeft {
		if _, ly := m.Bounds.Local(ev.X, ev.Y); ly >= 1 && ly <= len(filtered) {
			value := filtered[ly-1]
			m.Input.SetValue(value)
			m.justAccepted = true
			m.highlight = 0
			return m, func() tui.Msg { return AcceptedMsg{Value: value} }
		}
	}
	// Input row (and drag/release that began there).
	in := m.Input
	in.Mouse = true
	in.Bounds = hittest.Rect{X: m.Bounds.X, Y: m.Bounds.Y, W: m.Bounds.W, H: 1}
	next, cmd := in.Update(ev)
	next.Mouse, next.Bounds = m.Input.Mouse, m.Input.Bounds
	m.Input = next
	return m, cmd
}

// View renders the embedded Input, plus a suggestion dropdown beneath it
// while IsOpen.
func (m Model) View() string {
	filtered := m.filtered()
	if len(filtered) == 0 {
		return m.Input.View()
	}

	dim := ansi.NewStyle().Faint()
	highlight := m.themed().ResolvedStates().Selected.Bold()

	var b strings.Builder
	b.WriteString(m.Input.View())
	for i, s := range filtered {
		b.WriteByte('\n')
		if i == m.highlight {
			b.WriteString(highlight.Render("> " + s))
		} else {
			b.WriteString(dim.Render("  " + s))
		}
	}
	return b.String()
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}
