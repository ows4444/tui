package drawer

import (
	"math"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
)

// slideInterval is how often a running slide asks for a tick. The drawer's
// position comes from the time each tick carries, not from counting ticks, so
// a late tick lands where the clock says and the slide always ends on time.
const slideInterval = 16 * time.Millisecond

// slideTickMsg drives a slide. It is internal: never forwarded to the app's
// own Update; a drawer only reacts to ticks of its current slide (gen).
type slideTickMsg struct {
	gen int
	at  time.Time
}

// animated reports whether SlideIn and SlideOut animate: a SlideDuration is
// set and motion is not reduced.
func (m Model) animated() bool { return m.SlideDuration > 0 && !m.Motion.Reduced() }

// shown is how far the drawer is in place: 1 when fully in, 0 when it has
// slid all the way out of its edge.
func (m Model) shown() float64 { return 1 - m.hidden }

// cancelSlide stops a running slide; its late ticks are then ignored.
func (m *Model) cancelSlide() {
	m.sliding = false
	m.slideStarted = false
	m.gen++
}

// SlideIn opens the drawer and slides it in from its edge over SlideDuration,
// shaped by Ease, and returns the Cmd that drives the animation: run it (return
// it from Update). Open reports true from the start. With no SlideDuration, or
// with Motion reduced, the drawer is in place at once and the Cmd is nil.
// Called on an open, resting drawer it does nothing; called during a slide out
// it reverses from where the drawer is, without a jump.
func (m *Model) SlideIn() tui.Cmd {
	if !m.animated() {
		m.Show()
		return nil
	}
	if m.open && !m.sliding && m.hidden == 0 {
		return nil
	}
	return m.startSlide(0)
}

// SlideOut slides the drawer out through its edge over SlideDuration, shaped by
// Ease, and returns the Cmd that drives the animation. When the slide finishes
// the drawer is closed (Open reports false) and a DismissedMsg is delivered
// once. With no SlideDuration, or with Motion reduced, the drawer closes at
// once and the Cmd only delivers the DismissedMsg. Called on a closed drawer it
// does nothing; called during a slide in it reverses from where the drawer is,
// without a jump.
func (m *Model) SlideOut() tui.Cmd {
	if !m.open {
		return nil
	}
	if !m.animated() {
		m.Hide()
		return dismissed
	}
	return m.startSlide(1)
}

// startSlide begins (or reverses into) a slide toward hidden == to.
func (m *Model) startSlide(to float64) tui.Cmd {
	if m.sliding && m.slideTo == to {
		return nil // already heading there
	}
	from := m.hidden
	if !m.open {
		from = 1
	}
	m.open = true
	m.hidden = from
	m.sliding = true
	m.slideFrom, m.slideTo = from, to
	m.slideStarted = false
	m.gen++
	return m.tickCmd()
}

func (m Model) tickCmd() tui.Cmd {
	gen := m.gen
	return tui.FromCtx(motion.After(slideInterval, func(at time.Time) tui.Msg { return slideTickMsg{gen: gen, at: at} }))
}

// tween is the running slide as a motion.Tween. Its duration is SlideDuration
// scaled by the distance still to cover, so a reversal near the end is quick.
func (m Model) tween() motion.Tween {
	dist := m.slideTo - m.slideFrom
	if dist < 0 {
		dist = -dist
	}
	return motion.Tween{
		From:     m.slideFrom,
		To:       m.slideTo,
		Duration: time.Duration(float64(m.SlideDuration) * dist),
		Ease:     m.Ease,
		Motion:   m.Motion,
	}
}

// slideTick advances the running slide to the time the tick carries. The first
// tick of a slide is its origin. A tick from a cancelled or replaced slide does
// nothing.
func (m Model) slideTick(t slideTickMsg) (Model, tui.Cmd) {
	if !m.sliding || t.gen != m.gen {
		return m, nil
	}
	if !m.slideStarted {
		m.slideStart, m.slideStarted = t.at, true
	}
	tw := m.tween()
	elapsed := t.at.Sub(m.slideStart)
	m.hidden = tw.At(elapsed)
	if !tw.Done(elapsed) {
		return m, m.tickCmd()
	}
	m.sliding, m.slideStarted = false, false
	m.hidden = m.slideTo
	if m.slideTo >= 1 { // slid out: closed
		m.open, m.hidden = false, 0
		return m, dismissed
	}
	return m, nil
}

// slideOverlay draws the drawer's box while it is partly outside its edge.
// (x, y) is where the box sits at rest, bw x bh its size, and hidden in (0, 1]
// the fraction of it still outside: the box is shifted toward its own edge by
// that fraction of its extent along the slide, and what lies outside the base
// is cut away, so the result keeps the base's rows and width. At hidden 1 the
// box is entirely outside and base is returned as it is.
func slideOverlay(base, box string, x, y, bw, bh, baseW, baseH int, edge Edge, hidden float64) string {
	along := bw
	if edge == EdgeTop || edge == EdgeBottom {
		along = bh
	}
	off := int(math.Round(hidden * float64(along)))
	switch edge {
	case EdgeRight:
		x += off
	case EdgeLeft:
		x -= off
	case EdgeBottom:
		y += off
	case EdgeTop:
		y -= off
	}

	lines := strings.Split(box, "\n")
	if y < 0 { // the rows above the base are outside it
		if -y >= len(lines) {
			return base
		}
		lines, y = lines[-y:], 0
	}
	if x < 0 { // the columns left of the base are outside it
		for i, l := range lines {
			lines[i] = ansi.TrimLeftWidth(l, -x)
		}
		x = 0
	}
	room := baseW - x // the columns from x to the base's right edge
	if room <= 0 || y >= baseH {
		return base
	}
	visible := false
	for i, l := range lines {
		if ansi.Width(l) > room {
			l = ansi.Truncate(l, room)
		}
		lines[i] = l
		if l != "" {
			visible = true
		}
	}
	if !visible {
		return base
	}
	return layout.Overlay(base, strings.Join(lines, "\n"), x, y)
}
