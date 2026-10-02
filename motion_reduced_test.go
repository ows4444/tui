package tui_test

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/faces"
	"github.com/ows4444/tui/loadingbar"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/skeleton"
	"github.com/ows4444/tui/spinner"
	"github.com/ows4444/tui/streamtext"
	"github.com/ows4444/tui/textarea"
	"github.com/ows4444/tui/textinput"
)

// Under motion.Reduced each animated widget schedules no tick and renders a
// static state; under the zero value (motion.Normal) it still ticks.

func TestSpinnerReducedSchedulesNoTick(t *testing.T) {
	m := spinner.New()
	still := m.View()
	m.Motion = motion.Reduced
	if cmd := m.Start(); cmd != nil {
		t.Error("Start scheduled a tick under reduced motion")
	}
	if m.View() != still {
		t.Error("a reduced spinner should stay on its first frame")
	}
	n := spinner.New()
	if cmd := n.Start(); cmd == nil {
		t.Error("Start scheduled no tick under normal motion")
	}
}

func TestSkeletonReducedSchedulesNoTick(t *testing.T) {
	m := skeleton.New()
	m.Width, m.Lines = 10, 2
	m.Motion = motion.Reduced
	if cmd := m.Start(); cmd != nil {
		t.Error("Start scheduled a tick under reduced motion")
	}
	if m.View() == "" {
		t.Error("a reduced skeleton should still draw its placeholder")
	}
	n := skeleton.New()
	if cmd := n.Start(); cmd == nil {
		t.Error("Start scheduled no tick under normal motion")
	}
}

func TestLoadingBarReducedSchedulesNoTick(t *testing.T) {
	m := loadingbar.New(12)
	m.Motion = motion.Reduced
	if cmd := m.Start(); cmd != nil {
		t.Error("Start scheduled a tick under reduced motion")
	}
	if m.View() == "" {
		t.Error("a reduced loading bar should still draw its track")
	}
	n := loadingbar.New(12)
	if cmd := n.Start(); cmd == nil {
		t.Error("Start scheduled no tick under normal motion")
	}
}

func TestFacesReducedSchedulesNoTick(t *testing.T) {
	m := faces.New()
	m.Motion = motion.Reduced
	if cmd := m.Start(); cmd != nil || m.Running() {
		t.Error("Start under reduced motion should schedule nothing and not run")
	}
	if cmd := m.PlayOnce(); cmd != nil || m.Running() || m.Frame() != 0 {
		t.Error("PlayOnce under reduced motion should rest on the first frame")
	}
	if m.View() == "" {
		t.Error("a reduced face should still draw its first frame")
	}
	n := faces.New()
	if cmd := n.Start(); cmd == nil {
		t.Error("Start scheduled no tick under normal motion")
	}
}

func TestStreamTextReducedRevealsAtOnce(t *testing.T) {
	m := streamtext.NewTypewriter()
	m.Motion = motion.Reduced
	cmd := m.SetText("hello world")
	if cmd != nil {
		t.Error("SetText scheduled a tick under reduced motion")
	}
	if start := m.Start(); start != nil {
		t.Error("Start scheduled a tick under reduced motion")
	}
	if !m.Done() || !strings.Contains(m.View(), "hello world") {
		t.Errorf("text should be fully shown at once, got %q", m.View())
	}
	n := streamtext.NewTypewriter()
	n.SetText("hello")
	if cmd := n.Start(); cmd == nil {
		t.Error("Start scheduled no tick under normal motion")
	}
}

func TestCursorBlinkReducedSchedulesNoTick(t *testing.T) {
	in := textinput.New()
	in.Motion = motion.Reduced
	if cmd := in.Focus(); cmd != nil {
		t.Error("textinput.Focus scheduled a blink under reduced motion")
	}
	ta := textarea.New()
	ta.Motion = motion.Reduced
	if cmd := ta.Focus(); cmd != nil {
		t.Error("textarea.Focus scheduled a blink under reduced motion")
	}
	in2, ta2 := textinput.New(), textarea.New()
	if in2.Focus() == nil || ta2.Focus() == nil {
		t.Error("Focus scheduled no blink under normal motion")
	}
}
