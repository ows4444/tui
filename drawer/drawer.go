// Package drawer is a click/key-triggered overlay holding arbitrary
// (possibly multi-line) content, anchored to an edge of the base view
// (left/right/top/bottom) rather than centered over it (unlike
// dialog.Model) or anchored to a point (unlike popover.Model). Its
// open/dismiss lifecycle mirrors popover.Model/dialog.Model exactly. This
// is a static anchored panel unless SlideDuration is set, in which case SlideIn
// and SlideOut animate it with motion.Tween.
//
// # Sliding
//
// Set SlideDuration (and optionally Ease) and call SlideIn or SlideOut from
// Update, returning the Cmd they give back. The drawer moves through its own
// edge, and only the part that has come through is drawn. Its position comes
// from the time each tick carries, so it is where the clock says and the slide
// ends exactly one SlideDuration after the first tick. Enter and Esc slide it
// out, and DismissedMsg is delivered when a slide out finishes. Show and Hide
// stay immediate and cancel a slide. See the Example.
//
// # Reduced motion
//
// The slide is an animation, so it honours reduced motion: with Motion set to
// motion.Reduced, SlideIn and SlideOut complete at once, draw no intermediate
// frame and schedule no tick (SlideOut still delivers DismissedMsg). Take the
// value from motion.Detect(), which is Reduced when the NO_ANIMATION
// environment variable is set, or from the Program:
//
//	d.Motion = motion.Normal
//	if p.ReducedMotion() { // WithReducedMotion(true), NO_ANIMATION, or WithAccessible
//		d.Motion = motion.Reduced
//	}
package drawer

import (
	"math"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/focus"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
)

// Edge is the side of the base view a Drawer is anchored to.
type Edge int

const (
	// EdgeRight anchors the drawer to the right edge of base — the zero
	// value, matching termcn's own default.
	EdgeRight Edge = iota
	// EdgeLeft anchors the drawer to the left edge of base.
	EdgeLeft
	// EdgeTop anchors the drawer to the top edge of base.
	EdgeTop
	// EdgeBottom anchors the drawer to the bottom edge of base.
	EdgeBottom
)

// Model is an edge-anchored drawer. Render composites it over a base view
// when open; when closed, Render returns base unchanged, so a parent can
// unconditionally call it every frame without checking Open itself.
type Model struct {
	// Content holds arbitrary, possibly multi-line text rendered inside
	// the drawer's bordered box, unwrapped and unsplit (like
	// popover.Model's Content, unlike dialog.Model's Title/Message
	// split).
	Content string
	Theme   theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// Edge is the side of base the drawer is anchored to. The zero value
	// is EdgeRight.
	Edge Edge

	// Width is the drawer's fixed content width, used when Edge is
	// EdgeLeft or EdgeRight. Height is its fixed content height, used
	// when Edge is EdgeTop or EdgeBottom.
	Width, Height int

	// SlideDuration is how long SlideIn and SlideOut take to move the drawer
	// all the way in or out. Zero (the default) means no animation: the drawer
	// appears and disappears in one frame, exactly as it always has.
	SlideDuration time.Duration
	// Ease shapes the slide; the zero value is motion.Linear.
	Ease motion.Ease
	// Motion set to motion.Reduced skips the animation (see motion.Detect and
	// Program.ReducedMotion): SlideIn and SlideOut then complete at once.
	Motion motion.Preference

	// KeyMap holds the keys that dismiss the drawer. New fills it with
	// DefaultKeyMap; a Model built as a struct literal with a zero KeyMap
	// behaves as if it held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent while the drawer
	// is open: a left click outside the drawer dismisses it exactly as Esc
	// does (sliding it out when SlideDuration is set) and a click inside it is
	// swallowed. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle of the base view the drawer is rendered
	// over; the drawer's own rectangle is worked out from it as Render does,
	// fully slid in.
	Bounds hittest.Rect

	open bool

	// The slide in progress, if any; see slide.go.
	hidden       float64   // fraction of the drawer still outside its edge: 0 in place, 1 out
	sliding      bool      // a slide is running
	slideFrom    float64   // hidden at the start of the slide
	slideTo      float64   // hidden it is heading for: 0 in, 1 out
	slideStarted bool      // the first tick has fixed slideStart
	slideStart   time.Time // the time carried by the first tick
	gen          int       // bumped whenever a slide starts or is cancelled, so old ticks are ignored
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Dismiss keymap.Binding // close (or slide out) the drawer and deliver DismissedMsg
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

// Bindings returns the actions the drawer currently honours, with
// descriptions, for help text. A closed drawer honours none.
func (m Model) Bindings() []keymap.Binding {
	if !m.open {
		return nil
	}
	return []keymap.Binding{m.keys().Dismiss}
}

// New returns an already-open Model with Content, Theme: theme.Dark,
// Edge: EdgeRight and sane default Width/Height — the common case is
// showing a drawer immediately in response to some trigger, not
// constructing one ahead of time and opening it later (though Show/Hide
// support that too).
func New(content string) Model {
	return Model{
		Content: content,
		Theme:   theme.DarkTheme(),
		Edge:    EdgeRight,
		Width:   40,
		Height:  10,
		KeyMap:  DefaultKeyMap(),
		open:    true,
	}
}

// Open reports whether the drawer is currently shown.
func (m Model) Open() bool { return m.open }

// Show opens the drawer at once, cancelling any slide in progress.
func (m *Model) Show() { m.open, m.hidden = true, 0; m.cancelSlide() }

// Hide closes the drawer at once without dismissing it via Update (no
// DismissedMsg is sent), cancelling any slide in progress.
func (m *Model) Hide() { m.open, m.hidden = false, 0; m.cancelSlide() }

// Trap shows the drawer and traps focus in it: it pushes a focus scope of
// items focusable items (at least 1) over r and returns the new Ring. While
// the scope is open Tab and Shift+Tab cycle among those items only, never
// reaching the widgets behind the drawer. Call Release with the Ring when the
// drawer closes.
func (m *Model) Trap(r focus.Ring, items int) focus.Ring {
	m.open = true
	if items < 1 {
		items = 1
	}
	return r.Push(items)
}

// Release hides the drawer and pops the scope Trap pushed, returning the Ring
// with focus back on the widget that had it before the drawer opened. Call it
// when DismissedMsg arrives, then Sync the Ring so that widget is told it has
// focus again. On a Ring with no open scope it only hides the drawer.
func (m *Model) Release(r focus.Ring) focus.Ring {
	m.open = false
	return r.Pop()
}

// DismissedMsg is delivered (via the Cmd Update returns) when the
// drawer is dismissed with Enter or Esc, or when SlideOut finishes.
type DismissedMsg struct{}

func dismissed() tui.Msg { return DismissedMsg{} }

// Update dismisses the drawer on Enter or Esc while open; it's a no-op
// (including for Enter/Esc) once closed, so a parent doesn't need to
// guard forwarding to it on Model.Open(). With a SlideDuration set (and motion
// not reduced) Enter and Esc slide the drawer out instead of closing it at
// once. Update also advances a running slide on its ticks.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if t, ok := msg.(slideTickMsg); ok {
		return m.slideTick(t)
	}
	if !m.open {
		return m, nil
	}
	if ev, ok := msg.(tui.MouseEvent); ok {
		if !m.Mouse || ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress ||
			m.Rect().Contains(ev.X, ev.Y) {
			return m, nil
		}
		return m.dismiss()
	}
	if keymap.Matches(msg, m.keys().Dismiss) {
		return m.dismiss()
	}
	return m, nil
}

// dismiss closes the drawer (or slides it out when animated), as Esc does.
func (m Model) dismiss() (Model, tui.Cmd) {
	if m.animated() {
		cmd := m.SlideOut()
		return m, cmd
	}
	m.open, m.hidden = false, 0
	m.cancelSlide()
	return m, dismissed
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

// padToHeight appends blank lines to content until it has at least
// height lines, so a box built from it has a fixed row count for
// EdgeTop/EdgeBottom (layout.Box has no direct height knob, only Width).
func padToHeight(content string, height int) string {
	lines := strings.Split(content, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// Render composites the drawer, anchored to Edge, over base. If the
// drawer is closed, it returns base unchanged. While a slide is running only the
// part of the drawer that has come through its edge is drawn (see SlideIn).
//
// EdgeLeft/EdgeRight size the box to a fixed Width and place it flush
// against the left/right edge of base, vertically centered. EdgeTop/
// EdgeBottom size the box to a fixed Height and place it flush against
// the top/bottom edge of base, horizontally centered.
func (m Model) Render(base string) string {
	if !m.open {
		return base
	}
	baseW, baseH := extent(base)
	box, x, y, bw, bh := m.place(baseW, baseH)
	if h := math.Min(m.hidden, 1); h > 0 {
		return slideOverlay(base, box, x, y, bw, bh, baseW, baseH, m.Edge, h)
	}
	return layout.Overlay(base, box, x, y)
}

// place renders the drawer box and returns it with its offset and size inside
// a base of baseW x baseH, fully slid in.
func (m Model) place(baseW, baseH int) (box string, x, y, bw, bh int) {
	content := m.Content
	b := layout.NewBox().Border(m.themed().Border).BorderColor(m.themed().BorderColor).PaddingAll(1)

	switch m.Edge {
	case EdgeLeft, EdgeRight:
		b = b.Width(m.Width)
	case EdgeTop, EdgeBottom:
		content = padToHeight(content, m.Height)
	}

	box = b.Render(content)
	bw, bh = extent(box)

	switch m.Edge {
	case EdgeLeft:
		x = 0
		y = (baseH - bh) / 2
	case EdgeRight:
		x = baseW - bw
		y = (baseH - bh) / 2
	case EdgeTop:
		y = 0
		x = (baseW - bw) / 2
	case EdgeBottom:
		y = baseH - bh
		x = (baseW - bw) / 2
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return box, x, y, bw, bh
}

// Rect is the screen rectangle the drawer occupies, fully slid in, when
// rendered over a base view filling Bounds. It is the zero Rect while the
// drawer is closed.
func (m Model) Rect() hittest.Rect {
	if !m.open {
		return hittest.Rect{}
	}
	_, x, y, w, h := m.place(m.Bounds.W, m.Bounds.H)
	return hittest.Rect{X: m.Bounds.X + x, Y: m.Bounds.Y + y, W: w, H: h}
}

// Compile-time proof that Model satisfies tui.Overlay[Model] — see
// tui.Overlay's doc comment for what this contract means and why.
var _ tui.Overlay[Model] = Model{}
