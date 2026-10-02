// Package taginput is a small composite widget for entering a list of
// short tags: a textinput.Model for typing plus a []string of committed
// tags, rendered as widgets.Tag chips followed by the live input field.
package taginput

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// Model holds an embedded textinput.Model for the in-progress value and
// the already-committed Tags. MaxTags, when positive, caps how many tags
// Enter will add (0 means unlimited).
type Model struct {
	Input   textinput.Model
	Tags    []string
	MaxTags int

	Theme theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens  theme.Tokens
	Variant widgets.Variant

	// Mouse, when true, makes Update handle tui.MouseEvent: a left click on the
	// input part of the row moves the cursor and a drag selects, as in
	// textinput. Clicks on the tag chips do nothing. Off (the default) ignores
	// the mouse.
	Mouse bool
	// Bounds is the screen rectangle of the row the app draws the Model on (tag
	// chips first, then the input). Update derives Input's own Bounds from it.
	Bounds hittest.Rect

	// KeyMap holds the keys for committing and removing tags. New fills it
	// with DefaultKeyMap; a Model built as a struct literal with a zero
	// KeyMap behaves as if it held DefaultKeyMap. Keys it does not name go
	// to Input, whose own KeyMap governs editing.
	KeyMap KeyMap
}

// KeyMap names the keys of the tag actions of a Model.
type KeyMap struct {
	Commit     keymap.Binding // add the typed text as a tag
	RemoveLast keymap.Binding // with empty input, remove the last tag
}

// DefaultKeyMap returns the keys a Model used before KeyMap existed.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Commit:     keymap.NewBinding("add tag", "enter"),
		RemoveLast: keymap.NewBinding("remove last tag", "backspace"),
	}
}

func (m Model) keys() KeyMap {
	if len(m.KeyMap.Commit.Keys) == 0 && len(m.KeyMap.RemoveLast.Keys) == 0 {
		return DefaultKeyMap()
	}
	return m.KeyMap
}

// Bindings returns the tag actions followed by the editing actions of Input,
// with descriptions, for help text.
func (m Model) Bindings() []keymap.Binding {
	km := m.keys()
	return append([]keymap.Binding{km.Commit, km.RemoveLast}, m.Input.Bindings()...)
}

// New returns a Model with textinput's default styling and theme.Dark.
func New() Model {
	return Model{
		Input:  textinput.New(),
		Theme:  theme.DarkTheme(),
		KeyMap: DefaultKeyMap(),
	}
}

// Update handles Enter (commit the current input value as a new tag,
// unless MaxTags is already reached) and Backspace on an empty input
// (pop the last tag). With Mouse on, a tui.MouseEvent goes to Input (see
// Bounds). Every other Msg is forwarded to Input.Update.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev)
	}
	km := m.keys()
	switch {
	case keymap.Matches(msg, km.Commit):
		value := strings.TrimSpace(m.Input.Value())
		if value == "" {
			return m, nil
		}
		if m.MaxTags > 0 && len(m.Tags) >= m.MaxTags {
			return m, nil
		}
		m.Tags = append(m.Tags, value)
		m.Input.Reset()
		return m, nil
	case keymap.Matches(msg, km.RemoveLast):
		if m.Input.Value() == "" {
			if len(m.Tags) > 0 {
				m.Tags = m.Tags[:len(m.Tags)-1]
			}
			return m, nil
		}
	}

	next, cmd := m.Input.Update(msg)
	m.Input = next
	return m, cmd
}

// updateMouse forwards a mouse event to Input when Mouse is on, with Input's
// Bounds set to the part of Bounds right of the tag chips.
func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse {
		return m, nil
	}
	used := 0
	for _, tag := range m.Tags {
		used += ansi.Width(widgets.Tag(tag, widgets.TagSolid, m.Variant, m.themed())) + 1
	}
	in := m.Input
	in.Mouse = true
	in.Bounds = hittest.Rect{X: m.Bounds.X + used, Y: m.Bounds.Y, W: m.Bounds.W - used, H: 1}
	if m.Bounds.H > 0 && in.Bounds.H > m.Bounds.H {
		in.Bounds.H = m.Bounds.H
	}
	next, cmd := in.Update(ev)
	next.Mouse, next.Bounds = m.Input.Mouse, m.Input.Bounds
	m.Input = next
	return m, cmd
}

// View renders each committed tag via widgets.Tag, space-separated,
// followed by the live Input field.
func (m Model) View() string {
	var b strings.Builder
	for _, tag := range m.Tags {
		b.WriteString(widgets.Tag(tag, widgets.TagSolid, m.Variant, m.themed()))
		b.WriteString(" ")
	}
	b.WriteString(m.Input.View())
	return b.String()
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the tag list and the input as plain text for accessible
// output (see tui.Linearizer): a first line naming the tags and how many
// there are ("Tags: go, tui (2 tags)", or "2 of 5 tags" with MaxTags, or "No
// tags"), then the embedded input's own line.
func (m Model) Linearize() string {
	tags := "No tags"
	if n := len(m.Tags); n > 0 {
		count := strconv.Itoa(n) + " tags"
		if n == 1 {
			count = "1 tag"
		}
		if m.MaxTags > 0 {
			count = strconv.Itoa(n) + " of " + strconv.Itoa(m.MaxTags) + " tags"
		}
		tags = "Tags: " + strings.Join(m.Tags, ", ") + " (" + count + ")"
	}
	return tags + "\n" + m.Input.Linearize()
}
