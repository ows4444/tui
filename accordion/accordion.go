// Package accordion is a list of collapsible sections — InkUI's
// "Accordion". Multiple sections can be expanded independently (not
// exclusive-open), since that's the simpler model and the caller can
// still enforce single-open themselves by collapsing others on Toggle if
// they want that.
package accordion

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Section is one entry: a header always shown, and content shown only
// while expanded.
type Section struct {
	Title   string
	Content string
}

// Model is a keyboard-navigable list of collapsible sections: Up/Down move
// the cursor, Enter/Space toggle the section under it.
type Model struct {
	Sections []Section
	Theme    theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with
	// DefaultKeyMap; a Model built as a struct literal with a zero KeyMap
	// behaves as if it held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left click on a
	// section header moves the cursor to it and toggles it, and the wheel
	// moves the cursor. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the accordion (its
	// first row is the first header); mouse events outside it are ignored.
	Bounds hittest.Rect

	cursor   int
	expanded map[int]bool
}

// New builds a Model from sections, all initially collapsed.
func New(sections ...Section) Model {
	return Model{Sections: sections, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Up     keymap.Binding // move the cursor to the previous section
	Down   keymap.Binding // move the cursor to the next section
	Toggle keymap.Binding // expand or collapse the section under the cursor
}

// DefaultKeyMap returns the keys a Model used before KeyMap existed.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:     keymap.NewBinding("previous section", "up"),
		Down:   keymap.NewBinding("next section", "down"),
		Toggle: keymap.NewBinding("expand/collapse", "enter", "space"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	if len(km.Up.Keys)+len(km.Down.Keys)+len(km.Toggle.Keys) == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the actions the accordion currently honours, with
// descriptions, for help text.
func (m Model) Bindings() []keymap.Binding {
	km := m.keys()
	return []keymap.Binding{km.Up, km.Down, km.Toggle}
}

// Cursor returns the index of the section currently under the cursor.
func (m Model) Cursor() int { return m.cursor }

// SetCursor moves the cursor to i, clamped to a valid section index (or 0
// with no sections).
func (m *Model) SetCursor(i int) {
	if len(m.Sections) == 0 {
		m.cursor = 0
		return
	}
	m.cursor = clamp(i, 0, len(m.Sections)-1)
}

// IsExpanded reports whether section i is expanded.
func (m Model) IsExpanded(i int) bool { return m.expanded[i] }

// Toggle flips whether section i is expanded. Like
// multiselect.Model.Toggle, it always rebuilds the map rather than
// mutating it in place, since Model is used with value semantics and a
// map field's header (unlike its contents) copies by value — mutating in
// place would corrupt any other Model value still sharing that map.
func (m *Model) Toggle(i int) {
	next := make(map[int]bool, len(m.expanded)+1)
	for k, v := range m.expanded {
		next[k] = v
	}
	next[i] = !next[i]
	m.expanded = next
}

// Update moves the cursor on Up/Down and toggles the section under it on
// Enter or Space. With Mouse on it also handles tui.MouseEvent (see
// Model.Mouse). Other Msgs and an empty Model are no-ops.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev), nil
	}
	if _, ok := msg.(tui.Key); !ok || len(m.Sections) == 0 {
		return m, nil
	}

	km := m.keys()
	switch {
	case keymap.Matches(msg, km.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case keymap.Matches(msg, km.Down):
		if m.cursor < len(m.Sections)-1 {
			m.cursor++
		}
	case keymap.Matches(msg, km.Toggle):
		m.Toggle(m.cursor)
	}
	return m, nil
}

// updateMouse applies a mouse press inside Bounds when Mouse is on.
func (m Model) updateMouse(ev tui.MouseEvent) Model {
	if !m.Mouse || len(m.Sections) == 0 || !m.Bounds.Contains(ev.X, ev.Y) || ev.Action != tui.MouseActionPress {
		return m
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp:
		m.SetCursor(m.cursor - 1)
	case tui.MouseButtonWheelDown:
		m.SetCursor(m.cursor + 1)
	case tui.MouseButtonLeft:
		_, row := m.Bounds.Local(ev.X, ev.Y)
		if i, header := m.sectionAtRow(row); i >= 0 && header {
			m.SetCursor(i)
			m.Toggle(i)
		}
	}
	return m
}

// sectionAtRow maps a row of the rendered View to the section it belongs to
// and whether it is that section's header (as opposed to a content line). It
// returns -1 for a row below the last section.
func (m Model) sectionAtRow(row int) (section int, header bool) {
	y := 0
	for i, s := range m.Sections {
		if row == y {
			return i, true
		}
		y++
		if m.expanded[i] && s.Content != "" {
			n := strings.Count(s.Content, "\n") + 1
			if row < y+n {
				return i, false
			}
			y += n
		}
	}
	return -1, false
}

// View renders every section's header, with content shown beneath any
// expanded one.
func (m Model) View() string {
	if len(m.Sections) == 0 {
		return ""
	}

	cursorStyle := m.themed().ResolvedStates().Selected.Bold()
	dim := ansi.NewStyle().Faint()

	var b strings.Builder
	for i, s := range m.Sections {
		prefix := "  "
		title := s.Title
		if i == m.cursor {
			prefix = "> "
			title = cursorStyle.Render(title)
		}
		arrow := m.themed().GlyphSet().Collapsed
		if m.expanded[i] {
			arrow = m.themed().GlyphSet().Expanded
		}
		b.WriteString(prefix + arrow + " " + title)
		if m.expanded[i] && s.Content != "" {
			for _, line := range strings.Split(s.Content, "\n") {
				b.WriteString("\n    " + dim.Render(line))
			}
		}
		if i < len(m.Sections)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
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

// Linearize renders the sections as plain text for accessible output (see
// tui.Linearizer): one line per section with its title, position and
// "expanded" or "collapsed", ", selected" on the cursor row, and for an
// expanded section one "Title content: line" line per content line, e.g.
// "Display, section 1 of 2, expanded, selected". No arrows or indentation.
// An accordion with no sections reads "No sections".
func (m Model) Linearize() string {
	if len(m.Sections) == 0 {
		return "No sections"
	}
	var out []string
	for i, s := range m.Sections {
		state := "collapsed"
		if m.expanded[i] {
			state = "expanded"
		}
		line := s.Title + ", section " + strconv.Itoa(i+1) + " of " + strconv.Itoa(len(m.Sections)) + ", " + state
		if i == m.cursor {
			line += ", selected"
		}
		out = append(out, line)
		if m.expanded[i] && s.Content != "" {
			for _, c := range strings.Split(s.Content, "\n") {
				out = append(out, s.Title+" content: "+c)
			}
		}
	}
	return strings.Join(out, "\n")
}
