package rangeslider

import (
	"strconv"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
)

func ready() Model {
	m := New(0, 100)
	m.Focus()
	return m
}

func send(m Model, msg tui.Msg) (Model, tui.Msg) {
	m, cmd := m.Update(msg)
	if cmd != nil {
		return m, cmd()
	}
	return m, nil
}

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func TestStartsOnTheWholeSpan(t *testing.T) {
	m := New(10, 50)
	if m.Low() != 10 || m.High() != 50 {
		t.Fatalf("%v to %v", m.Low(), m.High())
	}
}

func TestKeysMoveTheActiveThumbOnly(t *testing.T) {
	m := ready()
	var msg tui.Msg
	m, msg = send(m, key(tui.KeyRight))
	if m.Low() != 1 || m.High() != 100 {
		t.Fatalf("Right moved %v..%v", m.Low(), m.High())
	}
	if c, ok := msg.(ChangedMsg); !ok || c.Low != 1 || c.High != 100 {
		t.Fatalf("msg %+v", msg)
	}
	m, _ = send(m, tui.Key{Type: tui.KeySpace}) // the high thumb
	m, _ = send(m, key(tui.KeyLeft))
	m, _ = send(m, key(tui.KeyPgDown))
	if m.Low() != 1 || m.High() != 89 {
		t.Fatalf("%v..%v", m.Low(), m.High())
	}
	m, _ = send(m, key(tui.KeyHome)) // the high thumb cannot pass the low one
	if m.High() != 1 {
		t.Fatalf("High went to %v; want it held at Low (1)", m.High())
	}
	m, msg = send(m, key(tui.KeyLeft))
	if msg != nil || m.High() != 1 {
		t.Fatalf("a move into the other thumb should do nothing: %v %+v", m.High(), msg)
	}
}

func TestLowThumbCannotPassTheHigh(t *testing.T) {
	m := ready()
	m.SetRange(40, 60)
	m, _ = send(m, key(tui.KeyEnd))
	if m.Low() != 60 || m.High() != 60 {
		t.Fatalf("%v..%v", m.Low(), m.High())
	}
}

func TestSwitchAndDirectPicks(t *testing.T) {
	m := ready()
	m, _ = send(m, tui.Key{Type: tui.KeyRunes, Text: "]"})
	if m.Active() != 1 {
		t.Fatal("] should pick the high thumb")
	}
	m, _ = send(m, tui.Key{Type: tui.KeyRunes, Text: "["})
	if m.Active() != 0 {
		t.Fatal("[ should pick the low thumb")
	}
	m, _ = send(m, tui.Key{Type: tui.KeySpace})
	if m.Active() != 1 {
		t.Fatal("Space should switch")
	}
}

func TestSetRangeSnapsClampsAndSwaps(t *testing.T) {
	m := New(0, 10)
	m.Step = 2
	m.SetRange(9, -5)
	if m.Low() != 0 || m.High() != 10 {
		t.Fatalf("%v..%v", m.Low(), m.High())
	}
	m.SetRange(3, 7)
	if m.Low() != 4 || m.High() != 8 {
		t.Fatalf("not rounded to steps of 2: %v..%v", m.Low(), m.High())
	}
}

func TestMouseTakesTheNearestThumbAndDrags(t *testing.T) {
	m := ready()
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 30, H: 1}
	m.Width = 21 // cells 0..20 map to 0..100, 5 per cell
	m.SetRange(20, 80)
	// A press near the high thumb (cell 16 at x=17 with the bracket cell).
	m, _ = send(m, tui.MouseEvent{X: 15, Y: 0, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Active() != 1 || m.High() >= 80 {
		t.Fatalf("active %d high %v", m.Active(), m.High())
	}
	m, _ = send(m, tui.MouseEvent{X: 12, Y: 0, Button: tui.MouseButtonLeft, Action: tui.MouseActionMotion})
	if m.High() != 55 {
		t.Fatalf("a drag should follow the pointer: High = %v", m.High())
	}
	m, _ = send(m, tui.MouseEvent{X: 12, Y: 0, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease})
	m, _ = send(m, tui.MouseEvent{X: 3, Y: 0, Button: tui.MouseButtonLeft, Action: tui.MouseActionMotion})
	if m.High() != 55 {
		t.Fatal("motion after release moved a thumb")
	}
	// Both thumbs on one cell: a press left of them takes the low one.
	m.SetRange(50, 50)
	m, _ = send(m, tui.MouseEvent{X: 5, Y: 0, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Active() != 0 {
		t.Fatal("a press to the left of stacked thumbs should take the low one")
	}
}

func TestDisabledAndUnfocusedIgnoreKeysAndMouse(t *testing.T) {
	m := ready()
	m.Disabled = true
	m.Mouse = true
	m.Bounds = hittest.Rect{W: 30, H: 1}
	m2, cmd := m.Update(key(tui.KeyRight))
	if cmd != nil || m2.Low() != 0 || m.Bindings() != nil {
		t.Fatal("a disabled slider moved")
	}
	m3, cmd := m.Update(tui.MouseEvent{X: 10, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if cmd != nil || m3.Low() != 0 {
		t.Fatal("a disabled slider took the mouse")
	}
	u := New(0, 10)
	if u2, cmd := u.Update(key(tui.KeyRight)); cmd != nil || u2.Low() != 0 {
		t.Fatal("an unfocused slider moved")
	}
}

func TestViewAndLinearize(t *testing.T) {
	m := New(0, 10)
	m.Width = 11
	m.ShowValue = true
	m.SetRange(2, 8)
	v := ansi.StripANSI(m.View())
	if ansi.Width(v) != 1+11+1+1+len("2 to 8") {
		t.Fatalf("width %d: %q", ansi.Width(v), v)
	}
	if !strings.HasSuffix(v, " 2 to 8") {
		t.Fatalf("no value text: %q", v)
	}
	m.Focus()
	if f := ansi.StripANSI(m.View()); !strings.HasPrefix(f, "<") || !strings.Contains(f, ">") {
		t.Fatalf("focus not shown: %q", f)
	}
	if got := m.Linearize(); got != "range slider, 2 to 8, from 0 to 10" {
		t.Fatalf("linearize %q", got)
	}
	m.Disabled = true
	if !strings.HasSuffix(m.Linearize(), ", unavailable") || !strings.HasPrefix(ansi.StripANSI(m.View()), "(") {
		t.Fatal("disabled not shown")
	}
	same := New(5, 5)
	_ = same.View()
	m.Format = func(v float64) string { return "$" + strconv.Itoa(int(v)) }
	if !strings.Contains(m.Linearize(), "$2 to $8") {
		t.Fatalf("Format ignored: %q", m.Linearize())
	}
	_ = m.LayoutNode()
	_ = m.Tokens()
	_ = m.SetTheme(m.Theme).WithTokens(m.Tokens())
}
