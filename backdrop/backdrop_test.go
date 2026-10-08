package backdrop

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func press(x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Action: tui.MouseActionPress, Button: tui.MouseButtonLeft}
}

// The text and every row's width stay; the frame's own styling goes, and
// each row that has text is drawn faint.
func TestRenderDimsTheFrame(t *testing.T) {
	red := ansi.NewStyle().Foreground(ansi.RGB{R: 255}).Bold().Render("alert")
	base := "one " + red + "\n\nthree  "
	m := New()
	out := m.Render(base)
	if got := ansi.StripANSI(out); got != "one alert\n\nthree  " {
		t.Errorf("text = %q", got)
	}
	if strings.Contains(out, "\x1b[1m") || strings.Contains(out, "255;0;0") {
		t.Errorf("the frame kept its styling: %q", out)
	}
	lines := strings.Split(out, "\n")
	if !strings.HasPrefix(lines[0], "\x1b[") || !strings.Contains(lines[0], "2") || lines[1] != "" {
		t.Errorf("rows = %q", lines)
	}
	if lines[0] == "one alert" || lines[2] == "three  " {
		t.Errorf("a row was not dimmed: %q", lines)
	}
	m.Hide()
	if m.Open() || m.Render(base) != base {
		t.Error("a closed backdrop changed the frame")
	}
	m.Show()
	if !m.Open() {
		t.Error("Show left it closed")
	}
	var zero Model
	if zero.Open() {
		t.Error("the zero Model is open")
	}
}

func TestALeftPressOutsideTheHoleIsReported(t *testing.T) {
	m := New()
	m.ID = "dim"
	m.Mouse, m.Bounds = true, hittest.Rect{W: 40, H: 10}
	m.Hole = hittest.Rect{X: 10, Y: 3, W: 20, H: 4}
	next, cmd := m.Update(press(2, 2))
	if cmd == nil || !next.Open() {
		t.Fatal("a press on the backdrop was not reported, or closed it")
	}
	if p, ok := cmd().(PressedMsg); !ok || p.ID != "dim" {
		t.Fatalf("message = %+v", cmd())
	}
	quiet := map[string]tui.Msg{
		"a press in the hole": press(12, 4),
		"a press outside":     press(50, 2),
		"the right button":    tui.MouseEvent{X: 2, Y: 2, Action: tui.MouseActionPress, Button: tui.MouseButtonRight},
		"a release":           tui.MouseEvent{X: 2, Y: 2, Action: tui.MouseActionRelease, Button: tui.MouseButtonLeft},
		"a key":               tui.Key{Type: tui.KeyEsc},
	}
	for name, msg := range quiet {
		if _, cmd := m.Update(msg); cmd != nil {
			t.Errorf("%s was reported", name)
		}
	}
	m.Hide()
	if _, cmd := m.Update(press(2, 2)); cmd != nil {
		t.Error("a closed backdrop reported a press")
	}
	m.Show()
	m.Mouse = false
	if _, cmd := m.Update(press(2, 2)); cmd != nil {
		t.Error("a press was reported with Mouse off")
	}
}

var _ tui.Linearizer = Model{}

func TestThemeTokensLayoutNodeAndLinearize(t *testing.T) {
	m := New().SetTheme(theme.LightTheme())
	if m.Theme.Primary != theme.LightTheme().Primary {
		t.Error("SetTheme did not apply the theme")
	}
	red := ansi.RGB{R: 255}
	m = m.WithTokens(theme.Tokens{Muted: red})
	if m.Tokens().Muted != red || !strings.Contains(m.Render("x"), "255;0;0") {
		t.Error("WithTokens did not override the muted colour")
	}
	if s := m.LayoutNode().Measure(layout.Constraints{MaxW: layout.Unbounded, MaxH: layout.Unbounded}); s.W != 0 {
		t.Fatalf("Measure = %+v", s)
	}
	if m.Linearize() != "" {
		t.Error("a backdrop has something to read out")
	}
}
