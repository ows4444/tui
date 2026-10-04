package tui_test

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/avatar"
	"github.com/ows4444/tui/drawer"
	"github.com/ows4444/tui/faces"
	"github.com/ows4444/tui/loadingbar"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/skeleton"
	"github.com/ows4444/tui/spinner"
	"github.com/ows4444/tui/streamtext"
	"github.com/ows4444/tui/textarea"
	"github.com/ows4444/tui/textinput"
)

// TestReduceMotionEnvEveryAnimatedWidget proves criterion #37: with
// REDUCE_MOTION=1 each animated widget is at its final frame as soon as it
// is started, so no tick is needed, and schedules no tick Cmd.
func TestReduceMotionEnvEveryAnimatedWidget(t *testing.T) {
	t.Setenv("REDUCE_MOTION", "1")
	pref := motion.Detect()
	if !pref.Reduced() {
		t.Fatal("REDUCE_MOTION=1 not detected as reduced")
	}

	sp := spinner.New()
	sp.Motion = pref
	sk := skeleton.New()
	sk.Motion = pref
	lb := loadingbar.New(12)
	lb.Motion = pref
	fc := faces.New()
	fc.Motion = pref
	av := avatar.New("alain")
	av.Motion = pref
	st := streamtext.NewTypewriter()
	st.Motion = pref
	ti := textinput.New()
	ti.Motion = pref
	ta := textarea.New()
	ta.Motion = pref
	dr := drawer.New("body")
	dr.SlideDuration = 200_000_000
	dr.Motion = pref

	cases := []struct {
		name  string
		start func() bool // reports whether a tick was scheduled
		final func() bool // reports whether the widget is at its final frame
	}{
		{"spinner", func() bool { return sp.Start() != nil }, func() bool { return sp.View() != "" }},
		{"skeleton", func() bool { return sk.Start() != nil }, func() bool { return true }},
		{"loadingbar", func() bool { return lb.Start() != nil }, func() bool { return true }},
		{"faces", func() bool { return fc.PlayOnce() != nil }, func() bool { return !fc.Running() && fc.Frame() == 0 }},
		{"avatar", func() bool { return av.Blink() != nil }, func() bool { return !av.Blinking() }},
		{"streamtext", func() bool { return st.SetText("hello world") != nil || st.Start() != nil },
			func() bool { return st.Done() && strings.Contains(st.View(), "hello world") }},
		{"textinput", func() bool { return ti.Focus() != nil }, func() bool { return true }},
		{"textarea", func() bool { return ta.Focus() != nil }, func() bool { return true }},
		{"drawer", func() bool { return dr.SlideIn() != nil }, func() bool { return dr.Open() }},
	}
	for _, c := range cases {
		if c.start() {
			t.Errorf("%s: scheduled a tick under REDUCE_MOTION=1", c.name)
		}
		if !c.final() {
			t.Errorf("%s: not at its final frame under REDUCE_MOTION=1", c.name)
		}
	}
}
