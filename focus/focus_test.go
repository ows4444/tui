package focus

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/input"
)

func tab() tui.Key      { return tui.Key{Type: tui.KeyTab} }
func shiftTab() tui.Key { return tui.Key{Type: tui.KeyTab, Mod: input.ModShift} }

func TestNextAndPrevWrap(t *testing.T) {
	r := New(3)
	var seen []int
	for i := 0; i < 4; i++ {
		seen = append(seen, r.Current())
		r = r.Next()
	}
	if fmt.Sprint(seen) != "[0 1 2 0]" {
		t.Errorf("forward = %v", seen)
	}
	r = New(3)
	seen = nil
	for i := 0; i < 4; i++ {
		r = r.Prev()
		seen = append(seen, r.Current())
	}
	if fmt.Sprint(seen) != "[2 1 0 2]" {
		t.Errorf("backward = %v", seen)
	}
}

func TestUpdateHandlesTabAndShiftTabOnly(t *testing.T) {
	r := New(3)
	r, moved := r.Update(tab())
	if !moved || r.Current() != 1 {
		t.Errorf("Tab: moved=%v current=%d", moved, r.Current())
	}
	r, moved = r.Update(shiftTab())
	if !moved || r.Current() != 0 {
		t.Errorf("Shift+Tab: moved=%v current=%d", moved, r.Current())
	}
	for _, msg := range []tui.Msg{tui.Key{Type: tui.KeyEnter}, tui.Key{Type: tui.KeyRunes, Text: "a"}, tui.ResizeMsg{Width: 1, Height: 1}} {
		if got, moved := r.Update(msg); moved || got.Current() != r.Current() {
			t.Errorf("%#v should not move focus", msg)
		}
	}
}

func TestDisabledItemsAreSkipped(t *testing.T) {
	r := New(5).SetDisabled(1, true).SetDisabled(2, true)
	r = r.Next()
	if r.Current() != 3 {
		t.Errorf("Next skipped to %d, want 3", r.Current())
	}
	r = r.Prev()
	if r.Current() != 0 {
		t.Errorf("Prev skipped to %d, want 0", r.Current())
	}
	r = r.SetDisabled(1, false).Next()
	if r.Current() != 1 {
		t.Errorf("re-enabled item not reachable: %d", r.Current())
	}
}

func TestDisablingTheFocusedItemMovesFocus(t *testing.T) {
	r := New(3).SetDisabled(0, true)
	if r.Current() != 1 {
		t.Errorf("focus stayed on a disabled item: %d", r.Current())
	}
}

func TestNoEnabledItemLeavesFocusAlone(t *testing.T) {
	r := New(2).SetDisabled(0, true).SetDisabled(1, true)
	before := r.Current()
	if r.Next().Current() != before || r.Prev().Current() != before {
		t.Error("focus moved with nothing enabled")
	}
	if got := New(0); got.Current() != -1 || got.Next().Current() != -1 || got.Len() != 0 {
		t.Errorf("empty ring: current=%d", got.Current())
	}
	if got := New(-3); got.Len() != 0 {
		t.Error("negative n should give an empty ring")
	}
}

func TestSetIgnoresBadTargets(t *testing.T) {
	r := New(3).SetDisabled(2, true)
	for _, i := range []int{-1, 3, 2} {
		if r.Set(i).Current() != 0 {
			t.Errorf("Set(%d) moved focus", i)
		}
	}
	if r.Set(1).Current() != 1 || !r.Set(1).Focused(1) || r.Set(1).Focused(0) {
		t.Error("Set(1) failed")
	}
}

func TestValueSemantics(t *testing.T) {
	a := New(3)
	b := a.SetDisabled(1, true)
	c := b.Next() // skips the disabled item
	if a.Enabled(1) != true || b.Enabled(1) != false {
		t.Error("SetDisabled leaked into the original Ring")
	}
	if a.Current() != 0 || b.Current() != 0 || c.Current() != 2 {
		t.Errorf("currents = %d %d %d", a.Current(), b.Current(), c.Current())
	}
	d := b.SetDisabled(1, false) // must not re-enable b's own copy
	if b.Enabled(1) || !d.Enabled(1) {
		t.Error("re-enabling changed the source Ring")
	}
}

// A form with three fields: Tab cycles forward, Shift+Tab back.
func Example() {
	ring := New(3)
	for _, k := range []tui.Key{tab(), tab(), tab(), shiftTab()} {
		ring, _ = ring.Update(k)
		fmt.Print(ring.Current(), " ")
	}
	// Output: 1 2 0 2
}

// TestRawTerminalBytesDriveTheRing feeds what terminals actually send for
// Tab (0x09) and Shift+Tab (ESC [ Z, and the kitty form ESC [ 9 ; 2 u)
// through the real input reader into a Ring.
func TestRawTerminalBytesDriveTheRing(t *testing.T) {
	rd := input.NewReader(strings.NewReader("\t\t\x1b[Z\x1b[9;2u\t"))
	ring := New(4)
	var seen []int
	for i := 0; i < 5; i++ {
		ev, err := rd.ReadEvent()
		if err != nil {
			t.Fatalf("ReadEvent: %v", err)
		}
		var moved bool
		if ring, moved = ring.Update(ev); !moved {
			t.Fatalf("event %d (%#v) did not move focus", i, ev)
		}
		seen = append(seen, ring.Current())
	}
	// Tab, Tab, back-tab, kitty shift+tab, Tab.
	if fmt.Sprint(seen) != "[1 2 1 0 1]" {
		t.Errorf("focus path = %v, want [1 2 1 0 1]", seen)
	}
}
