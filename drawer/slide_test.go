package drawer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/motion"
)

var t0 = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

const d = 200 * time.Millisecond

// tickAt delivers the tick a running slide would receive at time at.
func tickAt(m Model, at time.Time) (Model, tui.Cmd) {
	return m.Update(slideTickMsg{gen: m.gen, at: at})
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func slider() Model {
	m := New("content")
	m.Hide()
	m.SlideDuration = d
	m.Ease = motion.OutQuad
	return m
}

// SlideIn opens the drawer at once (Open is true while it slides in) with
// nothing showing, then follows the eased curve as ticks arrive, reaching fully
// open exactly at t0+d.
func TestSlideInFollowsTheEasedCurveAndEndsExactlyAtTheDuration(t *testing.T) {
	m := slider()
	cmd := m.SlideIn()
	if cmd == nil {
		t.Fatal("SlideIn returned no Cmd to drive the animation")
	}
	if !m.Open() || m.shown() != 0 {
		t.Fatalf("right after SlideIn: Open=%v shown=%v, want true and 0", m.Open(), m.shown())
	}

	m, cmd = tickAt(m, t0) // the first tick is the origin
	if !near(m.shown(), 0) || cmd == nil {
		t.Errorf("at t0: shown=%v cmd=%v, want 0 and a next tick", m.shown(), cmd != nil)
	}
	m, cmd = tickAt(m, t0.Add(d/2))
	if want := motion.OutQuad(0.5); !near(m.shown(), want) || cmd == nil {
		t.Errorf("at t0+d/2: shown=%v, want %v (and a next tick)", m.shown(), want)
	}
	m, cmd = tickAt(m, t0.Add(d))
	if !near(m.shown(), 1) {
		t.Errorf("at t0+d: shown=%v, want 1", m.shown())
	}
	if cmd != nil {
		t.Error("a finished slide in scheduled another tick")
	}
	if m.sliding || !m.Open() {
		t.Errorf("after the slide in: sliding=%v Open=%v, want false and true", m.sliding, m.Open())
	}
}

// A tick past the end does not overshoot.
func TestSlideNeverOvershoots(t *testing.T) {
	m := slider()
	m.SlideIn()
	m, _ = tickAt(m, t0)
	m, _ = tickAt(m, t0.Add(10*d))
	if !near(m.shown(), 1) {
		t.Errorf("shown = %v after a late tick, want 1", m.shown())
	}
}

// Reduced motion, or no duration, completes at once with no ticks.
func TestReducedMotionAndZeroDurationCompleteImmediately(t *testing.T) {
	for name, mut := range map[string]func(*Model){
		"reduced":       func(m *Model) { m.Motion = motion.Reduced },
		"zero duration": func(m *Model) { m.SlideDuration = 0 },
	} {
		m := slider()
		mut(&m)
		if cmd := m.SlideIn(); cmd != nil || !m.Open() || m.shown() != 1 || m.sliding {
			t.Errorf("%s: SlideIn cmd=%v Open=%v shown=%v sliding=%v, want nil/true/1/false", name, cmd != nil, m.Open(), m.shown(), m.sliding)
		}
		cmd := m.SlideOut()
		if m.Open() || m.sliding {
			t.Errorf("%s: after SlideOut Open=%v sliding=%v, want false", name, m.Open(), m.sliding)
		}
		if cmd == nil {
			t.Fatalf("%s: SlideOut returned no Cmd; the dismissal must still be reported", name)
		}
		if _, ok := cmd().(DismissedMsg); !ok {
			t.Errorf("%s: the SlideOut Cmd did not yield DismissedMsg", name)
		}
	}
}

// A slide out finishes with Open false and DismissedMsg once; stale ticks after
// that do nothing.
func TestSlideOutFinishesClosedAndDismissesOnce(t *testing.T) {
	m := New("content")
	m.SlideDuration, m.Ease = d, motion.Linear
	cmd := m.SlideOut()
	if cmd == nil || !m.Open() {
		t.Fatalf("SlideOut: cmd=%v Open=%v, want a Cmd and still open while it slides", cmd != nil, m.Open())
	}
	gen := m.gen
	m, _ = tickAt(m, t0)
	m, cmd = tickAt(m, t0.Add(d/2))
	if !near(m.shown(), 0.5) || !m.Open() || cmd == nil {
		t.Fatalf("halfway: shown=%v Open=%v", m.shown(), m.Open())
	}
	m, cmd = tickAt(m, t0.Add(d))
	if m.Open() || m.sliding {
		t.Fatalf("after the slide out: Open=%v sliding=%v, want closed", m.Open(), m.sliding)
	}
	if cmd == nil {
		t.Fatal("the finished slide out delivered no DismissedMsg")
	}
	if _, ok := cmd().(DismissedMsg); !ok {
		t.Errorf("the final Cmd yields %T, want DismissedMsg", cmd())
	}
	// A stale tick from the finished slide is ignored: no second dismissal.
	m2, cmd2 := m.Update(slideTickMsg{gen: gen, at: t0.Add(2 * d)})
	if cmd2 != nil || m2.Open() {
		t.Errorf("a stale tick after the end produced cmd=%v Open=%v", cmd2 != nil, m2.Open())
	}
}

// Reversing mid-slide continues from where the drawer is: no jump.
func TestReversingMidSlideDoesNotJump(t *testing.T) {
	m := slider()
	m.SlideIn()
	m, _ = tickAt(m, t0)
	m, _ = tickAt(m, t0.Add(d/2))
	before := m.shown()
	if before <= 0 || before >= 1 {
		t.Fatalf("setup: shown=%v, want strictly between 0 and 1", before)
	}
	if cmd := m.SlideOut(); cmd == nil {
		t.Fatal("reversing returned no Cmd")
	}
	if !near(m.shown(), before) {
		t.Errorf("SlideOut jumped from %v to %v", before, m.shown())
	}
	m, _ = tickAt(m, t0.Add(d)) // origin of the reversed slide
	if !near(m.shown(), before) {
		t.Errorf("the first tick of the reversed slide jumped from %v to %v", before, m.shown())
	}
	// It now moves toward closed and monotonically so.
	prev := m.shown()
	for i := 1; i <= 8 && m.Open(); i++ {
		m, _ = tickAt(m, t0.Add(d).Add(time.Duration(i)*d/8))
		if m.Open() && m.shown() > prev+1e-9 {
			t.Fatalf("shown rose from %v to %v while sliding out", prev, m.shown())
		}
		prev = m.shown()
	}
	if m.Open() {
		t.Errorf("Open is still true after the reversed slide finished (shown=%v)", m.shown())
	}
}

// Reversing takes only as long as the distance left, not the full duration.
func TestReversalDurationScalesWithTheRemainingDistance(t *testing.T) {
	m := slider()
	m.Ease = motion.Linear
	m.SlideIn()
	m, _ = tickAt(m, t0)
	m, _ = tickAt(m, t0.Add(d/4)) // a quarter of the way in
	m.SlideOut()
	m, _ = tickAt(m, t0.Add(d))
	m, _ = tickAt(m, t0.Add(d).Add(d/4)) // a quarter of the duration is enough to go back a quarter
	if m.Open() {
		t.Errorf("a quarter-way slide did not finish after a quarter of the duration (shown=%v)", m.shown())
	}
}

// SlideIn on an open drawer, and SlideOut on a closed one, do nothing.
func TestSlideIsANoOpWhenAlreadyThere(t *testing.T) {
	open := New("content")
	open.SlideDuration = d
	if cmd := open.SlideIn(); cmd != nil || open.sliding {
		t.Errorf("SlideIn on an open drawer: cmd=%v sliding=%v", cmd != nil, open.sliding)
	}
	closed := slider()
	if cmd := closed.SlideOut(); cmd != nil || closed.sliding || closed.Open() {
		t.Errorf("SlideOut on a closed drawer: cmd=%v sliding=%v Open=%v", cmd != nil, closed.sliding, closed.Open())
	}
	// Asking for the direction it is already moving is also a no-op.
	m := slider()
	m.SlideIn()
	gen := m.gen
	if cmd := m.SlideIn(); cmd != nil || m.gen != gen {
		t.Errorf("a second SlideIn restarted the slide (cmd=%v, gen %d -> %d)", cmd != nil, gen, m.gen)
	}
}

// Show and Hide are immediate and cancel a slide in progress.
func TestShowAndHideCancelASlide(t *testing.T) {
	m := slider()
	m.SlideIn()
	m, _ = tickAt(m, t0)
	m.Hide()
	if m.Open() || m.sliding {
		t.Errorf("after Hide: Open=%v sliding=%v", m.Open(), m.sliding)
	}
	m2, cmd := m.Update(slideTickMsg{gen: m.gen - 1, at: t0.Add(d)})
	if cmd != nil || m2.Open() {
		t.Error("a tick from the cancelled slide changed the drawer")
	}
	m.SlideIn()
	m.Show()
	if !m.Open() || m.sliding || m.shown() != 1 {
		t.Errorf("after Show: Open=%v sliding=%v shown=%v, want true/false/1", m.Open(), m.sliding, m.shown())
	}
}

// With a slide configured, Enter and Esc slide the drawer out instead of
// closing it at once; without one they close it at once, as always.
func TestDismissKeysSlideOutOnlyWhenSlideIsConfigured(t *testing.T) {
	m := New("content")
	m.SlideDuration = d
	next, cmd := m.Update(tui.Key{Type: tui.KeyEsc})
	if !next.Open() || !next.sliding || cmd == nil {
		t.Errorf("Esc with a slide: Open=%v sliding=%v cmd=%v, want it to start sliding out", next.Open(), next.sliding, cmd != nil)
	}
	plain := New("content")
	next, cmd = plain.Update(tui.Key{Type: tui.KeyEnter})
	if next.Open() || next.sliding {
		t.Errorf("Enter without a slide: Open=%v sliding=%v, want closed at once", next.Open(), next.sliding)
	}
	if _, ok := cmd().(DismissedMsg); !ok {
		t.Error("Enter without a slide did not yield DismissedMsg")
	}
	reduced := New("content")
	reduced.SlideDuration, reduced.Motion = d, motion.Reduced
	next, cmd = reduced.Update(tui.Key{Type: tui.KeyEsc})
	if next.Open() {
		t.Error("Esc under reduced motion should close at once")
	}
	if _, ok := cmd().(DismissedMsg); !ok {
		t.Error("Esc under reduced motion did not yield DismissedMsg")
	}
}

// The package documentation says how reduced motion makes the slide immediate,
// so the two ways of asking for it stay written down.
func TestPackageDocExplainsReducedMotion(t *testing.T) {
	files := parseSources(t, ".", parser.ParseComments)
	doc := ""
	for _, f := range files {
		if f.Doc != nil {
			doc += f.Doc.Text()
		}
	}
	for _, want := range []string{"Reduced motion", "NO_ANIMATION", "motion.Reduced", "WithReducedMotion", "at once", "SlideDuration"} {
		if !strings.Contains(doc, want) {
			t.Errorf("the package doc does not mention %q", want)
		}
	}
}

// parseSources parses the non-test Go files in dir. It replaces
// parser.ParseDir, deprecated since Go 1.25; these checks read declarations
// and comments only, so the build tags ParseDir ignored do not matter here.
func parseSources(t *testing.T, dir string, mode parser.Mode) []*ast.File {
	t.Helper()
	names, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, mode)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	return files
}
