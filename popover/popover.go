// Package popover is a click/key-triggered overlay holding arbitrary
// (possibly multi-line) content, anchored near a given point rather than
// centered over the whole base — unlike dialog.Model, which always
// centers. Its open/dismiss lifecycle mirrors dialog.Model exactly.
package popover

import (
	"strings"

	"github.com/ows4444/tui/focus"
	"github.com/ows4444/tui/hittest"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// Model is an anchored popover. Render composites it over a base view
// when open; when closed, Render returns base unchanged, so a parent can
// unconditionally call it every frame without checking Open itself.
type Model struct {
	// Content holds arbitrary, possibly multi-line text rendered inside
	// the popover's bordered box, unwrapped and unsplit (unlike
	// dialog.Model's Title/Message split).
	Content string
	Theme   theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// AnchorX, AnchorY is the point the popover is placed near. Default
	// placement is below and right of the anchor, matching
	// widgets.TooltipOverlay; Render clamps/flips it to stay within base.
	AnchorX, AnchorY int

	// KeyMap holds the keys that dismiss the popover. New fills it with
	// DefaultKeyMap; a Model built as a struct literal with a zero KeyMap
	// behaves as if it held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent while the popover
	// is open: a left click outside the popover dismisses it (DismissedMsg, as
	// Esc does) and a click inside it is swallowed. Off (the default) ignores
	// the mouse.
	Mouse bool
	// Bounds is the screen rectangle of the base view the popover is rendered
	// over; the popover's own rectangle is worked out from it as Render does.
	Bounds hittest.Rect

	open bool
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Dismiss keymap.Binding // close the popover and deliver DismissedMsg
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

// Bindings returns the actions the popover currently honours, with
// descriptions, for help text. A closed popover honours none.
func (m Model) Bindings() []keymap.Binding {
	if !m.open {
		return nil
	}
	return []keymap.Binding{m.keys().Dismiss}
}

// New returns an already-open Model anchored at (anchorX, anchorY) — the
// common case is showing a popover immediately in response to some
// trigger, not constructing one ahead of time and opening it later
// (though Show/Hide support that too).
func New(content string, anchorX, anchorY int) Model {
	return Model{Content: content, Theme: theme.DarkTheme(), AnchorX: anchorX, AnchorY: anchorY, KeyMap: DefaultKeyMap(), open: true}
}

// Open reports whether the popover is currently shown.
func (m Model) Open() bool { return m.open }

// Show opens the popover.
func (m *Model) Show() { m.open = true }

// Hide closes the popover without dismissing it via Update (no
// DismissedMsg is sent).
func (m *Model) Hide() { m.open = false }

// Trap shows the popover and traps focus in it: it pushes a focus scope of
// items focusable items (at least 1) over r and returns the new Ring. While
// the scope is open Tab and Shift+Tab cycle among those items only, never
// reaching the widgets behind the popover. Call Release with the Ring when the
// popover closes.
func (m *Model) Trap(r focus.Ring, items int) focus.Ring {
	m.open = true
	if items < 1 {
		items = 1
	}
	return r.Push(items)
}

// Release hides the popover and pops the scope Trap pushed, returning the Ring
// with focus back on the widget that had it before the popover opened. Call it
// when DismissedMsg arrives, then Sync the Ring so that widget is told it has
// focus again. On a Ring with no open scope it only hides the popover.
func (m *Model) Release(r focus.Ring) focus.Ring {
	m.open = false
	return r.Pop()
}

// DismissedMsg is delivered (via the Cmd Update returns) when the
// popover is dismissed with Enter or Esc.
type DismissedMsg struct{}

// Update dismisses the popover on Enter or Esc while open; it's a no-op
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

// extent returns s's rendered width (the widest line) and height (line
// count).
func extent(s string) (w, h int) {
	lines := strings.Split(s, "\n")
	h = len(lines)
	for _, line := range lines {
		if lw := ansi.Width(line); lw > w {
			w = lw
		}
	}
	return w, h
}

// place renders the popover box and returns it with its offset and size
// inside a base of baseW x baseH.
func (m Model) place(baseW, baseH int) (box string, x, y, bw, bh int) {
	box = layout.NewBox().Border(m.themed().Border).BorderColor(m.themed().BorderColor).PaddingAll(m.themed().ResolvedSpacing().S).Render(m.Content)
	bw, bh = extent(box)

	x = m.AnchorX
	if x+bw > baseW {
		x = baseW - bw
	}
	if x < 0 {
		x = 0
	}

	y = m.AnchorY + 1
	if y+bh > baseH {
		y = m.AnchorY - bh
	}
	if y < 0 {
		y = 0
	}
	return box, x, y, bw, bh
}

// Rect is the screen rectangle the popover occupies when rendered over a base
// view filling Bounds. It is the zero Rect while the popover is closed.
func (m Model) Rect() hittest.Rect {
	if !m.open {
		return hittest.Rect{}
	}
	_, x, y, w, h := m.place(m.Bounds.W, m.Bounds.H)
	return hittest.Rect{X: m.Bounds.X + x, Y: m.Bounds.Y + y, W: w, H: h}
}

// Render composites the popover, anchored near (AnchorX, AnchorY), over
// base. If the popover is closed, it returns base unchanged.
//
// Default placement is below and right-aligned to the anchor: the
// popover's top-left corner starts at (AnchorX, AnchorY+1). If that
// would push the popover's right edge past base's rendered width, it's
// shifted left just enough to stay within bounds. If placing it below
// the anchor would push it past base's last line, it's placed above the
// anchor instead — the same semantics as widgets.TooltipOverlay.
func (m Model) Render(base string) string {
	if !m.open {
		return base
	}
	baseW, baseH := extent(base)
	box, x, y, _, _ := m.place(baseW, baseH)
	return layout.Overlay(base, box, x, y)
}

// Compile-time proof that Model satisfies tui.Overlay[Model] — see
// tui.Overlay's doc comment for what this contract means and why.
var _ tui.Overlay[Model] = Model{}
