// Package faces is a catalog of 50 animated characters drawn in Braille
// dots, and a widget that plays them. Each character is a solid head with
// its face cut out, looping through 4, 8 or 12 frames, at a Small (6x3
// cell head) or Large (10x5 cell head) Size. All frames at one Size are
// the same block of cells, so swapping faces never moves the layout.
//
// The characters are ready-made and picked by index or name; For and
// IndexFor pick one for any string, the same one every time. To give any
// string a picture of its own, with a figure and a colour derived from it,
// use package avatar.
//
// Stability: experimental. Its API may change in any minor release.
package faces

import (
	"fmt"
	"math/bits"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
)

// Model plays one face's animation, using the same self-rescheduling
// tui.Tick pattern as spinner: a tick only reschedules while running, so
// Stop needs no cancellation. View works before Start (it shows frame 0),
// which is the static rendering for callers honouring reduced motion.
type Model struct {
	// Size is Small or Large.
	Size Size
	// Interval overrides the face's own pace when non-zero.
	Interval time.Duration
	Theme    theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// ShowLabel adds a "Name · anim · frame/total" line under the sprite.
	ShowLabel bool
	// Motion is the reduced-motion preference. The zero value is
	// motion.Normal, the behaviour before this field existed. With
	// motion.Reduced, Start and PlayOnce schedule no tick and leave the face on its
	// current frame, not running.
	Motion motion.Preference

	face    int
	frame   int
	running bool
	gen     int // bumped by Start so a stale tick from an earlier run is ignored
	once    bool
}

// New returns a stopped Model showing the first face.
func New() Model { return Model{Theme: theme.DarkTheme()} }

type tickMsg struct{ gen int }

func (m Model) delay() time.Duration {
	if m.Interval > 0 {
		return m.Interval
	}
	return m.Face().Interval
}

func (m Model) tick() tui.Cmd {
	gen := m.gen
	return tui.FromCtx(motion.After(m.delay(), func(time.Time) tui.Msg { return tickMsg{gen: gen} }))
}

func (m Model) frames() []string { return m.Face().Frames(m.Size) }

// Face returns the face being played.
func (m Model) Face() Face { return Get(m.face) }

// Index returns the catalog index of the face being played.
func (m Model) Index() int { return ((m.face % Count()) + Count()) % Count() }

// Frame returns the current frame number within the face's loop.
func (m Model) Frame() int { return m.frame }

// Running reports whether the animation is playing.
func (m Model) Running() bool { return m.running }

// Set switches to face i (wrapping) and restarts its loop. If the model is
// running, the pending tick keeps going at the new face's pace.
func (m *Model) Set(i int) {
	m.face = ((i % Count()) + Count()) % Count()
	m.frame = 0
}

// Next steps to the next face in the catalog, wrapping at the end.
func (m *Model) Next() { m.Set(m.Index() + 1) }

// Prev steps to the previous face in the catalog, wrapping at the start.
func (m *Model) Prev() { m.Set(m.Index() - 1) }

// Start begins playing; return the Cmd from your own Init or Update.
// Restarting is safe at any moment: ticks left over from before are ignored.
func (m *Model) Start() tui.Cmd {
	m.once = false
	m.gen++
	if m.Motion.Reduced() {
		m.running = false
		return nil
	}
	m.running = true
	return m.tick()
}

// PlayOnce plays the current face's loop a single time from its first
// frame and then rests there: the click-to-animate behaviour. Return the
// Cmd from your Update. Calling it again mid-loop restarts the loop.
func (m *Model) PlayOnce() tui.Cmd {
	m.frame = 0
	cmd := m.Start()
	m.once = true
	return cmd
}

// Stop pauses on the current frame.
func (m *Model) Stop() { m.running, m.once = false, false }

// Update advances one frame per tick; it ignores every other Msg.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if t, ok := msg.(tickMsg); !ok || !m.running || t.gen != m.gen {
		return m, nil
	}
	m.frame = (m.frame + 1) % len(m.frames())
	if m.once && m.frame == 0 {
		m.running, m.once = false, false // the single loop is done
		return m, nil
	}
	return m, m.tick()
}

// View renders the current frame in the theme's primary colour. Under an ASCII
// glyph set (theme.DarkTheme().ASCII()) the braille dots are drawn as a density ramp of
// ASCII characters instead, one per 2x4 cell, so the face keeps its shape and
// size on a terminal that cannot show braille.
func (m Model) View() string {
	f, frames := m.Face(), m.frames()
	frame := frames[m.frame%len(frames)]
	if g := m.themed().GlyphSet(); g.ASCII() {
		frame = asciiFrame(frame, g.Shades)
	}
	body := ansi.NewStyle().Foreground(m.themed().Primary).Render(frame)
	if !m.ShowLabel {
		return body
	}
	dot := m.themed().GlyphSet().Middot
	label := fmt.Sprintf("%s %s %s %s %d/%d", f.Name, dot, f.Anim, dot, m.frame%len(frames)+1, len(frames))
	w, _ := m.Size.Cells()
	pad := (w - ansi.Width(label)) / 2
	if pad < 0 {
		pad = 0
	}
	return body + "\n" + ansi.NewStyle().Faint().Render(strings.Repeat(" ", pad)+label)
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// asciiFrame redraws a braille frame with the characters of shades, a ramp
// from light to dark: a cell with no dots is a space, and a cell with n dots
// (1 to 8) takes the shade at (n-1)*len(shades)/8, so a fuller cell is never
// lighter. Anything that is not a braille pattern (the frame's spaces and
// newlines) is kept. It is the rule chart.Gauge uses for its braille arc.
func asciiFrame(frame, shades string) string {
	ramp := []rune(shades)
	var b strings.Builder
	for _, r := range frame {
		if r < 0x2800 || r > 0x28FF {
			b.WriteRune(r)
			continue
		}
		n := bits.OnesCount8(uint8(r - 0x2800))
		if n == 0 {
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(ramp[min((n-1)*len(ramp)/8, len(ramp)-1)])
	}
	return b.String()
}
