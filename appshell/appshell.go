// Package appshell composes a common interactive-CLI-tool layout — a
// header, a full-width input, a scrollable content area, and an optional
// key-hints footer — out of existing widgets, without introducing any new
// rendering primitive of its own.
//
// Stability: experimental. Its API may change in any minor release.
package appshell

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/viewport"
	"github.com/ows4444/tui/widgets"
)

// Model is a composed application shell: a title rendered via
// widgets.Header, a single-line Input, a scrollable Content viewport, and
// an optional row of Hints rendered via widgets.KeyHints.
//
// v1 scope: AppShell composes exactly one input field, and that field is
// always treated as focused — Update always forwards keys to Input (after
// handling Enter itself), so there's no separate focus-tracking state.
type Model struct {
	Title   string
	Input   textinput.Model
	Content viewport.Model
	Hints   []widgets.Hint
	Theme   theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
}

// New returns a Model with an Input and a Content viewport sized to width
// (content height contentHeight), and the Input focused and ready to type
// into.
func New(title string, width, contentHeight int) Model {
	input := textinput.New()
	input.Width = width
	input.Focus()

	return Model{
		Title:   title,
		Input:   input,
		Content: viewport.New(width, contentHeight),
		Theme:   theme.DarkTheme(),
	}
}

// SubmitMsg is emitted when Enter is pressed, carrying the Input's value
// at the moment of submission. The Input is cleared as part of handling
// the key that produced it.
type SubmitMsg struct {
	Value string
}

// View renders the shell: Header, Input, Content, and — only when Hints is
// non-empty — a footer built from widgets.KeyHints, stacked
// with a layout.Column. With no Hints, the footer row is omitted entirely
// rather than rendered as a stray empty line.
func (m Model) View() string {
	blocks := []string{
		widgets.Header(m.Title, m.themed()),
		m.Input.View(),
		m.Content.View(),
	}
	if len(m.Hints) > 0 {
		blocks = append(blocks, widgets.KeyHints(" ", m.Hints...))
	}
	kids := make([]layout.FlexChild, len(blocks))
	for i, b := range blocks {
		kids[i] = layout.FlexChild{Node: layout.Block(b)}
	}
	return layout.Draw(layout.Column(0, kids...), layout.Unconstrained())
}

// Update handles a message. On Enter it emits a SubmitMsg carrying the
// Input's current value via a Cmd and clears the Input; any other key is
// forwarded to Input.Update.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if key, ok := msg.(tui.Key); ok && key.Type == tui.KeyEnter {
		value := m.Input.Value()
		m.Input.Reset()
		return m, func() tui.Msg { return SubmitMsg{Value: value} }
	}

	var cmd tui.Cmd
	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}
