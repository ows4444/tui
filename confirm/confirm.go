// Package confirm is a yes/no prompt widget — InkUI's "Confirm".
package confirm

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Model is a two-option Yes/No prompt: Left/Right/Tab move which option is
// highlighted, Enter confirms the highlighted one, and 'y'/'n' confirm
// directly regardless of what's highlighted.
type Model struct {
	Prompt   string
	YesLabel string
	NoLabel  string
	Theme    theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with
	// DefaultKeyMap; a Model built as a struct literal with a zero KeyMap
	// behaves as if it held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left click on the
	// Yes or No label answers the prompt as that key would. Off (the default)
	// ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the prompt (its first
	// row is the prompt's first line); clicks outside it are ignored.
	Bounds hittest.Rect

	yes bool // which option is highlighted; true = Yes
}

// New returns a Model with "Yes"/"No" labels and Yes highlighted, the
// common default for a prompt where doing nothing is the safe choice.
func New(prompt string) Model {
	return Model{Prompt: prompt, YesLabel: "Yes", NoLabel: "No", Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap(), yes: true}
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Toggle keymap.Binding // move the highlight to the other option
	Accept keymap.Binding // confirm the highlighted option
	Yes    keymap.Binding // confirm Yes directly
	No     keymap.Binding // confirm No directly
}

// DefaultKeyMap returns the keys a Model used before KeyMap existed.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Toggle: keymap.NewBinding("switch option", "left", "right", "tab"),
		Accept: keymap.NewBinding("confirm", "enter"),
		Yes:    keymap.NewBinding("yes", "y", "Y"),
		No:     keymap.NewBinding("no", "n", "N"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	if len(km.Toggle.Keys)+len(km.Accept.Keys)+len(km.Yes.Keys)+len(km.No.Keys) == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the actions the prompt currently honours, with
// descriptions, for help text.
func (m Model) Bindings() []keymap.Binding {
	km := m.keys()
	return []keymap.Binding{km.Toggle, km.Accept, km.Yes, km.No}
}

// hit reports whether msg triggers b, ignoring Shift so a shifted letter
// such as a capital 'Y' still matches as it did before KeyMap existed.
func hit(msg tui.Msg, b keymap.Binding) bool {
	if keymap.Matches(msg, b) {
		return true
	}
	if k, ok := msg.(tui.Key); ok && k.Mod.Shift() {
		k.Mod &^= input.ModShift
		return keymap.Matches(k, b)
	}
	return false
}

// Highlighted reports which option the cursor is currently on — not
// which was chosen; that only happens via ConfirmedMsg.
func (m Model) Highlighted() bool { return m.yes }

// ConfirmedMsg is delivered (via the Cmd Update returns) once the prompt
// is answered, by Enter or by a direct 'y'/'n' keypress.
type ConfirmedMsg struct {
	Yes bool
}

// Update moves the highlight on Left/Right/Tab, and confirms on Enter (the
// highlighted option) or a direct 'y'/'n' keypress, returning a Cmd that
// delivers ConfirmedMsg. With Mouse on, a left click on the Yes or No label
// confirms it the same way.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev)
	}
	km := m.keys()
	switch {
	case hit(msg, km.Toggle):
		m.yes = !m.yes
	case hit(msg, km.Accept):
		yes := m.yes
		return m, func() tui.Msg { return ConfirmedMsg{Yes: yes} }
	case hit(msg, km.Yes):
		return m, func() tui.Msg { return ConfirmedMsg{Yes: true} }
	case hit(msg, km.No):
		return m, func() tui.Msg { return ConfirmedMsg{Yes: false} }
	}
	return m, nil
}

// updateMouse answers the prompt when a left press lands on the Yes or No
// label (padding or brackets included) in the options row, the last line of the
// prompt.
func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse || ev.Action != tui.MouseActionPress || ev.Button != tui.MouseButtonLeft ||
		!m.Bounds.Contains(ev.X, ev.Y) {
		return m, nil
	}
	lx, ly := m.Bounds.Local(ev.X, ev.Y)
	if ly != strings.Count(m.Prompt, "\n") {
		return m, nil
	}
	last := m.Prompt[strings.LastIndexByte(m.Prompt, '\n')+1:]
	yesStart := ansi.Width(last) + 2
	yesEnd := yesStart + ansi.Width(m.YesLabel) + 2
	noStart := yesEnd + 2
	noEnd := noStart + ansi.Width(m.NoLabel) + 2
	switch {
	case lx >= yesStart && lx < yesEnd:
		m.yes = true
		return m, func() tui.Msg { return ConfirmedMsg{Yes: true} }
	case lx >= noStart && lx < noEnd:
		m.yes = false
		return m, func() tui.Msg { return ConfirmedMsg{Yes: false} }
	}
	return m, nil
}

// View renders the prompt and its two options, with the highlighted one
// shown in reverse video.
func (m Model) View() string { return m.Prompt + "  " + m.options() }

// options is the styled "Yes  No" pair with the highlighted one reversed.
func (m Model) options() string {
	yesLabel, noLabel := " "+m.YesLabel+" ", " "+m.NoLabel+" "
	highlight := m.themed().ResolvedStates().Selected.Bold()
	// Brackets replace the padding on the highlighted option, so the
	// highlight reads without colour or reverse video and the width holds.
	if m.yes {
		yesLabel = highlight.Render("[" + m.YesLabel + "]")
	} else {
		noLabel = highlight.Render("[" + m.NoLabel + "]")
	}
	return yesLabel + "  " + noLabel
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the prompt as plain text for accessible output (see
// tui.Linearizer): the prompt line, then one line per option with its
// position, ", selected" on the highlighted one, e.g. "Yes, option 1 of 2,
// selected". No brackets or reverse-video.
func (m Model) Linearize() string {
	yes, no := m.YesLabel+", option 1 of 2", m.NoLabel+", option 2 of 2"
	if m.yes {
		yes += ", selected"
	} else {
		no += ", selected"
	}
	return m.Prompt + "\n" + yes + "\n" + no
}
