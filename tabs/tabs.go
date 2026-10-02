// Package tabs is a horizontal tab bar — InkUI's "Tabs". It only manages
// which tab is active and renders the bar itself; actual per-tab content
// switching stays app-owned (the parent reads Active() and renders
// whichever content it has for that index), the same way picker doesn't
// own what "selected" means beyond reporting an index.
package tabs

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Model is a horizontal tab bar: Left/Right move the active tab (clamped
// at the ends), Tab wraps around.
type Model struct {
	Labels []string
	Theme  theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the key bindings Update obeys. New fills it with
	// DefaultKeyMap; a Model built as a literal with no bindings set behaves
	// as if it held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left click on
	// a tab label activates it. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the bar (its first
	// row holds the tabs); clicks outside it are ignored.
	Bounds hittest.Rect

	active int
}

// KeyMap is the set of keys Update reacts to.
type KeyMap struct {
	Prev  keymap.Binding
	Next  keymap.Binding
	Cycle keymap.Binding
}

// DefaultKeyMap returns the default bindings: left and right move, tab
// cycles with wrap-around.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Prev:  keymap.NewBinding("previous tab", "left"),
		Next:  keymap.NewBinding("next tab", "right"),
		Cycle: keymap.NewBinding("cycle tabs", "tab"),
	}
}

// keys returns m.KeyMap, or DefaultKeyMap when it is unset.
func (m Model) keys() KeyMap {
	k := m.KeyMap
	if len(k.Prev.Keys)+len(k.Next.Keys)+len(k.Cycle.Keys) == 0 {
		return DefaultKeyMap()
	}
	return k
}

// Bindings returns the active bindings, for help text.
func (m Model) Bindings() []keymap.Binding {
	k := m.keys()
	return []keymap.Binding{k.Prev, k.Next, k.Cycle}
}

// New builds a Model from labels, with the first tab active.
func New(labels ...string) Model {
	return Model{Labels: labels, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// Active returns the index of the currently active tab.
func (m Model) Active() int { return m.active }

// SetActive sets the active tab to i, clamped to a valid index (or 0 with
// no labels).
func (m *Model) SetActive(i int) {
	if len(m.Labels) == 0 {
		m.active = 0
		return
	}
	m.active = clamp(i, 0, len(m.Labels)-1)
}

// ChangedMsg is delivered (via the Cmd Update returns) when the active tab
// changes.
type ChangedMsg struct{ Index int }

// Update moves the active tab on the Prev/Next bindings (clamped at the
// ends) or Cycle (wrapping around). With Mouse on, a left click on a tab
// label activates it. Any other Msg is a no-op.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if len(m.Labels) == 0 {
		return m, nil
	}
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev)
	}
	if _, ok := msg.(tui.Key); !ok {
		return m, nil
	}

	k := m.keys()
	switch {
	case keymap.Matches(msg, k.Prev):
		if m.active == 0 {
			return m, nil
		}
		m.active--
	case keymap.Matches(msg, k.Next):
		if m.active == len(m.Labels)-1 {
			return m, nil
		}
		m.active++
	case keymap.Matches(msg, k.Cycle):
		m.active = (m.active + 1) % len(m.Labels)
	default:
		return m, nil
	}

	idx := m.active
	return m, func() tui.Msg { return ChangedMsg{Index: idx} }
}

// updateMouse activates the tab under a left click in Bounds' first row.
// Each tab is its label padded with a space on both sides, and tabs are one
// cell apart, as View draws them.
func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse || !m.Bounds.Contains(ev.X, ev.Y) ||
		ev.Action != tui.MouseActionPress || ev.Button != tui.MouseButtonLeft {
		return m, nil
	}
	lx, ly := m.Bounds.Local(ev.X, ev.Y)
	if ly != 0 {
		return m, nil
	}
	x := 0
	for i, l := range m.Labels {
		w := ansi.Width(l) + 2
		if lx >= x && lx < x+w {
			if i == m.active {
				return m, nil
			}
			m.active = i
			return m, func() tui.Msg { return ChangedMsg{Index: i} }
		}
		x += w + 1
	}
	return m, nil
}

// View renders the tab bar with the active tab reverse-styled.
func (m Model) View() string {
	active := m.themed().ResolvedStates().Selected.Bold()
	inactive := ansi.NewStyle().Foreground(m.themed().Muted)

	parts := make([]string, len(m.Labels))
	for i, l := range m.Labels {
		label := " " + l + " "
		if i == m.active {
			// Brackets replace the padding: visible without colour, same width.
			parts[i] = active.Render("[" + l + "]")
		} else {
			parts[i] = inactive.Render(label)
		}
	}
	return strings.Join(parts, " ")
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the tab bar as plain text for accessible output (see
// tui.Linearizer): one line per tab with its label and position, and
// ", selected" on the active tab, e.g. "Logs, tab 2 of 3, selected". No
// padding or reverse-video styling. With no tabs it reads "No tabs".
func (m Model) Linearize() string {
	if len(m.Labels) == 0 {
		return "No tabs"
	}
	lines := make([]string, len(m.Labels))
	for i, l := range m.Labels {
		lines[i] = l + ", tab " + strconv.Itoa(i+1) + " of " + strconv.Itoa(len(m.Labels))
		if i == m.active {
			lines[i] += ", selected"
		}
	}
	return strings.Join(lines, "\n")
}
