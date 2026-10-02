// Package commandpalette is a text input with a fuzzy-filtered dropdown
// of Commands — a "Ctrl+K" style command palette.
//
// It reuses autocomplete.Model's entire filter-dropdown skeleton (embedded
// tui/textinput, re-filter-per-keystroke, highlight index, Up/Down/Enter,
// dropdown View), swapping two things: the match function is fuzzy
// subsequence matching instead of prefix matching, and accepting a result
// emits a Msg identifying the chosen Command instead of filling the input
// value — a palette runs a command, it doesn't complete text.
//
// Stability: experimental. Its API may change in any minor release.
package commandpalette

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
)

// Command is a single palette entry: a Name to match and display, plus an
// optional Description shown alongside it.
type Command struct {
	Name        string
	Description string
}

// Model is a textinput.Model plus a fuzzy-filtered dropdown of Commands.
type Model struct {
	Input    textinput.Model
	Commands []Command
	Theme    theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// Raw, when true, draws command names and descriptions unchanged. By
	// default they are sanitised (ansi.Sanitize) so untrusted text cannot
	// carry terminal escape sequences.
	Raw bool

	// KeyMap holds the navigation keys Update reacts to while the dropdown
	// is open; every other key goes to Input. New sets DefaultKeyMap; a Model
	// built as a struct literal with a zero KeyMap uses DefaultKeyMap.
	KeyMap KeyMap

	// Mouse turns on mouse handling in Update (off by default). Bounds is the
	// screen rectangle where the app draws the palette (the input on its
	// first row, the dropdown beneath); events outside it, or while the
	// dropdown is closed, are ignored. WheelStep is the rows one wheel notch
	// moves the highlight (0 means 3). A left press on a dropdown row selects
	// that command, exactly as Enter on it would.
	Mouse     bool
	Bounds    hittest.Rect
	WheelStep int

	highlight    int
	justAccepted bool // suppresses the dropdown right after Enter/Esc
}

// New returns a focused Model (typing works immediately) with an
// unstarted cursor blink — call Init to start it, the same as
// textinput.Model.Focus would require.
func New(commands ...Command) Model {
	m := Model{Commands: commands, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
	m.Input = textinput.New()
	m.Input.Focus()
	return m
}

// KeyMap is the set of navigation keys a palette reacts to while its
// dropdown is open.
type KeyMap struct {
	Up   keymap.Binding
	Down keymap.Binding
	// Select runs the highlighted command.
	Select keymap.Binding
	// Close closes the dropdown without running anything.
	Close keymap.Binding
}

// DefaultKeyMap returns the default keys: up/down, enter to select, esc to
// close.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:     keymap.NewBinding("up", "up"),
		Down:   keymap.NewBinding("down", "down"),
		Select: keymap.NewBinding("run command", "enter"),
		Close:  keymap.NewBinding("close", "esc"),
	}
}

func (k KeyMap) all() []keymap.Binding {
	return []keymap.Binding{k.Up, k.Down, k.Select, k.Close}
}

// keys is the KeyMap in force: KeyMap, or the defaults when it is unset.
func (m Model) keys() KeyMap {
	for _, b := range m.KeyMap.all() {
		if len(b.Keys) > 0 {
			return m.KeyMap
		}
	}
	return DefaultKeyMap()
}

// Bindings returns the active key bindings, with descriptions, for help text.
func (m Model) Bindings() []keymap.Binding { return m.keys().all() }

// Init returns the Cmd that starts the embedded input's cursor blink.
// Without running it, the input still renders and accepts typing, just
// with a static (non-blinking) cursor.
func (m Model) Init() tui.Cmd {
	return m.Input.Focus()
}

// Focus and Blur delegate to the embedded Input, so a parent composing
// several fields can drive focus across all of them the same way.
func (m *Model) Focus() tui.Cmd { return m.Input.Focus() }

// Blur unfocuses the embedded Input; see Focus.
func (m *Model) Blur() { m.Input.Blur() }

// IsOpen reports whether the command dropdown is currently showing.
func (m Model) IsOpen() bool { return len(m.filtered()) > 0 }

// filtered returns the Commands that fuzzy-match the current input value:
// every rune of the (lowercased) query must appear in order within the
// (lowercased) Command Name, not necessarily contiguously — so "cp"
// matches "CommandPalette". It returns nil (no dropdown) for an empty
// value or right after an accept/close, until the next keystroke, the
// same convention autocomplete.Model uses: an empty query in a command
// palette usually means "nothing typed yet", not "show every command",
// since palettes are typically opened over dozens of commands where an
// unfiltered dump isn't useful until the user starts typing.
func (m Model) filtered() []Command {
	if m.justAccepted {
		return nil
	}
	q := strings.ToLower(m.Input.Value())
	if q == "" {
		return nil
	}
	var out []Command
	for _, c := range m.Commands {
		if fuzzyMatch(q, strings.ToLower(c.Name)) {
			out = append(out, c)
		}
	}
	return out
}

// fuzzyMatch reports whether every rune of q appears in order within s
// (not necessarily contiguously). Both q and s are assumed already
// case-normalized by the caller.
func fuzzyMatch(q, s string) bool {
	if q == "" {
		return true
	}
	qr := []rune(q)
	qi := 0
	for _, r := range s {
		if r == qr[qi] {
			qi++
			if qi == len(qr) {
				return true
			}
		}
	}
	return false
}

// SelectedMsg is delivered (via the Cmd Update returns) when a command is
// selected with Enter. Unlike autocomplete.Model's AcceptedMsg, this does
// not fill the input value — a chosen command is meant to run, not to be
// edited further.
type SelectedMsg struct {
	Command Command
}

// Update forwards most keys to Input, intercepting the KeyMap keys (by default
// Up/Down/Enter/Esc) to drive the dropdown while it's open (has at least one filtered command).
// If your app also uses Esc to quit, check that globally before
// forwarding to CommandPalette — otherwise Esc will only ever close the
// dropdown, never reach your quit handling, since CommandPalette consumes
// it whenever the dropdown is open.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.mouse(ev)
	}
	if _, ok := msg.(tui.Key); !ok {
		var cmd tui.Cmd
		m.Input, cmd = m.Input.Update(msg)
		return m, cmd
	}

	km := m.keys()
	if filtered := m.filtered(); len(filtered) > 0 {
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
		case keymap.Matches(msg, km.Select):
			return m.accept(filtered[m.highlight])
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

// accept closes the dropdown and returns a Cmd delivering SelectedMsg for c.
func (m Model) accept(c Command) (Model, tui.Cmd) {
	m.justAccepted = true
	m.highlight = 0
	return m, func() tui.Msg { return SelectedMsg{Command: c} }
}

func (m Model) mouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	filtered := m.filtered()
	if !m.Mouse || len(filtered) == 0 || !m.Bounds.Contains(ev.X, ev.Y) || ev.Action != tui.MouseActionPress {
		return m, nil
	}
	step := m.WheelStep
	if step <= 0 {
		step = 3
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp:
		m.highlight = clamp(m.highlight-step, 0, len(filtered)-1)
	case tui.MouseButtonWheelDown:
		m.highlight = clamp(m.highlight+step, 0, len(filtered)-1)
	case tui.MouseButtonLeft:
		_, ly := m.Bounds.Local(ev.X, ev.Y)
		if i := ly - 1; i >= 0 && i < len(filtered) { // row 0 is the input
			m.highlight = i
			return m.accept(filtered[i])
		}
	}
	return m, nil
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

// View renders the embedded Input, plus a command dropdown beneath it
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
	for i, c := range filtered {
		b.WriteByte('\n')
		line := ansi.Clean(m.Raw, c.Name)
		if c.Description != "" {
			line += " " + m.themed().GlyphSet().Dash + " " + ansi.Clean(m.Raw, c.Description)
		}
		if i == m.highlight {
			b.WriteString(highlight.Render("> " + line))
		} else {
			b.WriteString(dim.Render("  " + line))
		}
	}
	return b.String()
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}
