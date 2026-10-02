package popover

import (
	"fmt"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/focus"
)

// scenario is a screen of three background widgets plus the overlay's own
// two focusable items; log records every Focus and Blur call by widget name.
type scenario struct {
	ring focus.Ring
	log  []string
}

func (s *scenario) fields(names ...string) []focus.Field {
	fs := make([]focus.Field, len(names))
	for i, n := range names {
		n := n
		fs[i] = focus.Field{
			Focus: func() tui.Cmd { s.log = append(s.log, "focus "+n); return nil },
			Blur:  func() { s.log = append(s.log, "blur "+n) },
		}
	}
	return fs
}

var tabKey = tui.Key{Type: tui.KeyTab}

func TestTrapKeepsTabInsideThepopover(t *testing.T) {
	s := &scenario{ring: focus.New(3).Set(1)} // "b" has focus
	bg := s.fields("a", "b", "c")
	in := s.fields("ok", "cancel")
	m := New("c", 1, 1)
	m.Hide()

	// Open: the popover pushes its scope. Its two widgets get the ring.
	s.ring = m.Trap(s.ring, 2)
	if !m.Open() || s.ring.Depth() != 1 {
		t.Fatalf("open=%v depth=%d", m.Open(), s.ring.Depth())
	}
	s.ring, _ = s.ring.Route(tabKey, in...) // ok -> cancel
	for i := 0; i < 6; i++ {
		s.ring, _ = s.ring.Route(tabKey, in...)
		if c := s.ring.Current(); c < 0 || c > 1 {
			t.Fatalf("Tab %d moved focus to %d, outside the popover", i, c)
		}
	}
	for _, l := range s.log {
		for _, n := range []string{"a", "b", "c"} {
			if l == "focus "+n {
				t.Fatalf("background widget focused while the popover was open: %v", s.log)
			}
		}
	}
	_ = bg
}

func TestReleaseReturnsFocusToTheWidgetFocusedBefore(t *testing.T) {
	s := &scenario{ring: focus.New(3).Set(2)} // "c" had focus
	bg := s.fields("a", "b", "c")
	in := s.fields("ok", "cancel")
	m := New("c", 1, 1)
	m.Hide()
	s.ring = m.Trap(s.ring, 2)
	s.ring, _ = s.ring.Route(tabKey, in...)

	s.ring = m.Release(s.ring)
	s.log = nil
	s.ring.Sync(bg...)
	if m.Open() || s.ring.Depth() != 0 || s.ring.Current() != 2 {
		t.Fatalf("open=%v depth=%d current=%d, want closed, 0, 2", m.Open(), s.ring.Depth(), s.ring.Current())
	}
	if got := fmt.Sprint(s.log); got != "[blur a blur b focus c]" {
		t.Errorf("sync log = %s", got)
	}
}

func TestTrapAllowsAtLeastOneItem(t *testing.T) {
	m := New("c", 1, 1)
	if r := m.Trap(focus.New(2), 0); r.Len() != 1 || r.Depth() != 1 {
		t.Errorf("len=%d depth=%d", r.Len(), r.Depth())
	}
}
