// Package colorpicker is a palette-swatch + hex-input color picker.
//
// It is deliberately not built on picker.Model's View: round 11 of the
// widget-catalog gap analysis found that wrapping a separately-Render-ed
// swatch style in an outer cursor-highlight Render call clobbers the
// highlight, because ansi.Style.Render always appends a full SGR reset
// with no save/restore (see ansi/style.go). Model.View instead computes
// ONE combined ansi.Style per swatch — swatch color and cursor highlight
// together — and calls Render exactly once per swatch.
package colorpicker

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
)

// swatch is the plain-text block rendered for each palette entry. Two
// cells wide so a Background color is visible as a solid block.
const swatch = "  "

// Model is a keyboard-navigable palette of ansi.Color swatches plus an
// embedded hex text input, for callers who want an arbitrary RGB color
// rather than one of Palette's fixed entries.
type Model struct {
	Palette  []ansi.Color
	HexInput textinput.Model
	Theme    theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with
	// DefaultKeyMap; a Model built as a struct literal with a zero KeyMap
	// behaves as if it held DefaultKeyMap. While the hex input has focus,
	// keys other than SwitchPane and Accept go to HexInput.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left click on a
	// swatch moves the cursor to it and gives the palette focus (a click on
	// the highlighted swatch confirms it, like Enter), a left click on the hex
	// input focuses it, and the wheel moves the palette cursor. Off (the
	// default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the picker (one
	// row); mouse events outside it are ignored.
	Bounds hittest.Rect

	cursor     int
	hexFocused bool
}

// New builds a Model from palette, with the cursor on the first entry and
// focus on the palette (not the hex input).
func New(palette ...ansi.Color) Model {
	return Model{
		Palette:  palette,
		HexInput: textinput.New(),
		Theme:    theme.DarkTheme(),
		KeyMap:   DefaultKeyMap(),
	}
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Prev       keymap.Binding // move the palette cursor left
	Next       keymap.Binding // move the palette cursor right
	Accept     keymap.Binding // confirm the swatch, or the hex value
	SwitchPane keymap.Binding // toggle focus between palette and hex input
}

// DefaultKeyMap returns the keys a Model used before KeyMap existed.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Prev:       keymap.NewBinding("previous color", "left"),
		Next:       keymap.NewBinding("next color", "right"),
		Accept:     keymap.NewBinding("select color", "enter"),
		SwitchPane: keymap.NewBinding("switch palette/hex", "tab"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	if len(km.Prev.Keys)+len(km.Next.Keys)+len(km.Accept.Keys)+len(km.SwitchPane.Keys) == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the actions active for the focused pane, with
// descriptions, for help text: SwitchPane and Accept, then the palette
// movement keys, or the editing keys of HexInput while it has focus.
func (m Model) Bindings() []keymap.Binding {
	km := m.keys()
	if m.hexFocused {
		return append([]keymap.Binding{km.SwitchPane, km.Accept}, m.HexInput.Bindings()...)
	}
	return []keymap.Binding{km.Prev, km.Next, km.Accept, km.SwitchPane}
}

// Cursor returns the index of the palette entry currently under the
// cursor.
func (m Model) Cursor() int { return m.cursor }

// HexFocused reports whether keys are currently forwarded to HexInput
// rather than moving the palette cursor.
func (m Model) HexFocused() bool { return m.hexFocused }

// SelectedMsg is delivered (via the Cmd Update returns) when a color is
// confirmed, either from the palette (Enter on a swatch) or from the hex
// input (Enter while hex-focused, with a valid value).
type SelectedMsg struct {
	Color ansi.Color
}

// Update moves the cursor among Palette on Left/Right (clamped at the
// ends, not wrapping — Home/End style navigation isn't in scope here, so
// clamping keeps repeated Left/Right presses inert at the boundary
// instead of silently jumping to the opposite end), confirms the
// highlighted swatch on Enter, and toggles hex-input focus on Tab. While
// hex-focused, keys other than Tab are forwarded to HexInput.Update, and
// Enter attempts to parse HexInput.Value() as a "#RRGGBB" hex color: a
// valid value confirms it via SelectedMsg, an invalid one is a no-op (no
// panic, no Cmd). With Mouse on it also handles tui.MouseEvent (see
// Model.Mouse).
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev)
	}
	// A paste goes to the hex field: a bracketed paste, or the answer to its
	// own paste key (tui.PasteCopied).
	if pe, ok := msg.(tui.PasteEvent); ok && m.hexFocused {
		var cmd tui.Cmd
		m.HexInput, cmd = m.HexInput.Update(pe)
		return m, cmd
	}
	key, ok := msg.(tui.Key)
	if !ok {
		return m, nil
	}
	km := m.keys()

	if keymap.Matches(key, km.SwitchPane) {
		m.hexFocused = !m.hexFocused
		if m.hexFocused {
			cmd := m.HexInput.Focus()
			return m, cmd
		}
		m.HexInput.Blur()
		return m, nil
	}

	if m.hexFocused {
		if keymap.Matches(key, km.Accept) {
			color, ok := parseHex(m.HexInput.Value())
			if !ok {
				return m, nil
			}
			return m, func() tui.Msg { return SelectedMsg{Color: color} }
		}
		var cmd tui.Cmd
		m.HexInput, cmd = m.HexInput.Update(key)
		return m, cmd
	}

	switch {
	case keymap.Matches(key, km.Prev):
		if m.cursor > 0 {
			m.cursor--
		}
	case keymap.Matches(key, km.Next):
		if m.cursor < len(m.Palette)-1 {
			m.cursor++
		}
	case keymap.Matches(key, km.Accept):
		if len(m.Palette) == 0 {
			return m, nil
		}
		color := m.Palette[m.cursor]
		return m, func() tui.Msg { return SelectedMsg{Color: color} }
	}
	return m, nil
}

// updateMouse applies a mouse press inside Bounds when Mouse is on.
func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse || !m.Bounds.Contains(ev.X, ev.Y) || ev.Action != tui.MouseActionPress {
		return m, nil
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp:
		if !m.hexFocused && m.cursor > 0 {
			m.cursor--
		}
	case tui.MouseButtonWheelDown:
		if !m.hexFocused && m.cursor < len(m.Palette)-1 {
			m.cursor++
		}
	case tui.MouseButtonLeft:
		col, _ := m.Bounds.Local(ev.X, ev.Y)
		// Swatch i spans columns 3i and 3i+1; column 3i+2 is the gap.
		if i := col / 3; i < len(m.Palette) && col%3 != 2 {
			wasHex := m.hexFocused
			if wasHex {
				m.hexFocused = false
				m.HexInput.Blur()
			}
			if i == m.cursor && !wasHex {
				color := m.Palette[i]
				return m, func() tui.Msg { return SelectedMsg{Color: color} }
			}
			m.cursor = i
			return m, nil
		}
		if col >= 3*len(m.Palette) && !m.hexFocused {
			m.hexFocused = true
			cmd := m.HexInput.Focus()
			return m, cmd
		}
	}
	return m, nil
}

// View renders the palette swatches followed by the hex input. Each
// swatch is rendered with a single combined ansi.Style built for that
// swatch specifically (background color alone for a plain swatch,
// background + bold + inverse-text for the cursor swatch), then Render is
// called exactly once on the whole swatch block — never a base style
// Render wrapped inside a separate cursor-highlight Render.
func (m Model) View() string {
	var b strings.Builder
	for i, color := range m.Palette {
		style := ansi.NewStyle().Background(color)
		if i == m.cursor && !m.hexFocused {
			style = style.Bold().Foreground(m.themed().TextInverse)
		}
		b.WriteString(style.Render(swatch))
		if i < len(m.Palette)-1 {
			b.WriteString(" ")
		}
	}
	b.WriteString("  ")
	b.WriteString(m.HexInput.View())
	return b.String()
}

// parseHex parses s as "#RRGGBB" (case-insensitive), returning ok=false
// for anything else instead of panicking: wrong length, missing '#', or
// non-hex-digit characters.
func parseHex(s string) (ansi.RGB, bool) {
	if len(s) != 7 || s[0] != '#' {
		return ansi.RGB{}, false
	}
	r, err := strconv.ParseUint(s[1:3], 16, 8)
	if err != nil {
		return ansi.RGB{}, false
	}
	g, err := strconv.ParseUint(s[3:5], 16, 8)
	if err != nil {
		return ansi.RGB{}, false
	}
	bl, err := strconv.ParseUint(s[5:7], 16, 8)
	if err != nil {
		return ansi.RGB{}, false
	}
	return ansi.RGB{R: uint8(r), G: uint8(g), B: uint8(bl)}, true
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}
