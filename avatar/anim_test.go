package avatar_test

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/avatar"
	"github.com/ows4444/tui/motion"
)

func large(name string) avatar.Model {
	m := avatar.New(name)
	m.Width, m.Height = 24, 12
	return m
}

// eyeCells counts the cells drawn with the eye colour as their background,
// and returns the mean column and row of those cells.
func eyeCells(m avatar.Model) (n int, col, row float64) {
	_, eyes, _ := m.Colors()
	code := strings.TrimSuffix(strings.TrimPrefix(strings.SplitN(ansi.NewStyle().Background(eyes).Render("x"), "x", 2)[0], "\x1b["), "m")
	for y, line := range strings.Split(m.View(), "\n") {
		x := 0
		for _, part := range strings.Split(line, "\x1b[") {
			sgr, text, ok := strings.Cut(part, "m")
			if !ok {
				x += ansi.Width(part)
				continue
			}
			w := ansi.Width(text)
			if strings.Contains(sgr, code) {
				n += w
				col += float64(w) * (float64(x) + float64(w-1)/2)
				row += float64(w * y)
			}
			x += w
		}
	}
	if n > 0 {
		col, row = col/float64(n), row/float64(n)
	}
	return n, col, row
}

// A blink closes the eyes, reopens them and rests: three ticks, then no Cmd,
// and the View is the resting one again.
func TestBlinkPlaysOnceAndRests(t *testing.T) {
	m := large("alain00")
	rest := m.View()
	open, _, _ := eyeCells(m)
	if m.Blinking() {
		t.Fatal("a new avatar is blinking")
	}
	cmd := m.Blink()
	if cmd == nil || !m.Blinking() {
		t.Fatal("Blink did not start")
	}
	fewest := open
	steps := 0
	for cmd != nil {
		if n, _, _ := eyeCells(m); n < fewest {
			fewest = n
		}
		if steps++; steps > 10 {
			t.Fatal("the blink does not end")
		}
		m, cmd = m.Update(tui.RunCmd(context.Background(), cmd))
	}
	if steps != 3 {
		t.Errorf("a blink took %d ticks, want 3", steps)
	}
	if fewest >= open {
		t.Errorf("the eyes never closed: %d eye cells at rest, %d at the least", open, fewest)
	}
	if m.Blinking() || m.View() != rest {
		t.Error("the avatar did not return to rest")
	}
}

// A tick from a blink that was restarted is ignored, as is any other Msg.
func TestStaleBlinkTicksAreIgnored(t *testing.T) {
	m := large("alain00")
	first := m.Blink()
	stale := tui.RunCmd(context.Background(), first)
	second := m.Blink()
	before := m.View()
	n, cmd := m.Update(stale)
	if cmd != nil || n.View() != before {
		t.Error("a stale tick advanced the blink")
	}
	if n, cmd = m.Update(tui.Key{Type: tui.KeyEnter}); cmd != nil || n.View() != before {
		t.Error("a key advanced the blink")
	}
	if n, cmd = m.Update(tui.RunCmd(context.Background(), second)); cmd == nil || n.View() == before {
		t.Error("the current tick did not advance the blink")
	}
	rest := large("alain00")
	if n, cmd = rest.Update(stale); cmd != nil || n.Blinking() {
		t.Error("a tick started a blink on an avatar at rest")
	}
}

func TestBlinkUnderReducedMotionPlaysNothing(t *testing.T) {
	m := large("alain00")
	rest := m.View()
	m.Motion = motion.Reduced
	if cmd := m.Blink(); cmd != nil || m.Blinking() || m.View() != rest {
		t.Error("Blink under reduced motion should schedule nothing and leave the eyes open")
	}
}

// The eyes move toward the target, on both axes, and LookAt(0, 0) returns
// them to rest.
func TestLookAtTurnsTheEyesTowardTheTarget(t *testing.T) {
	for _, name := range []string{"alain00", "kasper", "mdawais"} {
		m := large(name)
		rest := m.View()
		_, col, row := eyeCells(m)
		look := func(dx, dy int) (float64, float64) {
			c := m
			c.LookAt(dx, dy)
			n, x, y := eyeCells(c)
			if n == 0 {
				t.Fatalf("%s looking (%d, %d) has no eyes", name, dx, dy)
			}
			return x, y
		}
		if x, _ := look(100, 0); x <= col {
			t.Errorf("%s: looking right moved the eyes from column %.2f to %.2f", name, col, x)
		}
		if x, _ := look(-100, 0); x >= col {
			t.Errorf("%s: looking left moved the eyes from column %.2f to %.2f", name, col, x)
		}
		if _, y := look(0, 100); y <= row {
			t.Errorf("%s: looking down moved the eyes from row %.2f to %.2f", name, row, y)
		}
		if _, y := look(0, -100); y >= row {
			t.Errorf("%s: looking up moved the eyes from row %.2f to %.2f", name, row, y)
		}
		m.LookAt(5, 5)
		m.LookAt(0, 0)
		if m.LookX != 0 || m.LookY != 0 || m.View() != rest {
			t.Errorf("%s: LookAt(0, 0) did not return the eyes to rest", name)
		}
	}
}

// A look is a direction with a length of at most 1, reached at one and a
// half avatar widths; values outside [-1, 1] set by hand are clamped.
func TestLookIsBounded(t *testing.T) {
	m := avatar.New("alain00")
	m.LookAt(1000, 0)
	if m.LookX != 1 || m.LookY != 0 {
		t.Errorf("a far target = (%v, %v)", m.LookX, m.LookY)
	}
	m.LookAt(6, 0) // half of the 12 columns a full look needs
	if math.Abs(m.LookX-0.5) > 1e-9 {
		t.Errorf("a target half way = %v", m.LookX)
	}
	m.LookAt(0, -3) // three rows are six columns
	if math.Abs(m.LookY+0.5) > 1e-9 || m.LookX != 0 {
		t.Errorf("a target three rows up = (%v, %v)", m.LookX, m.LookY)
	}
	m.LookAt(1000, 0)
	full := m.View()
	m.LookX = 50
	if m.View() != full {
		t.Error("LookX above 1 is not clamped")
	}
	m.LookX = math.NaN()
	if m.View() != avatar.New("alain00").View() {
		t.Error("a NaN look is not treated as rest")
	}
	zero := avatar.Model{Name: "alain00"}
	zero.LookAt(3, 0)
	if zero.LookX != 1 {
		t.Errorf("LookAt on a Model with no Width = %v", zero.LookX)
	}
}

// SVG is the figure at rest, whatever the eyes are doing.
func TestSVGIgnoresLookAndBlink(t *testing.T) {
	m := large("alain00")
	rest := m.SVG()
	m.LookAt(40, -9)
	m.Blink()
	if m.SVG() != rest {
		t.Error("SVG changed with the look or the blink")
	}
}

// A look never draws an eye outside the body: without a plate, no cell is an
// eye over nothing.
func TestEyesStayOnTheBody(t *testing.T) {
	for _, name := range names {
		for _, d := range [][2]int{{100, 0}, {-100, 0}, {0, 100}, {0, -100}, {100, 100}} {
			m := large(name)
			m.LookAt(d[0], d[1])
			_, eyes, _ := m.Colors()
			fg := strings.SplitN(ansi.NewStyle().Foreground(eyes).Render("x"), "x", 2)[0]
			if strings.Contains(m.View(), fg) {
				t.Errorf("%q looking %v draws an eye in the foreground, off the body", name, d)
			}
		}
	}
}
