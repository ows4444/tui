// Package dialog is a modal overlay — InkUI's "Dialog": a bordered box
// with a title and message, composited on top of the rest of the screen
// via layout.Overlay rather than replacing it, dismissed with Enter/Esc.
package dialog

import (
	"github.com/ows4444/tui/focus"
	"github.com/ows4444/tui/keymap"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// Model is a centered modal dialog. Render composites it over a base view
// when open; when closed, Render returns base unchanged, so a parent can
// unconditionally call it every frame without checking Open itself.
type Model struct {
	Title   string
	Message string
	Theme   theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// Raw, when true, draws Title and Message unchanged. By default it is sanitised
	// (ansi.Sanitize) so untrusted text cannot carry terminal escape
	// sequences.
	Raw bool
	// Width constrains Message to word-wrap within it (Title is never
	// wrapped — dialog titles are expected to be short). 0 means no
	// wrapping: Message renders as-is, honoring only its own embedded
	// newlines. A Message wider than the base it's rendered over will
	// make the dialog box itself wider than its base, since Overlay
	// deliberately never clips an overlay to fit — leaving Width unset
	// for anything but a short, controlled message is asking for that.
	Width int

	// KeyMap holds the keys that dismiss the dialog. New fills it with
	// DefaultKeyMap; a Model built as a struct literal with a zero KeyMap
	// behaves as if it held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent while the dialog is
	// open: a left click outside the dialog box dismisses it (DismissedMsg, as
	// Esc does), and every mouse event is swallowed so nothing reaches the
	// widgets behind a modal. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle of the base view the dialog is rendered
	// over; the box's own rectangle is worked out from it as Render does.
	Bounds hittest.Rect

	open bool
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Dismiss keymap.Binding // close the dialog and deliver DismissedMsg
}

// DefaultKeyMap returns the keys a Model used before KeyMap existed.
func DefaultKeyMap() KeyMap {
	return KeyMap{Dismiss: keymap.NewBinding("dismiss", "enter", "esc")}
}

func (m Model) keys() KeyMap {
	if len(m.KeyMap.Dismiss.Keys) == 0 {
		return DefaultKeyMap()
	}
	return m.KeyMap
}

// Bindings returns the actions the dialog currently honours, with
// descriptions, for help text. A closed dialog honours none.
func (m Model) Bindings() []keymap.Binding {
	if !m.open {
		return nil
	}
	return []keymap.Binding{m.keys().Dismiss}
}

// New returns an already-open Model — the common case is showing a dialog
// immediately in response to some event, not constructing one ahead of
// time and opening it later (though Show/Hide support that too).
func New(title, message string) Model {
	return Model{Title: title, Message: message, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap(), open: true}
}

// Open reports whether the dialog is currently shown.
func (m Model) Open() bool { return m.open }

// Show opens the dialog.
func (m *Model) Show() { m.open = true }

// Hide closes the dialog without dismissing it via Update (no DismissedMsg
// is sent).
func (m *Model) Hide() { m.open = false }

// Trap shows the dialog and traps focus in it: it pushes a focus scope of
// items focusable items (at least 1) over r and returns the new Ring. While
// the scope is open Tab and Shift+Tab cycle among those items only, never
// reaching the widgets behind the dialog. Call Release with the Ring when the
// dialog closes.
func (m *Model) Trap(r focus.Ring, items int) focus.Ring {
	m.open = true
	if items < 1 {
		items = 1
	}
	return r.Push(items)
}

// Release hides the dialog and pops the scope Trap pushed, returning the Ring
// with focus back on the widget that had it before the dialog opened. Call it
// when DismissedMsg arrives, then Sync the Ring so that widget is told it has
// focus again. On a Ring with no open scope it only hides the dialog.
func (m *Model) Release(r focus.Ring) focus.Ring {
	m.open = false
	return r.Pop()
}

// DismissedMsg is delivered (via the Cmd Update returns) when the dialog
// is dismissed with Enter or Esc.
type DismissedMsg struct{}

// Update dismisses the dialog on Enter or Esc while open; it's a no-op
// (including for Enter/Esc) once closed, so a parent doesn't need to
// guard forwarding to it on Model.Open().
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if !m.open {
		return m, nil
	}
	if ev, ok := msg.(tui.MouseEvent); ok {
		if m.Mouse && ev.Button == tui.MouseButtonLeft && ev.Action == tui.MouseActionPress &&
			!m.Rect().Contains(ev.X, ev.Y) {
			m.open = false
			return m, func() tui.Msg { return DismissedMsg{} }
		}
		return m, nil
	}
	if keymap.Matches(msg, m.keys().Dismiss) {
		m.open = false
		return m, func() tui.Msg { return DismissedMsg{} }
	}
	return m, nil
}

// build renders the dialog box and returns it with its size.
func (m Model) build() (box string, w, h int) {
	titleStyled := m.themed().ResolvedTypography().H2.Render(ansi.Clean(m.Raw, m.Title))
	body := titleStyled
	if m.Message != "" {
		msg := ansi.Clean(m.Raw, m.Message)
		if m.Width > 0 {
			msg = ansi.WrapStyled(msg, m.Width)
		}
		body += "\n\n" + msg
	}
	boxBuilder := layout.NewBox().Border(m.themed().Border).BorderColor(m.themed().BorderColor).PaddingAll(1)
	if m.Width > 0 {
		boxBuilder = boxBuilder.Width(m.Width)
	}
	box = boxBuilder.Render(body)
	lines := strings.Split(box, "\n")
	return box, ansi.Width(lines[0]), len(lines)
}

// center is the offset that centers a box inside a base, never negative.
func center(boxW, boxH, baseW, baseH int) (x, y int) {
	x, y = (baseW-boxW)/2, (baseH-boxH)/2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}

// Rect is the screen rectangle the dialog box occupies when rendered over a
// base view filling Bounds. It is the zero Rect while the dialog is closed.
func (m Model) Rect() hittest.Rect {
	if !m.open {
		return hittest.Rect{}
	}
	_, w, h := m.build()
	x, y := center(w, h, m.Bounds.W, m.Bounds.H)
	return hittest.Rect{X: m.Bounds.X + x, Y: m.Bounds.Y + y, W: w, H: h}
}

// Render composites the dialog, centered, over base. If the dialog is
// closed, it returns base unchanged.
func (m Model) Render(base string) string {
	if !m.open {
		return base
	}

	baseLines := strings.Split(base, "\n")
	baseWidth := 0
	for _, l := range baseLines {
		if w := ansi.Width(l); w > baseWidth {
			baseWidth = w
		}
	}
	box, boxWidth, boxHeight := m.build()
	x, y := center(boxWidth, boxHeight, baseWidth, len(baseLines))
	return layout.Overlay(base, box, x, y)
}

// Compile-time proof that Model satisfies tui.Overlay[Model] — see
// tui.Overlay's doc comment for what this contract means and why.
var _ tui.Overlay[Model] = Model{}
